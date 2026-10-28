package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

func TestCrawlersSayWhatEachFeeds(t *testing.T) {
	today, future := 0, 0
	for _, c := range domain.Crawlers() {
		if c.Lab == "" || c.Gets == "" {
			t.Errorf("%s has no lab or purpose", c.Bot)
		}
		switch c.Feeds {
		case domain.FeedsToday:
			if future > 0 {
				t.Errorf("%s feeds answers today but comes after a future one", c.Bot)
			}
			today++
		case domain.FeedsFuture:
			future++
		}
	}
	if today != 6 || future != 4 {
		t.Errorf("today %d, future %d", today, future)
	}
}

func TestMostNamedAndTheTrend(t *testing.T) {
	old := domain.Run{ID: "old", StartedAt: time.Date(2026, 10, 26, 6, 0, 0, 0, time.UTC), Answers: []domain.Answer{
		{Named: []string{"twentebelt"}}, {Named: []string{"tribelt"}},
	}}
	newest := domain.Run{ID: "new", Answers: []domain.Answer{
		{Named: []string{"ashworth", "martens", "de-spiraal"}},
		{Named: []string{"martens", "twentebelt", "de-spiraal"}},
		{Named: []string{"tribelt", "de-spiraal"}},
		{Named: []string{"ashworth"}},
	}}
	most := domain.MostNamed(newest, companies(), 2)
	if len(most) != 2 || most[0].CompanyID != "ashworth" || most[1].CompanyID != "martens" || most[0].Share != 0.5 {
		t.Errorf("most named = %+v", most)
	}
	if all := domain.MostNamed(newest, companies(), 10); len(all) != 3 {
		t.Errorf("a competitor no answer names is left out, the group never counts: %+v", all)
	}
	trend := domain.Trend([]domain.Run{newest, old}, []string{"tribelt", "twentebelt"})
	if len(trend) != 2 || trend[0].RunID != "old" || trend[0].Shares[0].Share != 0.5 || trend[1].Shares[1].Share != 0.25 {
		t.Errorf("trend = %+v", trend)
	}
}

func TestKeepOutNamesTheCompetitorsThatBlockCrawlers(t *testing.T) {
	readiness := map[string]domain.Readiness{
		"tribelt":    {Blocked: []string{"CCBot"}},
		"ashworth":   {Blocked: []string{"GPTBot"}},
		"martens":    {Blocked: []string{"GPTBot", "CCBot"}},
		"twentebelt": {},
		"de-spiraal": {Blocked: []string{"GPTBot"}},
	}
	got := domain.KeepOut(companies(), readiness)
	if len(got) != 2 || got[0].Name != "Märtens" || got[1].Name != "Ashworth" {
		t.Errorf("keeping out = %+v", got)
	}
}

func TestRankingKeepsTribeltInView(t *testing.T) {
	readiness := map[string]domain.Readiness{
		"tribelt":    {Crawlers: domain.AccessOpen},
		"ashworth":   {Crawlers: domain.AccessOpen, Sitemap: true},
		"martens":    {Crawlers: domain.AccessOpen, Sitemap: true, LLMs: true},
		"twentebelt": {Crawlers: domain.AccessOpen, Sitemap: true},
		"de-spiraal": {Crawlers: domain.AccessOpen, Sitemap: true, LLMs: true, LLMsFull: true},
	}
	names := func(rs []domain.Ranked) []string {
		var out []string
		for _, r := range rs {
			out = append(out, r.Name)
		}
		return out
	}
	if got := names(domain.Ranking(companies(), readiness, 10)); len(got) != 4 || got[0] != "Märtens" || got[3] != "Tribelt B.V." {
		t.Errorf("everyone = %v", got)
	}
	// Two rows: Märtens leads, and Tribelt takes the last row although Ashworth scores higher.
	if got := names(domain.Ranking(companies(), readiness, 2)); len(got) != 2 || got[0] != "Märtens" || got[1] != "Tribelt B.V." {
		t.Errorf("two rows = %v", got)
	}
	if got := names(domain.Ranking(companies(), readiness, 3)); got[0] != "Märtens" || got[1] != "Ashworth" || got[2] != "Tribelt B.V." {
		t.Errorf("three rows = %v", got)
	}
	delete(readiness, "tribelt")
	if got := names(domain.Ranking(companies(), readiness, 2)); got[1] != "Ashworth" {
		t.Errorf("without Tribelt = %v", got)
	}
}

func TestTargetsHaveDefaultsAndRules(t *testing.T) {
	d := domain.DefaultTargets()
	if len(d) != len(domain.Targets()) || d[domain.TargetShareOfVoice] != 0.40 || d[domain.TargetTermsUsed] != 90 || d[domain.TargetCitations] != 50 {
		t.Errorf("defaults = %v", d)
	}
	// Every default is above where Tribelt stood when they were set (October 2026): a target
	// already reached is no goal.
	for target, then := range map[domain.Target]float64{
		domain.TargetShareOfVoice: 0.05, domain.TargetTwenteShareOfVoice: 0.16, domain.TargetReadiness: 3, domain.TargetTermsUsed: 45, domain.TargetCitations: 28,
	} {
		if d[target] <= then {
			t.Errorf("the default for %s, %v, is not above %v", target, d[target], then)
		}
	}
	for _, tc := range []struct {
		target domain.Target
		value  float64
		ok     bool
	}{
		{domain.TargetShareOfVoice, 0.4, true},
		{domain.TargetShareOfVoice, 1.2, false},
		{domain.TargetTwenteShareOfVoice, 0, false},
		{domain.TargetCitations, 20, true},
		{domain.TargetCitations, 0.5, false},
		{domain.TargetReadiness, 4.5, true},
		{domain.TargetReadiness, 4.2, false},
		{domain.TargetReadiness, 6, false},
		{domain.TargetTermsUsed, 35, true},
		{domain.TargetTermsUsed, 35.5, false},
		{domain.TargetTermsUsed, 20_000, false},
		{"nonsense", 1, false},
	} {
		v, err := domain.CheckTarget(tc.target, tc.value)
		if (err == nil) != tc.ok || (tc.ok && v != tc.value) || (!tc.ok && !errors.Is(err, domain.ErrInvalid)) {
			t.Errorf("CheckTarget(%s, %v) = %v, %v", tc.target, tc.value, v, err)
		}
	}
	// The readiness target is every check by default, and no more can be asked.
	if most, ok := domain.MaxOf(domain.TargetReadiness); !ok || most != float64(domain.ReadinessChecks()) || d[domain.TargetReadiness] != most {
		t.Errorf("readiness most %v, default %v", most, d[domain.TargetReadiness])
	}
	if most, ok := domain.MaxOf(domain.TargetTwenteShareOfVoice); !ok || most != 1 {
		t.Errorf("a share's most = %v", most)
	}
	if _, ok := domain.MaxOf(domain.TargetCitations); ok {
		t.Error("a count has a most")
	}
}
