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
	e.repo.runs[0].run.PromptSetVersion = 1
	if _, err := e.svc.Retry(t.Context(), run.ID, nil); !errors.Is(err, app.ErrEarlierSet) {
		t.Errorf("an earlier set = %v", err)
	}
	e.repo.runs[0].run.PromptSetVersion = domain.PromptSetVersion
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
