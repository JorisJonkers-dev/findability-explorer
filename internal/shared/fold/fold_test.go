package fold_test

import (
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
)

func TestFold(t *testing.T) {
	cases := map[string]string{
		"Drahtösengliederband":       "drahtosengliederband",
		"Straße":                     "strasse",
		"Ø 2 mm":                     "o 2 mm",
		"taśma pętlicowa":            "tasma petlicowa",
		"Łódź":                       "lodz",
		"Tri-Flex / spiral_belt":     "tri flex spiral belt",
		"  -leading and trailing-  ": "leading and trailing",
		"tapis à œillets":            "tapis a œillets",
		"":                           "",
	}
	for in, want := range cases {
		if got := fold.Fold(in); got != want {
			t.Errorf("Fold(%q) = %q, want %q", in, got, want)
		}
	}
}
