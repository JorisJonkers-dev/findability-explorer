package domain_test

import (
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

func companies() []domain.Company {
	return []domain.Company{
		{ID: "tribelt", Name: "Tribelt B.V.", Role: domain.RoleTribelt},
		{ID: "de-spiraal", Name: "De Spiraal", Role: domain.RoleGroup},
		{ID: "tribelt-mirror", Name: "Mirror of tribelt.nl", Role: domain.RoleMirror},
		{ID: "ashworth", Name: "Ashworth", Role: domain.RoleCompetitor},
		{ID: "martens", Name: "Märtens", Role: domain.RoleCompetitor},
		{ID: "twentebelt", Name: "Twentebelt", Role: domain.RoleCompetitor},
	}
}

func TestMedianIsTheMiddleValueOrTheMeanOfTheTwo(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []float64
		want   float64
		ok     bool
	}{
		{"none", nil, 0, false},
		{"one", []float64{4}, 4, true},
		{"odd, unsorted", []float64{9, 1, 5}, 5, true},
		{"even", []float64{1, 2, 10, 4}, 3, true},
		{"an outlier does not move it", []float64{1, 2, 3, 1000}, 2.5, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := domain.Median(tc.values)
			if got != tc.want || ok != tc.ok {
				t.Errorf("Median(%v) = %v %v, want %v %v", tc.values, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestReadinessScoreCountsTheFiveChecks(t *testing.T) {
	for _, tc := range []struct {
		name string
		r    domain.Readiness
		want float64
	}{
		{"nothing", domain.Readiness{Crawlers: domain.AccessUnknown}, 0},
		{"crawlers kept out", domain.Readiness{Crawlers: domain.AccessClosed, Sitemap: true}, 1},
		{"some crawlers kept out is half", domain.Readiness{Crawlers: domain.AccessSome, Sitemap: true}, 1.5},
		{"everything", domain.Readiness{Crawlers: domain.AccessOpen, Sitemap: true, LLMs: true, LLMsFull: true, Markdown: true}, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.r.ReadinessScore(); got != tc.want {
				t.Errorf("score = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSharesCountAnswersAndWebAnswers(t *testing.T) {
	answers := []domain.Answer{
		{Mode: domain.ModeKnowledge, Named: []string{"ashworth", "tribelt"}},
		{Mode: domain.ModeWeb, Named: []string{"tribelt"}, Cited: []string{"tribelt"}},
		{Mode: domain.ModeWeb, Named: []string{"ashworth"}},
		{Mode: domain.ModeKnowledge},
	}
	if v, ok := domain.ShareOfVoice(answers, "tribelt"); v != 0.5 || !ok {
		t.Errorf("share of voice = %v %v", v, ok)
	}
	if v, ok := domain.CitationShare(answers, "tribelt"); v != 0.5 || !ok {
		t.Errorf("citation share = %v %v", v, ok)
	}
	if _, ok := domain.CitationShare(answers[:1], "tribelt"); ok {
		t.Error("a run without web answers has no citation share")
	}
	if _, ok := domain.ShareOfVoice(nil, "tribelt"); ok {
		t.Error("no answers, no share of voice")
	}
	run := domain.Run{Answers: answers}
	if web := run.Keep(func(a domain.Answer) bool { return a.Mode == domain.ModeWeb }); len(web) != 2 {
		t.Errorf("kept %d web answers", len(web))
	}
}

func facts() domain.Facts {
	return domain.Facts{
		Run: &domain.Run{Answers: []domain.Answer{
			{Mode: domain.ModeKnowledge, Named: []string{"ashworth", "tribelt", "de-spiraal"}},
			{Mode: domain.ModeWeb, Named: []string{"ashworth", "martens", "de-spiraal"}, Cited: []string{"de-spiraal", "martens"}},
			{Mode: domain.ModeWeb, Named: []string{"martens", "de-spiraal"}, Cited: []string{"tribelt", "de-spiraal"}},
			{Mode: domain.ModeKnowledge, Named: []string{"de-spiraal"}},
		}},
		Readiness: map[string]domain.Readiness{
			"tribelt":        {Crawlers: domain.AccessOpen, Sitemap: true, SitemapPages: 124},
			"tribelt-mirror": {Crawlers: domain.AccessOpen, Sitemap: true, LLMs: true, LLMsFull: true, Markdown: true, SitemapPages: 9000},
			"ashworth":       {Crawlers: domain.AccessSome, Sitemap: true, LLMs: true, SitemapPages: 2140},
			"martens":        {Crawlers: domain.AccessOpen, Sitemap: true, LLMs: true, LLMsFull: true, Markdown: true, SitemapPages: 300},
		},
		TermsUsed: map[string]int{"tribelt": 38, "ashworth": 12, "martens": 30, "twentebelt": 30, "de-spiraal": 90},
	}
}

func TestStandingsCompareTribeltWithTheCompetitorsOnly(t *testing.T) {
	got := domain.Standings(companies(), facts())
	want := []struct {
		measure     domain.Measure
		tribelt     *float64
		median      *float64
		leader      string
		value       float64
		competitors int
		trails      bool
		leads       bool
	}{
		// De Spiraal is named in every answer, but a group company is never the leader.
		{domain.MeasureShareOfVoice, new(0.25), new(0.5), "ashworth", 0.5, 3, true, false},
		// Märtens and De Spiraal are cited as often; only Märtens is a competitor.
		{domain.MeasureCitationShare, new(0.5), new(0.0), "martens", 0.5, 3, false, true},
		// Twentebelt was never scanned for readiness, so it is not in the median; the mirror never counts.
		{domain.MeasureReadiness, new(2.0), new(3.75), "martens", 5, 2, true, false},
		{domain.MeasureSitemapPages, new(124.0), new(1220.0), "ashworth", 2140, 2, true, false},
		// Märtens and Twentebelt tie on 30; Tribelt is ahead.
		{domain.MeasureTermsUsed, new(38.0), new(30.0), "martens", 30, 3, false, true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d standings", len(got))
	}
	for i, w := range want {
		g := got[i]
		if g.Measure != w.measure || *g.Tribelt != *w.tribelt || *g.Median != *w.median || g.Leader.CompanyID != w.leader ||
			g.Leader.Value != w.value || g.Competitors != w.competitors || g.Trails() != w.trails || g.Leads() != w.leads {
			t.Errorf("%s: got tribelt %v median %v leader %+v (%d competitors), trails %v leads %v",
				w.measure, *g.Tribelt, *g.Median, *g.Leader, g.Competitors, g.Trails(), g.Leads())
		}
	}
	if n := domain.Trailing(got); n != 3 {
		t.Errorf("trailing = %d, want 3", n)
	}
	if l := domain.LeaderOf(got, domain.MeasureReadiness); l == nil || l.Name != "Märtens" {
		t.Errorf("leader of readiness = %v", l)
	}
	if l := domain.LeaderOf(nil, domain.MeasureReadiness); l != nil {
		t.Errorf("no standings, no leader: %v", l)
	}
}

func TestATieForTheLeadGoesToTheFirstNameAlphabetically(t *testing.T) {
	cs := []domain.Company{
		{ID: "zeta", Name: "Zeta", Role: domain.RoleCompetitor},
		{ID: "abc", Name: "Äbc", Role: domain.RoleCompetitor},
		{ID: "abd", Name: "abd", Role: domain.RoleCompetitor},
	}
	f := domain.Facts{TermsUsed: map[string]int{"zeta": 5, "abc": 5, "abd": 5}}
	s := domain.Standings(cs, f)[4]
	if s.Leader.CompanyID != "abc" {
		t.Errorf("leader = %s, want abc (Äbc folds before abd)", s.Leader.CompanyID)
	}
	same := []domain.Company{{ID: "b", Name: "Same", Role: domain.RoleCompetitor}, {ID: "a", Name: "Same", Role: domain.RoleCompetitor}}
	if s := domain.Standings(same, domain.Facts{TermsUsed: map[string]int{"a": 1, "b": 1}})[4]; s.Leader.CompanyID != "b" {
		t.Errorf("an exact tie keeps the first listed: %s", s.Leader.CompanyID)
	}
}

func TestWithoutDataNothingTrails(t *testing.T) {
	s := domain.Standings(companies(), domain.Facts{})
	for _, st := range s {
		if st.Tribelt != nil || st.Median != nil || st.Leader != nil || st.Trails() || st.Leads() || st.Competitors != 0 {
			t.Errorf("%s: %+v", st.Measure, st)
		}
	}
	// Tribelt not scanned yet: a leader, but Tribelt is not said to trail.
	f := domain.Facts{TermsUsed: map[string]int{"ashworth": 4}}
	if st := domain.Standings(companies(), f)[4]; st.Leader == nil || st.Trails() || st.Leads() {
		t.Errorf("unscanned Tribelt: %+v", st)
	}
	if _, ok := (domain.Facts{}).Value("nonsense", "tribelt"); ok {
		t.Error("an unknown measure has no value")
	}
}

func TestAccessOfCountsTheCrawlersKeptOut(t *testing.T) {
	for _, tc := range []struct {
		readable         bool
		blocked, crawler int
		want             domain.Access
	}{
		{true, 0, 10, domain.AccessOpen},
		{true, 2, 10, domain.AccessSome},
		{true, 10, 10, domain.AccessClosed},
		{false, 0, 10, domain.AccessUnknown},
		{true, 0, 0, domain.AccessUnknown},
	} {
		if got := domain.AccessOf(tc.readable, tc.blocked, tc.crawler); got != tc.want {
			t.Errorf("AccessOf(%v, %d, %d) = %s, want %s", tc.readable, tc.blocked, tc.crawler, got, tc.want)
		}
	}
}
