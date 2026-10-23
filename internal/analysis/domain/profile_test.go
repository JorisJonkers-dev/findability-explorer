package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func kw(phrase, lang string, score float64, pages int, tier, family string) domain.Keyword {
	return domain.Keyword{Phrase: phrase, Language: lang, Score: score, Pages: pages, Classification: domain.Classification{Tier: tier, Family: family}}
}

func TestGroupOf(t *testing.T) {
	for c, want := range map[domain.Classification]domain.Group{
		{Tier: "spec", Family: "eyelink"}:                    domain.GroupSpec,
		{Tier: "broad", Family: "spec_all"}:                  domain.GroupSpec,
		{Tier: "product", Family: "open_spiral"}:             domain.GroupOpenSpiral,
		{Tier: "product", Family: "other"}:                   domain.GroupOther,
		{Tier: "broad", Family: "all"}:                       domain.GroupBroad,
		{Tier: "broad"}:                                      domain.GroupBroad,
		{Tier: "application", Family: "tri_flex_turn_curve"}: domain.GroupTriFlex,
	} {
		if got := domain.GroupOf(c); got != want {
			t.Errorf("GroupOf(%+v) = %s, want %s", c, got, want)
		}
	}
}

func TestBuildProfile(t *testing.T) {
	latest := domain.Analysis{
		ID: uuid.New(), Number: 4, Status: domain.StatusDone,
		Pages: []domain.PageSummary{{URL: "u"}},
		Keywords: []domain.Keyword{
			kw("eyelink belt", "en", 10, 2, "product", "eyelink"),
			kw("drahtösengliederband", "de", 30, 3, "product", "eyelink"),
			kw("metal belts", "en", 5, 1, "broad", "all"),
			kw("wire diameter", "en", 3, 1, "spec", "spec_all"),
		},
	}
	p := domain.BuildProfile(latest, 3, map[domain.PhraseKey]int{{Phrase: "eyelink belt", Language: "en"}: 1})
	if p.Scan.Number != 4 || p.Scan.Pages != nil || p.Scan.Keywords != nil || p.FinishedScans != 3 {
		t.Fatalf("scan %+v", p.Scan)
	}
	if strings.Join(p.Languages, " ") != "en de" || len(p.Groups) != 7 {
		t.Fatalf("languages %v, %d groups", p.Languages, len(p.Groups))
	}
	eyelink := p.Groups[0]
	if eyelink.Group != domain.GroupEyelink || len(eyelink.Terms) != 2 || eyelink.Terms[0].Phrase != "drahtösengliederband" ||
		eyelink.Terms[0].FirstScan != 4 || eyelink.Terms[1].FirstScan != 1 {
		t.Fatalf("eyelink %+v", eyelink)
	}
	if p.Groups[1].Terms == nil || len(p.Groups[5].Terms) != 1 || len(p.Groups[6].Terms) != 1 {
		t.Fatalf("groups %+v", p.Groups)
	}
}

func TestFinished(t *testing.T) {
	got := domain.Finished([]domain.Analysis{{Status: domain.StatusDone}, {Status: domain.StatusFailed}, {Status: domain.StatusRunning}})
	if len(got) != 1 {
		t.Fatalf("Finished = %+v", got)
	}
}

func TestCompare(t *testing.T) {
	from := domain.Analysis{Keywords: []domain.Keyword{
		kw("eyelink belt", "en", 10, 2, "product", "eyelink"),            // moves by score
		kw("spiral belt", "en", 10, 2, "product", "tri_flex_turn_curve"), // moves by pages
		kw("wire mesh", "en", 20, 3, "product", "other"),                 // stays: +4 points
		kw("old term", "en", 8, 1, "broad", ""),                          // removed
		kw("older term", "en", 9, 1, "broad", ""),                        // removed
		kw("balanced weave", "en", 40, 3, "product", "open_spiral"),      // moves down
	}}
	to := domain.Analysis{Keywords: []domain.Keyword{
		kw("eyelink belt", "en", 20, 2, "product", "eyelink"),
		kw("spiral belt", "en", 11, 4, "product", "tri_flex_turn_curve"),
		kw("wire mesh", "en", 24, 3, "product", "other"),
		kw("new term", "de", 6, 1, "spec", "spec_all"),
		kw("newer term", "de", 7, 1, "spec", "spec_all"),
		kw("balanced weave", "en", 4, 1, "product", "open_spiral"),
		kw("closed spiral", "en", 2, 1, "product", "closed_spiral"),
		kw("closed spiral two", "en", 1, 2, "product", "closed_spiral"),
	}}
	groups := domain.Compare(from, to)
	byGroup := map[domain.Group]domain.ComparisonGroup{}
	for _, g := range groups {
		byGroup[g.Group] = g
	}
	if len(groups) != 7 || groups[0].Group != domain.GroupEyelink {
		t.Fatalf("groups %+v", groups)
	}
	if m := byGroup[domain.GroupEyelink].Moved; len(m) != 1 || m[0].From.Score != 10 || m[0].To.Score != 20 {
		t.Errorf("eyelink moved %+v", m)
	}
	if m := byGroup[domain.GroupTriFlex].Moved; len(m) != 1 || m[0].To.Pages != 4 {
		t.Errorf("tri-flex moved %+v", m)
	}
	if m := byGroup[domain.GroupOpenSpiral].Moved; len(m) != 1 || m[0].To.Pages != 1 {
		t.Errorf("open spiral moved %+v", m)
	}
	if g := byGroup[domain.GroupOther]; len(g.Moved)+len(g.Added)+len(g.Removed) != 0 {
		t.Errorf("other changed %+v", g)
	}
	if r := byGroup[domain.GroupBroad].Removed; len(r) != 2 || r[0].Phrase != "older term" || r[0].To != nil {
		t.Errorf("broad removed %+v", r)
	}
	if a := byGroup[domain.GroupSpec].Added; len(a) != 2 || a[0].Phrase != "newer term" || a[0].From != nil {
		t.Errorf("spec added %+v", a)
	}
	if a := byGroup[domain.GroupClosedSpiral].Added; len(a) != 2 {
		t.Errorf("closed spiral added %+v", a)
	}
}

func TestCompareSortsMovesByShift(t *testing.T) {
	from := domain.Analysis{Keywords: []domain.Keyword{kw("a", "en", 10, 1, "product", "eyelink"), kw("b", "en", 10, 1, "product", "eyelink")}}
	to := domain.Analysis{Keywords: []domain.Keyword{kw("a", "en", 20, 1, "product", "eyelink"), kw("b", "en", 50, 1, "product", "eyelink")}}
	if m := domain.Compare(from, to)[0].Moved; len(m) != 2 || m[0].Phrase != "b" {
		t.Fatalf("moved %+v", m)
	}
}

func TestPick(t *testing.T) {
	a := domain.Analysis{ID: uuid.New(), Number: 5}
	b := domain.Analysis{ID: uuid.New(), Number: 3}
	c := domain.Analysis{ID: uuid.New(), Number: 1}
	finished := []domain.Analysis{a, b, c}
	check := func(from, to uuid.UUID, wantFrom, wantTo int) {
		t.Helper()
		x, y, err := domain.Pick(finished, from, to)
		if err != nil || x.Number != wantFrom || y.Number != wantTo {
			t.Errorf("Pick = %d, %d, %v; want %d, %d", x.Number, y.Number, err, wantFrom, wantTo)
		}
	}
	check(uuid.Nil, uuid.Nil, 3, 5)
	check(uuid.Nil, b.ID, 1, 3)
	check(c.ID, uuid.Nil, 1, 5)
	check(a.ID, c.ID, 1, 5) // given the wrong way round
	for name, ids := range map[string][2]uuid.UUID{
		"same":          {a.ID, a.ID},
		"unknown from":  {uuid.New(), uuid.Nil},
		"unknown to":    {uuid.Nil, uuid.New()},
		"nothing older": {uuid.Nil, c.ID},
	} {
		if _, _, err := domain.Pick(finished, ids[0], ids[1]); !errors.Is(err, domain.ErrNotComparable) {
			t.Errorf("%s: Pick error = %v", name, err)
		}
	}
	if _, _, err := domain.Pick(nil, uuid.Nil, uuid.Nil); !errors.Is(err, domain.ErrNotComparable) {
		t.Errorf("Pick of nothing = %v", err)
	}
}
