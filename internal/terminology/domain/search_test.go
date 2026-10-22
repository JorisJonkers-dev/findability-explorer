package domain_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

func sample() []domain.Term {
	return []domain.Term{
		{Text: "Drahtösengliederband", Language: domain.LanguageDe, Family: domain.FamilyEyelink, Tier: domain.TierProduct, UsedBy: []string{"Märtens", "Tribelt"}},
		{Text: "TwenteFlex", Language: domain.LanguageEn, Family: domain.FamilyTriFlex, Tier: domain.TierProduct, UsedBy: []string{"Twentebelt"}},
		{Text: "TwenteSideFlex", Language: domain.LanguageEn, Family: domain.FamilyTriFlex, Tier: domain.TierProduct, UsedBy: []string{"Twentebelt"}},
		{Text: "spiral belt", Language: domain.LanguageEn, Family: domain.FamilyTriFlex, Tier: domain.TierBroad, UsedBy: []string{"Twentebelt"}},
		{Text: "Eye-Link belt", Language: domain.LanguageEn, Family: domain.FamilyEyelink, Tier: domain.TierProduct, UsedBy: []string{"Ashworth"}},
		{Text: "flex belt", Language: domain.LanguageEnUS, Family: domain.FamilyOther, Tier: domain.TierProduct, UsedBy: []string{"Wire Belt"}},
		{Text: "Edelstahl 1.4301", Language: domain.LanguageDe, Family: domain.FamilySpecAll, Tier: domain.TierSpec, UsedBy: []string{"Märtens"}},
	}
}

func texts(terms []domain.Term) []string {
	out := make([]string, 0, len(terms))
	for _, t := range terms {
		out = append(out, t.Text)
	}
	return out
}

func TestSearchRanksExactThenPrefixThenWordThenContainsThenUser(t *testing.T) {
	terms := append(sample(), domain.Term{Text: "Belt", Language: domain.LanguageEn, Family: domain.FamilyAll, Tier: domain.TierBroad})
	got := domain.Search(terms, domain.Query{Text: "belt", Limit: 10})
	// Word matches order by language then family; the last three match only through a user whose
	// name contains "belt" (Tribelt, Twentebelt), German first.
	want := []string{"Belt", "Eye-Link belt", "spiral belt", "flex belt", "Drahtösengliederband", "TwenteFlex", "TwenteSideFlex"}
	if !slices.Equal(texts(got.Items), want) {
		t.Fatalf("Search(belt) = %v, want %v", texts(got.Items), want)
	}
	if got.Total != len(want) {
		t.Fatalf("total = %d, want %d", got.Total, len(want))
	}
	prefix := domain.Search(sample(), domain.Query{Text: "twente", Limit: 10})
	if !slices.Equal(texts(prefix.Items), []string{"TwenteFlex", "TwenteSideFlex", "spiral belt"}) {
		t.Fatalf("Search(twente) = %v", texts(prefix.Items))
	}
	contains := domain.Search(sample(), domain.Query{Text: "glieder", Limit: 10})
	if !slices.Equal(texts(contains.Items), []string{"Drahtösengliederband"}) {
		t.Fatalf("Search(glieder) = %v", texts(contains.Items))
	}
}

func TestSearchFoldsAccentsCaseAndSeparators(t *testing.T) {
	for _, q := range []string{"drahtosengliederband", "DRAHTÖSENGLIEDERBAND", "Drahtösengliederband "} {
		if got := domain.Search(sample(), domain.Query{Text: q, Limit: 5}); len(got.Items) != 1 {
			t.Errorf("Search(%q) found %d terms, want 1", q, len(got.Items))
		}
	}
	if got := domain.Search(sample(), domain.Query{Text: "eye link", Limit: 5}); len(got.Items) != 1 || got.Items[0].Text != "Eye-Link belt" {
		t.Errorf("Search(eye link) = %v", texts(got.Items))
	}
	if got := domain.Search(sample(), domain.Query{Text: "nothing like this", Limit: 5}); got.Total != 0 || len(got.Items) != 0 {
		t.Errorf("Search(nothing) = %+v", got)
	}
}

func TestSearchFiltersAndCountsFacetsWithoutTheirOwnFilter(t *testing.T) {
	got := domain.Search(sample(), domain.Query{Language: domain.LanguageEn, Tier: domain.TierProduct, Limit: 10})
	if !slices.Equal(texts(got.Items), []string{"Eye-Link belt", "TwenteFlex", "TwenteSideFlex"}) {
		t.Fatalf("filtered = %v", texts(got.Items))
	}
	count := func(fs []domain.FacetCount, v string) int {
		for _, f := range fs {
			if f.Value == v {
				return f.Count
			}
		}
		t.Fatalf("facet %q missing", v)
		return 0
	}
	// Languages ignore the language filter but keep the tier filter: product terms per language.
	if count(got.Facets.Languages, "de") != 1 || count(got.Facets.Languages, "en") != 3 || count(got.Facets.Languages, "en-US") != 1 {
		t.Errorf("language facets = %+v", got.Facets.Languages)
	}
	// Tiers ignore the tier filter but keep the language filter: English terms per tier.
	if count(got.Facets.Tiers, "product") != 3 || count(got.Facets.Tiers, "broad") != 1 || count(got.Facets.Tiers, "spec") != 0 {
		t.Errorf("tier facets = %+v", got.Facets.Tiers)
	}
	if count(got.Facets.Families, "tri_flex_turn_curve") != 2 || count(got.Facets.Families, "eyelink") != 1 {
		t.Errorf("family facets = %+v", got.Facets.Families)
	}
	if len(got.Facets.Languages) != len(domain.Languages()) || len(got.Facets.Families) != len(domain.Families()) || len(got.Facets.Tiers) != len(domain.Tiers()) {
		t.Error("a facet does not list every value")
	}

	family := domain.Search(sample(), domain.Query{Family: domain.FamilySpecAll, Limit: 10})
	if !slices.Equal(texts(family.Items), []string{"Edelstahl 1.4301"}) {
		t.Errorf("family filter = %v", texts(family.Items))
	}
	user := domain.Search(sample(), domain.Query{User: "martens", Limit: 10})
	if !slices.Equal(texts(user.Items), []string{"Drahtösengliederband", "Edelstahl 1.4301"}) {
		t.Errorf("user filter = %v", texts(user.Items))
	}
}

func TestSearchOrdersEqualScoresByLanguageFamilyAndText(t *testing.T) {
	got := domain.Search(sample(), domain.Query{Limit: 3})
	if !slices.Equal(texts(got.Items), []string{"Drahtösengliederband", "Edelstahl 1.4301", "Eye-Link belt"}) {
		t.Fatalf("unfiltered = %v", texts(got.Items))
	}
	if got.Total != len(sample()) {
		t.Fatalf("total = %d", got.Total)
	}
	none := domain.Search(sample(), domain.Query{Limit: 0})
	if len(none.Items) != 0 || none.Total != len(sample()) {
		t.Fatalf("limit 0 = %d items, total %d", len(none.Items), none.Total)
	}
}

func TestParse(t *testing.T) {
	if l, err := domain.ParseLanguage("en-US"); err != nil || l != domain.LanguageEnUS {
		t.Errorf("ParseLanguage = %q, %v", l, err)
	}
	if f, err := domain.ParseFamily("spec_all"); err != nil || f != domain.FamilySpecAll {
		t.Errorf("ParseFamily = %q, %v", f, err)
	}
	if tr, err := domain.ParseTier("broad"); err != nil || tr != domain.TierBroad {
		t.Errorf("ParseTier = %q, %v", tr, err)
	}
	if _, err := domain.ParseTier("vague"); !errors.Is(err, domain.ErrUnknownValue) {
		t.Errorf("ParseTier(vague) error = %v", err)
	}
}

func TestFold(t *testing.T) {
	if got := domain.Fold("  Großer   Ofen-Band "); got != "grosser ofen band" {
		t.Fatalf("Fold = %q", got)
	}
}

func TestSortCompaniesPutsCompetitorsFirst(t *testing.T) {
	got := domain.SortCompanies([]domain.Company{
		{Name: "De Spiraal", GroupCompany: true},
		{Name: "unibelt"},
		{Name: "Ashworth"},
		{Name: "Lesjöfors", GroupCompany: true},
	})
	var names []string
	for _, c := range got {
		names = append(names, c.Name)
	}
	if !slices.Equal(names, []string{"Ashworth", "unibelt", "De Spiraal", "Lesjöfors"}) {
		t.Fatalf("sorted = %v", names)
	}
}
