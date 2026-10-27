// Package spelling is the one rule for when a phrase is one of a map term's spellings. The scans
// judge a site's phrases by it, and the review queue checks by it whether the map holds a phrase,
// so a phrase a scan found in the map is never also a candidate, and the other way round. Whether
// the phrase and the term are in the same language is lang.Same's to say.
package spelling

import (
	"slices"
	"strings"
	"unicode"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
)

// Tokens splits text into lower-case words. A word is a run of letters and digits; a hyphen or a
// dot between two such characters stays inside the word, so "tri-flex" and "1.4301" are one word.
func Tokens(text string) []string {
	runes := []rune(strings.ToLower(text))
	var out []string
	start := -1
	for i, r := range runes {
		inWord := unicode.IsLetter(r) || unicode.IsDigit(r)
		joiner := (r == '-' || r == '.') && start >= 0 && i+1 < len(runes) &&
			(unicode.IsLetter(runes[i+1]) || unicode.IsDigit(runes[i+1]))
		switch {
		case inWord || joiner:
			if start < 0 {
				start = i
			}
		case start >= 0:
			out = append(out, string(runes[start:i]))
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, string(runes[start:]))
	}
	return out
}

// Words is s as the folded words phrases are compared by: its Tokens, folded (fold.Fold).
func Words(s string) []string {
	return strings.Fields(fold.Fold(strings.Join(Tokens(s), " ")))
}

// Alternatives splits a glossary entry into the spellings it stands for: "a / b" is a and b, and
// every parenthesised remark is dropped, so "belt(s)" is belt (a plural matches too; see Key).
func Alternatives(term string) []string {
	var out []string
	add := func(s string) {
		if s = strings.Join(strings.Fields(s), " "); s != "" && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	for _, part := range strings.Split(term, " / ") {
		add(dropRemarks(part))
	}
	return out
}

// dropRemarks removes every "(…)" from s.
func dropRemarks(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '(':
			depth++
		case r == ')' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Forms are the spellings a map term stands for, each as its Words; an alternative without a
// word is left out.
func Forms(term string) [][]string {
	var out [][]string
	for _, alt := range Alternatives(term) {
		if ws := Words(alt); len(ws) > 0 {
			out = append(out, ws)
		}
	}
	return out
}

// Singular drops a plural s, so "belts" is "belt"; "glass" and a word of three letters keep theirs.
func Singular(w string) string {
	if len(w) > 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") {
		return w[:len(w)-1]
	}
	return w
}

// SingularAll is every word Singular.
func SingularAll(words []string) []string {
	out := make([]string, 0, len(words))
	for _, w := range words {
		out = append(out, Singular(w))
	}
	return out
}

// Key is what a phrase's folded words are compared by: each word singular. A phrase is one of a
// term's spellings when its Key is the Key of one of the term's Forms, so "eyelink belts" is the
// term "eye-link belt / eyelink belt".
func Key(words []string) string {
	return strings.Join(SingularAll(words), " ")
}
