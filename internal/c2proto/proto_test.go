package c2proto

import (
	"strings"
	"testing"
)

func TestValidateAgentID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"uuid with dashes", "e51ec4c6-a873-4f01-9a2b-0c1d2e3f4a5b", false},
		{"exactly the minimum", strings.Repeat("a", MinAgentIDLen), false},
		{"exactly the maximum", strings.Repeat("a", MaxAgentIDLen), false},
		{"fallback prefix", "fallback-1709566971", false},
		{"too short", "ab", true},
		{"one below minimum", strings.Repeat("a", MinAgentIDLen-1), true},
		{"too long", strings.Repeat("a", MaxAgentIDLen+1), true},
		{"empty", "", true},
		{"space", "abcd efgh", true},
		{"newline", "abcd\nefgh", true},
		{"slash", "abcd/efgh", true},
		{"ansi escape", "abcd\x1b[0mefgh", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAgentID(tc.id)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateAgentID(%q) = %v, wantErr %v", tc.id, err, tc.wantErr)
			}
		})
	}
}

// The Team Server console used to slice ID[:8] and panic on a short ID.
func TestShortIDNeverPanics(t *testing.T) {
	for _, id := range []string{"", "a", "ab", "abcdefg", "abcdefgh", "abcdefghi", strings.Repeat("x", 200)} {
		got := ShortID(id)
		if len(got) > 8 {
			t.Errorf("ShortID(%q) length = %d, want <= 8", id, len(got))
		}
		if id != "" && !strings.HasPrefix(id, got) {
			t.Errorf("ShortID(%q) = %q, want a prefix of the input", id, got)
		}
	}
}

func TestValidateCommand(t *testing.T) {
	if err := ValidateCommand(strings.Repeat("a", MaxCommandLen)); err != nil {
		t.Fatalf("command at the limit: %v", err)
	}
	if err := ValidateCommand(strings.Repeat("a", MaxCommandLen+1)); err == nil {
		t.Fatal("command over the limit: want error")
	}
}

func TestValidateTaskID(t *testing.T) {
	if err := ValidateTaskID(strings.Repeat("1", TaskIDLen)); err != nil {
		t.Fatalf("task ID at the limit: %v", err)
	}
	for _, bad := range []string{"", "1", strings.Repeat("1", TaskIDLen-1), strings.Repeat("1", TaskIDLen+1)} {
		if err := ValidateTaskID(bad); err == nil {
			t.Errorf("ValidateTaskID(%q) = nil, want error", bad)
		}
	}
}

func TestSanitizeHeader(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"plain", "ubuntu-target", 32, "ubuntu-target"},
		{"newline becomes space", "abc\ndef", 32, "abc def"},
		{"null becomes space", "abc\x00def", 32, "abc def"},
		{"escape becomes space", "abc\x1b[0mdef", 32, "abc [0mdef"},
		{"truncates", "abcdefghij", 4, "abcd..."},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SanitizeHeader(tc.in, tc.max); got != tc.want {
				t.Fatalf("SanitizeHeader(%q, %d) = %q, want %q", tc.in, tc.max, got, tc.want)
			}
		})
	}
}

func TestResultIsError(t *testing.T) {
	if (Result{}).IsError() {
		t.Error("empty Result reports an error")
	}
	if !(Result{Error: "boom"}).IsError() {
		t.Error("Result with Error does not report an error")
	}
}
