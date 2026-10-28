package app_test

import (
	"context"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

// Every page shows one number for one statistic. These tests read the same data through each use
// case and check that the home page, Catching up, the plan, AI search and the products agree.

// agreeingWorld has Tribelt named in one answer of four, one about Twente of two, cited in one web
// answer of two, five map terms on its newest scan and two more the research credits it with.
func agreeingWorld() *world {
	w := newWorld()
	w.runs[0].Answers = []domain.Answer{
		{Mode: domain.ModeWeb, Scope: domain.ScopeTwente, Family: "eyelink", Named: []string{"tribelt", "ashworth"}, Cited: []string{"tribelt"}},
		{Mode: domain.ModeWeb, Scope: domain.ScopeTwente, Family: "eyelink", Named: []string{"ashworth"}},
		{Mode: domain.ModeKnowledge, Scope: "germany", Family: "eyelink", Named: []string{"martens", "ashworth"}},
		{Mode: domain.ModeKnowledge, Scope: "english", Family: "other", Named: []string{"ashworth"}},
	}
	w.terms = termMap(12, 5)
	return w
}

func standing(t *testing.T, s app.Standings, m domain.Measure) float64 {
	t.Helper()
	for _, st := range s.Standings {
		if st.Measure == m && st.Tribelt != nil {
			return *st.Tribelt
		}
	}
	t.Fatalf("no value of Tribelt on %s", m)
	return 0
}

func watchedNow(t *testing.T, p app.Plan, target domain.Target) float64 {
	t.Helper()
	for _, w := range p.Watched {
		if w.Target == target && w.Now != nil {
			return *w.Now
		}
	}
	t.Fatalf("no value now on %s", target)
	return 0
}

func readAll(t *testing.T, w *world) (domain.Overview, app.Standings, app.Plan, app.AiSearch, app.Products) {
	t.Helper()
	svc, ctx := service(w), context.Background()
	overview, err := svc.Overview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	standings, err := svc.Standings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := svc.Plan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ai, err := svc.AiSearch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	products, err := svc.Products(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return overview, standings, plan, ai, products
}

func TestEveryPageShowsTheSameShareOfAnswersNamingTribelt(t *testing.T) {
	overview, standings, plan, ai, _ := readAll(t, agreeingWorld())
	var home float64
	for _, c := range overview.AiSearch.Companies {
		if c.CompanyID == domain.TribeltID {
			home = c.Share
		}
	}
	catchingUp := standing(t, standings, domain.MeasureShareOfVoice)
	watched := watchedNow(t, plan, domain.TargetShareOfVoice)
	trend := ai.Trend[len(ai.Trend)-1].Shares[0]
	if home != 0.25 || catchingUp != home || watched != home || trend.CompanyID != domain.TribeltID || trend.Share != home {
		t.Errorf("home %v, Catching up %v, plan %v, AI search %v", home, catchingUp, watched, trend.Share)
	}
}

func TestThePlanWatchesTheSameTwenteShareAndCitationsAsTheAnswers(t *testing.T) {
	_, standings, plan, _, products := readAll(t, agreeingWorld())
	if twente := watchedNow(t, plan, domain.TargetTwenteShareOfVoice); twente != 0.5 {
		t.Errorf("Twente share = %v", twente)
	}
	// The eyelink answers about Twente are the Twente answers: the products page agrees.
	for _, s := range products.Products[0].ByScope {
		if s.Scope == domain.ScopeTwente && (s.Answers != 2 || s.Named != 1) {
			t.Errorf("eyelink in Twente = %+v", s)
		}
	}
	// One web answer of two cites tribelt.nl: the plan's count, Catching up's share.
	if n, share := watchedNow(t, plan, domain.TargetCitations), standing(t, standings, domain.MeasureCitationShare); n != 1 || share != 0.5 {
		t.Errorf("citations = %v, citation share = %v", n, share)
	}
}

func TestEveryPageShowsTheSameCountOfTermsTribeltUses(t *testing.T) {
	for _, from := range []domain.TermsSource{domain.TermsFromScan, domain.TermsFromResearch} {
		w := agreeingWorld()
		w.terms.TribeltFrom = from
		overview, standings, plan, _, products := readAll(t, w)
		catchingUp := standing(t, standings, domain.MeasureTermsUsed)
		watched := watchedNow(t, plan, domain.TargetTermsUsed)
		perFamily := 0
		for _, p := range products.Products {
			perFamily += p.TermsUsed
		}
		perLanguage := 0
		for _, l := range overview.Terms.Languages {
			perLanguage += l.Tribelt
		}
		if overview.Terms.Tribelt != 5 || catchingUp != 5 || watched != 5 || perFamily != 5 || perLanguage != 5 {
			t.Errorf("%s: home %d, Catching up %v, plan %v, products %d, languages %d", from, overview.Terms.Tribelt, catchingUp, watched, perFamily, perLanguage)
		}
		// Each page says where the count comes from, the same everywhere.
		if overview.Terms.From != from || standings.TermsFrom != from || plan.TermsFrom != from || products.TermsFrom != from {
			t.Errorf("%s: from %s, %s, %s, %s", from, overview.Terms.From, standings.TermsFrom, plan.TermsFrom, products.TermsFrom)
		}
	}
}

func TestEveryPageShowsTheSameReadinessChecks(t *testing.T) {
	_, standings, plan, ai, _ := readAll(t, agreeingWorld())
	var ranked float64
	for _, r := range ai.Ranking {
		if r.CompanyID == domain.TribeltID {
			ranked = r.Score
		}
	}
	if catchingUp, watched := standing(t, standings, domain.MeasureReadiness), watchedNow(t, plan, domain.TargetReadiness); catchingUp != 1 || watched != 1 || ranked != 1 {
		t.Errorf("Catching up %v, plan %v, AI search %v", catchingUp, watched, ranked)
	}
}

// fieldWorld has Tribelt, its group, the mirror and three competitors on the list: two the desk
// research found and one added since, whose belt names and readiness only its scan knows.
func fieldWorld() *world {
	w := agreeingWorld()
	w.companies = append(w.companies,
		domain.Company{ID: "added", Name: "Added Belts", Role: domain.RoleCompetitor, BeltNames: []domain.BeltName{
			{Name: "Spiral", Family: "open_spiral", FromScan: true}, {Name: "Eye", Family: "eyelink", FromScan: true},
		}},
		domain.Company{ID: "sister", Name: "Sister", Role: domain.RoleGroup, BeltNames: []domain.BeltName{{Name: "S", Family: "eyelink"}}},
		domain.Company{ID: "tribelt-mirror", Name: "Mirror", Role: domain.RoleMirror},
	)
	w.companies[1].BeltNames = []domain.BeltName{{Name: "Omni-Grid", Family: "eyelink"}}
	w.companies[2].BeltNames = []domain.BeltName{{Name: "Eyelink", Family: "eyelink", FromScan: true}}
	w.readiness = map[string]domain.Readiness{
		"tribelt":        {Crawlers: domain.AccessSome, Sitemap: true, SitemapCoverage: new(0.5), Product: true},
		"martens":        {Crawlers: domain.AccessOpen, Sitemap: true, LLMs: true, SitemapCoverage: new(0.8)},
		"added":          {Crawlers: domain.AccessOpen, Sitemap: true, LLMs: true, LLMsFull: true, SitemapCoverage: new(1.0), Organization: true},
		"ashworth":       {Crawlers: domain.AccessClosed},
		"sister":         {Crawlers: domain.AccessOpen, Sitemap: true, LLMs: true, LLMsFull: true, Markdown: true, SitemapCoverage: new(1.0)},
		"tribelt-mirror": {Crawlers: domain.AccessOpen, Sitemap: true, LLMs: true, LLMsFull: true, Markdown: true, SitemapCoverage: new(1.0)},
	}
	return w
}

func competitorsIn(companies []domain.Company) int {
	n := 0
	for _, c := range companies {
		if c.Role == domain.RoleCompetitor {
			n++
		}
	}
	return n
}

func TestEveryPageCountsTheCompetitorsOnTheListNow(t *testing.T) {
	overview, standings, _, _, products := readAll(t, fieldWorld())
	// Ashworth, Märtens and the company added since; not Tribelt, its group or the mirror.
	if overview.Competitors.Makers != 3 || competitorsIn(standings.Companies) != 3 || competitorsIn(products.Companies) != 3 {
		t.Errorf("home %d, Catching up %d, products %d", overview.Competitors.Makers, competitorsIn(standings.Companies), competitorsIn(products.Companies))
	}
	// Each family's makers on the home page are the products page's sellers, scan-matched names included.
	for i, f := range overview.Competitors.Families {
		if p := products.Products[i]; p.Family != f.Family || f.Makers != len(p.Sellers) {
			t.Errorf("%s: home %d, products %d sellers", f.Family, f.Makers, len(p.Sellers))
		}
	}
	if overview.Competitors.Families[0].Makers != 3 || overview.Competitors.Families[1].Makers != 1 {
		t.Errorf("families = %+v", overview.Competitors.Families)
	}
}

func TestEveryPageNamesTheSameLeadersAndTypicalCompetitor(t *testing.T) {
	_, standings, plan, ai, _ := readAll(t, fieldWorld())
	leaders := map[domain.Measure]*domain.Leader{}
	for _, s := range standings.Standings {
		leaders[s.Measure] = s.Leader
	}
	// The plan names each improvement's measure's leader, as Catching up does.
	for _, it := range plan.Items {
		want := leaders[it.Measure]
		if (want == nil) != (it.Leader == nil) || (want != nil && *want != *it.Leader) {
			t.Errorf("%s: plan %+v, Catching up %+v", it.ID, it.Leader, want)
		}
	}
	// The company added since leads the readiness checks and the sitemap coverage; the group and
	// the mirror, though better, never lead.
	if l := leaders[domain.MeasureReadiness]; l == nil || l.CompanyID != "added" || l.Value != 4 {
		t.Errorf("readiness leader = %+v", l)
	}
	if l := leaders[domain.MeasureSitemapCoverage]; l == nil || l.CompanyID != "added" || l.Value != 1 {
		t.Errorf("sitemap leader = %+v", l)
	}
	// The best prepared competitor to keep an eye on, and the top of the AI search ranking, are
	// the readiness leader.
	for _, e := range plan.EyeOn {
		if e.Why == domain.WatchBestPrepared && e.Leader != *leaders[domain.MeasureReadiness] {
			t.Errorf("best prepared = %+v", e.Leader)
		}
	}
	if top := ai.Ranking[0]; top.CompanyID != "added" || top.Score != leaders[domain.MeasureReadiness].Value {
		t.Errorf("ranking = %+v", ai.Ranking)
	}
	// The typical competitor's sitemap coverage is the median of the competitors measured: Ashworth's
	// scan did not measure it.
	for _, s := range standings.Standings {
		if s.Measure == domain.MeasureSitemapCoverage && (s.Median == nil || *s.Median != 0.9 || s.Competitors != 2) {
			t.Errorf("sitemap standing = %+v", s)
		}
	}
}

func TestEveryPageShowsTheSameSitemapCoverageAndCheckCounts(t *testing.T) {
	w := fieldWorld()
	_, standings, plan, ai, _ := readAll(t, w)
	coverage := standing(t, standings, domain.MeasureSitemapCoverage)
	for _, it := range plan.Items {
		if it.Measure == domain.MeasureSitemapCoverage && (it.Tribelt == nil || *it.Tribelt != coverage) {
			t.Errorf("plan %v, Catching up %v", it.Tribelt, coverage)
		}
	}
	if coverage != 0.5 {
		t.Errorf("coverage = %v", coverage)
	}
	// The readiness page's summary counts the same sites the AI search page counts, and Tribelt's
	// passes add up to the score Catching up shows, the crawlers kept partly out aside.
	if ai.Sites != len(w.readiness) || len(ai.Checks) != domain.ReadinessChecks()+len(domain.AlsoChecked()) {
		t.Fatalf("sites %d, checks %+v", ai.Sites, ai.Checks)
	}
	passed := 0.0
	for _, c := range ai.Checks {
		n := 0
		for _, r := range w.readiness {
			if r.Passes(c.Check) {
				n++
			}
		}
		if c.Passing != n {
			t.Errorf("%s: %d passing, the sites say %d", c.Check, c.Passing, n)
		}
		if c.Scored && *c.Tribelt {
			passed++
		}
	}
	if score := standing(t, standings, domain.MeasureReadiness); score != passed+0.5 {
		t.Errorf("Catching up %v, the summary's passes %v", score, passed)
	}
	if most, _ := domain.MaxOf(domain.TargetReadiness); most != float64(domain.ReadinessChecks()) || watchedNow(t, plan, domain.TargetReadiness) != standing(t, standings, domain.MeasureReadiness) {
		t.Errorf("the plan's readiness is out of %v", most)
	}
}

func TestTheHomePageAndAISearchShowTheSameSharesForTheCompetitors(t *testing.T) {
	overview, standings, _, ai, _ := readAll(t, fieldWorld())
	last := ai.Trend[len(ai.Trend)-1]
	for _, c := range overview.AiSearch.Companies {
		for _, s := range last.Shares {
			if s.CompanyID == c.CompanyID && s.Share != c.Share {
				t.Errorf("%s: home %v, AI search %v", c.CompanyID, c.Share, s.Share)
			}
		}
	}
	// The leader on AI answers is the competitor the home page lists first.
	if l := domain.LeaderOf(standings.Standings, domain.MeasureShareOfVoice); l == nil || overview.AiSearch.Companies[0].CompanyID != l.CompanyID || overview.AiSearch.Companies[0].Share != l.Value {
		t.Errorf("leader %+v, home %+v", l, overview.AiSearch.Companies)
	}
}
