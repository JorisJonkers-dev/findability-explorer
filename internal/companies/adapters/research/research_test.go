package research_test

import (
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/adapters/research"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/seed"
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
	for _, want := range []string{"pelger", "martens-transportbander", "tribelt", "de-spiraal", "esfo-transportbanden"} {
		if _, ok := byID[want]; !ok {
			t.Errorf("no company %q", want)
		}
	}
}
