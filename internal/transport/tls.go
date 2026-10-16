package transport

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// CertValidity is how long a generated certificate stays valid.
const CertValidity = 365 * 24 * time.Hour

// Errors returned by the certificate helpers.
var (
	ErrNoCertPath  = errors.New("the certificate path is empty")
	ErrCertExpired = errors.New("the certificate is not valid yet or has expired")
)

// CertPaths returns the certificate and key paths for a state directory.
func CertPaths(dir string) (certPath, keyPath string) {
	return filepath.Join(dir, "team-server.crt"), filepath.Join(dir, "team-server.key")
}

// LoadOrCreateCert reads a certificate from dir. When the files do not exist,
// it makes a self-signed certificate and writes it.
//
// The certificate has a single purpose: it stops an observer on the network from
// reading the traffic. A lab operator must pin the certificate file, because a
// self-signed certificate proves nothing about the server identity.
func LoadOrCreateCert(dir string, hosts []string) (tls.Certificate, error) {
	certPath, keyPath := CertPaths(dir)

	if cert, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
		if leaf, perr := x509.ParseCertificate(cert.Certificate[0]); perr == nil {
			if now := time.Now(); now.Before(leaf.NotBefore) || now.After(leaf.NotAfter) {
				return tls.Certificate{}, fmt.Errorf("%s: %w", certPath, ErrCertExpired)
			}
			return cert, nil
		}
	}

	certPEM, keyPEM, err := makeSelfSigned(hosts)
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := writePair(dir, certPath, certPEM, keyPath, keyPEM); err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(certPEM, keyPEM)
}

// NewClientTLS returns the TLS configuration for the Beacon. When caFile is
// empty, the client does not verify the server certificate. That is acceptable
// in a closed lab network and is not acceptable anywhere else.
func NewClientTLS(caFile string) (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if caFile == "" {
		cfg.InsecureSkipVerify = true // #nosec G402 -- a lab has no certificate authority. This is a default, not a requirement.
		return cfg, nil
	}
	pemBytes, err := os.ReadFile(caFile) // #nosec G304 -- the path comes from the operator flag -ca, not from the network
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", caFile, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		return nil, fmt.Errorf("%s: no certificate found", caFile)
	}
	cfg.RootCAs = pool
	return cfg, nil
}

func writePair(dir, certPath string, certPEM []byte, keyPath string, keyPEM []byte) error {
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", certPath, err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", keyPath, err)
	}
	return nil
}

func makeSelfSigned(hosts []string) (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}
	serialMax := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialMax)
	if err != nil {
		return nil, nil, fmt.Errorf("generate serial: %w", err)
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "Go-ServerC2-Payload Team Server", Organization: []string{"Educational lab use"}},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(CertValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	if len(hosts) == 0 {
		hosts = []string{"localhost", "127.0.0.1", "::1"}
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
			continue
		}
		tmpl.DNSNames = append(tmpl.DNSNames, h)
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("create certificate: %w", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal key: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, nil
}
