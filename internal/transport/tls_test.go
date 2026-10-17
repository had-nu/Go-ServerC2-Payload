package transport

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func leafOf(t *testing.T, cert tls.Certificate) *x509.Certificate {
	t.Helper()
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse leaf: %v", err)
	}
	return leaf
}

func TestLoadOrCreateCertIsReused(t *testing.T) {
	dir := t.TempDir()

	first, err := LoadOrCreateCert(dir, nil)
	if err != nil {
		t.Fatalf("LoadOrCreateCert: %v", err)
	}
	certPath, keyPath := CertPaths(dir)

	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("key mode = %o, want 600", perm)
	}
	if _, err := os.Stat(certPath); err != nil {
		t.Fatalf("stat cert: %v", err)
	}

	second, err := LoadOrCreateCert(dir, nil)
	if err != nil {
		t.Fatalf("LoadOrCreateCert again: %v", err)
	}
	if string(first.Certificate[0]) != string(second.Certificate[0]) {
		t.Fatal("LoadOrCreateCert must reuse the stored certificate")
	}
}

func TestLoadOrCreateCertCoversLocalhost(t *testing.T) {
	leaf := leafOf(t, mustCert(t, t.TempDir()))

	for _, want := range []string{"127.0.0.1", "::1"} {
		if !containsIP(leaf.IPAddresses, want) {
			t.Errorf("certificate does not cover %s", want)
		}
	}
	if !containsName(leaf.DNSNames, "localhost") {
		t.Errorf("certificate DNS names = %v, want localhost", leaf.DNSNames)
	}

	now := time.Now()
	if now.Before(leaf.NotBefore) || now.After(leaf.NotAfter) {
		t.Error("certificate is not valid now")
	}
	if leaf.NotAfter.Sub(leaf.NotBefore) < CertValidity-time.Hour {
		t.Errorf("validity window = %v, want about %v", leaf.NotAfter.Sub(leaf.NotBefore), CertValidity)
	}
}

func TestLoadOrCreateCertAcceptsExplicitHosts(t *testing.T) {
	leaf := leafOf(t, mustCert(t, t.TempDir(), "lab.internal", "10.0.0.5"))
	if !containsName(leaf.DNSNames, "lab.internal") {
		t.Errorf("DNS names = %v, want lab.internal", leaf.DNSNames)
	}
	if !containsIP(leaf.IPAddresses, "10.0.0.5") {
		t.Errorf("IP addresses = %v, want 10.0.0.5", leaf.IPAddresses)
	}
}

func TestLoadOrCreateCertRejectsExpired(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := CertPaths(dir)

	expiredPEM, keyPEM := expiredPair(t)
	if err := os.WriteFile(certPath, expiredPEM, 0o644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	if _, err := LoadOrCreateCert(dir, nil); err == nil {
		t.Fatal("LoadOrCreateCert must reject an expired certificate")
	}
}

func TestCertPaths(t *testing.T) {
	cert, key := CertPaths(filepath.Join("a", "b"))
	if filepath.Base(cert) != "team-server.crt" || filepath.Base(key) != "team-server.key" {
		t.Fatalf("CertPaths = %q, %q", cert, key)
	}
}

func mustCert(t *testing.T, dir string, hosts ...string) tls.Certificate {
	t.Helper()
	cert, err := LoadOrCreateCert(dir, hosts)
	if err != nil {
		t.Fatalf("LoadOrCreateCert: %v", err)
	}
	return cert
}

func mustPair(t *testing.T) tls.Certificate {
	t.Helper()
	return mustCert(t, t.TempDir())
}

func containsIP(list []net.IP, want string) bool {
	for _, ip := range list {
		if ip.Equal(net.ParseIP(want)) {
			return true
		}
	}
	return false
}

func containsName(list []string, want string) bool {
	for _, n := range list {
		if n == want {
			return true
		}
	}
	return false
}

func expiredPair(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "expired"},
		NotBefore:             time.Now().Add(-48 * time.Hour),
		NotAfter:              time.Now().Add(-24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}
