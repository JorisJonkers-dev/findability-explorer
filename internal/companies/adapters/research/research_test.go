package research_test

import (
	"slices"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/adapters/research"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/seed"
	terminologydomain "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

func TestCompaniesFromTheSeed(t *testing.T) {
	c, err := seed.Load()
	if err != nil {
		t.Fatal(err)
	}
	got := research.Companies(c.Companies())
	byID := map[string]int{}
	for i, co := range got {
		if _, dup := byID[co.ID]; dup {
			t.Errorf("two companies have id %q", co.ID)
		}
		byID[co.ID] = i
	}
	tw := got[byID["twentebelt"]]
	if tw.URL == nil || tw.URL.String() != "https://www.twentebelt.com/" || tw.Host != "twentebelt.com" || len(tw.BeltNames) == 0 {
		t.Errorf("Twentebelt = %+v", tw)
	}
	if ntm := got[byID["ntm"]]; ntm.URL != nil || ntm.Host != "" {
		t.Errorf("NTM, which has no website, = %+v", ntm)
	}
	// Only the website's Dutch, German and English versions count, and names in those languages
	// come first; Codina's Spanish names stay, as they show which families it offers.
	for _, co := range got {
		for _, l := range co.Languages {
			if l != "nl" && l != "de" && l != "en" {
				t.Errorf("%s lists language %q", co.ID, l)
			}
		}
	}
	if codina := got[byID["codina-metal"]]; len(codina.BeltNames) == 0 {
		t.Errorf("Codina lost its belt names: %+v", codina)
	}
	for _, want := range []string{"pelger", "martens-transportbander", "tribelt", "de-spiraal", "esfo-transportbanden"} {
		if _, ok := byID[want]; !ok {
			t.Errorf("no company %q", want)
		}
	}
}

func TestNamesInTheFocusLanguagesComeFirst(t *testing.T) {
	got := research.Companies([]terminologydomain.Company{{
		Name: "Example", Languages: []string{"es", "en", "fr", "de"},
		BeltNames: []terminologydomain.BeltName{
			{Name: "Banda de malla", Family: "eyelink", Language: "es"},
			{Name: "Eyelink belt", Family: "eyelink", Language: "en"},
			{Name: "Ösenband", Family: "eyelink", Language: "de"},
		},
	}})
	if len(got) != 1 || !slices.Equal(got[0].Languages, []string{"en", "de"}) ||
		got[0].BeltNames[0].Name != "Eyelink belt" || got[0].BeltNames[1].Name != "Ösenband" || got[0].BeltNames[2].Name != "Banda de malla" {
		t.Fatalf("Companies = %+v", got)
	}
}
