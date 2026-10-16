package ste

import (
	"regexp"
	"strings"
	"unicode"
)

// WordCount counts the words of a sentence with the rules of the standard.
//
// The rules that change the count are:
//   - rule 8.5: text in parentheses counts as one word
//   - rule 8.6: a number, a number with a unit, an abbreviation, an
//     alphanumeric identifier, quoted text, a title, and a proper name count
//     as one word
//   - rule 8.7: a hyphenated word counts as one word
func WordCount(sentence string) int {
	if sentence == "" {
		return 0
	}
	s := collapseParentheticals(sentence)
	s = stripQuoteBlocks(s)

	fields := strings.FieldsFunc(s, func(r rune) bool { return unicode.IsSpace(r) })
	count := 0
	for i := 0; i < len(fields); i++ {
		f := trimPunctuation(fields[i])
		if f == "" {
			continue
		}
		count++
		// Rule 8.6: a number with a unit is one word. Merge the unit into the
		// number so that "50 psi" counts once.
		if isNumber(f) && i+1 < len(fields) && isUnit(trimPunctuation(fields[i+1])) {
			i++
		}
	}
	return count
}

var numberPattern = regexp.MustCompile(`^[-+]?\d+([.,]\d+)*$`)

func isNumber(s string) bool { return numberPattern.MatchString(s) }

// units is the list of unit symbols that the checker merges with a number. The
// standard does not give a list of units. The checker uses this list and the
// writer reads the sentence by hand when a unit is not in the list.
var units = map[string]bool{
	// pressure
	"psi": true, "bar": true, "Pa": true, "kPa": true, "MPa": true, "mbar": true,
	// electrical
	"V": true, "A": true, "mA": true, "ohm": true, "Ω": true, "kΩ": true, "MΩ": true,
	"Hz": true, "kHz": true, "MHz": true, "GHz": true,
	// length
	"m": true, "cm": true, "mm": true, "km": true, "in": true, "ft": true,
	// mass
	"kg": true, "g": true, "mg": true, "lb": true, "oz": true,
	// force and energy
	"N": true, "kN": true, "J": true, "kJ": true, "W": true, "kW": true, "MW": true,
	// time
	"s": true, "ms": true, "min": true, "h": true, "seconds": true, "minutes": true,
	"hours": true, "milliseconds": true,
	// angle and ratio
	"deg": true, "°": true, "%": true, "rpm": true, "dB": true,
	// other
	"L": true, "l": true, "ml": true, "gal": true, "mol": true,
}

func isUnit(s string) bool { return units[s] }

// collapseParentheticals replaces each balanced parenthetical group with one
// placeholder token.
func collapseParentheticals(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '(':
			if depth == 0 {
				b.WriteString(" (P) ")
			}
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

var quotedText = regexp.MustCompile("`[^`]*`|\"[^\"]*\"")

func stripQuoteBlocks(s string) string {
	return quotedText.ReplaceAllString(s, " (Q) ")
}

func trimPunctuation(f string) string {
	return strings.Trim(f, ".,;:!?()[]{}\"'")
}

// contractions is a list, not a pattern. A pattern cannot tell a contraction
// from a possessive: both use an apostrophe. The list names the forms that the
// standard does not allow, so a possessive such as "someone else's" does not
// give a false positive.
var contractions = []string{
	"ain't", "aren't", "can't", "cannot've", "could've", "couldn't", "didn't",
	"doesn't", "don't", "hadn't", "hasn't", "haven't", "he'd", "he'll", "he's",
	"how'd", "how'll", "how's", "i'd", "i'll", "i'm", "i've", "isn't", "it'd",
	"it'll", "it's", "let's", "mightn't", "mustn't", "shan't", "she'd", "she'll",
	"she's", "should've", "shouldn't", "that'd", "that'll", "that's", "there'd",
	"there's", "they'd", "they'll", "they're", "they've", "wasn't", "we'd",
	"we'll", "we're", "we've", "weren't", "what'd", "what'll", "what's",
	"when's", "where'd", "where'll", "where's", "who'd", "who'll", "who's",
	"why's", "won't", "would've", "wouldn't", "you'd", "you'll", "you're",
	"you've",
}

var contractionPattern = regexp.MustCompile(`(?i)\b(?:` + strings.Join(contractions, "|") + `)\b`)

// Contractions returns the contractions in a text. Rule 4.2 does not allow them.
func Contractions(text string) []string {
	var out []string
	for _, m := range contractionPattern.FindAllString(text, -1) {
		out = append(out, m)
	}
	return out
}

// Semicolons returns the positions of the semicolons. Rule 8.1 does not allow
// them.
func Semicolons(text string) []int {
	var out []int
	for i, r := range text {
		if r == ';' {
			out = append(out, i)
		}
	}
	return out
}

var ingWord = regexp.MustCompile(`\b[A-Za-z]+ing\b`)

// IngestiveForms returns the "-ing" words of a text. Rule 3.5 allows them only
// as a technical noun or inside a technical noun, so a program cannot decide
// the case. The caller must review each result.
func IngestiveForms(text string) []string {
	var out []string
	for _, m := range ingWord.FindAllString(text, -1) {
		out = append(out, m)
	}
	return out
}

// quantityWord follows ABOUT in the restricted meaning that the dictionary
// warns about. ABOUT is approved only as a preposition with the meaning
// "concerned with".
var quantityAfterAbout = regexp.MustCompile(`(?i)\bABOUT\s+(\d|about\s+\d|one|two|three|four|five|six|seven|eight|nine|ten|\d+\s)`)

// AboutAsApproximate returns the fragments where ABOUT seems to carry the
// meaning that the dictionary gives to APPROXIMATELY.
func AboutAsApproximate(text string) []string {
	var out []string
	for _, m := range quantityAfterAbout.FindAllString(text, -1) {
		out = append(out, m)
	}
	return out
}

var unapprovedModal = regexp.MustCompile(`(?i)\b(MAY|MIGHT|COULD|SHALL|SHOULD|OUGHT)\b`)

// UnapprovedModalUses returns the modals that the dictionary does not approve,
// with the approved alternative.
func UnapprovedModalUses(text string) []string {
	var out []string
	for _, m := range unapprovedModal.FindAllString(text, -1) {
		up := strings.ToUpper(m)
		out = append(out, up+" -> "+UnapprovedModals[up])
	}
	return out
}

// multiWordVerbs are the approved verbs that have two words. A sentence that
// opens with one of them names the verb with both words.
var multiWordVerbs = []string{"MAKE SURE", "COME ON", "GO OFF", "PUT ON"}

var leadingWord = regexp.MustCompile(`(?i)^[a-z][a-z-]*`)

// FirstVerb returns the verb that opens a sentence, in upper case. It returns
// an empty string when it cannot tell.
func FirstVerb(sentence string) string {
	s := strings.TrimSpace(sentence)
	// A table row or a list marker is not the start of a sentence.
	s = strings.TrimLeft(s, "-*0123456789. \t")
	if s == "" {
		return ""
	}
	upper := strings.ToUpper(s)
	for _, v := range multiWordVerbs {
		if strings.HasPrefix(upper, v) {
			return v
		}
	}
	return strings.ToUpper(leadingWord.FindString(s))
}

// IsApprovedVerb reports whether the dictionary approves the word as a verb.
func IsApprovedVerb(word string) bool {
	return ApprovedVerbs[strings.ToUpper(word)]
}

// SentenceMode gives the word limit that applies to a sentence.
type SentenceMode int

// Sentence modes.
const (
	ModeDescriptive SentenceMode = iota
	ModeProcedure
	ModeNote
)

// MaxWords returns the limit for a mode.
func MaxWords(m SentenceMode) int {
	switch m {
	case ModeProcedure:
		return MaxWordsProcedure
	case ModeNote:
		return MaxWordsNote
	default:
		return MaxWordsDescriptive
	}
}

// LooksLikeProcedure reports whether a sentence gives an instruction. The
// standard gives no test for this, so the result is a hint for the writer.
func LooksLikeProcedure(sentence string) bool {
	s := strings.TrimSpace(sentence)
	if s == "" {
		return false
	}
	first := FirstVerb(s)
	if first == "" {
		return false
	}
	// An instruction opens with an imperative verb and has no modal.
	if unapprovedModal.MatchString(s) {
		return false
	}
	for _, m := range ModalVerbs {
		if strings.HasPrefix(strings.ToUpper(s), m+" ") {
			return false
		}
	}
	return IsApprovedVerb(first)
}
