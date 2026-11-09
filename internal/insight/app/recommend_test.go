package app_test

import (
	"context"
	"errors"
	"sync"
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

func (j *journal) Beat(context.Context, uuid.UUID, time.Time) error { return nil }

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
	w.recs = []domain.Recommendation{
		{ID: "r1", Gap: domain.Gap{Kind: domain.GapFact}, Title: "Correct it", Benefit: 3, Effort: 3},
		{ID: "r2", Gap: domain.Gap{Kind: domain.GapListing, Key: "listing:x.com"}, Title: "List it", Benefit: 3, Effort: 3},
	}
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
	// A listing a rule decides is not ticked by hand.
	if err := svc.Tick(t.Context(), "joris", "r2", true); !errors.Is(err, domain.ErrCheckedByScan) {
		t.Errorf("a scan-checked one = %v", err)
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
			svc := writing(w, wr, j, c.budget, &reported)
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
	// The store holds the run: one it cannot begin does not start.
	if err := svc2.StartRecommendationRun(t.Context(), "joris"); !errors.Is(err, errDown) || len(reported) != 0 {
		t.Errorf("the store failing: %v; reported %v", err, reported)
	}
}

// gone is a store that lost a recommendation between reading and ticking it.
type gone struct{ *world }

func (g gone) SetRecDone(context.Context, string, string, time.Time) (bool, error) { return false, nil }

func TestARecommendationGoneMeanwhileIsUnknown(t *testing.T) {
	w := newWorld()
	w.recs = []domain.Recommendation{{ID: "r1", Gap: domain.Gap{Kind: domain.GapFact}, Title: "Correct it", Benefit: 3, Effort: 3}}
	svc := app.New(app.Ports{Directory: w, PromptRuns: w, Readiness: w, Terms: w, Store: gone{w}}, func() time.Time { return today }, counter())
	if err := svc.Tick(t.Context(), "joris", "r1", true); !errors.Is(err, domain.ErrUnknownImprovement) {
		t.Errorf("gone meanwhile = %v", err)
	}
	broken := app.New(app.Ports{Directory: w, PromptRuns: w, Readiness: w, Terms: w, Store: brokenTick{w}}, func() time.Time { return today }, counter())
	if err := broken.Tick(t.Context(), "joris", "r1", true); !errors.Is(err, errDown) {
		t.Errorf("a failing tick = %v", err)
	}
}

// brokenTick is a store whose SetRecDone fails.
type brokenTick struct{ *world }

func (brokenTick) SetRecDone(context.Context, string, string, time.Time) (bool, error) {
	return false, errDown
}

func TestTheRulesDecideAWrittenRecommendationAndARunKeepsIt(t *testing.T) {
	w := gappy()
	// The newest scan has a page with the term on it, and tribelt.nl has Organization markup.
	r := w.readiness["tribelt"]
	r.Organization = true
	w.readiness["tribelt"] = r
	w.recs = []domain.Recommendation{
		{ID: "org", RunID: "run0", Gap: domain.Gap{Key: "readiness:organization", Kind: domain.GapReadiness}, Title: "Organization", Benefit: 3, Effort: 2},
		{ID: "lang", RunID: "run0", Gap: domain.Gap{Key: "readiness:languages", Kind: domain.GapReadiness}, Title: "Languages", Benefit: 3, Effort: 4},
	}
	w.recRuns = []domain.RecRun{{ID: "run0", Ending: "done"}}
	var reported []error
	svc := writing(w, &writer{}, &journal{}, nil, &reported)
	plan, err := svc.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range plan.Items {
		switch it.ID {
		case "org":
			if !it.Auto || !it.Done || len(it.Parts) != 1 {
				t.Errorf("Organization = %+v", it)
			}
		case "lang":
			if !it.Auto || it.Done || !it.New {
				t.Errorf("languages = %+v", it)
			}
		}
	}
	before, err := svc.RecommendationRuns(t.Context())
	if err != nil || before.Replaces != 1 {
		t.Fatalf("before = %+v, %v", before, err)
	}
	if err := svc.StartRecommendationRun(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if run := w.recRuns[0]; run.Replaced != 1 || run.Kept != 1 {
		t.Errorf("run = %+v", run)
	}
	broken := writing(w, &writer{}, &journal{}, nil, &reported)
	broken.WithWriting(writers{w: &writer{}}, pages{err: errDown}, nil, nil, func(error) {}, func(f func()) { f() })
	if _, err := broken.Plan(t.Context()); !errors.Is(err, errDown) {
		t.Errorf("the plan with the pages failing = %v", err)
	}
}

func TestAnAdministratorEditsARecommendationARunWrote(t *testing.T) {
	w := newWorld()
	w.recs = []domain.Recommendation{{ID: "r1", Gap: domain.Gap{Kind: domain.GapFact}, Title: "Correct it", Benefit: 3, Effort: 3, WhereNote: domain.ForTheAgency}}
	svc := service(w)
	e := domain.Edit{Title: "Correct the address", Why: "It is wrong.", Steps: []string{"Fix it."}, DoneWhen: "Fixed.", Benefit: 4, Effort: 1}
	if err := svc.EditRecommendation(t.Context(), "r1", e); err != nil {
		t.Fatal(err)
	}
	if r := w.recs[0]; r.Title != "Correct the address" || r.Benefit != 4 || !r.EditedAt.Equal(today) || r.Open() {
		t.Errorf("edited = %+v", r)
	}
	if err := svc.EditRecommendation(t.Context(), "listings", e); !errors.Is(err, domain.ErrNotEditable) || !app.IsInputError(err) {
		t.Errorf("a hand-written one = %v", err)
	}
	if err := svc.EditRecommendation(t.Context(), "nothing", e); !errors.Is(err, domain.ErrUnknownImprovement) {
		t.Errorf("an unknown one = %v", err)
	}
	e.Benefit = 9
	if err := svc.EditRecommendation(t.Context(), "r1", e); !app.IsInputError(err) {
		t.Errorf("a bad edit = %v", err)
	}
	e.Benefit = 4
	w.recErr = errDown
	if err := svc.EditRecommendation(t.Context(), "r1", e); !errors.Is(err, errDown) {
		t.Errorf("a failing store = %v", err)
	}
	failing := &failingUpdate{world: newWorld()}
	failing.recs = w.recs
	failing.recErr = nil
	if err := app.New(app.Ports{Directory: failing, PromptRuns: failing, Readiness: failing, Terms: failing, Store: failing}, func() time.Time { return today }, counter()).
		EditRecommendation(t.Context(), "r1", e); !errors.Is(err, errDown) {
		t.Errorf("a failing update = %v", err)
	}
}

// failingUpdate is a store whose UpdateRecommendation fails.
type failingUpdate struct{ *world }

func (failingUpdate) UpdateRecommendation(context.Context, domain.Recommendation) error {
	return errDown
}

// A stop asked on another instance reaches the run, through the store.
func TestAStopAskedOnAnotherInstanceStopsTheRun(t *testing.T) {
	w := gappy()
	wr, j := &writer{}, &journal{}
	var reported []error
	running := writing(w, wr, j, nil, &reported)
	other := service(w)
	wr.during = func() {
		if err := other.StopRecommendationRun(context.Background()); err != nil {
			panic(err)
		}
	}
	if err := running.StartRecommendationRun(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if w.recRuns[0].Ending != string(work.EndingStoppedByHand) || len(reported) != 0 {
		t.Errorf("run %+v; reported %v", w.recRuns[0], reported)
	}
	w.stopErr = errDown
	if err := other.StopRecommendationRun(t.Context()); !errors.Is(err, errDown) {
		t.Errorf("a stop the store cannot record = %v", err)
	}
}

// A run beats while it runs; a beat the store cannot record is reported, and the run goes on.
func TestARunBeatsWhileItRuns(t *testing.T) {
	w := gappy()
	w.beatErr, w.beaten = errDown, make(chan struct{})
	// The first batch is written while two beats land.
	wr := &writer{during: func() {
		for range 2 {
			<-w.beaten
		}
		w.beaten = nil
	}}
	var mu sync.Mutex
	var reported []error
	svc := service(w)
	svc.WithWriting(writers{w: wr}, pages{}, nil, &journal{}, func(err error) {
		mu.Lock()
		defer mu.Unlock()
		reported = append(reported, err)
	}, func(f func()) { f() })
	svc.SetRecBeat(time.Millisecond)
	if err := svc.StartRecommendationRun(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if w.recRuns[0].Ending != string(work.EndingDone) || len(reported) == 0 || !errors.Is(reported[0], errDown) {
		t.Errorf("run %+v; reported %v", w.recRuns[0], reported)
	}
}

// Whether a run runs, and whether it was asked to stop, are read from the store; when that fails,
// the page says so and the run ends as failed.
func TestTheRunsStoredStateFailing(t *testing.T) {
	w := gappy()
	var reported []error
	svc := writing(w, &writer{}, &journal{}, nil, &reported)
	w.runningErr = errDown
	if _, err := svc.RecommendationRuns(t.Context()); !errors.Is(err, errDown) {
		t.Errorf("RecommendationRuns = %v", err)
	}
	w.runningErr, w.askedErr = nil, errDown
	if err := svc.StartRecommendationRun(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if w.recRuns[0].Ending != string(work.EndingFailed) || len(reported) != 1 || !errors.Is(reported[0], errDown) {
		t.Errorf("run %+v; reported %v", w.recRuns[0], reported)
	}
}

// A sweep ends the runs a stopped server left, with the window of work.Stale, and an hour for runs
// begun before runs beat.
func TestASweepEndsTheRunsLeftRunning(t *testing.T) {
	w := gappy()
	svc := service(w)
	if n, err := svc.EndAbandonedRecRuns(t.Context()); n != 1 || err != nil ||
		!w.alive.HeardSince.Equal(today.Add(-work.Stale)) || !w.alive.BegunSince.Equal(today.Add(-app.Unbeaten)) {
		t.Errorf("EndAbandonedRecRuns = %d, %v; alive %+v", n, err, w.alive)
	}
	w.recErr = errDown
	if _, err := svc.EndAbandonedRecRuns(t.Context()); !errors.Is(err, errDown) {
		t.Errorf("a sweep the store fails = %v", err)
	}
}
