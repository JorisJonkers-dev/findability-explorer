package domain_test

import (
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// A stored scan is read by the same-language rule: each keyword classified again from its phrase
// and language, and only the known terms in a language of a page the scan read.
func TestAStoredScanIsReadByTheMap(t *testing.T) {
	lex := domain.NewLexicon([]domain.GlossaryTerm{
		{Text: "Transportband", Language: "de", Family: "all", Tier: "broad"},
		{Text: "spiral belt", Language: "en-US", Family: "tri_flex_turn_curve", Tier: "product"},
	})
	stored := domain.Analysis{
		Pages: []domain.PageSummary{{Language: "nl"}, {Language: "en"}, {Language: "und"}},
		Keywords: []domain.Keyword{
			// Stored in the map through the German term, as scans before the rule did.
			{Phrase: "transportband", Language: "nl", Score: 9, Classification: domain.Classification{Tier: "broad", Family: "all", Closest: "Transportband", InMap: true}},
			{Phrase: "spiral belts", Language: "en", Score: 5},
			{Phrase: "spiral belts", Language: "und", Score: 1, Classification: domain.Classification{Tier: "product", Closest: "spiral belt", InMap: true}},
		},
		KnownTerms: []domain.KnownTerm{
			{GlossaryTerm: domain.GlossaryTerm{Text: "Transportband", Language: "de"}},
			{GlossaryTerm: domain.GlossaryTerm{Text: "spiral belt", Language: "en-US"}},
		},
	}
	got := stored.ByTheMap(lex)
	want := []domain.Classification{
		{Tier: "broad"},
		{Tier: "product", Family: "tri_flex_turn_curve", Closest: "spiral belt", InMap: true},
		{Tier: "broad"},
	}
	for i, w := range want {
		if got.Keywords[i].Classification != w || got.Keywords[i].Score != stored.Keywords[i].Score {
			t.Errorf("keyword %d = %+v, want %+v", i, got.Keywords[i], w)
		}
	}
	if len(got.KnownTerms) != 1 || got.KnownTerms[0].Text != "spiral belt" {
		t.Errorf("known terms = %+v", got.KnownTerms)
	}
	if !stored.Keywords[0].InMap || len(stored.KnownTerms) != 2 {
		t.Error("ByTheMap changed the stored analysis")
	}
	// A scan without results stays without them.
	if empty := (domain.Analysis{}).ByTheMap(lex); empty.Keywords != nil || empty.KnownTerms != nil {
		t.Errorf("empty = %+v", empty)
	}
}

// A sighting counts only when the scan read a page in the term's language; American English is
// English, and a page in a language not told is no language.
func TestSightingsKeepToTheirLanguage(t *testing.T) {
	seen := []domain.Sighting{
		{CompetitorID: "a", Term: "Transportband", Language: "de", PageLanguages: []string{"en", "und"}},
		{CompetitorID: "a", Term: "conveyor belt", Language: "en-US", PageLanguages: []string{"en", "und"}},
		{CompetitorID: "b", Term: "Transportband", Language: "de", PageLanguages: []string{"de"}},
		{CompetitorID: "c", Term: "spiral belt", Language: "en"},
	}
	got := domain.SightedInTheirLanguage(seen)
	if len(got) != 2 || got[0].Term != "conveyor belt" || got[1].CompetitorID != "b" {
		t.Fatalf("SightedInTheirLanguage = %+v", got)
	}
	if len(seen) != 4 {
		t.Error("SightedInTheirLanguage changed its input")
	}
}
