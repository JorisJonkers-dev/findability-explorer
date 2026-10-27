package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

var (
	today   = time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	errDown = errors.New("down")
)

// world is a stand-in for the other contexts: each port answers from its fields.
type world struct {
	companies    []domain.Company
	runs         []domain.Run
	readiness    map[string]domain.Readiness
	terms        domain.TermMap
	ticks        []domain.Tick
	set          map[domain.Target]float64
	ticksErr     error
	targetsErr   error
	writeErr     error
	companiesErr error
	runsErr      error
	readyErr     error
	termsErr     error
	asked        int
}

func (w *world) Companies(context.Context) ([]domain.Company, error) {
	return w.companies, w.companiesErr
}

func (w *world) Runs(_ context.Context, limit int) ([]domain.Run, error) {
	w.asked = limit
	return w.runs[:min(limit, len(w.runs))], w.runsErr
}

func (w *world) Readiness(context.Context) (map[string]domain.Readiness, error) {
	return w.readiness, w.readyErr
}

func (w *world) TermMap(context.Context) (domain.TermMap, error) { return w.terms, w.termsErr }

// termMap is a map of n terms, all found by Ashworth's newest scan, the first used of them
// tribelt.nl's newest scan found.
func termMap(n, used int) domain.TermMap {
	m := domain.TermMap{TribeltFrom: domain.TermsFromScan, Scanned: []string{"tribelt", "ashworth"}}
	for i := range n {
		t := domain.MapTerm{Text: fmt.Sprintf("term %d", i), Language: "en", Family: "eyelink", Tier: "product", FoundBy: []string{"ashworth"}}
		if i < used {
			t.UsedByTribelt, t.FoundBy = true, []string{"ashworth", "tribelt"}
		}
		m.Terms = append(m.Terms, t)
	}
	return m
}

func (w *world) Ticks(context.Context) ([]domain.Tick, error) { return w.ticks, w.ticksErr }

func (w *world) Tick(_ context.Context, t domain.Tick) error {
	if w.writeErr != nil {
		return w.writeErr
	}
	w.ticks = append(w.ticks, t)
	return nil
}

func (w *world) Untick(_ context.Context, id string) error {
	if w.writeErr != nil {
		return w.writeErr
	}
	w.ticks = slices.DeleteFunc(w.ticks, func(t domain.Tick) bool { return t.ImprovementID == id })
	return nil
}

func (w *world) Targets(context.Context) (map[domain.Target]float64, error) {
	return w.set, w.targetsErr
}

func (w *world) SetTarget(_ context.Context, t domain.Target, v float64, _ string, _ time.Time) error {
	if w.writeErr != nil {
		return w.writeErr
	}
	w.set[t] = v
	return nil
}

func newWorld() *world {
	return &world{
		companies: []domain.Company{
			{ID: "tribelt", Name: "Tribelt B.V.", Role: domain.RoleTribelt},
			{ID: "ashworth", Name: "Ashworth", Role: domain.RoleCompetitor},
			{ID: "martens", Name: "Märtens", Role: domain.RoleCompetitor},
		},
		runs: []domain.Run{
			{ID: "newest", StartedAt: today, Answers: []domain.Answer{
				{Mode: domain.ModeWeb, Named: []string{"ashworth"}, Cited: []string{"tribelt"}},
				{Mode: domain.ModeKnowledge, Named: []string{"tribelt", "ashworth"}},
			}},
			{ID: "older", StartedAt: today.AddDate(0, 0, -7)},
		},
		readiness: map[string]domain.Readiness{
			"tribelt": {Crawlers: domain.AccessOpen, SitemapPages: 10},
			"martens": {Crawlers: domain.AccessOpen, Sitemap: true, LLMs: true, SitemapPages: 40},
		},
		terms: termMap(9, 3),
		set:   map[domain.Target]float64{},
	}
}

func service(w *world) *app.Service {
	return app.New(app.Ports{Directory: w, PromptRuns: w, Readiness: w, Terms: w, Store: w}, func() time.Time { return today })
}

func TestStandingsComeFromTheNewestRunAndScans(t *testing.T) {
	w := newWorld()
	got, err := service(w).Standings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if w.asked != 1 || got.Run == nil || got.Run.ID != "newest" || len(got.Companies) != 3 {
		t.Fatalf("run = %+v, asked for %d", got.Run, w.asked)
	}
	leaders := map[domain.Measure]string{}
	for _, s := range got.Standings {
		if s.Leader != nil {
			leaders[s.Measure] = s.Leader.CompanyID
		}
	}
	want := map[domain.Measure]string{
		domain.MeasureShareOfVoice: "ashworth", domain.MeasureCitationShare: "ashworth",
		domain.MeasureReadiness: "martens", domain.MeasureSitemapPages: "martens", domain.MeasureTermsUsed: "ashworth",
	}
	for m, id := range want {
		if leaders[m] != id {
			t.Errorf("leader of %s = %q, want %q", m, leaders[m], id)
		}
	}
	// Tribelt trails on share of voice (50% against 100%), readiness (1 against 3), the sitemap (10
	// against 40) and terms (3 against 9), and leads on citation share (100% against 0%).
	if got.Trailing != 4 {
		t.Errorf("trailing = %d, want 4", got.Trailing)
	}
}

func TestStandingsBeforeAnyRun(t *testing.T) {
	w := newWorld()
	w.runs = nil
	delete(w.readiness, "tribelt")
	got, err := service(w).Standings(context.Background())
	if err != nil || got.Run != nil || got.Standings[0].Tribelt != nil || len(got.Open) != app.OpenShown {
		t.Fatalf("standings = %+v, %v", got, err)
	}
}

func TestStandingsSayWhichSourceFailed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		breaks func(*world)
	}{
		{"companies", func(w *world) { w.companiesErr = errDown }},
		{"runs", func(w *world) { w.runsErr = errDown }},
		{"readiness", func(w *world) { w.readyErr = errDown }},
		{"terms", func(w *world) { w.termsErr = errDown }},
		{"ticks", func(w *world) { w.ticksErr = errDown }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld()
			tc.breaks(w)
			if _, err := service(w).Standings(context.Background()); !errors.Is(err, errDown) {
				t.Errorf("err = %v", err)
			}
		})
	}
}
