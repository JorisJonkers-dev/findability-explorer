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
	w.companies[1].Researched = true
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

func TestTheHomePageAndTheProductsCountTheSameCompetitors(t *testing.T) {
	w := agreeingWorld()
	w.companies[1].BeltNames = []domain.BeltName{{Name: "Omni-Grid", Family: "eyelink"}}
	w.companies[2].BeltNames = []domain.BeltName{{Name: "Eyelink", Family: "eyelink", FromScan: true}}
	overview, _, _, _, products := readAll(t, w)
	// Two competitors are on the list and both sell eyelink belts; the research found one of them.
	if overview.Competitors.Listed != len(products.Products[0].Sellers) || overview.Competitors.Makers != 1 || overview.Competitors.Families[0].Makers != 1 {
		t.Errorf("listed %d, sellers %d, researched %+v", overview.Competitors.Listed, len(products.Products[0].Sellers), overview.Competitors)
	}
}
