package domain_test

import (
	"reflect"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

func TestTermsOfCountsEachLanguageAndWhatTribeltUses(t *testing.T) {
	got := domain.TermsOf([]domain.MapTerm{
		{Text: "spiral belt", Language: "en", UsedByTribelt: true},
		{Text: "conveyor belt", Language: "en-US"},
		{Text: "Spiralband", Language: "de", UsedByTribelt: true},
		{Text: "Gliederband", Language: "de"},
		{Text: "spiraalband", Language: "nl"},
		{Text: "bande", Language: "fr"},
	})
	want := domain.TermsOverview{Total: 6, Tribelt: 2, Languages: []domain.LanguageTerms{
		{Language: "de", Terms: 2, Tribelt: 1}, {Language: "en", Terms: 2, Tribelt: 1}, {Language: "fr", Terms: 1}, {Language: "nl", Terms: 1},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("terms = %+v", got)
	}
	if empty := domain.TermsOf(nil); empty.Total != 0 || empty.Languages == nil || len(empty.Languages) != 0 {
		t.Errorf("an empty map = %+v", empty)
	}
}

func TestCompetitorsOfCountsTheResearchedCompetitorsPerFamily(t *testing.T) {
	got := domain.CompetitorsOf([]domain.Company{
		{ID: "tribelt", Role: domain.RoleTribelt, Researched: true, BeltNames: []domain.BeltName{{Family: "eyelink"}}},
		{ID: "twentebelt", Role: domain.RoleCompetitor, Researched: true, BeltNames: []domain.BeltName{{Family: "eyelink"}, {Family: "eyelink"}, {Family: "other"}}},
		{ID: "martens", Role: domain.RoleCompetitor, Researched: true},
		{ID: "added", Role: domain.RoleCompetitor, BeltNames: []domain.BeltName{{Family: "eyelink"}}},
		{ID: "sister", Role: domain.RoleGroup, Researched: true, BeltNames: []domain.BeltName{{Family: "eyelink"}}},
	})
	want := domain.CompetitorsOverview{Makers: 2, Families: []domain.FamilyMakers{
		{Family: "eyelink", Makers: 1}, {Family: "open_spiral"}, {Family: "closed_spiral"}, {Family: "tri_flex_turn_curve"}, {Family: "other", Makers: 1},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("competitors = %+v", got)
	}
}

func voiceCompanies() []domain.Company {
	return []domain.Company{
		{ID: "tribelt", Name: "Tribelt B.V.", Role: domain.RoleTribelt},
		{ID: "a", Name: "A", Role: domain.RoleCompetitor},
		{ID: "b", Name: "B", Role: domain.RoleCompetitor},
		{ID: "c", Name: "C", Role: domain.RoleCompetitor},
		{ID: "d", Name: "D", Role: domain.RoleCompetitor},
		{ID: "e", Name: "E", Role: domain.RoleCompetitor},
		{ID: "sister", Name: "Sister", Role: domain.RoleGroup},
	}
}

func TestAiSearchOfListsTribeltAndTheFourCompetitorsNamedMost(t *testing.T) {
	run := &domain.Run{Answers: []domain.Answer{
		{Named: []string{"e", "d", "tribelt", "sister"}, First: "e"},
		{Named: []string{"d", "c", "tribelt", "gone"}, First: "d"},
		{Named: []string{"b", "c", "gone"}},
		{Named: []string{"a", "b", "c"}},
	}}
	got := domain.AiSearchOf(run, voiceCompanies())
	// c is named most. b, d and tribelt tie on two: d was named first once, and Tribelt follows
	// the competitors it ties with. a and e tie on one; e was named first, so a is left out.
	want := domain.AiSearchOverview{Answers: 4, Companies: []domain.Mentioned{
		{CompanyID: "c", Name: "C", Role: domain.RoleCompetitor, Answers: 3},
		{CompanyID: "d", Name: "D", Role: domain.RoleCompetitor, Answers: 2},
		{CompanyID: "b", Name: "B", Role: domain.RoleCompetitor, Answers: 2},
		{CompanyID: "tribelt", Name: "Tribelt", Role: domain.RoleTribelt, Answers: 2},
		{CompanyID: "e", Name: "E", Role: domain.RoleCompetitor, Answers: 1},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ai search = %+v", got)
	}
}

func TestAiSearchOfListsTribeltWithNoAnswersWhenNoneNamesIt(t *testing.T) {
	got := domain.AiSearchOf(&domain.Run{Answers: []domain.Answer{{Named: []string{"b"}}, {Named: []string{"a", "b"}}}}, voiceCompanies())
	want := domain.AiSearchOverview{Answers: 2, Companies: []domain.Mentioned{
		{CompanyID: "b", Name: "B", Role: domain.RoleCompetitor, Answers: 2},
		{CompanyID: "a", Name: "A", Role: domain.RoleCompetitor, Answers: 1},
		{CompanyID: "tribelt", Name: "Tribelt", Role: domain.RoleTribelt},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ai search = %+v", got)
	}
}

func TestAiSearchOfIsEmptyBeforeAnyAnswer(t *testing.T) {
	for _, run := range []*domain.Run{nil, {ID: "empty"}} {
		if got := domain.AiSearchOf(run, voiceCompanies()); got.Answers != 0 || got.Companies == nil || len(got.Companies) != 0 {
			t.Errorf("%v: ai search = %+v", run, got)
		}
	}
}

func TestOverviewOfCountsEveryTile(t *testing.T) {
	got := domain.OverviewOf(
		[]domain.Company{{ID: "a", Name: "A", Role: domain.RoleCompetitor, Researched: true}},
		[]domain.MapTerm{{Text: "spiral belt", Language: "en", UsedByTribelt: true}},
		&domain.Run{Answers: []domain.Answer{{Named: []string{"a"}}}},
	)
	if got.Terms.Total != 1 || got.Competitors.Makers != 1 || got.AiSearch.Answers != 1 || len(got.AiSearch.Companies) != 2 {
		t.Errorf("overview = %+v", got)
	}
}

func TestCreditsTribeltNamesTribeltOrOneOfItsProductLines(t *testing.T) {
	for credits, want := range map[string]bool{
		"Tribelt":                       true,
		"Tribelt (Tri-Flex)":            true,
		"  Tribelt  (variabler Spaltweite) ": true,
		"Tribelt B.V.":                  false,
		"Tribeltech":                    false,
		"Märtens (Tribelt)":             false,
		"Tribelt (a) b)":                false,
		"(Tribelt)":                     false,
		"Tribelt)":                      false,
	} {
		if got := domain.CreditsTribelt([]string{"Märtens", credits}); got != want {
			t.Errorf("%q: %v", credits, got)
		}
	}
	if domain.CreditsTribelt(nil) {
		t.Error("no credits name Tribelt")
	}
}
