package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain/domaintest"
)

// partialRun stores a finished run of 40 questions that got three responses, each read: the rest
// failed, most because OpenRouter refused the key.
func partialRun(r *repo) domain.Run {
	run := domain.Run{
		ID: uuid.New(), PromptSetVersion: 1, FactSheet: "facts", StartedAt: start.Add(-time.Hour), Status: domain.RunDone, Tasks: 40,
		Models: []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}},
	}
	r.runs = append(r.runs, &storedRun{run: run, owner: "a"})
	tasks := domain.Tasks(domaintest.Version2().PromptSet(), run.Models)
	for i, t := range tasks[:10] {
		a := domain.Answer{ID: uuid.New(), RunID: run.ID, Prompt: t.Prompt, PromptSetVersion: 1, Provider: t.Provider, Mode: t.Mode}
		switch {
		case i < 3:
			a.Text, a.Extraction = "Tribelt.", &domain.Extraction{NamesTribelt: true}
		case i < 8:
			a.Failure = "OpenRouter answered 401: User not found."
		default:
			a.Failure = "The model is rate limited."
		}
		r.answers = append(r.answers, a)
	}
	return run
}

func TestTheHistoryListsTheRunsWithResponsesReadOldestFirstWithWhyAPartialRunFailed(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Providers = []string{"openai"}
	e.repo.sets = append(e.repo.sets, domain.PublishedSet{Version: 1, Note: "The first.", Questions: []domain.Question{{ID: "q"}}})
	partial := partialRun(e.repo)
	// A run whose every question failed read nothing, so it is left out.
	e.gw.askErr = errors.New("the provider is down")
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	e.gw.askErr = nil
	e.clock.set(start.Add(time.Hour))
	full, err := e.svc.Start(t.Context(), "joris")
	if err != nil {
		t.Fatal(err)
	}
	h, err := e.svc.History(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Versions) != 2 || h.Versions[0] != (domain.Version{Number: 1, Questions: 1, Note: "The first."}) || h.Versions[1].Number != 2 || h.Versions[1].Questions != 20 {
		t.Errorf("versions = %+v", h.Versions)
	}
	if len(h.Runs) != 2 || h.Runs[0].RunID != partial.ID || h.Runs[1].RunID != full.ID {
		t.Fatalf("history = %+v, want the partial run, then the full one", h.Runs)
	}
	p, f := h.Runs[0], h.Runs[1]
	if !p.Partial || p.Version != 1 || p.Answered != 3 || p.Read != 3 || p.All == nil || *p.All != 1 {
		t.Errorf("partial run = %+v", p)
	}
	if f.Partial || f.Version != 2 || f.Read != 40 || f.Citation == nil || *f.Citation != 1 {
		t.Errorf("full run = %+v", f)
	}
	if len(h.FailureOf) != 1 || h.FailureOf[partial.ID] != "OpenRouter answered 401: User not found." {
		t.Errorf("failures = %v, want the commonest of the partial run only", h.FailureOf)
	}
}

func TestTheHistoryFailsWhenTheRunsOrThePromptSetsCannotBeRead(t *testing.T) {
	for _, op := range []string{"Runs", "PromptSets", "PromptScopes"} {
		e := newEnv(t, newRepo(op), "a")
		if _, err := e.svc.History(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("%s failing: err = %v, want the repository's", op, err)
		}
	}
}
