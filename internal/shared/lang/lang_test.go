package lang_test

import (
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/lang"
)

func TestInFocus(t *testing.T) {
	for tag, want := range map[string]bool{
		"nl": true, "de": true, "en": true, "en-US": true, "DE-de": true, "en_GB": true,
		"fr": false, "es": false, "pl": false, "sv": false, "": false, "und": false,
	} {
		if got := lang.InFocus(tag); got != want {
			t.Errorf("InFocus(%q) = %v, want %v", tag, got, want)
		}
	}
}

func TestOtherLanguagePath(t *testing.T) {
	for path, want := range map[string]bool{
		"/fr/produits": true, "/es-es/productos": true, "/pl_pl/": true, "/it": true, "/PT/": true,
		"/de/produkte": false, "/en-us/belts": false, "/nl/": false, "/uk/belts": false,
		"/products": false, "/": false, "/frx/": false, "/es-/x": false, "/abcde/": false,
	} {
		if got := lang.OtherLanguagePath(path); got != want {
			t.Errorf("OtherLanguagePath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestSame(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"en", "en-US", true},
		{"en-US", "EN", true},
		{"de", "de-DE", true},
		{"nl", "nl", true},
		{"nl", "de", false},
		{"en", "de", false},
		{"und", "und", false},
		{"", "", false},
		{"und", "en", false},
		{"en", "", false},
	} {
		if got := lang.Same(c.a, c.b); got != c.want {
			t.Errorf("Same(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
