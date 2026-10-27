package app_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// failedRun runs one provider's run whose first prompt fails in both modes, and returns the env.
func failedRun(t *testing.T) *env {
	t.Helper()
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Providers = []string{"openai"}
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	// Mark the first prompt's two answers failed, as a rate limit would leave them.
	first := domain.CurrentPromptSet().Prompts[0].ID
	for i, a := range e.repo.answers {
		if a.Prompt.ID == first {
			a.Failure, a.Extraction, a.Text = "Rate limited.", nil, ""
			e.repo.answers[i] = a
		}
	}
	return e
}

func failures(e *env) int {
	n := 0
	for _, a := range e.repo.answers {
		if a.Failure != "" {
			n++
		}
	}
	return n
}

func TestRetryAsksEveryFailedAnswerAgainInTheSameRun(t *testing.T) {
	e := failedRun(t)
	run := e.run(t)
	before := len(e.repo.answers)
	got, err := e.svc.Retry(t.Context(), run.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tasks != 2 || got.Run.ID != run.ID || got.Run.Status != domain.RunRunning || got.Estimate != 24 {
		t.Errorf("retry = %+v", got)
	}
	if failures(e) != 0 || len(e.repo.answers) != before {
		t.Errorf("after asking again: %d failures in %d answers", failures(e), len(e.repo.answers))
	}
	if r := e.run(t); r.Status != domain.RunDone {
		t.Errorf("the run ends %s", r.Status)
	}
	// Each failed try cost 10 and so did the new one; reading it cost 2.
	if a := e.repo.answers[0]; a.Cost != 20 || a.Extraction == nil {
		t.Errorf("the replaced answer = %+v", a)
	}
}

func TestRetryAsksOneQuestionOfOneModel(t *testing.T) {
	e := failedRun(t)
	run := e.run(t)
	key := domain.TaskKey{PromptID: domain.CurrentPromptSet().Prompts[0].ID, Provider: "openai", Mode: domain.ModeWeb}
	got, err := e.svc.Retry(t.Context(), run.ID, &key)
	if err != nil || got.Tasks != 1 || failures(e) != 1 {
		t.Fatalf("retry = %+v, %v, %d failures left", got, err, failures(e))
	}
	answered := domain.TaskKey{PromptID: domain.CurrentPromptSet().Prompts[1].ID, Provider: "openai", Mode: domain.ModeWeb}
	if _, err := e.svc.Retry(t.Context(), run.ID, &answered); !errors.Is(err, app.ErrNothingFailed) || !app.IsInputError(err) {
		t.Errorf("asking an answered question again = %v", err)
	}
}

func TestRetryRespectsTheBudgetAndTheLock(t *testing.T) {
	e := failedRun(t)
	run := e.run(t)
	e.repo.settings.MonthlyBudget = 1
	if _, err := e.svc.Retry(t.Context(), run.ID, nil); !errors.Is(err, domain.ErrBudgetReached) {
		t.Errorf("over budget = %v", err)
	}
	e.repo.settings.MonthlyBudget = domain.DefaultBudget
	// Another run is running: the lock holds.
	e.svc = app.New(e.repo, gateways{gw: e.gw}, e.dir, e.clock.Now, app.Config{Owner: "b"}, func(uuid.UUID, error) {}, app.WithSpawn(func(func()) {}))
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Retry(t.Context(), run.ID, nil); !errors.Is(err, domain.ErrRunActive) {
		t.Errorf("while another runs = %v", err)
	}
	if _, err := e.svc.Retry(t.Context(), e.run(t).ID, nil); !errors.Is(err, domain.ErrRunActive) {
		t.Errorf("a running run = %v", err)
	}
}

func TestRetryRefusals(t *testing.T) {
	e := failedRun(t)
	run := e.run(t)
	if _, err := e.svc.Retry(t.Context(), uuid.New(), nil); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("no such run = %v", err)
	}
	e.repo.settings.CredentialID = ""
	if _, err := e.svc.Retry(t.Context(), run.ID, nil); !errors.Is(err, domain.ErrNoKey) {
		t.Errorf("no key = %v", err)
	}
	e.repo.settings.CredentialID = "key"
	for _, op := range []string{"Answers", "SpentSince", "ReopenRun", "Settings"} {
		e.repo.failOn[op] = true
		if _, err := e.svc.Retry(t.Context(), run.ID, nil); !errors.Is(err, errBoom) {
			t.Errorf("%s failing: %v", op, err)
		}
		e.repo.failOn[op] = false
	}
	e.repo.failOn["AddAnswer"] = true
	if _, err := e.svc.Retry(t.Context(), run.ID, nil); err != nil {
		t.Fatal(err)
	}
	if len(e.failures) == 0 {
		t.Error("a retry that could not be recorded was not reported")
	}
}

// earlierRun is a finished run of prompt set 1, which this release no longer asks: two questions,
// one asked twice, the other failed once and never asked in the other mode.
func earlierRun(t *testing.T) (*env, domain.Run) {
	t.Helper()
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Providers = []string{"openai"}
	run := domain.Run{
		ID: uuid.New(), PromptSetVersion: 1, FactSheet: "facts", Status: domain.RunStopped, Tasks: 3 * 2,
		Models: []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}},
	}
	e.repo.runs = append(e.repo.runs, &storedRun{run: run, owner: "a"})
	supplier := domain.Prompt{ID: "eyelink-supplier-twente", Family: domain.FamilyEyelink, Kind: domain.KindSupplier, Scope: domain.ScopeTwente, Language: "nl", Text: "Wie levert oogjesbanden in Twente?"}
	problem := domain.Prompt{ID: "eyelink-problem-germany", Family: domain.FamilyEyelink, Kind: domain.KindProblem, Scope: domain.ScopeGermany, Language: "de", Text: "Unser Ösenband reißt."}
	read := &domain.Extraction{Cost: 2}
	e.repo.answers = []domain.Answer{
		{ID: uuid.New(), RunID: run.ID, Prompt: supplier, PromptSetVersion: 1, Provider: "openai", Mode: domain.ModeKnowledge, Reply: domain.Reply{Text: "Twentebelt.", Cost: 10}, Extraction: read},
		{ID: uuid.New(), RunID: run.ID, Prompt: supplier, PromptSetVersion: 1, Provider: "openai", Mode: domain.ModeWeb, Reply: domain.Reply{Text: "Twentebelt.", Cost: 10}, Extraction: read},
		{ID: uuid.New(), RunID: run.ID, Prompt: problem, PromptSetVersion: 1, Provider: "openai", Mode: domain.ModeKnowledge, Failure: "Rate limited."},
	}
	return e, run
}

func TestRetryAsksARunOfAnEarlierSetWithTheQuestionsItKept(t *testing.T) {
	e, run := earlierRun(t)
	got, err := e.svc.Retry(t.Context(), run.ID, nil)
	// The failed answer and the mode never asked; the third question no answer kept.
	if err != nil || got.Tasks != 2 {
		t.Fatalf("retry = %+v, %v", got, err)
	}
	if failures(e) != 0 || len(e.repo.answers) != 4 || e.run(t).Status != domain.RunDone {
		t.Fatalf("after asking again: %d failures in %d answers, run %s", failures(e), len(e.repo.answers), e.run(t).Status)
	}
	for _, a := range e.repo.answers {
		if a.Prompt.ID == "eyelink-problem-germany" && (a.Prompt.Text != "Unser Ösenband reißt." || a.PromptSetVersion != 1 || a.Extraction == nil) {
			t.Errorf("asked again = %+v", a)
		}
	}
}

// unreadRun is failedRun with the second prompt's web answer read but not understood.
func unreadRun(t *testing.T) (*env, domain.TaskKey) {
	t.Helper()
	e := failedRun(t)
	key := domain.TaskKey{PromptID: domain.CurrentPromptSet().Prompts[1].ID, Provider: "openai", Mode: domain.ModeWeb}
	for i, a := range e.repo.answers {
		if a.Key() == key {
			e.repo.answers[i].Extraction = &domain.Extraction{Failed: true, Cost: 1}
		}
	}
	return e, key
}

func TestThePlanSaysWhatAskingAndReadingAgainCost(t *testing.T) {
	e, _ := unreadRun(t)
	run := e.run(t)
	plan, err := e.svc.RetryPlan(t.Context(), run.ID)
	// Two failed answers, each 10 to ask and about 2 to read; one reading failed. The price list has
	// no price for the reader, so reading goes by what readings cost: (37 × 2 + 1) / 38 = 1.
	if err != nil || plan.Ask != 2 || plan.Unknown != 0 || plan.Read != 1 || plan.Reading != 1 || plan.ReadCost != 1 || plan.AskCost != 2*(10+1) {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	if len(plan.Costs) != 2 || plan.Costs[1] != (domain.ModeCost{Provider: "openai", Mode: domain.ModeWeb, Cost: 11}) {
		t.Errorf("costs = %+v", plan.Costs)
	}
	// With the reader on the price list: a reading used 48 in and 19 out on average (37 of 50 and 20,
	// one failed of none), at 0.1 µ$ and 0.4 µ$ a token, 12 µ$.
	e.gw.catalogue = append(e.gw.catalogue, domain.CatalogueModel{ID: domain.ExtractorModel, PromptPrice: 0.0000001, CompletionPrice: 0.0000004})
	if plan, err := e.svc.RetryPlan(t.Context(), run.ID); err != nil || plan.Reading != 12 || plan.ReadCost != 12 || plan.AskCost != 2*(10+12) {
		t.Errorf("by the reader's price = %+v, %v", plan, err)
	}
	// Haiku reads: its own price, 48 × 0.1 + 19 × 0.5 = 14 µ$.
	e.svc = e.withExtractors(extractors{x: &haiku{}})
	if plan, err := e.svc.RetryPlan(t.Context(), run.ID); err != nil || plan.Reading != 14 {
		t.Errorf("by Haiku's price = %+v, %v", plan, err)
	}
	if _, err := e.svc.RetryPlan(t.Context(), uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("no such run = %v", err)
	}
	for _, op := range []string{"Answers", "Settings"} {
		e.repo.failOn[op] = true
		if _, err := e.svc.RetryPlan(t.Context(), run.ID); !errors.Is(err, errBoom) {
			t.Errorf("%s failing: %v", op, err)
		}
		e.repo.failOn[op] = false
	}
	e.svc = e.withExtractors(extractors{err: errBoom})
	if _, err := e.svc.RetryPlan(t.Context(), run.ID); !errors.Is(err, errBoom) {
		t.Errorf("the reader unknown = %v", err)
	}
}

func TestThePlanOfAnEarlierSetCountsTheQuestionsNoAnswerKept(t *testing.T) {
	e, run := earlierRun(t)
	if plan, err := e.svc.RetryPlan(t.Context(), run.ID); err != nil || plan.Ask != 2 || plan.Unknown != 2 {
		t.Errorf("plan = %+v, %v", plan, err)
	}
}

func TestRereadReadsTheStoredAnswerAgainWithoutAskingAgain(t *testing.T) {
	e, key := unreadRun(t)
	run := e.run(t)
	asked, read := e.gw.asked["openai/gpt-oss-20b"], e.gw.extracted
	got, err := e.svc.Reread(t.Context(), run.ID, nil)
	if err != nil || got.Tasks != 1 || got.Estimate != 1 || got.Run.Status != domain.RunRunning {
		t.Fatalf("reread = %+v, %v", got, err)
	}
	if e.gw.asked["openai/gpt-oss-20b"] != asked || e.gw.extracted != read+1 || e.run(t).Status != domain.RunDone {
		t.Fatalf("asked %d more, read %d more, run %s", e.gw.asked["openai/gpt-oss-20b"]-asked, e.gw.extracted-read, e.run(t).Status)
	}
	for _, a := range e.repo.answers {
		if a.Key() == key && (a.Extraction == nil || a.Extraction.Failed || len(a.Extraction.Companies) == 0) {
			t.Errorf("read again = %+v", a.Extraction)
		}
	}
	// The failed answers stay failed: reading again asks nothing.
	if failures(e) != 2 {
		t.Errorf("%d failures", failures(e))
	}
	if _, err := e.svc.Reread(t.Context(), run.ID, nil); !errors.Is(err, app.ErrNothingUnread) || !app.IsInputError(err) {
		t.Errorf("nothing left to read = %v", err)
	}
}

func TestRereadReadsOneAnswer(t *testing.T) {
	e, key := unreadRun(t)
	run := e.run(t)
	other := domain.TaskKey{PromptID: key.PromptID, Provider: "openai", Mode: domain.ModeKnowledge}
	if _, err := e.svc.Reread(t.Context(), run.ID, &other); !errors.Is(err, app.ErrNothingUnread) {
		t.Errorf("an answer read already = %v", err)
	}
	if got, err := e.svc.Reread(t.Context(), run.ID, &key); err != nil || got.Tasks != 1 {
		t.Errorf("one answer = %+v, %v", got, err)
	}
}

func TestRereadRefusals(t *testing.T) {
	e, _ := unreadRun(t)
	run := e.run(t)
	if _, err := e.svc.Reread(t.Context(), uuid.New(), nil); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("no such run = %v", err)
	}
	e.repo.settings.MonthlyBudget = 1
	if _, err := e.svc.Reread(t.Context(), run.ID, nil); !errors.Is(err, domain.ErrBudgetReached) {
		t.Errorf("over budget = %v", err)
	}
	e.repo.settings.MonthlyBudget = domain.DefaultBudget
	e.repo.settings.CredentialID = ""
	if _, err := e.svc.Reread(t.Context(), run.ID, nil); !errors.Is(err, domain.ErrNoKey) {
		t.Errorf("no key = %v", err)
	}
	e.repo.settings.CredentialID = "key"
	// The reader cannot be had: nothing is reopened.
	e.svc = e.withExtractors(extractors{err: errBoom})
	if _, err := e.svc.Reread(t.Context(), run.ID, nil); !errors.Is(err, errBoom) || e.run(t).Status == domain.RunRunning {
		t.Errorf("the reader unknown = %v, run %s", err, e.run(t).Status)
	}
	e.svc = e.service("a")
	e.repo.failOn["ReopenRun"] = true
	if _, err := e.svc.Reread(t.Context(), run.ID, nil); !errors.Is(err, errBoom) {
		t.Errorf("reopening failing = %v", err)
	}
	e.repo.failOn["ReopenRun"] = false
	e.repo.runs[0].run.Status = domain.RunRunning
	if _, err := e.svc.Reread(t.Context(), run.ID, nil); !errors.Is(err, domain.ErrRunActive) {
		t.Errorf("a running run = %v", err)
	}
}
