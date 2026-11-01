package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
)

// writer is a fake model: it places each gap on the first page, words it from its key, and moves
// no score; each request uses 1,000 tokens in and 100 out. fail fails a request; during runs
// something while a request is asked.
type writer struct {
	asked  [][]string
	fail   error
	during func()
}

func (w *writer) Write(_ context.Context, gaps []domain.Gap, pages []domain.Page) ([]domain.Written, app.Usage, error) {
	var keys []string
	var out []domain.Written
	for _, g := range gaps {
		keys = append(keys, g.Key)
		out = append(out, domain.Written{Key: g.Key, Title: "Write " + g.Key, Where: pages[0].URL, Why: "Because.", Steps: []string{"Do it."}, DoneWhen: "Found."})
	}
	w.asked = append(w.asked, keys)
	if w.during != nil {
		w.during()
	}
	return out, app.Usage{InputTokens: 1000, OutputTokens: 100}, w.fail
}

type writers struct {
	w   app.Writer
	err error
}

func (ws writers) Current(context.Context) (app.Writer, string, error) {
	return ws.w, "claude-haiku-5-5", ws.err
}

// pages is tribelt.nl's newest scan: one page, scanned an hour before the newest Prompt run.
type pages struct{ err error }

func (p pages) Pages(context.Context) ([]domain.Page, time.Time, error) {
	return []domain.Page{{URL: "https://www.tribelt.nl/", Title: "Tribelt", Language: "en", Family: "eyelink", Text: "Metal belts."}}, today.Add(-time.Hour), p.err
}

// journal records how each run ended; beginErr fails Begin.
type journal struct {
	begun    []work.Kind
	ended    []work.Outcome
	beginErr error
}

func (j *journal) Begin(_ context.Context, kind work.Kind, _ string, _ time.Time) (uuid.UUID, error) {
	j.begun = append(j.begun, kind)
	return uuid.New(), j.beginErr
}

func (j *journal) End(_ context.Context, _ uuid.UUID, o work.Outcome, _ time.Time) error {
	j.ended = append(j.ended, o)
	return nil
}

// writing is the world's service with a model writing recommendations, runs in place.
func writing(w *world, wr app.Writer, j *journal, budget app.BudgetSpent, reported *[]error) *app.Service {
	svc := service(w)
	svc.WithWriting(writers{w: wr}, pages{}, budget, j, func(err error) { *reported = append(*reported, err) }, func(f func()) { f() })
	return svc
}

// gappy is the world with 15 search terms tribelt.nl does not use, and one it does.
func gappy() *world {
	w := newWorld()
	w.terms = termMap(16, 1)
	return w
}

func TestARecommendationRunWritesAndPlacesEveryGap(t *testing.T) {
	w := gappy()
	// An open recommendation an earlier run wrote, and one an administrator ticked off.
	w.recs = []domain.Recommendation{
		{ID: "old-open", RunID: "run0", Gap: domain.Gap{Key: "term:en:term 1", Kind: domain.GapTerm}, Title: "Old", Benefit: 2, Effort: 2},
		{ID: "old-done", RunID: "run0", Gap: domain.Gap{Key: "term:en:term 2", Kind: domain.GapTerm}, Title: "Done", Benefit: 2, Effort: 2, DoneAt: today, DoneBy: "joris"},
	}
	w.recRuns = []domain.RecRun{{ID: "run0", Ending: "done"}}
	wr, j := &writer{}, &journal{}
	var reported []error
	svc := writing(w, wr, j, func(context.Context) (bool, error) { return false, nil }, &reported)

	before, err := svc.RecommendationRuns(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// 15 terms and the readiness tribelt.nl fails (Organization, languages); the page is older than the run.
	if before.Running || before.NoKey || before.Replaces != 1 || !before.Stale || before.Gaps != 17 || before.Terms != 16 || before.Estimate <= 0 ||
		!before.PromptRunAt.Equal(today) || before.Model != "claude-haiku-5-5" || len(before.Runs) != 1 {
		t.Fatalf("before = %+v", before)
	}
	if err := svc.StartRecommendationRun(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	// Two batches, the budget allowing each; the run replaced the open one and kept the done one, which covers its gap.
	if len(wr.asked) != 2 || len(wr.asked[0]) != app.WriteBatch || len(reported) != 0 {
		t.Fatalf("asked %v; reported %v", wr.asked, reported)
	}
	run := w.recRuns[0]
	if run.Ending != "done" || run.Added != 16 || run.Replaced != 1 || run.Kept != 1 || run.Gaps != 17 || run.InputTokens != 2000 || run.By != "joris" ||
		run.Model != "claude-haiku-5-5" || run.FinishedAt.IsZero() {
		t.Errorf("run = %+v", run)
	}
	if len(j.ended) != 1 || j.begun[0] != work.KindRecommendationRun || j.ended[0].Ending != work.EndingDone || j.ended[0].Counts != (work.Counts{Done: 17, Total: 17}) {
		t.Errorf("journal = %v, %+v", j.begun, j.ended)
	}
	plan, err := svc.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	written, fresh := 0, 0
	for _, it := range plan.Items {
		if it.Written != nil {
			written++
		}
		if it.New {
			fresh++
		}
	}
	if written != 17 || fresh != 16 {
		t.Errorf("the plan has %d written, %d new", written, fresh)
	}
}

func TestAWrittenRecommendationIsTickedOffByHand(t *testing.T) {
	w := newWorld()
	w.recs = []domain.Recommendation{{ID: "r1", Gap: domain.Gap{Kind: domain.GapListing}, Title: "List it", Benefit: 3, Effort: 3}}
	svc := service(w)
	if err := svc.Tick(t.Context(), "joris", "r1", true); err != nil || w.recs[0].DoneBy != "joris" || !w.recs[0].DoneAt.Equal(today) {
		t.Fatalf("tick = %v, %+v", err, w.recs[0])
	}
	if err := svc.Tick(t.Context(), "joris", "r1", false); err != nil || !w.recs[0].DoneAt.IsZero() {
		t.Fatalf("untick = %v, %+v", err, w.recs[0])
	}
	if err := svc.Tick(t.Context(), "joris", "nothing", true); !errors.Is(err, domain.ErrUnknownImprovement) {
		t.Errorf("an unknown one = %v", err)
	}
	w.recErr = errDown
	if err := svc.Tick(t.Context(), "joris", "r1", true); !errors.Is(err, errDown) {
		t.Errorf("a failing store = %v", err)
	}
	if _, err := svc.Plan(t.Context()); !errors.Is(err, errDown) {
		t.Errorf("the plan with the store failing = %v", err)
	}
	if _, err := svc.Standings(t.Context()); !errors.Is(err, errDown) {
		t.Errorf("the standings with the store failing = %v", err)
	}
	w.recErr, w.runsErr2 = nil, errDown
	if _, err := svc.Plan(t.Context()); !errors.Is(err, errDown) {
		t.Errorf("the plan with the past runs failing = %v", err)
	}
	svc.WithWriting(writers{w: &writer{}}, pages{}, nil, nil, func(error) {}, func(f func()) { f() })
	if _, err := svc.RecommendationRuns(t.Context()); !errors.Is(err, errDown) {
		t.Errorf("RecommendationRuns with the past runs failing = %v", err)
	}
}

func TestARunStoppedOrCutShortChangesNothing(t *testing.T) {
	cases := map[string]struct {
		setup  func(svc **app.Service, wr *writer)
		budget app.BudgetSpent
		want   work.Ending
		report bool
	}{
		"stopped by hand": {setup: func(svc **app.Service, wr *writer) {
			wr.during = func() {
				if err := (*svc).StopRecommendationRun(context.Background()); err != nil {
					panic(err)
				}
			}
		}, want: work.EndingStoppedByHand},
		"the budget":         {budget: func(context.Context) (bool, error) { return true, nil }, want: work.EndingStoppedByBudget},
		"the model":          {setup: func(_ **app.Service, wr *writer) { wr.fail = errDown }, want: work.EndingFailed, report: true},
		"the budget unknown": {budget: func(context.Context) (bool, error) { return false, errDown }, want: work.EndingFailed, report: true},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			w := gappy()
			w.recs = []domain.Recommendation{{ID: "old-open", Gap: domain.Gap{Key: "x"}}}
			wr, j := &writer{}, &journal{}
			var reported []error
			var svc *app.Service
			svc = writing(w, wr, j, c.budget, &reported)
			if c.setup != nil {
				c.setup(&svc, wr)
			}
			if err := svc.StartRecommendationRun(t.Context(), "joris"); err != nil {
				t.Fatal(err)
			}
			if len(w.recs) != 1 || w.recs[0].ID != "old-open" || w.recRuns[0].Ending != string(c.want) || j.ended[0].Ending != c.want || (len(reported) > 0) != c.report {
				t.Errorf("recs %+v; run %+v; journal %+v; reported %v", w.recs, w.recRuns[0], j.ended, reported)
			}
		})
	}
}

func TestStartingAndStoppingARunIsRefusedWhenItCannotBe(t *testing.T) {
	w := gappy()
	var reported []error
	svc := service(w)
	if err := svc.StartRecommendationRun(t.Context(), "joris"); !errors.Is(err, app.ErrNoWriter) {
		t.Errorf("without a key = %v", err)
	}
	if err := svc.StopRecommendationRun(t.Context()); !errors.Is(err, app.ErrNothingToStop) {
		t.Errorf("a stop with none running = %v", err)
	}
	if v, err := svc.RecommendationRuns(t.Context()); err != nil || !v.NoKey || v.Model != "" {
		t.Errorf("without a key = %+v, %v", v, err)
	}
	// One runs already: it is not started twice.
	var later func()
	held := service(w)
	held.WithWriting(writers{w: &writer{}}, pages{}, nil, nil, func(err error) { reported = append(reported, err) }, func(f func()) { later = f })
	if err := held.StartRecommendationRun(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if v, _ := held.RecommendationRuns(t.Context()); !v.Running {
		t.Error("the run does not show as running")
	}
	if err := held.StartRecommendationRun(t.Context(), "joris"); !errors.Is(err, app.ErrRunning) {
		t.Errorf("a second start = %v", err)
	}
	later()
	if len(reported) != 0 {
		t.Errorf("reported %v", reported)
	}
	broken := service(w)
	broken.WithWriting(writers{err: errDown}, pages{}, nil, nil, func(error) {}, func(f func()) { f() })
	if err := broken.StartRecommendationRun(t.Context(), "joris"); !errors.Is(err, errDown) {
		t.Errorf("the settings failing = %v", err)
	}
	if _, err := broken.RecommendationRuns(t.Context()); !errors.Is(err, errDown) {
		t.Errorf("RecommendationRuns with the settings failing = %v", err)
	}
}

func TestARunWhoseInputsOrStoreFailIsRecordedAsFailed(t *testing.T) {
	for name, breaks := range map[string]func(*world, *pages){
		"the companies": func(w *world, _ *pages) { w.companiesErr = errDown },
		"the terms":     func(w *world, _ *pages) { w.termsErr = errDown },
		"the pages":     func(_ *world, p *pages) { p.err = errDown },
	} {
		t.Run(name, func(t *testing.T) {
			w, p := gappy(), pages{}
			breaks(w, &p)
			var reported []error
			svc := service(w)
			svc.WithWriting(writers{w: &writer{}}, p, nil, &journal{}, func(err error) { reported = append(reported, err) }, func(f func()) { f() })
			if _, err := svc.RecommendationRuns(t.Context()); !errors.Is(err, errDown) {
				t.Errorf("RecommendationRuns = %v", err)
			}
			if err := svc.StartRecommendationRun(t.Context(), "joris"); err != nil || len(reported) != 1 || w.recRuns[0].Ending != "failed" {
				t.Errorf("start = %v; reported %v; runs %+v", err, reported, w.recRuns)
			}
		})
	}
	// The store failing: reading what was written, beginning, and finishing.
	w := gappy()
	var reported []error
	svc := writing(w, &writer{}, &journal{beginErr: errDown}, nil, &reported)
	w.finishErr = errDown
	if err := svc.StartRecommendationRun(t.Context(), "joris"); err != nil || len(reported) != 2 {
		t.Errorf("finish failing: %v; reported %v", err, reported)
	}
	w2 := gappy()
	reported = nil
	svc2 := writing(w2, &writer{}, &journal{}, nil, &reported)
	if v, err := svc2.RecommendationRuns(t.Context()); err != nil || v.Replaces != 0 {
		t.Fatalf("before = %+v, %v", v, err)
	}
	w2.recErr = errDown
	if _, err := svc2.RecommendationRuns(t.Context()); !errors.Is(err, errDown) {
		t.Errorf("RecommendationRuns with the store failing = %v", err)
	}
	if err := svc2.StartRecommendationRun(t.Context(), "joris"); err != nil || len(reported) != 2 {
		t.Errorf("the store failing: %v; reported %v", err, reported)
	}
}
