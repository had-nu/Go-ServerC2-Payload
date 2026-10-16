// Command stelint checks the Markdown documents of this repository against the
// mechanical rules of ASD-STE100 Simplified Technical English, Issue 9.
//
// The tool checks the rules that a program can decide. It cannot check the
// approved meaning of a word, or the approved part of speech of a noun. The
// tool does not certify a document against the standard. It reports the rules
// that a writer must review by hand.
//
// Usage:
//
//	stelint [flags] <file>...
//
// Exit status is 1 when the tool finds an error, and 0 otherwise.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/had-nu/Go-ServerC2-Payload/tools/stelint/ste"
)

type finding struct {
	file     string
	line     int
	rule     string
	severity string
	message  string
}

func (f finding) String() string {
	return fmt.Sprintf("%s:%d: [%s] %s: %s", f.file, f.line, f.rule, f.severity, f.message)
}

func main() {
	strict := flag.Bool("strict", false, "treat a review finding as an error")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "stelint checks Markdown against the mechanical rules of ASD-STE100 Issue 9.\n\nusage: stelint [flags] <file>...\n\nflags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	files := flag.Args()
	if len(files) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	var all []finding
	for _, f := range files {
		found, err := checkFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stelint: %v\n", err)
			os.Exit(2)
		}
		all = append(all, found...)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].file != all[j].file {
			return all[i].file < all[j].file
		}
		return all[i].line < all[j].line
	})

	errors, reviews := 0, 0
	for _, f := range all {
		fmt.Println(f)
		switch f.severity {
		case "error":
			errors++
		default:
			reviews++
		}
	}

	fmt.Fprintf(os.Stderr, "\nstelint: %d files, %d errors, %d items for review\n", len(files), errors, reviews)
	if errors > 0 {
		os.Exit(1)
	}
	if *strict && reviews > 0 {
		os.Exit(1)
	}
}

func checkFile(path string) ([]finding, error) {
	src, err := os.ReadFile(path) // #nosec G304 -- the path comes from the command line of a documentation checker, not from the network
	if err != nil {
		return nil, err
	}
	p := &parser{file: filepath.ToSlash(path), allowances: map[int]map[string]bool{}}
	p.parse(string(src))
	return p.findings, nil
}

type parser struct {
	file       string
	lines      []string
	para       []string
	paraLine   int
	findings   []finding
	inFence    bool
	allowances map[int]map[string]bool
}

var (
	tableRow = regexp.MustCompile(`^\s*\|`)
	listItem = regexp.MustCompile(`^\s*([-*+]|\d+\.)\s+`)
	heading  = regexp.MustCompile(`^\s*#{1,6}\s`)
	htmlTag  = regexp.MustCompile(`^\s*<`)
)

func (p *parser) parse(text string) {
	p.lines = strings.Split(text, "\n")
	p.collectAllowances()
	for i, line := range p.lines {
		lineNo := i + 1

		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			p.flushParagraph()
			p.inFence = !p.inFence
			continue
		}
		if p.inFence {
			continue
		}
		if strings.TrimSpace(line) == "" {
			p.flushParagraph()
			continue
		}
		if tableRow.MatchString(line) {
			p.flushParagraph()
			// A header row names the columns. The text of a column name is not
			// a use of the word.
			isHeader := i+1 < len(p.lines) && tableSeparator.MatchString(p.lines[i+1])
			p.checkTableRow(line, lineNo, isHeader)
			continue
		}
		if heading.MatchString(line) || htmlTag.MatchString(line) {
			p.flushParagraph()
			p.checkStandaloneLine(line, lineNo)
			continue
		}

		if listItem.MatchString(line) || len(p.para) == 0 {
			p.flushParagraph()
			p.para = append(p.para, line)
			p.paraLine = lineNo
		} else {
			p.para = append(p.para, line)
		}
	}
	p.flushParagraph()
}

func (p *parser) add(line int, rule, severity, format string, args ...any) {
	p.findings = append(p.findings, finding{
		file:     p.file,
		line:     line,
		rule:     rule,
		severity: severity,
		message:  fmt.Sprintf(format, args...),
	})
}

// flushParagraph treats the buffered lines as one paragraph and checks it.
func (p *parser) flushParagraph() {
	if len(p.para) == 0 {
		return
	}
	text := strings.Join(p.para, " ")
	line := p.paraLine
	p.para = nil

	sentences := SplitSentences(text)
	if len(sentences) > ste.MaxSentencesPerParagraph {
		p.add(line, "6.6", "error",
			"the paragraph has %d sentences, the standard allows %d", len(sentences), ste.MaxSentencesPerParagraph)
	}
	for _, s := range sentences {
		p.checkSentence(s, line)
	}
}

func (p *parser) checkStandaloneLine(line string, lineNo int) {
	// A heading is a title. Rule 8.6 counts a title as one word, so a heading
	// is not a sentence. The tool checks the text of a heading for tokens only.
	p.checkTokens(line, lineNo, false)
}

func (p *parser) checkTableRow(line string, lineNo int, isHeader bool) {
	cells := strings.Split(strings.Trim(line, "| "), "|")
	for _, c := range cells {
		c = strings.TrimSpace(c)
		if c == "" || setextSeparator.MatchString(c) {
			continue
		}
		p.checkTokens(c, lineNo, !isHeader)
	}
}

var setextSeparator = regexp.MustCompile(`^:?-{2,}:?$`)

// tableSeparator matches the row under a table header, such as "|---|---|".
var tableSeparator = regexp.MustCompile(`^\s*\|[\s:|-]*-?[\s:|-]*\|?\s*$`)

func (p *parser) checkSentence(sentence string, line int) {
	words := ste.WordCount(sentence)

	mode := ste.ModeDescriptive
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sentence)), ste.NoteWord) {
		mode = ste.ModeNote
	}
	limit := ste.MaxWords(mode)

	clean := ste.StripMarkdown(sentence)
	allow := p.allowances[line]

	if words > limit {
		p.addAllow(allow, line, "6.3", "error",
			"the sentence has %d words, the standard allows %d: %q", words, limit, truncate(clean, 90))
	} else if words > ste.MaxWordsProcedure && ste.LooksLikeProcedure(sentence) {
		p.addAllow(allow, line, "5.1", "error",
			"the instruction has %d words, the standard allows %d: %q", words, ste.MaxWordsProcedure, truncate(clean, 90))
	}

	p.checkTokens(clean, line, true)
}

// checkTokens checks the rules that apply to any piece of text. When checkVerb
// is false the text is not a sentence, so the rule 1.2 check does not apply.
func (p *parser) checkTokens(text string, line int, checkVerb bool) {
	allow := p.allowances[line]

	// The text of a code span is an identifier, not a word. Remove the
	// Markdown syntax before the token checks.
	plain := ste.StripMarkdown(text)
	// Keep the text inside the code spans for the rule 1.2 check.
	withCode := ste.UnwrapCode(text)

	for _, c := range ste.Contractions(plain) {
		p.addAllow(allow, line, "4.2", "error", "the text uses the contraction %q", c)
	}
	if n := len(ste.Semicolons(plain)); n > 0 {
		p.addAllow(allow, line, "8.1", "error", "the text uses %d semicolon(s)", n)
	}
	for _, m := range ste.UnapprovedModalUses(plain) {
		p.addAllow(allow, line, "1.1", "error",
			"the text uses a modal that the dictionary does not approve: %s", m)
	}
	for _, frag := range ste.AboutAsApproximate(plain) {
		p.addAllow(allow, line, "1.3", "error",
			"ABOUT seems to carry the meaning that the dictionary gives to APPROXIMATELY (%q)", frag)
	}

	if !checkVerb {
		return
	}
	// A writer writes "`CHECK (v)`" to name a form that the dictionary does not
	// approve. The checker must read the text inside, so that the writer can
	// suppress the finding with a directive.
	if first := ste.FirstVerb(withCode); first != "" {
		if note, ok := ste.NounOnlyVerbs[first]; ok {
			p.addAllow(allow, line, "1.2", "error",
				"%s opens the sentence as a verb, but the dictionary approves it only as a noun (%s)", first, note)
		}
	}
}

// allowDirective suppresses a rule on one line. A writer uses it when the line
// documents a form that the standard does not approve.
var allowDirective = regexp.MustCompile(`<!--\s*stelint:allow\s+([0-9]+\.[0-9]+)(?:\s+([0-9]+\.[0-9]+))?\s*-->`)

func (p *parser) collectAllowances() {
	p.allowances = map[int]map[string]bool{}
	for i, line := range p.lines {
		lineNo := i + 1
		for _, m := range allowDirective.FindAllStringSubmatch(line, -1) {
			for _, g := range m[1:] {
				if g == "" {
					continue
				}
				if p.allowances[lineNo] == nil {
					p.allowances[lineNo] = map[string]bool{}
				}
				p.allowances[lineNo][g] = true
			}
		}
	}
}

func (p *parser) addAllow(allow map[string]bool, line int, rule, severity, format string, args ...any) {
	if allow[rule] {
		return
	}
	p.add(line, rule, severity, format, args...)
}

// SplitSentences splits a text into sentences. It does not split on an
// abbreviation such as "e.g." or on a decimal number.
func SplitSentences(text string) []string {
	var out []string
	var cur strings.Builder
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		cur.WriteRune(r)
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		// Look ahead.
		j := i + 1
		for j < len(runes) && (runes[j] == '"' || runes[j] == '\'' || runes[j] == ')' || runes[j] == '`') {
			cur.WriteRune(runes[j])
			j++
		}
		if j < len(runes) && runes[j] != ' ' && runes[j] != '\n' {
			continue
		}
		word, ok := lastWord(cur.String())
		if ok && isAbbreviation(word) {
			continue
		}
		s := strings.TrimSpace(cur.String())
		if s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		out = append(out, s)
	}
	return out
}

func lastWord(s string) (string, bool) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return "", false
	}
	w := fields[len(fields)-1]
	w = strings.Trim(w, ".,!?")
	return strings.ToLower(w), w != ""
}

func isAbbreviation(w string) bool {
	switch w {
	case "e.g", "i.e", "etc", "vs", "mr", "mrs", "ms", "dr", "prof", "cf", "al", "fig", "no", "approx", "resp":
		return true
	}
	return false
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
