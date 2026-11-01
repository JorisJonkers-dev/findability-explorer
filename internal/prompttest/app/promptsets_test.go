package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain/domaintest"
)

// version3 is a published version newer than the imported ones: one product question.
func version3() domain.PublishedSet {
	return domain.PublishedSet{
		Version: 3, PublishedAt: start.Add(-time.Hour), PublishedBy: "joris", Note: "One question.",
		Questions: []domain.Question{{
			ID: "eyelink-product-german", Family: domain.FamilyEyelink, Intent: domain.IntentProduct,
			Scope: domaintest.Scope(domain.ScopeGerman), Text: "Wer liefert Ösenbänder?",
		}},
	}
}

func TestANewRunAsksTheNewestPublishedVersion(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.sets = []domain.PublishedSet{version3(), domaintest.Version2()}
	run, err := e.svc.Start(t.Context(), "joris")
	if err != nil {
		t.Fatal(err)
	}
	// One question, two providers, two modes.
	if run.PromptSetVersion != 3 || run.Tasks != 4 || len(e.repo.answers) != 4 {
		t.Fatalf("run = %+v with %d answers", run, len(e.repo.answers))
	}
	a := e.repo.answers[0]
	want := domain.Prompt{ID: "eyelink-product-german", Family: domain.FamilyEyelink, Kind: domain.KindSupplier, Scope: domain.ScopeGerman, Language: "de", Text: "Wer liefert Ösenbänder?"}
	if a.Prompt != want || a.PromptSetVersion != 3 {
		t.Errorf("answer = %+v", a)
	}
	view, err := e.svc.Settings(t.Context())
	if err != nil || view.PromptSet.Version != 3 || len(view.PromptSet.Prompts) != 1 {
		t.Errorf("the settings show %+v, %v", view.PromptSet, err)
	}
	res, err := e.svc.Results(t.Context())
	if err != nil || res.PromptSetVersion != 3 {
		t.Errorf("the results are of version %d, %v", res.PromptSetVersion, err)
	}
}

func TestARetryOfAnEarlierRunAsksTheQuestionsOfItsPublishedVersion(t *testing.T) {
	e, run := earlierRun(t)
	// Version 1 as published: the two questions the answers kept, and a third none kept.
	asked := []domain.Prompt{e.repo.answers[0].Prompt, e.repo.answers[2].Prompt}
	v1 := domain.PublishedSet{Version: 1, Note: "Three questions."}
	for _, p := range asked {
		intent := domain.Intent(p.Kind)
		if p.Kind == domain.KindSupplier {
			intent = domain.IntentProduct
		}
		v1.Questions = append(v1.Questions, domain.Question{ID: p.ID, Family: p.Family, Intent: intent, Scope: domaintest.Scope(p.Scope), Text: p.Text})
	}
	v1.Questions = append(v1.Questions, domain.Question{
		ID: "eyelink-application-english", Family: domain.FamilyEyelink, Intent: domain.IntentApplication,
		Scope: domaintest.Scope(domain.ScopeEnglish), Text: "Which belt for an oven?",
	})
	e.repo.sets = append(e.repo.sets, v1)
	plan, err := e.svc.RetryPlan(t.Context(), run.ID)
	// The failed answer, the mode it was never asked in, and both modes of the third question.
	if err != nil || plan.Ask != 4 || plan.Unknown != 0 {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	got, err := e.svc.Retry(t.Context(), run.ID, nil)
	if err != nil || got.Tasks != 4 {
		t.Fatalf("retry = %+v, %v", got, err)
	}
	if failures(e) != 0 || len(e.repo.answers) != 6 {
		t.Fatalf("after asking again: %d failures in %d answers", failures(e), len(e.repo.answers))
	}
	for _, a := range e.repo.answers[3:] {
		if a.PromptSetVersion != 1 {
			t.Errorf("asked again = %+v", a)
		}
	}
}

func TestNoRunStartsWhileNoPromptSetIsPublished(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.sets = nil
	if _, err := e.svc.Start(t.Context(), "joris"); !errors.Is(err, domain.ErrNoPromptSet) {
		t.Errorf("start = %v", err)
	}
	if _, err := e.svc.Settings(t.Context()); !errors.Is(err, domain.ErrNoPromptSet) {
		t.Errorf("settings = %v", err)
	}
}

func TestWhatReadsThePromptSetsFailsWhenTheyCannotBeRead(t *testing.T) {
	e := failedRun(t)
	run := e.run(t)
	e.repo.failOn["PromptSets"] = true
	if _, err := e.svc.Start(t.Context(), "joris"); !errors.Is(err, errBoom) {
		t.Errorf("start = %v", err)
	}
	if _, err := e.svc.Settings(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("settings = %v", err)
	}
	if _, err := e.svc.Results(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("results = %v", err)
	}
	if _, err := e.svc.RetryPlan(t.Context(), run.ID); !errors.Is(err, errBoom) {
		t.Errorf("retry plan = %v", err)
	}
	if _, err := e.svc.Retry(t.Context(), run.ID, nil); !errors.Is(err, errBoom) {
		t.Errorf("retry = %v", err)
	}
	if _, err := e.svc.PromptSets(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("prompt sets = %v", err)
	}
	e = newEnv(t, newRepo("PromptScopes"), "a")
	if _, err := e.svc.PromptSets(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("prompt sets without their scopes = %v", err)
	}
}

func TestThePromptSetsAreThePublishedVersionsWithEveryScope(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.sets = []domain.PublishedSet{version3(), domaintest.Version2()}
	got, err := e.svc.PromptSets(t.Context())
	if err != nil || len(got.Sets) != 2 || got.Sets[0].Version != 3 || got.Sets[1].Version != 2 || len(got.Sets[1].Questions) != 20 {
		t.Fatalf("sets = %+v, %v", got.Sets, err)
	}
	if len(got.Scopes) != 6 || got.Scopes[0] != domaintest.Scope(domain.ScopeTwente) {
		t.Errorf("scopes = %+v", got.Scopes)
	}
}
