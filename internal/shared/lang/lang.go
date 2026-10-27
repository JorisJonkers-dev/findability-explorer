// Package lang names the languages that matter to Tribelt: the ones its buyers search in, Dutch,
// German and English. Terms, scans, the review queue and the views keep to them; what companies
// offer in other languages is left out, as it is not Tribelt's market.
package lang

import (
	"slices"
	"strings"
)

// Focus returns the languages that matter, in the order the views list them.
func Focus() []string { return []string{"nl", "de", "en"} }

// Primary is a language tag's primary subtag, lower case: "en-US" is "en", "DE_de" is "de".
func Primary(tag string) string {
	p, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
	p, _, _ = strings.Cut(p, "_")
	return p
}

// InFocus reports whether tag is Dutch, German or English, in any region (en-US is English).
func InFocus(tag string) bool { return slices.Contains(Focus(), Primary(tag)) }

// Unknown is the tag of a language that could not be told.
const Unknown = "und"

// Same reports whether two language tags name the same language: their primary subtags agree, so
// American English (en-US) is English (en). A tag that names no language, empty or Unknown, is
// the same as none, not even itself: text in a language not told matches nothing. This is the one
// rule for pairing a phrase with a term; a phrase never matches, is never closest to and is never
// merged with a term in another language.
func Same(a, b string) bool {
	return Told(a) && Primary(a) == Primary(b)
}

// Told reports whether tag names a language: it is neither empty nor Unknown.
func Told(tag string) bool {
	p := Primary(tag)
	return p != "" && p != Unknown
}

// others are language codes a site's address may start with (/fr/, /es-es/) that are not in
// focus. "uk" is left out: sites use /uk/ for the United Kingdom far more than for Ukrainian.
//
//nolint:gochecknoglobals // a read-only table
var others = map[string]bool{
	"fr": true, "es": true, "it": true, "pl": true, "sv": true, "da": true, "pt": true, "ru": true,
	"cs": true, "sk": true, "hu": true, "ro": true, "tr": true, "zh": true, "ja": true, "ko": true,
	"ar": true, "fi": true, "no": true, "nb": true, "el": true, "bg": true, "hr": true, "sl": true,
	"lt": true, "lv": true, "et": true, "sr": true, "he": true, "th": true, "vi": true, "id": true,
	"ms": true, "hi": true, "ca": true, "eu": true, "gl": true, "is": true, "ga": true,
}

// OtherLanguagePath reports whether an address's first path segment names a language not in
// focus: /fr/produits, /es-es/productos, /pl_pl/.
func OtherLanguagePath(path string) bool {
	first, _, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/")
	if len(first) != 2 && len(first) != 5 {
		return false
	}
	if len(first) == 5 && first[2] != '-' && first[2] != '_' {
		return false
	}
	return others[Primary(first)]
}
