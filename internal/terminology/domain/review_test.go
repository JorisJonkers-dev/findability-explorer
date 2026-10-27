package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

var at = time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)

func obs(scan, company, phrase, lang string, score float64, pages int, tier, family, closest string, started time.Time) domain.Observation {
	return domain.Observation{
		ScanID: scan, CompanyID: company, CompanyName: strings.ToUpper(company[:1]) + company[1:], ScanURL: "https://" + company + ".test/",
		StartedAt: started, Phrase: phrase, Language: lang, Score: score, Pages: pages, Tier: tier, Family: family, Closest: closest,
	}
}

func TestCandidatesAggregateAcrossCompanies(t *testing.T) {
	observed := []domain.Observation{
		obs("s1", "zeta", "spiral freezer", "en", 10, 2, "application", "tri_flex_turn_curve", "spiral belt", at.Add(-2*time.Hour)),
		obs("s2", "alpha", "spiral freezer", "en", 30, 1, "application", "tri_flex_turn_curve", "spiral belt", at),
		obs("s2", "alpha", "spiral freezer", "de", 5, 5, "broad", "", "", at),
		obs("s3", "alpha", "eyelink belts", "en", 50, 3, "product", "eyelink", "eyelink belt", at), // in the map now
		obs("s3", "alpha", "odd phrase", "en", 1, 1, "nonsense", "nonsense", "", at),
		obs("s3", "alpha", "rejected one", "en", 2, 1, "broad", "", "", at),
	}
	terms := []domain.Term{{Text: "eyelink belt / eye-link belt (s)", Language: domain.LanguageEn}}
	decisions := []domain.Decision{{CandidateID: domain.CandidateID("Rejected One", "en"), Status: domain.StatusRejected, By: "j", At: at}}
	got := domain.Candidates(observed, decisions, terms)
	if len(got) != 4 {
		t.Fatalf("candidates %+v", got)
	}
	// The German use of the phrase is a candidate of its own.
	c := got[0]
	if c.Phrase != "spiral freezer" || c.Status != domain.StatusPending || c.Scans != 2 || c.BestScore != 30 || c.MaxPages != 2 ||
		c.Language != "en" || strings.Join(c.Languages, " ") != "en" || c.Companies[0].Name != "Alpha" || c.Companies[1].ID != "zeta" ||
		c.SuggestedTier != domain.TierApplication || c.SuggestedFamily != domain.FamilyTriFlex || c.Closest != "spiral belt" ||
		!c.LastSeenAt.Equal(at) || c.Source != "https://alpha.test/" || c.Decision != nil || len(c.ID) != 16 {
		t.Fatalf("spiral freezer = %+v", c)
	}
	// An unknown judgement suggests a broad term of no family.
	byPhrase := map[string]domain.Candidate{}
	for _, c := range got {
		byPhrase[c.Phrase] = c
	}
	if odd := byPhrase["odd phrase"]; odd.SuggestedTier != domain.TierBroad || odd.SuggestedFamily != "" {
		t.Fatalf("odd phrase = %+v", odd)
	}
	if r := byPhrase["rejected one"]; r.Status != domain.StatusRejected || r.Decision == nil || r.Decision.By != "j" {
		t.Fatalf("rejected one = %+v", r)
	}
	list, counts := domain.Count(append(got, domain.Candidate{Status: domain.StatusAccepted}), domain.StatusPending)
	if len(list) != 3 || counts != (domain.Counts{Pending: 3, Accepted: 1, Rejected: 1, Unreviewed: 3}) {
		t.Fatalf("Count = %d, %+v", len(list), counts)
	}
}

func TestCandidatesBreakTiesByPhrase(t *testing.T) {
	got := domain.Candidates([]domain.Observation{
		obs("s", "a", "b phrase", "en", 1, 1, "broad", "", "", at),
		obs("s", "a", "a phrase", "en", 1, 1, "broad", "", "", at),
		obs("s", "a", "a phrase", "en", 1, 1, "spec", "spec_all", "", at),
	}, nil, nil)
	if got[0].Phrase != "a phrase" || got[0].SuggestedTier != domain.TierBroad {
		t.Fatalf("candidates %+v", got)
	}
}

func TestNewTerm(t *testing.T) {
	existing := []domain.Term{{Text: "Spiral belt", Language: domain.LanguageEn}}
	ok := domain.Acceptance{Text: "  spiral  freezer belt ", Language: "en", Family: "tri_flex_turn_curve", Tier: "application", UsedBy: []string{" Twentebelt "}}
	term, err := domain.NewTerm(ok, strings.Repeat("s", 600), existing)
	if err != nil || term.Text != "spiral freezer belt" || term.UsedBy[0] != "Twentebelt" || term.Origin != domain.OriginReview || len(term.Source) != 500 {
		t.Fatalf("NewTerm = %+v, %v", term, err)
	}
	bad := func(mut func(*domain.Acceptance)) domain.Acceptance {
		a := ok
		a.UsedBy = []string{"x"}
		mut(&a)
		return a
	}
	cases := map[string]struct {
		a    domain.Acceptance
		want error
	}{
		"empty":          {bad(func(a *domain.Acceptance) { a.Text = " " }), domain.ErrInvalidTerm},
		"too long":       {bad(func(a *domain.Acceptance) { a.Text = strings.Repeat("x", 201) }), domain.ErrInvalidTerm},
		"many companies": {bad(func(a *domain.Acceptance) { a.UsedBy = make([]string, 21) }), domain.ErrInvalidTerm},
		"empty company":  {bad(func(a *domain.Acceptance) { a.UsedBy = []string{" "} }), domain.ErrInvalidTerm},
		"language":       {bad(func(a *domain.Acceptance) { a.Language = "xx" }), domain.ErrUnknownValue},
		"other language": {bad(func(a *domain.Acceptance) { a.Language = "fr" }), domain.ErrUnknownValue},
		"family":         {bad(func(a *domain.Acceptance) { a.Family = "xx" }), domain.ErrUnknownValue},
		"tier":           {bad(func(a *domain.Acceptance) { a.Tier = "xx" }), domain.ErrUnknownValue},
		"exists":         {bad(func(a *domain.Acceptance) { a.Text = "SPIRAL BELT" }), domain.ErrTermExists},
	}
	for name, c := range cases {
		if _, err := domain.NewTerm(c.a, "", existing); !errors.Is(err, c.want) {
			t.Errorf("%s: NewTerm = %v, want %v", name, err, c.want)
		}
	}
}
