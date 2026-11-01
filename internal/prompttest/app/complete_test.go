package app_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain/domaintest"
)

// readRun stores a run with status, started at, with one response read that names Tribelt.
func readRun(r *repo, status domain.RunStatus, at time.Time) domain.Run {
	run := domain.Run{
		ID: uuid.New(), PromptSetVersion: 2, FactSheet: "facts", StartedAt: at, Status: status, Tasks: 1,
		Models: []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}},
	}
	if status != domain.RunRunning {
		run.FinishedAt = at.Add(time.Minute)
	}
	r.runs = append(r.runs, &storedRun{run: run, owner: "a"})
	t := domain.Tasks(domaintest.Version2().PromptSet(), run.Models)[0]
	r.answers = append(r.answers, domain.Answer{
		ID: uuid.New(), RunID: run.ID, Prompt: t.Prompt, PromptSetVersion: 2, Provider: t.Provider, Mode: t.Mode, Reply: domain.Reply{Text: "Tribelt."},
		Extraction: &domain.Extraction{NamesTribelt: true, Companies: []domain.Company{{Name: "Tribelt", CompetitorID: "tribelt", Position: 1}}},
	})
	return run
}

func TestOtherContextsCompareOnTheNewestCompleteRun(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	complete := readRun(e.repo, domain.RunDone, start)
	readRun(e.repo, domain.RunStopped, start.Add(time.Hour))
	readRun(e.repo, domain.RunFailed, start.Add(2*time.Hour))
	readRun(e.repo, domain.RunRunning, start.Add(3*time.Hour))
	counted, err := e.svc.Counted(t.Context(), 1)
	if err != nil || len(counted) != 1 || counted[0].Run.ID != complete.ID {
		t.Fatalf("Counted = %+v, %v", counted, err)
	}
	res, err := e.svc.Results(t.Context())
	if err != nil || len(res.Runs) != 4 || len(res.Mentions) != 1 || res.Mentions[0].Answers != 1 {
		t.Fatalf("results = %d runs, mentions %+v, %v", len(res.Runs), res.Mentions, err)
	}
	// The mentions are the complete run's: with none, there are none.
	e2 := newEnv(t, newRepo(), "a")
	readRun(e2.repo, domain.RunStopped, start)
	if res, err := e2.svc.Results(t.Context()); err != nil || len(res.Mentions) != 0 {
		t.Errorf("without a complete run: mentions %+v, %v", res.Mentions, err)
	}
	if counted, err := e2.svc.Counted(t.Context(), 1); err != nil || len(counted) != 0 {
		t.Errorf("Counted without a complete run = %+v, %v", counted, err)
	}
}

func TestTheHistoryListsEveryRunReadAndLeavesOutTheOneStillRunning(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	for i := range 30 {
		readRun(e.repo, domain.RunDone, start.Add(time.Duration(i)*time.Hour))
	}
	running := readRun(e.repo, domain.RunRunning, start.Add(40*time.Hour))
	h, err := e.svc.History(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Runs) != 30 {
		t.Errorf("the history has %d runs, want every finished one", len(h.Runs))
	}
	for _, r := range h.Runs {
		if r.RunID == running.ID {
			t.Error("the running run is in the history")
		}
	}
}
