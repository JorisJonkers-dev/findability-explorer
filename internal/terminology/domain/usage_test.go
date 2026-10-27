package domain_test

import (
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
