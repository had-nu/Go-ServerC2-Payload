package ste

import (
	"regexp"
	"strings"
)

var (
	codeSpan    = regexp.MustCompile("`[^`]*`")
	linkText    = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	refLink     = regexp.MustCompile(`\[([^\]]*)\]\[[^\]]*\]`)
	boldItalic  = regexp.MustCompile(`\*\*|\*\*|__|\*|~~`)
	image       = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	linkDef     = regexp.MustCompile(`^\[[^\]]+\]:`)
	htmlComment = regexp.MustCompile(`<!--.*?-->`)
)

// StripMarkdown removes the Markdown syntax so that a word counter reads only
// the words of the text. A code span becomes one placeholder, because a code
// span is an identifier and rule 8.6 counts an identifier as one word.
func StripMarkdown(s string) string {
	s = htmlComment.ReplaceAllString(s, " ")
	s = image.ReplaceAllString(s, " ")
	s = codeSpan.ReplaceAllString(s, " (C) ")
	s = linkText.ReplaceAllString(s, "$1")
	s = refLink.ReplaceAllString(s, "$1")
	s = boldItalic.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

// codeSpanWithGroup captures the text inside a code span. StripMarkdown uses
// the plain pattern because it removes the text. UnwrapCode needs the group.
var codeSpanWithGroup = regexp.MustCompile("`([^`]*)`")

// UnwrapCode removes the backticks of a code span but keeps the text inside.
// A writer uses a code span to name a form that the dictionary does not
// approve, as in "`CHECK (v)`". The checker must read the text inside, so that
// the writer can suppress the finding with a directive.
func UnwrapCode(s string) string {
	return codeSpanWithGroup.ReplaceAllString(s, "$1")
}
