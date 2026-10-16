// Package transport holds the HTTP layer of the Team Server: authentication,
// shared-secret handling, and the server limits.
package transport

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"time"

	"github.com/had-nu/Go-ServerC2-Payload/internal/c2proto"
)

// Server limits. Each one closes a weakness that the first version of this
// project had.
const (
	ReadHeaderTimeout = 10 * time.Second
	ReadTimeout       = 30 * time.Second
	WriteTimeout      = 30 * time.Second
	IdleTimeout       = 60 * time.Second
	ShutdownTimeout   = 5 * time.Second
	MaxHeaderBytes    = 16 << 10
	MaxRequestBody    = 1 << 20 // 1 MiB
	SecretLen         = 32
)

// Errors returned by the transport helpers.
var (
	ErrNoSecret       = errors.New("the shared secret is empty")
	ErrShortSecret    = fmt.Errorf("the shared secret has fewer than %d bytes", SecretLen)
	ErrUnauthorized   = errors.New("the request does not carry a valid shared secret")
	ErrSecretNotASCII = errors.New("the shared secret contains a non-ASCII byte")
)

// GenerateSecret returns a new random shared secret, encoded in hex.
func GenerateSecret() (string, error) {
	buf := make([]byte, SecretLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// LoadSecret reads a shared secret from a file, or makes one and writes it when
// the file does not exist. The file is written for the owner only.
func LoadSecret(path string) (string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- the path comes from the operator flag -secret-file or from the -state directory, not from the network
	switch {
	case err == nil:
		secret := trimNewline(string(data))
		if err := ValidateSecret(secret); err != nil {
			return "", fmt.Errorf("%s: %w", path, err)
		}
		return secret, nil
	case !os.IsNotExist(err):
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	secret, err := GenerateSecret()
	if err != nil {
		return "", err
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", fmt.Errorf("create %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, []byte(secret+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return secret, nil
}

// ValidateSecret checks a shared secret.
func ValidateSecret(secret string) error {
	if secret == "" {
		return ErrNoSecret
	}
	if len(secret) < SecretLen {
		return ErrShortSecret
	}
	for i := range len(secret) {
		if secret[i] >= 0x80 {
			return fmt.Errorf("%w: byte %d", ErrSecretNotASCII, i)
		}
	}
	return nil
}

func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

// CheckSecret compares the shared secret that a request carries with the one the
// server holds. The comparison takes the same time for every input.
func CheckSecret(r *http.Request, secret string) error {
	const prefix = "Bearer "

	got := r.Header.Get(c2proto.HeaderAuth)
	if !hasBearerPrefix(got) {
		return ErrUnauthorized
	}
	got = got[len(prefix):]

	wantSum := sha256.Sum256([]byte(secret))
	gotSum := sha256.Sum256([]byte(got))
	if subtle.ConstantTimeCompare(gotSum[:], wantSum[:]) != 1 {
		return ErrUnauthorized
	}
	return nil
}

func hasBearerPrefix(v string) bool {
	const prefix = "Bearer "
	return len(v) > len(prefix) && v[:len(prefix)] == prefix
}

// RemoteIP returns the address of the peer, without the port.
//
// The old code used strings.Split on a colon, which broke on IPv6 and returned
// "[2001" for the address 2001:db8::1.
func RemoteIP(r *http.Request) netip.Addr {
	addr, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return netip.Addr{}
	}
	return addr.Addr().Unmap()
}

// NewLogger returns the logger that the Team Server uses.
func NewLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
