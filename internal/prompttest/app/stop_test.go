package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// stoppedRun is a run of one provider an administrator stopped while its fourth question was asked.
func stoppedRun(t *testing.T) *env {
	t.Helper()
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Providers = []string{"openai"}
	e.gw.onAsk = func(n int) {
		if n != 4 {
			return
		}
		if err := e.svc.StopRun(t.Context(), "joris", e.run(t).ID); err != nil {
			t.Error(err)
		}
	}
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	e.gw.onAsk = nil
	return e
}

func TestAStopDuringARunKeepsTheResponsesSoFarAndIsRecordedAsStoppedByHand(t *testing.T) {
	e := stoppedRun(t)
	run := e.run(t)
	if run.Status != domain.RunStopped || run.Failure != domain.StoppedByHand {
		t.Fatalf("run %s (%q), want stopped by hand", run.Status, run.Failure)
	}
	// The question asked when the stop came is answered and kept; no question is asked after it.
	if n := len(e.repo.answers); n != 4 || run.Tasks != 40 {
		t.Fatalf("%d of %d responses kept, want the 4 asked before the stop", n, run.Tasks)
	}
	for _, a := range e.repo.answers {
		if a.Failure != "" || a.Extraction == nil {
			t.Errorf("a kept response is failed or unread: %+v", a)
		}
	}
}

func TestARetryOfARunStoppedByHandAsksOnlyTheQuestionsItDidNotGet(t *testing.T) {
	e := stoppedRun(t)
	run := e.run(t)
	retry, err := e.svc.Retry(t.Context(), run.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if retry.Tasks != 36 {
		t.Errorf("retry asks %d questions, want the 36 the stop left", retry.Tasks)
	}
	if got := e.run(t); got.Status != domain.RunDone || got.Failure != "" || len(e.repo.answers) != 40 || e.gw.asked["openai/gpt-oss-20b"] != 40 {
		t.Errorf("run %s (%q) with %d responses after %d questions", got.Status, got.Failure, len(e.repo.answers), e.gw.asked["openai/gpt-oss-20b"])
	}
}

func TestAStopRequestedBeforeTheFirstQuestionAsksNothing(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	run := domain.Run{ID: uuid.New(), PromptSetVersion: 2, FactSheet: "facts", Status: domain.RunRunning, Models: []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}}}
	e.repo.runs = append(e.repo.runs, &storedRun{run: run, owner: "a"})
	if err := e.svc.StopRun(t.Context(), "joris", run.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Execute(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	if got := e.run(t); got.Status != domain.RunStopped || got.Failure != domain.StoppedByHand || len(e.gw.asked) != 0 {
		t.Errorf("run %s (%q) after asking %v", got.Status, got.Failure, e.gw.asked)
	}
}

func TestAStopReachesARunWhileItWaitsOnASlowModel(t *testing.T) {
	r := newRepo()
	e := newEnv(t, r, "a")
	e.gw.slow = 40 * time.Millisecond
	run := domain.Run{ID: uuid.New(), PromptSetVersion: 2, FactSheet: "facts", Status: domain.RunRunning, Models: []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}}}
	r.runs = append(r.runs, &storedRun{run: run, owner: "a"})
	svc := app.New(r, gateways{gw: e.gw}, e.dir, time.Now, app.Config{Owner: "a", Stale: 8 * time.Millisecond}, func(uuid.UUID, error) {},
		app.WithSleep(e.sleeps.Sleep))
	e.gw.onAsk = func(n int) {
		if n == 1 {
			if err := svc.StopRun(t.Context(), "joris", run.ID); err != nil {
				t.Error(err)
			}
		}
	}
	if err := svc.Execute(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	if got := e.run(t); got.Status != domain.RunStopped || got.Failure != domain.StoppedByHand || len(r.answers) != 1 {
		t.Errorf("run %s (%q) with %d responses", got.Status, got.Failure, len(r.answers))
	}
}

func TestStoppingARunThatIsNotRunningIsNotFound(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Providers = []string{"openai"}
	if err := e.svc.StopRun(t.Context(), "joris", uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("stopping no run = %v, want ErrNotFound", err)
	}
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.StopRun(t.Context(), "joris", e.run(t).ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("stopping a finished run = %v, want ErrNotFound", err)
	}
	if got := e.run(t); got.Status != domain.RunDone {
		t.Errorf("the finished run is %s", got.Status)
	}
}

func TestAStopThatCannotBeStoredIsAnError(t *testing.T) {
	e := newEnv(t, newRepo("RequestStop"), "a")
	if err := e.svc.StopRun(t.Context(), "joris", uuid.New()); !errors.Is(err, errBoom) {
		t.Errorf("err = %v, want the repository's", err)
	}
}

func TestAStopRequestThatCannotBeReadLetsTheRunGoOn(t *testing.T) {
	e := newEnv(t, newRepo("StopRequested"), "a")
	e.repo.settings.Providers = []string{"openai"}
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if got := e.run(t); got.Status != domain.RunDone || len(e.repo.answers) != 40 {
		t.Errorf("run %s with %d responses", got.Status, len(e.repo.answers))
	}
}
