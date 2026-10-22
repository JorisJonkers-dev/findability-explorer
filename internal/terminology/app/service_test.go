package app_test

import (
	"fmt"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

type catalogue struct {
	terms     []domain.Term
	companies []domain.Company
}

func (c catalogue) Terms() []domain.Term        { return c.terms }
func (c catalogue) Companies() []domain.Company { return c.companies }

func many(n int) []domain.Term {
	out := make([]domain.Term, 0, n)
	for i := range n {
		out = append(out, domain.Term{Text: fmt.Sprintf("term %03d", i), Language: domain.LanguageEn, Family: domain.FamilyOther, Tier: domain.TierProduct})
	}
	return out
}

func TestSearchClampsTheLimit(t *testing.T) {
	svc := app.New(catalogue{terms: many(300)})
	cases := map[int]int{0: app.DefaultLimit, -4: app.DefaultLimit, 7: 7, 1000: app.MaxLimit}
	for limit, want := range cases {
		if got := svc.Search(domain.Query{Limit: limit}); len(got.Items) != want || got.Total != 300 {
			t.Errorf("Search(limit %d) = %d items of %d, want %d", limit, len(got.Items), got.Total, want)
		}
	}
	if len(svc.Terms()) != 300 {
		t.Error("Terms does not return the whole map")
	}
}

func TestCompaniesSorted(t *testing.T) {
	svc := app.New(catalogue{companies: []domain.Company{{Name: "Zeta", GroupCompany: true}, {Name: "Beta"}, {Name: "alpha"}}})
	got := svc.Companies()
	if got[0].Name != "alpha" || got[1].Name != "Beta" || got[2].Name != "Zeta" {
		t.Fatalf("Companies = %+v", got)
	}
}
