package app_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

var start = time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)

// env is a Service on in-memory fakes, running runs in place.
type env struct {
	repo     *repo
	gw       *gateway
	dir      *directory
	clock    *clock
	sleeps   *sleeps
	mu       sync.Mutex
	failures []error
	svc      *app.Service
}

// newEnv returns an env whose settings choose the key "key" and the providers openai and google.
func newEnv(t *testing.T, r *repo, owner string) *env {
	t.Helper()
	e := &env{repo: r, gw: newGateway(), dir: &directory{}, clock: &clock{now: start}, sleeps: &sleeps{}}
	if r.settings == nil {
		s := domain.DefaultSettings(start)
		s.CredentialID, s.Providers = "key", []string{"openai", "google"}
		r.settings = &s
	}
	e.svc = e.service(owner)
	return e
}

func (e *env) service(owner string) *app.Service {
	return app.New(e.repo, gateways{gw: e.gw}, e.dir, e.clock.Now, app.Config{Owner: owner, Pace: time.Second, RetryWait: 10 * time.Second, Retries: 2},
		func(_ uuid.UUID, err error) {
			e.mu.Lock()
			defer e.mu.Unlock()
			e.failures = append(e.failures, err)
		},
		app.WithSleep(e.sleeps.Sleep), app.WithSpawn(func(f func()) { f() }))
}

func (e *env) run(t *testing.T) domain.Run {
	t.Helper()
	if len(e.repo.runs) == 0 {
		t.Fatal("no run stored")
	}
	return e.repo.runs[len(e.repo.runs)-1].run
}

func TestARunAsksEveryPromptOfEveryModelInBothModesAndReadsEveryAnswer(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	run, err := e.svc.Start(t.Context(), "joris")
	if err != nil {
		t.Fatal(err)
	}
	if run.Tasks != 360 || len(run.Models) != 2 || run.Models[1] != (domain.ModelChoice{Provider: "google", Model: "google/gemma:free", Free: true, Fallback: "google/gemini-flash-lite"}) ||
		run.FactSheetVersion != 1 || run.Trigger != domain.TriggerManual || run.StartedBy != "joris" {
		t.Fatalf("run = %+v", run)
	}
	if got := e.run(t); got.Status != domain.RunDone || len(e.repo.answers) != 360 || e.gw.extracted != 360 {
		t.Fatalf("status %s, %d answers, %d read", got.Status, len(e.repo.answers), e.gw.extracted)
	}
	if e.gw.asked["google/gemma:free"] != 180 || e.gw.asked["openai/gpt-oss-20b"] != 180 {
		t.Errorf("asked = %v", e.gw.asked)
	}
	if e.sleeps.total != 180*time.Second {
		t.Errorf("free model paced %v, want a second before each of its 180 calls", e.sleeps.total)
	}
	a := e.repo.answers[0]
	if a.PromptSetVersion != 1 || a.Model == "" || a.Extraction == nil || a.Extraction.Failed {
		t.Errorf("answer = %+v", a)
	}

	res, err := e.svc.Results(t.Context())
	if err != nil || len(res.Runs) != 1 {
		t.Fatalf("results = %+v, %v", res, err)
	}
	p := res.Runs[0].Progress
	if p.Answered != 360 || p.Cost != 360*(10+2) || len(p.ByProvider) != 2 {
		t.Errorf("progress = %+v", p)
	}
	named, cited, answers := 0, 0, 0
	for _, c := range res.Runs[0].Cells {
		named, cited, answers = named+c.Named, cited+c.Cited, answers+c.Answers
	}
	// Dutch supplier and application prompts (30 of 90) name Tribelt; web answers (half) cite it.
	if answers != 360 || named != 120 || cited != 180 {
		t.Errorf("answers %d, named %d, cited %d", answers, named, cited)
	}
	if len(res.Mentions) != 2 || res.Mentions[0].CompetitorID != "twentebelt" || res.Mentions[0].Answers != 360 ||
		res.Mentions[0].Name != "Twentebelt" || res.Mentions[1].Role != domain.RoleTribelt {
		t.Errorf("mentions = %+v", res.Mentions)
	}
	if len(res.WrongClaims) != 120 || res.WrongClaims[0].Claim != "Tribelt is in Hengelo." {
		t.Errorf("%d wrong claims", len(res.WrongClaims))
	}
	run2, answers2, err := e.svc.Answers(t.Context(), run.ID, app.AnswerFilter{})
	if err != nil || run2.Run.ID != run.ID || run2.Progress.Answered != 360 || len(answers2) != 360 || answers2[0].Text == "" {
		t.Errorf("answers = %d, %v", len(answers2), err)
	}
	filter := app.AnswerFilter{Provider: "google", Scope: domain.ScopeTwente, Family: domain.FamilyEyelink, Mode: domain.ModeWeb}
	if _, kept, err := e.svc.Answers(t.Context(), run.ID, filter); err != nil || len(kept) != 3 {
		t.Errorf("filtered answers = %d, %v", len(kept), err)
	}
	if _, _, err := e.svc.Answers(t.Context(), uuid.New(), app.AnswerFilter{}); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("answers of no run = %v", err)
	}
}

func TestAFreeModelThatIsRateLimitedHandsOverToThePaidOne(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.gw.limited["google/gemma:free"] = true
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if e.gw.asked["google/gemma:free"] != 3 || e.gw.asked["google/gemini-flash-lite"] != 180 {
		t.Errorf("asked = %v: the free model three times, then the paid one for all 180", e.gw.asked)
	}
	var models []string
	for _, a := range e.repo.answers {
		if a.Provider == "google" && a.Failure == "" {
			models = append(models, a.Model)
		}
	}
	if len(models) != 180 || models[0] != "google/gemini-flash-lite" {
		t.Errorf("google answered %d times by %v", len(models), models[:1])
	}
	if e.run(t).Status != domain.RunDone {
		t.Errorf("status = %s", e.run(t).Status)
	}
}

func TestAModelThatCannotAnswerIsRecordedAsFailed(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Providers = []string{"openai"}
	e.gw.limited["openai/gpt-oss-20b"] = true
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if len(e.repo.answers) != 180 || e.repo.answers[0].Failure != "The model is rate limited." || e.repo.answers[0].Extraction != nil {
		t.Fatalf("answers %d, first %+v", len(e.repo.answers), e.repo.answers[0])
	}
	// Each try waits longer than the one before: 0, 10s and 20s.
	if e.sleeps.total != 180*30*time.Second {
		t.Errorf("waited %v", e.sleeps.total)
	}
	res, _ := e.svc.Results(t.Context())
	if res.Runs[0].Progress.Failed != 180 || len(res.Runs[0].Cells) != 0 {
		t.Errorf("progress = %+v", res.Runs[0].Progress)
	}
	e.gw.limited = map[string]bool{}
	e.gw.askErr = errors.New("the provider is down")
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if last := e.repo.answers[len(e.repo.answers)-1]; last.Failure != "The provider is down." {
		t.Errorf("failure = %q", last.Failure)
	}
}

func TestAnAnswerTheExtractorCannotReadIsRetriedThenMarked(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Providers = []string{"openai"}
	e.gw.failExtract = 3 // the first answer fails twice, the second once
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	first, second := e.repo.answers[0].Extraction, e.repo.answers[1].Extraction
	if !first.Failed || first.Cost != 2 || second.Failed || second.Cost != 1+2 {
		t.Errorf("first %+v, second %+v", first, second)
	}
}

func TestARunStopsCleanlyAtTheMonthsBudget(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Providers = []string{"openai"}
	e.repo.settings.MonthlyBudget = 12 * 50 // fifty answers, each 10 asked and 2 read
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	run := e.run(t)
	if run.Status != domain.RunStopped || run.Failure == "" || len(e.repo.answers) != 50 {
		t.Fatalf("run %s (%q) with %d answers", run.Status, run.Failure, len(e.repo.answers))
	}
	view, err := e.svc.Settings(t.Context())
	if err != nil || view.SpentThisMonth != 600 {
		t.Errorf("spent = %v, %v", view.SpentThisMonth, err)
	}
	// A spent budget starts no question at all.
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if run := e.run(t); run.Status != domain.RunStopped || len(e.repo.answers) != 50 {
		t.Errorf("second run %s, %d answers", run.Status, len(e.repo.answers))
	}
}

func TestOnlyOneRunAtATime(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	held := domain.Run{ID: uuid.New(), Status: domain.RunRunning}
	e.repo.runs = append(e.repo.runs, &storedRun{run: held, owner: "b", heartbeat: start})
	if _, err := e.svc.Start(t.Context(), "joris"); !errors.Is(err, domain.ErrRunActive) {
		t.Fatalf("a second run = %v", err)
	}
}

func TestStartNeedsAKeyAndModels(t *testing.T) {
	cases := map[string]struct {
		change func(*env)
		want   error
	}{
		"no key":            {func(e *env) { e.repo.settings.CredentialID = "" }, domain.ErrNoKey},
		"no models":         {func(e *env) { e.repo.settings.Providers = []string{"x-ai"} }, domain.ErrNoModels},
		"no price list":     {func(e *env) { e.gw.modelsErr = errBoom }, errBoom},
		"no settings":       {func(e *env) { e.repo.failOn["Settings"] = true }, errBoom},
		"the store failing": {func(e *env) { e.repo.failOn["CreateRun"] = true }, errBoom},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t, newRepo(), "a")
			c.change(e)
			if _, err := e.svc.Start(t.Context(), "joris"); !errors.Is(err, c.want) {
				t.Fatalf("Start = %v, want %v", err, c.want)
			}
		})
	}
	if !app.IsInputError(domain.ErrNoKey) || !app.IsInputError(domain.ErrNoModels) || app.IsInputError(errBoom) {
		t.Error("input errors")
	}
}

func TestTheScheduleStartsADueRunOnceAcrossInstances(t *testing.T) {
	r := newRepo()
	e := newEnv(t, r, "a")
	other := e.service("b")
	if err := e.svc.Tick(t.Context()); err != nil || len(r.runs) != 0 {
		t.Fatalf("a run started before it was due: %v, %d runs", err, len(r.runs))
	}
	e.clock.set(time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC))
	// The first instance claims the due run and, here, runs it to the end; the second finds the
	// schedule moved on.
	if err := e.svc.Tick(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := other.Tick(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(r.runs) != 1 || r.runs[0].run.Trigger != domain.TriggerSchedule || r.runs[0].run.Status != domain.RunDone {
		t.Fatalf("runs = %d, %+v", len(r.runs), r.runs[0].run)
	}
	if !r.settings.NextDue.Equal(time.Date(2026, 12, 1, 6, 0, 0, 0, time.UTC)) {
		t.Errorf("next due = %v", r.settings.NextDue)
	}
}

func TestADueRunWaitsForAKeyAndForTheRunningOne(t *testing.T) {
	r := newRepo()
	e := newEnv(t, r, "a")
	e.clock.set(time.Date(2026, 11, 1, 7, 0, 0, 0, time.UTC))
	r.settings.CredentialID = ""
	if err := e.svc.Tick(t.Context()); err != nil || len(r.runs) != 0 {
		t.Fatalf("a run started without a key: %v", err)
	}
	r.settings.CredentialID = "key"
	r.runs = append(r.runs, &storedRun{run: domain.Run{ID: uuid.New(), Status: domain.RunRunning}, owner: "b", heartbeat: e.clock.Now()})
	if err := e.svc.Tick(t.Context()); err != nil || len(r.runs) != 1 || r.settings.NextDue.After(e.clock.Now()) {
		t.Fatalf("a run started beside a running one: %v, %d runs", err, len(r.runs))
	}
}

func TestTickFailures(t *testing.T) {
	for _, op := range []string{"AdoptStale", "Settings", "ClaimDue"} {
		e := newEnv(t, newRepo(op), "a")
		e.clock.set(time.Date(2026, 11, 1, 7, 0, 0, 0, time.UTC))
		if err := e.svc.Tick(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("%s failing: %v", op, err)
		}
	}
	e := newEnv(t, newRepo(), "a")
	e.clock.set(time.Date(2026, 11, 1, 7, 0, 0, 0, time.UTC))
	e.gw.modelsErr = errBoom
	if err := e.svc.Tick(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("no price list: %v", err)
	}
}

func TestARunInterruptedByARestartIsResumedByAnotherInstance(t *testing.T) {
	r := newRepo()
	e := newEnv(t, r, "old")
	r.settings.Providers = []string{"openai"}
	run, err := e.svc.Start(t.Context(), "joris")
	if err != nil {
		t.Fatal(err)
	}
	// Pretend the old instance stopped after two answers, one of them not read yet.
	r.mu.Lock()
	r.answers = r.answers[:2]
	r.answers[1].Extraction = nil
	r.runs[0].run.Status, r.runs[0].heartbeat = domain.RunRunning, start
	r.mu.Unlock()
	e.gw.asked = map[string]int{}
	e.gw.extracted = 0

	fresh := e.service("new")
	e.clock.set(start.Add(time.Minute))
	if err := fresh.Tick(t.Context()); err != nil {
		t.Fatal(err)
	}
	if r.runs[0].owner != "" && r.runs[0].run.Status != domain.RunRunning {
		t.Fatal("the run was taken over before it went quiet for long")
	}
	e.clock.set(start.Add(5 * time.Minute))
	if err := fresh.Tick(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := e.run(t); got.ID != run.ID || got.Status != domain.RunDone || len(r.answers) != 180 {
		t.Fatalf("run %s with %d answers", got.Status, len(r.answers))
	}
	if e.gw.asked["openai/gpt-oss-20b"] != 178 || e.gw.extracted != 179 || r.answers[1].Extraction == nil {
		t.Errorf("asked %v, read %d: only what was left", e.gw.asked, e.gw.extracted)
	}
}

func TestARunTakenOverByAnotherInstanceIsLeftToIt(t *testing.T) {
	r := newRepo()
	r.heartbeatsLeft = 3
	e := newEnv(t, r, "a")
	r.settings.Providers = []string{"openai"}
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if got := e.run(t); got.Status != domain.RunRunning || len(r.answers) != 4 {
		t.Fatalf("run %s with %d answers", got.Status, len(r.answers))
	}
}

func TestARunThatCannotBeRecordedFails(t *testing.T) {
	for _, op := range []string{"AddAnswer", "Heartbeat"} {
		e := newEnv(t, newRepo(op), "a")
		e.repo.settings.Providers = []string{"openai"}
		if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
			t.Fatal(err)
		}
		if got := e.run(t); got.Status != domain.RunFailed || len(e.failures) != 1 || !errors.Is(e.failures[0], errBoom) {
			t.Errorf("%s failing: run %s, failures %v", op, got.Status, e.failures)
		}
	}
	// An unread answer whose extraction cannot be stored stops the run before it asks more.
	r := newRepo("SetExtraction")
	e := newEnv(t, r, "a")
	run := domain.Run{ID: uuid.New(), PromptSetVersion: 1, FactSheet: "facts", Status: domain.RunRunning, Models: []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}}}
	r.runs = append(r.runs, &storedRun{run: run, owner: "a"})
	r.answers = []domain.Answer{{ID: uuid.New(), RunID: run.ID, Prompt: domain.CurrentPromptSet().Prompts[0], Provider: "openai", Mode: domain.ModeKnowledge, Reply: domain.Reply{Text: "x"}}}
	if err := e.svc.Execute(t.Context(), run); !errors.Is(err, errBoom) || e.run(t).Status != domain.RunFailed || len(r.answers) != 1 {
		t.Errorf("Execute = %v, status %s", err, e.run(t).Status)
	}
}

func TestARunThatCannotStartIsFailed(t *testing.T) {
	cases := map[string]func(*env, *domain.Run){
		"no key":             func(e *env, _ *domain.Run) { e.repo.settings.CredentialID = "" },
		"no companies":       func(e *env, _ *domain.Run) { e.dir.err = errBoom },
		"no spend":           func(e *env, _ *domain.Run) { e.repo.failOn["SpentSince"] = true },
		"no answers":         func(e *env, _ *domain.Run) { e.repo.failOn["Answers"] = true },
		"no settings":        func(e *env, _ *domain.Run) { e.repo.failOn["Settings"] = true },
		"another prompt set": func(_ *env, r *domain.Run) { r.PromptSetVersion = 99 },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t, newRepo(), "a")
			run := domain.Run{ID: uuid.New(), PromptSetVersion: 1, Status: domain.RunRunning, Models: []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}}}
			e.repo.runs = append(e.repo.runs, &storedRun{run: run, owner: "a"})
			change(e, &run)
			if err := e.svc.Execute(t.Context(), run); err != nil {
				t.Fatal(err)
			}
			if got := e.run(t); got.Status != domain.RunFailed || got.Failure == "" {
				t.Errorf("run %s (%q)", got.Status, got.Failure)
			}
		})
	}
	e := newEnv(t, newRepo("FinishRun"), "a")
	e.repo.settings.CredentialID = ""
	if err := e.svc.Execute(t.Context(), domain.Run{ID: uuid.New()}); !errors.Is(err, errBoom) {
		t.Errorf("a failure that cannot be recorded = %v", err)
	}
}

func TestARunEndsWhenItsContextDoes(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Providers = []string{"google"}
	ctx, cancel := context.WithCancel(t.Context())
	svc := app.New(e.repo, gateways{gw: e.gw}, e.dir, e.clock.Now, app.Config{}, func(uuid.UUID, error) {},
		app.WithSleep(func(ctx context.Context, _ time.Duration) error { cancel(); return ctx.Err() }))
	run := domain.Run{ID: uuid.New(), PromptSetVersion: 1, Status: domain.RunRunning, Models: []domain.ModelChoice{{Provider: "google", Model: "google/gemma:free", Free: true}}}
	e.repo.runs = append(e.repo.runs, &storedRun{run: run, owner: "a"})
	if err := svc.Execute(ctx, run); err != nil || len(e.repo.answers) != 0 {
		t.Errorf("Execute = %v with %d answers", err, len(e.repo.answers))
	}
	// A paid model is not paced; an ended context stops it all the same.
	run.Models = []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}}
	if err := svc.Execute(ctx, run); err != nil || len(e.repo.answers) != 0 {
		t.Errorf("Execute = %v with %d answers", err, len(e.repo.answers))
	}
}

func TestLoopTicksUntilItsContextEnds(t *testing.T) {
	e := newEnv(t, newRepo("AdoptStale"), "a")
	ctx, cancel := context.WithCancel(t.Context())
	var errs []error
	ticks := 0
	svc := app.New(e.repo, gateways{gw: e.gw}, e.dir, e.clock.Now, app.Config{}, func(uuid.UUID, error) {},
		app.WithSleep(func(context.Context, time.Duration) error {
			ticks++
			if ticks == 2 {
				cancel()
				return context.Canceled
			}
			return nil
		}))
	svc.Loop(ctx, func(err error) { errs = append(errs, err) })
	if ticks != 2 || len(errs) != 2 {
		t.Errorf("%d ticks, errors %v", ticks, errs)
	}
}

func TestTheDefaultsRunInTheBackground(t *testing.T) {
	r := newRepo()
	e := newEnv(t, r, "a")
	r.settings.Providers = []string{"openai"}
	svc := app.New(r, gateways{gw: e.gw}, e.dir, e.clock.Now, app.Config{}, func(uuid.UUID, error) {})
	if _, err := svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(5 * time.Second); ; {
		r.mu.Lock()
		status := r.runs[0].run.Status
		r.mu.Unlock()
		if status == domain.RunDone {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the run in the background is still %s", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
	// The real pause waits, and ends with its context.
	quick := app.New(r, gateways{gw: e.gw}, e.dir, e.clock.Now, app.Config{Tick: time.Millisecond}, func(uuid.UUID, error) {})
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	quick.Loop(ctx, func(error) {})
}

func TestAResumedRunOverBudgetReadsAndAsksNoMore(t *testing.T) {
	r := newRepo()
	e := newEnv(t, r, "a")
	r.settings.MonthlyBudget = 1
	run := domain.Run{ID: uuid.New(), PromptSetVersion: 1, FactSheet: "facts", Status: domain.RunRunning, Models: []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}}}
	r.runs = append(r.runs, &storedRun{run: run, owner: "a"})
	prompt := domain.CurrentPromptSet().Prompts[0]
	for _, mode := range domain.Modes() {
		r.answers = append(r.answers, domain.Answer{ID: uuid.New(), RunID: run.ID, Prompt: prompt, Provider: "openai", Mode: mode, Reply: domain.Reply{Text: "x", Cost: 5}})
	}
	if err := e.svc.Execute(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	if e.gw.extracted != 0 || len(r.answers) != 2 || e.run(t).Status != domain.RunStopped {
		t.Errorf("read %d, %d answers, %s", e.gw.extracted, len(r.answers), e.run(t).Status)
	}
}

func TestSettingsFailures(t *testing.T) {
	e := newEnv(t, newRepo("Settings"), "a")
	if _, err := e.svc.Settings(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("settings failing: %v", err)
	}
}

func TestMentionsComeFromTheNewestRunWithAnswers(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Providers = []string{"openai"}
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	e.repo.runs = append(e.repo.runs, &storedRun{run: domain.Run{ID: uuid.New(), Status: domain.RunRunning, StartedAt: start.Add(time.Hour)}, owner: "a"})
	res, err := e.svc.Results(t.Context())
	if err != nil || len(res.Runs) != 2 || len(res.Runs[0].Cells) != 0 || len(res.Mentions) == 0 {
		t.Errorf("results = %d runs, %d mentions, %v", len(res.Runs), len(res.Mentions), err)
	}
}

func TestASlowRunStaysHeldWhileItWaitsOnAModel(t *testing.T) {
	r := newRepo()
	e := newEnv(t, r, "a")
	r.settings.Providers = []string{"openai"}
	e.gw.slow = 30 * time.Millisecond
	run := domain.Run{ID: uuid.New(), PromptSetVersion: 1, FactSheet: "facts", Status: domain.RunRunning, Models: []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}}}
	r.runs = append(r.runs, &storedRun{run: run, owner: "a"})
	r.answers = nil
	// Only two prompts are left to ask, each slower than the stale window.
	set := domain.CurrentPromptSet()
	for _, task := range domain.Tasks(set, run.Models)[2:] {
		r.answers = append(r.answers, domain.Answer{ID: uuid.New(), RunID: run.ID, Prompt: task.Prompt, Provider: task.Provider, Mode: task.Mode, Failure: "x"})
	}
	svc := app.New(r, gateways{gw: e.gw}, e.dir, time.Now, app.Config{Owner: "a", Stale: 20 * time.Millisecond}, func(uuid.UUID, error) {},
		app.WithSleep(e.sleeps.Sleep))
	if err := svc.Execute(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	// Two heartbeats come with the two answers; the rest kept the run held while the model was slow.
	if r.beats < 4 || e.run(t).Status != domain.RunDone {
		t.Errorf("%d heartbeats, status %s", r.beats, e.run(t).Status)
	}
}

func TestAResumedProviderKeepsToItsPaidFallback(t *testing.T) {
	r := newRepo()
	e := newEnv(t, r, "a")
	choice := domain.ModelChoice{Provider: "google", Model: "google/gemma:free", Free: true, Fallback: "google/gemini-flash-lite"}
	run := domain.Run{ID: uuid.New(), PromptSetVersion: 1, FactSheet: "facts", Status: domain.RunRunning, Models: []domain.ModelChoice{choice}}
	r.runs = append(r.runs, &storedRun{run: run, owner: "a"})
	first := domain.Tasks(domain.CurrentPromptSet(), run.Models)[0]
	r.answers = []domain.Answer{{ID: uuid.New(), RunID: run.ID, Prompt: first.Prompt, Provider: "google", Model: choice.Fallback, Mode: first.Mode, Failure: "x"}}
	if err := e.svc.Execute(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	if e.gw.asked["google/gemma:free"] != 0 || e.gw.asked[choice.Fallback] != 179 {
		t.Errorf("asked %v: the free model gave up before the restart, so the paid one answers the rest", e.gw.asked)
	}
}
