package domain_test

import (
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

func productCompanies() []domain.Company {
	return []domain.Company{
		{ID: "tribelt", Name: "Tribelt B.V.", Role: domain.RoleTribelt, BeltNames: []domain.BeltName{
			{Name: "Draadogenbanden", Family: "eyelink"}, {Name: "Eyelink belts", Family: "eyelink"}, {Name: "Draadogenbanden", Family: "eyelink"},
		}},
		{ID: "twentebelt", Name: "Twentebelt", Role: domain.RoleCompetitor, Website: "https://twentebelt.com/", BeltNames: []domain.BeltName{
			{Name: "Oogbanden", Family: "eyelink"}, {Name: "Spiraalbanden", Family: "open_spiral", FromScan: true},
		}},
		{ID: "ashworth", Name: "Ashworth", Role: domain.RoleCompetitor, BeltNames: []domain.BeltName{{Name: "Omni-Grid", Family: "tri_flex_turn_curve"}, {Name: "Eyelink", Family: "eyelink"}}},
		{ID: "de-spiraal", Name: "De Spiraal", Role: domain.RoleGroup, BeltNames: []domain.BeltName{{Name: "Spiraalbanden", Family: "open_spiral"}}},
	}
}

func TestProductsCompareEachFamily(t *testing.T) {
	terms := []domain.MapTerm{
		{Text: "eyelink belt", Language: "en", Family: "eyelink", UsedBy: 5, UsedByTribelt: true},
		{Text: "draadogenband", Language: "nl", Family: "eyelink", UsedBy: 9},
		{Text: "oogband", Language: "nl", Family: "eyelink", UsedBy: 9},
		{Text: "spiral belt", Language: "en", Family: "open_spiral", UsedBy: 2},
	}
	run := &domain.Run{Answers: []domain.Answer{
		{Family: "eyelink", Scope: "twente", Named: []string{"tribelt", "twentebelt"}},
		{Family: "eyelink", Scope: "twente", Named: []string{"twentebelt", "de-spiraal"}},
		{Family: "eyelink", Scope: "dutch", Named: []string{"ashworth"}},
		{Family: "open_spiral", Scope: "dutch"},
	}}
	got := domain.Products(productCompanies(), terms, run)
	if len(got) != 5 {
		t.Fatalf("products = %d", len(got))
	}
	eye := got[0]
	if eye.Family != "eyelink" || len(eye.TribeltNames) != 2 || len(eye.Sellers) != 2 || eye.Sellers[0].Name != "Ashworth" ||
		eye.Sellers[1].Website != "https://twentebelt.com/" || eye.Sellers[1].Names[0] != "Oogbanden" {
		t.Errorf("eyelink sellers = %+v, Tribelt %v", eye.Sellers, eye.TribeltNames)
	}
	if eye.TermsUsed != 1 || len(eye.Terms) != 3 || eye.TopTerm() != "draadogenband" {
		t.Errorf("eyelink terms = %+v", eye.Terms)
	}
	if eye.Answers != 3 || eye.Named != 1 || len(eye.ByScope) != 2 || eye.ByScope[0] != (domain.ScopeShare{Scope: "twente", Answers: 2, Named: 1}) {
		t.Errorf("eyelink answers = %d %d %+v", eye.Answers, eye.Named, eye.ByScope)
	}
	// The group company is never among the named; Ashworth and Tribelt tie on one answer each.
	if len(eye.MostNamed) != 3 || eye.MostNamed[0].CompanyID != "twentebelt" || eye.MostNamed[1].CompanyID != "ashworth" {
		t.Errorf("most named = %+v", eye.MostNamed)
	}
	open := got[1]
	if len(open.Sellers) != 1 || open.Answers != 1 || len(open.MostNamed) != 0 || open.TopTerm() != "spiral belt" {
		t.Errorf("open spiral = %+v", open)
	}
	if other := got[4]; other.TopTerm() != "" || other.Answers != 0 || len(other.Sellers) != 0 {
		t.Errorf("other = %+v", other)
	}
	if none := domain.Products(productCompanies(), nil, nil)[0]; none.Answers != 0 || len(none.ByScope) != 0 {
		t.Errorf("before any run = %+v", none)
	}
}

func TestStandoutsAreComputed(t *testing.T) {
	products := []domain.Product{
		{Family: "eyelink", Sellers: make([]domain.Seller, 3), Terms: make([]domain.MapTerm, 4), TermsUsed: 2, Answers: 10, Named: 1},
		{Family: "open_spiral", Sellers: make([]domain.Seller, 5), Terms: make([]domain.MapTerm, 4), TermsUsed: 0, Answers: 10, Named: 4},
		{Family: "other", Sellers: make([]domain.Seller, 5), Terms: make([]domain.MapTerm, 2), TermsUsed: 0},
	}
	got := domain.Standouts(products)
	want := []domain.Standout{
		{Finding: domain.FindingCrowded, Family: "open_spiral"},
		{Finding: domain.FindingFewestTerms, Family: "open_spiral"},
		{Finding: domain.FindingBestInAI, Family: "open_spiral"},
	}
	if len(got) != len(want) {
		t.Fatalf("standouts = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("standout %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if got := domain.Standouts([]domain.Product{{Family: "eyelink"}}); len(got) != 0 {
		t.Errorf("no data, no standouts: %+v", got)
	}
}
