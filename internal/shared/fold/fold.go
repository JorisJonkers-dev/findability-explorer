// Package fold is the one text normalisation every context compares with, so a term found by the
// terminology search is also found on a crawled page. The web client folds the same way.
package fold

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Fold reduces s to the form searches compare: lower case, accents removed, ß as ss, ø as o, ł as
// l, and every run of spaces, hyphens, slashes and underscores as one space. So "Förderband"
// matches "forderband" and "Eye-Link" matches "eye link". The web client folds the same way.
func Fold(s string) string {
	var b strings.Builder
	space := false
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case r == 'ß':
			b.WriteString("ss")
		case r == 'ø':
			b.WriteRune('o')
		case r == 'ł':
			b.WriteRune('l')
		case unicode.IsSpace(r) || r == '-' || r == '/' || r == '_':
			if !space && b.Len() > 0 {
				b.WriteRune(' ')
			}
			space = true
			continue
		default:
			b.WriteRune(r)
		}
		space = false
	}
	return strings.TrimSpace(b.String())
}
