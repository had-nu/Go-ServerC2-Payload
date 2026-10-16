package transport

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSecret(t *testing.T) {
	a, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	if len(a) != SecretLen*2 {
		t.Fatalf("secret length = %d, want %d", len(a), SecretLen*2)
	}
	if err := ValidateSecret(a); err != nil {
		t.Fatalf("ValidateSecret(generated) = %v", err)
	}
	b, err := GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret: %v", err)
	}
	if a == b {
		t.Fatal("two generated secrets must differ")
	}
}

func TestValidateSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"ok", strings.Repeat("a", SecretLen), false},
		{"empty", "", true},
		{"too short", strings.Repeat("a", SecretLen-1), true},
		{"non ascii", strings.Repeat("é", SecretLen), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSecret(tc.secret)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateSecret(%q) = %v, wantErr %v", tc.secret, err, tc.wantErr)
			}
		})
	}
}

func TestLoadSecretCreatesAndReuses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "c2.key")

	first, err := LoadSecret(path)
	if err != nil {
		t.Fatalf("LoadSecret: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("secret file mode = %o, want 600", perm)
	}

	second, err := LoadSecret(path)
	if err != nil {
		t.Fatalf("LoadSecret again: %v", err)
	}
	if first != second {
		t.Fatal("LoadSecret must reuse the stored secret")
	}
}

func TestLoadSecretRejectsShortFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c2.key")
	if err := os.WriteFile(path, []byte("short\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadSecret(path); err == nil {
		t.Fatal("LoadSecret must reject a short secret file")
	}
}

func req(t *testing.T, auth string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/checkin", nil)
	if auth != "" {
		r.Header.Set("Authorization", auth)
	}
	return r
}

func TestCheckSecret(t *testing.T) {
	secret := strings.Repeat("s", SecretLen)

	if err := CheckSecret(req(t, "Bearer "+secret), secret); err != nil {
		t.Fatalf("valid secret rejected: %v", err)
	}

	bad := []struct {
		name string
		auth string
	}{
		{"missing header", ""},
		{"no scheme", secret},
		{"wrong scheme", "Basic " + secret},
		{"scheme only", "Bearer "},
		{"wrong secret", "Bearer " + strings.Repeat("x", SecretLen)},
		{"prefix of the secret", "Bearer " + secret[:16]},
		{"extra whitespace", "Bearer  " + secret},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckSecret(req(t, tc.auth), secret)
			if !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("CheckSecret = %v, want ErrUnauthorized", err)
			}
		})
	}
}

// The old code used strings.Split(RemoteAddr, ":")[0].
func TestRemoteIPHandlesIPv6(t *testing.T) {
	tests := []struct {
		remote string
		want   string
	}{
		{"192.0.2.10:54321", "192.0.2.10"},
		{"[2001:db8::1]:54321", "2001:db8::1"},
		{"[::1]:443", "::1"},
		{"garbage", ""},
	}
	for _, tc := range tests {
		t.Run(tc.remote, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/checkin", nil)
			r.RemoteAddr = tc.remote
			got := RemoteIP(r)
			if tc.want == "" {
				if got.IsValid() {
					t.Fatalf("RemoteIP(%q) = %v, want an invalid address", tc.remote, got)
				}
				return
			}
			want := netip.MustParseAddr(tc.want)
			if got != want {
				t.Fatalf("RemoteIP(%q) = %v, want %v", tc.remote, got, want)
			}
		})
	}
}

func TestNewLoggerWrites(t *testing.T) {
	var buf strings.Builder
	NewLogger(&buf).Info("test", "key", "value")
	if !strings.Contains(buf.String(), "test") {
		t.Fatalf("logger wrote %q, want the message", buf.String())
	}
}

var _ = io.Discard
