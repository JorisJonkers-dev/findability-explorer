package domain_test

import (
	"slices"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

func TestTribeltUsesWhatItsNewestScanFoundOnceThereIsOne(t *testing.T) {
	credited := domain.Term{Text: "Drahtösenband", Language: domain.LanguageDe, UsedBy: []string{"Tribelt"}}
	found := domain.Term{Text: "eyelink belt", Language: domain.LanguageEn, UsedBy: []string{"Märtens"}}
	// The same text in another language is another term: the scan's find does not count for it.
	otherLanguage := domain.Term{Text: "eyelink belt", Language: domain.LanguageNl}
	seen := map[domain.TermKey][]string{found.Key(): {"martens", domain.TribeltID}}

	scanned := domain.SiteUse{FoundBy: seen, TribeltScanned: true}
	if scanned.Source() != domain.UsageScan || !scanned.TribeltUses(found) || scanned.TribeltUses(credited) || scanned.TribeltUses(otherLanguage) {
		t.Errorf("after a scan, the scan decides: %v %v %v", scanned.TribeltUses(found), scanned.TribeltUses(credited), scanned.TribeltUses(otherLanguage))
	}
	before := domain.SiteUse{FoundBy: seen}
	if before.Source() != domain.UsageResearch || before.TribeltUses(found) || !before.TribeltUses(credited) {
		t.Errorf("before a scan, the research decides: %v %v", before.TribeltUses(found), before.TribeltUses(credited))
	}
}

func TestCreditsTribeltNamesTribeltOrOneOfItsProductLines(t *testing.T) {
	for credits, want := range map[string]bool{
		"Tribelt":                            true,
		"Tribelt (Tri-Flex)":                 true,
		"  Tribelt  (variabler Spaltweite) ": true,
		"Tribelt B.V.":                       false,
		"Tribeltech":                         false,
		"Märtens (Tribelt)":                  false,
		"Tribelt (a) b)":                     false,
		"(Tribelt)":                          false,
		"Tribelt)":                           false,
	} {
		if got := domain.CreditsTribelt([]string{"Märtens", credits}); got != want {
			t.Errorf("%q: %v", credits, got)
		}
	}
	if domain.CreditsTribelt(nil) {
		t.Error("no credits name Tribelt")
	}
}

func TestMakersAreTheCompetitorsOnTheListThatUseATerm(t *testing.T) {
	terms := []domain.Term{
		{Text: "wire belt", Language: domain.LanguageEn, UsedBy: []string{"Wire Belt", "Märtens (Gliederband)", "Tribelt"}},
		{Text: "flat flex", Language: domain.LanguageEn, UsedBy: []string{"Wire Belt Osterloh", "Wire Belt", "Gone"}},
	}
	competitors := domain.CreditNames(terms, []domain.Listed{
		{ID: "wire-belt", Name: "Wire Belt Company Ltd"},
		{ID: "osterloh", Name: "Wire Belt Company Osterloh GmbH"},
		{ID: "martens", Name: "Märtens Transportbänder GmbH"},
		{ID: "added", Name: "Added Belts B.V."},
	})
	// The longest credit that fits wins: Osterloh is "Wire Belt Osterloh", not "Wire Belt".
	want := []domain.Competitor{{ID: "wire-belt", ShortName: "Wire Belt"}, {ID: "osterloh", ShortName: "Wire Belt Osterloh"}, {ID: "martens", ShortName: "Märtens"}, {ID: "added"}}
	if !slices.Equal(competitors, want) {
		t.Fatalf("credit names = %+v", competitors)
	}
	use := domain.SiteUse{Competitors: competitors, FoundBy: map[domain.TermKey][]string{terms[0].Key(): {"added", "martens", "archived"}}}
	// Tribelt's credit and a scan of a company no longer on the list make no maker; a scan of a
	// credited company is no scan-only maker.
	if got := use.Makers(terms[0]); !slices.Equal(got, []domain.Maker{{ID: "wire-belt"}, {ID: "martens"}, {ID: "added", FromScan: true}}) {
		t.Errorf("makers = %+v", got)
	}
	// "Gone" is credited but no longer on the list.
	if got := use.Makers(terms[1]); !slices.Equal(got, []domain.Maker{{ID: "wire-belt"}, {ID: "osterloh"}}) {
		t.Errorf("makers = %+v", got)
	}
	if got := (domain.SiteUse{}).Makers(terms[0]); got != nil {
		t.Errorf("without the list = %+v", got)
	}
}
