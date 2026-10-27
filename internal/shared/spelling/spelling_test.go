package spelling_test

import (
	"slices"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/spelling"
)

func TestTokens(t *testing.T) {
	got := spelling.Tokens("Tri-Flex belts in 1.4301, up to 1.150 °C! Edelstahl-Förderbänder -x. end-")
	want := []string{"tri-flex", "belts", "in", "1.4301", "up", "to", "1.150", "c", "edelstahl-förderbänder", "x", "end"}
	if !slices.Equal(got, want) {
		t.Fatalf("Tokens = %q, want %q", got, want)
	}
	if got := spelling.Tokens("Ofenband"); !slices.Equal(got, []string{"ofenband"}) {
		t.Fatalf("Tokens(single word) = %q", got)
	}
}

func TestAlternatives(t *testing.T) {
	cases := map[string][]string{
		"eye-link belt / eyelink belt":         {"eye-link belt", "eyelink belt"},
		"metal conveyor belt(s)":               {"metal conveyor belt"},
		"Eye-Link OGBL (Ashworth)":             {"Eye-Link OGBL"},
		"spiraaltorens / vries- en koeltorens": {"spiraaltorens", "vries- en koeltorens"},
		"a / a / (only a remark)":              {"a"},
		"odd ) bracket":                        {"odd ) bracket"},
	}
	for in, want := range cases {
		if got := spelling.Alternatives(in); !slices.Equal(got, want) {
			t.Errorf("Alternatives(%q) = %q, want %q", in, got, want)
		}
	}
}

// A phrase is a term's spelling when their words agree, each singular: "Eye-Link Belts" is
// "eye-link belt / eyelink belt", and "nid d'abeilles" folds its apostrophe away.
func TestKeysOfFormsAndPhrases(t *testing.T) {
	forms := spelling.Forms("Eye-Link belt / eyelink belts (OGBL) / (a remark) / °")
	keys := make([]string, 0, len(forms))
	for _, f := range forms {
		keys = append(keys, spelling.Key(f))
	}
	if !slices.Equal(keys, []string{"eye link belt", "eyelink belt"}) {
		t.Fatalf("keys = %q", keys)
	}
	if got := spelling.Key(spelling.Words("Eye-Link Belts")); got != "eye link belt" {
		t.Errorf("Key(Eye-Link Belts) = %q", got)
	}
	if got := spelling.Words("nid d'abeilles"); !slices.Equal(got, []string{"nid", "d", "abeilles"}) {
		t.Errorf("Words = %q", got)
	}
	for w, want := range map[string]string{"belts": "belt", "glass": "glass", "bus": "bus", "belt": "belt"} {
		if got := spelling.Singular(w); got != want {
			t.Errorf("Singular(%q) = %q, want %q", w, got, want)
		}
	}
}
