package domain_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
)

func TestRelevant(t *testing.T) {
	lex := testLexicon()
	for phrase, want := range map[string]bool{
		"spiral belts":           true,
		"bakery ovens":           true,  // application words
		"flachdrahtgliedergurte": false, // no lexicon word of five letters inside
		"drahtosengliedergurte":  true,
		"huidige arsenaal belt":  false, // a third of its content words
		"the belt":               true,  // stopwords do not count
		"our approach":           false,
		"d":                      false, // a one-letter map word does not count
	} {
		if got := lex.Relevant(splitWords(phrase)); got != want {
			t.Errorf("Relevant(%q) = %v, want %v", phrase, got, want)
		}
	}
}

func TestCompoundsCount(t *testing.T) {
	lex := domain.NewLexicon([]domain.GlossaryTerm{{Text: "Gliedergurt", Language: "de", Family: "open_spiral", Tier: "product"}})
	if !lex.Relevant([]string{"runddrahtgliedergurte"}) {
		t.Fatal("a compound holding a map word is not relevant")
	}
	if c := lex.Classify([]string{"runddrahtgliedergurt"}, "de"); c.Family != "open_spiral" || c.InMap {
		t.Fatalf("compound classified %+v", c)
	}
}

func TestClassify(t *testing.T) {
	lex := testLexicon()
	cases := []struct {
		phrase, lang string
		want         domain.Classification
	}{
		{"eyelink belts", "en", domain.Classification{Tier: "product", Family: "eyelink", Closest: "eye-link belt / eyelink belt", InMap: true}},
		{"eyelink belt", "sv", domain.Classification{Tier: "product", Family: "eyelink", Closest: "Eyelink-band", InMap: true}},
		{"spiral belt", "en", domain.Classification{Tier: "product", Family: "tri_flex_turn_curve", Closest: "spiral belt / spiral conveyor belt", InMap: true}},
		{"wire diameter", "en", domain.Classification{Tier: "spec", Family: "spec_all", Closest: "wire diameter / cross rod diameter", InMap: true}},
		{"belt options", "en", domain.Classification{Tier: "broad"}},
		{"metal conveyor belts", "en", domain.Classification{Tier: "broad", Family: "all", Closest: "metal conveyor belt(s)", InMap: true}},
	}
	for _, c := range cases {
		if got := lex.Classify(splitWords(c.phrase), c.lang); got != c.want {
			t.Errorf("Classify(%q, %s) = %+v, want %+v", c.phrase, c.lang, got, c.want)
		}
	}
}

func TestBrandWords(t *testing.T) {
	for host, want := range map[string][]string{
		"www.twentebelt.com":    {"twentebelt"},
		"esfo-metalbelts.com":   {"esfo", "metalbelts"},
		"pelger-drahtgewebe.de": {"pelger", "drahtgewebe"},
		"www.fenner.co.uk":      {"fenner"},
		"localhost":             nil,
		"abc.de":                nil,
	} {
		if got := domain.BrandWords(host); !slices.Equal(got, want) {
			t.Errorf("BrandWords(%q) = %q, want %q", host, got, want)
		}
	}
}

func splitWords(s string) []string { return strings.Fields(fold.Fold(s)) }

// Only words that point at a family decide a phrase's family; a map spelling inside a phrase
// still counts, and a specification term needs a specification word.
func TestClassifyNamesAFamilyOnlyWithEvidence(t *testing.T) {
	lex := domain.NewLexicon([]domain.GlossaryTerm{
		{Text: "spiral belt / spiral conveyor belt", Language: "en", Family: "tri_flex_turn_curve", Tier: "product"},
		{Text: "metal conveyor belt(s)", Language: "en", Family: "all", Tier: "broad"},
		{Text: "baking band", Language: "en", Family: "closed_spiral", Tier: "application"},
		{Text: "request spiral tension", Language: "en", Family: "tri_flex_turn_curve", Tier: "problem"},
		{Text: "pitch / spiral pitch", Language: "en", Family: "spec_all", Tier: "spec"},
		{Text: "honeycomb belt", Language: "en", Family: "other", Tier: "product"},
		{Text: "flat wire belt", Language: "en", Family: "other", Tier: "product"},
		{Text: "eye-link belt", Language: "en", Family: "eyelink", Tier: "product"},
		{Text: "balanced weave belt", Language: "en", Family: "open_spiral", Tier: "product"},
		{Text: "compound balanced belt / compound balance weave", Language: "en", Family: "closed_spiral", Tier: "product"},
		{Text: "rod belt", Language: "en", Family: "other", Tier: "product"},
	})
	cases := map[string]domain.Classification{
		// "conveyor" is in a broad term too, so it points at no family.
		"conveyor belts": {Tier: "broad", Family: "all", Closest: "metal conveyor belt(s)"},
		// "tension" is only in a problem term: broad, keeping the problem tier.
		"tension": {Tier: "problem", Closest: "request spiral tension"},
		// A map spelling inside the phrase.
		"cb5 baking band": {Tier: "application", Family: "closed_spiral", Closest: "baking band"},
		// Its words, but not the spelling: no family.
		"band baking": {Tier: "application", Closest: "baking band"},
		// A specification word makes a specification term.
		"belt pitch": {Tier: "spec", Family: "spec_all", Closest: "pitch / spiral pitch"},
		// "compound" points at closed spiral; no family leads, but the closest term is of it.
		"compound balanced weave": {Tier: "product", Family: "closed_spiral", Closest: "compound balanced belt / compound balance weave"},
		"honeycomb":               {Tier: "product", Family: "other", Closest: "honeycomb belt"},
		"stainless":               {Tier: "broad"},
	}
	for phrase, want := range cases {
		if got := lex.Classify(splitWords(phrase), "en"); got != want {
			t.Errorf("Classify(%q) = %+v, want %+v", phrase, got, want)
		}
	}
}
