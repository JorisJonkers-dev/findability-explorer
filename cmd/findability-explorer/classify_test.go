package main

import (
	"strings"
	"testing"

	analysisdomain "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/seed"
	terminologydomain "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// TestPhrasesSortIntoTheRightGroup classifies phrases a scan of ashworth.com found on 26 October
// 2026 against the embedded terminology map. A phrase of generic words (belt, conveyor, drive,
// tension, baking) names no belt family, however few map terms hold the word; a phrase that
// spells out a family's construction keeps that family.
func TestPhrasesSortIntoTheRightGroup(t *testing.T) {
	catalogue, err := seed.Load()
	if err != nil {
		t.Fatal(err)
	}
	lex := analysisdomain.NewLexicon(glossary(catalogue.Terms(), terminologydomain.SiteUse{}.TribeltUses))
	cases := map[string]analysisdomain.Group{
		// Generic: no family.
		"conveyor belts":         analysisdomain.GroupBroad,
		"wire conveyor belts":    analysisdomain.GroupBroad,
		"belt":                   analysisdomain.GroupBroad,
		"belt width":             analysisdomain.GroupBroad,
		"belt life":              analysisdomain.GroupBroad,
		"woven belts":            analysisdomain.GroupBroad,
		"tension":                analysisdomain.GroupBroad,
		"allowable tension":      analysisdomain.GroupBroad,
		"baking":                 analysisdomain.GroupBroad,
		"baking industry":        analysisdomain.GroupBroad,
		"band":                   analysisdomain.GroupBroad,
		"spiral":                 analysisdomain.GroupBroad,
		"links":                  analysisdomain.GroupBroad,
		"metal":                  analysisdomain.GroupBroad,
		"transportadoras":        analysisdomain.GroupBroad,
		"woven wire conveyor":    analysisdomain.GroupBroad,
		"straight-running belts": analysisdomain.GroupBroad,
		// Specification words.
		"sprocket options": analysisdomain.GroupSpec,
		"pitch":            analysisdomain.GroupSpec,
		"t316":             analysisdomain.GroupSpec,
		// A family's construction, spelled out.
		"compound balance weave":  analysisdomain.GroupClosedSpiral,
		"compound balanced weave": analysisdomain.GroupClosedSpiral,
		"balance weave baking":    analysisdomain.GroupOpenSpiral,
		"balanced weave conveyor": analysisdomain.GroupOpenSpiral,
		"eye-link belts":          analysisdomain.GroupEyelink,
		"small radius omni-flex":  analysisdomain.GroupTriFlex,
		"turn-curve":              analysisdomain.GroupTriFlex,
		"spiral and turn-curve":   analysisdomain.GroupTriFlex,
		"flat wire conveyor":      analysisdomain.GroupOther,
		"honeycomb belt":          analysisdomain.GroupOther,
		// The map's own spellings keep the map's family.
		"spiral conveyor belt": analysisdomain.GroupTriFlex,
		"balanced weave belt":  analysisdomain.GroupOpenSpiral,
	}
	for phrase, want := range cases {
		c := lex.Classify(strings.Fields(fold.Fold(phrase)), "en") // as Keywords folds a phrase
		if got := analysisdomain.GroupOf(c); got != want {
			t.Errorf("%q: group %s (family %q, tier %q, closest %q), want %s", phrase, got, c.Family, c.Tier, c.Closest, want)
		}
	}
}
