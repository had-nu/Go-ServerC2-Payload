package ste

import (
	"slices"
	"testing"
)

func TestWordCountAppliesTheCountingRules(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"plain", "Close the valve.", 3},
		// Rule 8.6: a number with a unit is one word.
		{"number with unit", "Tighten to 50 psi.", 3},
		{"bare number", "The value is 42.", 4},
		// Rule 8.6: an alphanumeric identifier is one word.
		{"identifier", "The agent e51ec4c6 is alive.", 5},
		{"seconds as a unit", "Wait 10 seconds.", 2},
		{"kilopascal", "Set the pressure to 800 kPa.", 5},
		// Rule 8.6: an abbreviation is one word.
		{"abbreviation", "Read the addr field.", 4},
		// Rule 8.5: text in parentheses counts as one word.
		{"parenthetical", "Install the screw (2) now.", 5},
		{"two parentheticals", "Install the screw (2) and the nut (14).", 8},
		// Rule 8.7: a hyphenated word is one word.
		{"hyphenated", "Open the high-pressure valve.", 4},
		// Quoted text counts as one word.
		{"quoted", "The comm value is \"[kworker/u4:2]\" now.", 6},
		// Markdown code is not part of the sentence.
		{"code span", "Set the flag `-server` first.", 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := WordCount(tc.in); got != tc.want {
				t.Fatalf("WordCount(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestContractionsIgnoresPossessives(t *testing.T) {
	// A possessive is not a contraction. An earlier pattern reported both.
	if got := Contractions("someone else's system"); len(got) != 0 {
		t.Fatalf("Contractions(possessive) = %v, want none", got)
	}
	if got := Contractions("the team's secret"); len(got) != 0 {
		t.Fatalf("Contractions(possessive) = %v, want none", got)
	}
}

func TestContractionsFindsRealContractions(t *testing.T) {
	for _, in := range []string{
		"you mustn't do that",
		"it doesn't work",
		"we aren't ready",
		"they won't come",
	} {
		if len(Contractions(in)) == 0 {
			t.Errorf("Contractions(%q) = none, want a finding", in)
		}
	}
}

func TestSemicolons(t *testing.T) {
	if n := len(Semicolons("no semicolon here")); n != 0 {
		t.Fatalf("Semicolons = %d, want 0", n)
	}
	if n := len(Semicolons("one; two; three")); n != 2 {
		t.Fatalf("Semicolons = %d, want 2", n)
	}
}

func TestUnapprovedModalUses(t *testing.T) {
	got := UnapprovedModalUses("the system may fail and it should stop")
	if len(got) != 2 {
		t.Fatalf("got %v, want 2 findings", got)
	}
	if got[0] != "MAY -> CAN" || got[1] != "SHOULD -> MUST" {
		t.Fatalf("got %v", got)
	}
	for _, approved := range []string{"CAN", "MUST", "WILL", "CANNOT", "BE", "HAVE"} {
		if n := len(UnapprovedModalUses("the system uses " + approved)); n != 0 {
			t.Errorf("%s must not be reported: %v", approved, UnapprovedModalUses("the system uses "+approved))
		}
	}
}

// ABOUT is approved only as a preposition with the meaning "concerned with".
func TestAboutAsApproximate(t *testing.T) {
	bad := []string{
		"Drain about 2 liters of fuel from the tank.",
		"The timeout is about 30 seconds.",
		"The interval is about one minute.",
	}
	for _, in := range bad {
		if len(AboutAsApproximate(in)) == 0 {
			t.Errorf("AboutAsApproximate(%q) = none, want a finding", in)
		}
	}

	good := []string{
		"For data about the location of circuit breakers, refer to the list.",
		"Turn the shaft around its axis.",
		"Read the section about the valve.",
	}
	for _, in := range good {
		if n := len(AboutAsApproximate(in)); n != 0 {
			t.Errorf("AboutAsApproximate(%q) = %d, want 0", in, n)
		}
	}
}

func TestIsApprovedVerb(t *testing.T) {
	approved := []string{"CLOSE", "EXAMINE", "START", "MAKE", "USE", "IDENTIFY", "MUST"}
	for _, v := range approved {
		if !IsApprovedVerb(v) {
			t.Errorf("%s must be an approved verb", v)
		}
	}
	// The first version of this project listed these as approved verbs. The
	// dictionary approves none of them as a verb.
	notApproved := []string{"VERIFY", "CLEAR", "DELETE", "DOWNLOAD", "CHECK", "TEST", "IMPROVE", "PROCESS", "DEPEND"}
	for _, v := range notApproved {
		if IsApprovedVerb(v) {
			t.Errorf("%s must not be an approved verb", v)
		}
	}
}

// The list has 208 entries and the letters J, N, Q, V, X, Y, Z have none.
func TestApprovedVerbList(t *testing.T) {
	if len(ApprovedVerbs) != 208 {
		t.Fatalf("the list has %d verbs, want 208", len(ApprovedVerbs))
	}
	for _, v := range []string{"CHECK", "TEST", "VERIFY", "CLEAR", "DELETE", "DOWNLOAD", "IMPROVE", "PROCESS"} {
		if ApprovedVerbs[v] {
			t.Errorf("%s must not be in the approved verb list", v)
		}
	}
}

func TestFirstVerb(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Close the valve.", "CLOSE"},
		{"The server returns a task.", "THE"},
		{"MAKE SURE that the room is clean.", "MAKE SURE"},
		{"", ""},
		{"- Install the screw.", "INSTALL"},
	}
	for _, tc := range tests {
		if got := FirstVerb(tc.in); got != tc.want {
			t.Errorf("FirstVerb(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMaxWords(t *testing.T) {
	if MaxWords(ModeProcedure) != 20 {
		t.Errorf("the procedure limit is %d, want 20", MaxWords(ModeProcedure))
	}
	if MaxWords(ModeDescriptive) != 25 {
		t.Errorf("the descriptive limit is %d, want 25", MaxWords(ModeDescriptive))
	}
	if MaxWords(ModeNote) != 25 {
		t.Errorf("the note limit is %d, want 25", MaxWords(ModeNote))
	}
}

func TestStripMarkdown(t *testing.T) {
	tests := []struct{ in, want string }{
		{"a `code` word", "a (C) word"},
		{"see [the docs](docs/A.md) now", "see the docs now"},
		{"**bold** text", "bold text"},
	}
	for _, tc := range tests {
		if got := StripMarkdown(tc.in); got != tc.want {
			t.Errorf("StripMarkdown(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIngestiveFormsAreFoundForReview(t *testing.T) {
	got := IngestiveForms("The cooling fan stops. The beacon is running.")
	if len(got) != 2 {
		t.Fatalf("IngestiveForms = %v, want 2 items for review", got)
	}
	if !slices.Contains(got, "cooling") || !slices.Contains(got, "running") {
		t.Fatalf("IngestiveForms = %v, want cooling and running", got)
	}
}
