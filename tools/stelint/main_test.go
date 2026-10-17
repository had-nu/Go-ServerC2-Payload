package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// write puts a Markdown document in a temporary file and returns its path.
func write(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func check(t *testing.T, content string) []finding {
	t.Helper()
	path := write(t, "doc.md", content)
	found, err := checkFile(path)
	if err != nil {
		t.Fatalf("checkFile: %v", err)
	}
	return found
}

func rules(found []finding) []string {
	out := make([]string, 0, len(found))
	for _, f := range found {
		out = append(out, f.rule)
	}
	return out
}

func hasRule(found []finding, rule string) bool {
	for _, f := range found {
		if f.rule == rule {
			return true
		}
	}
	return false
}

func TestCleanDocumentHasNoFindings(t *testing.T) {
	doc := `# Title

The Beacon checks in every 8 to 12 seconds. It sends the identifier in one
header. The Team Server stores the result.

Close the valve. Then start the system.

Make sure that the room is clean.
`
	if found := check(t, doc); len(found) != 0 {
		for _, f := range found {
			t.Errorf("unexpected finding: %s", f)
		}
	}
}

func TestDetectsSentenceTooLong(t *testing.T) {
	doc := "This is a sentence that contains far too many individual words which " +
		"will definitely push the total past twenty five words for sure and break the limit.\n"
	if got := check(t, doc); !hasRule(got, "6.3") {
		t.Fatalf("want a rule 6.3 finding, got %v", rules(got))
	}
}

func TestDetectsSemicolon(t *testing.T) {
	if got := check(t, "The system crashes; the operator restarts it.\n"); !hasRule(got, "8.1") {
		t.Fatalf("want a rule 8.1 finding, got %v", rules(got))
	}
}

func TestDetectsUnapprovedModals(t *testing.T) {
	doc := "The system may crash. The report should be sent. The build could fail.\n"
	got := check(t, doc)
	if len(got) != 3 {
		t.Fatalf("want 3 findings, got %v", rules(got))
	}
	if !hasRule(got, "1.1") {
		t.Fatalf("want a rule 1.1 finding, got %v", rules(got))
	}
}

func TestDetectsContractions(t *testing.T) {
	got := check(t, "You mustn't do that. It doesn't work.\n")
	if len(got) != 2 {
		t.Fatalf("want 2 findings, got %v", rules(got))
	}
}

func TestDoesNotReportPossessives(t *testing.T) {
	if got := check(t, "The team's secret is in someone's system.\n"); len(got) != 0 {
		t.Fatalf("want no findings, got %v", rules(got))
	}
}

func TestDetectsAboutAsApproximate(t *testing.T) {
	if got := check(t, "Drain about 2 liters of fuel from the tank.\n"); !hasRule(got, "1.3") {
		t.Fatalf("want a rule 1.3 finding, got %v", rules(got))
	}
}

func TestDoesNotReportAboutInTheApprovedMeaning(t *testing.T) {
	doc := "For data about the location of circuit breakers, refer to the list.\n"
	if got := check(t, doc); len(got) != 0 {
		t.Fatalf("want no findings, got %v", rules(got))
	}
}

func TestDetectsNounOnlyVerb(t *testing.T) {
	if got := check(t, "Check the valve for a leak now.\n"); !hasRule(got, "1.2") {
		t.Fatalf("want a rule 1.2 finding, got %v", rules(got))
	}
}

func TestDetectsTooManySentencesInAParagraph(t *testing.T) {
	doc := "One. Two. Three. Four. Five. Six. Seven.\n"
	if got := check(t, doc); !hasRule(got, "6.6") {
		t.Fatalf("want a rule 6.6 finding, got %v", rules(got))
	}
}

func TestAllowDirectiveSuppressesTheRule(t *testing.T) {
	doc := "| Use | Do not use |\n|---|---|\n" +
		"| `CHECK (n)` | `CHECK (v)` <!-- stelint:allow 1.2 -->\n"
	if got := check(t, doc); len(got) != 0 {
		t.Fatalf("the directive must suppress the finding, got %v", got)
	}

	// The same line without the directive must report the finding.
	doc2 := "| Use | Do not use |\n|---|---|\n" +
		"| `CHECK (n)` | `CHECK (v)` |\n"
	if got := check(t, doc2); !hasRule(got, "1.2") {
		t.Fatalf("want a rule 1.2 finding without the directive, got %v", rules(got))
	}
}

func TestAllowDirectiveOnlySuppressesTheNamedRule(t *testing.T) {
	doc := "Check the valve; the operator waits.\n" // rule 8.1 and rule 1.2
	if got := check(t, doc+"<!-- stelint:allow 1.2 -->\n"); !hasRule(got, "8.1") {
		t.Fatalf("rule 8.1 must still be reported, got %v", rules(got))
	}
}

func TestCodeFenceIsNotChecked(t *testing.T) {
	doc := "Run the program.\n\n```text\nThe system may crash; it might also hang.\n```\n"
	if got := check(t, doc); len(got) != 0 {
		t.Fatalf("code inside a fence must not be checked, got %v", got)
	}
}

func TestCheckAndTestInsideBackticksAreNotSentences(t *testing.T) {
	doc := "The flag `-check` and the value `-test` set the options.\n"
	if got := check(t, doc); len(got) != 0 {
		t.Fatalf("inline code must not be read as a sentence, got %v", got)
	}
}

func TestSplitSentences(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"One. Two.", 2},
		{"Read e.g. the file.", 1},
		{"The value is 1.5 units.", 1},
		{"No end of sentence", 1},
		{"", 0},
		{"One! Two? Three.", 3},
	}
	for _, tc := range tests {
		if got := SplitSentences(tc.in); len(got) != tc.want {
			t.Errorf("SplitSentences(%q) = %d sentences, want %d: %q", tc.in, len(got), tc.want, got)
		}
	}
}

func TestFindingStringHasFileAndRule(t *testing.T) {
	f := finding{file: "a.md", line: 3, rule: "6.3", severity: "error", message: "too long"}
	s := f.String()
	for _, want := range []string{"a.md", "3", "6.3", "error", "too long"} {
		if !strings.Contains(s, want) {
			t.Errorf("the finding %q does not contain %q", s, want)
		}
	}
}
