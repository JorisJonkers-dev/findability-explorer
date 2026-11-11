package persistence_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain/domaintest"
)

// A published version is stored with its new scope, read back as it was published, and its answers
// in the new scope, language and kind fit the answers' checks.
func TestAPublishedVersionIsStoredWithItsNewScopeAndItsAnswersFit(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	lyon := domain.ScopeInfo{ID: "fr-lyon", Language: "fr", Region: "Lyon", Name: "French, Lyon"}
	fr := domain.ScopeInfo{ID: "fr", Language: "fr", Name: "French, no region"}
	set := domain.PublishedSet{
		Version: 3, PublishedAt: at, PublishedBy: "joris", Note: "Adds 2 questions, changes none, and removes none. Adds the scopes French, Lyon; French, no region.",
		Questions: []domain.Question{
			{ID: "eyelink-application-fr-lyon", Family: domain.FamilyEyelink, Intent: domain.IntentApplication, Scope: lyon, Text: "Quelle bande ?", English: "Which belt?"},
			{ID: "other-problem-fr", Family: domain.FamilyOther, Intent: domain.IntentProblem, Scope: fr, Text: "Pourquoi ?"},
			{ID: "other-application-english", Family: domain.FamilyOther, Intent: domain.IntentApplication, Scope: domaintest.Scope(domain.ScopeEnglish), Text: "Which belt?"},
		},
	}
	if err := repo.PublishPromptSet(ctx, set, []domain.ScopeInfo{lyon, fr}); err != nil {
		t.Fatal(err)
	}
	sets, err := repo.PromptSets(ctx)
	if err != nil || len(sets) != 3 || !reflect.DeepEqual(sets[0], set) {
		t.Fatalf("newest = %+v, %v", sets[0], err)
	}
	scopes, err := repo.PromptScopes(ctx)
	if err != nil || len(scopes) != 8 {
		t.Fatalf("scopes = %+v, %v", scopes, err)
	}
	if err := repo.PublishPromptSet(ctx, set, nil); !errors.Is(err, domain.ErrVersionTaken) {
		t.Errorf("the same version again: err = %v", err)
	}

	r := run(domain.TriggerManual)
	r.PromptSetVersion = 3
	if err := repo.CreateRun(ctx, r, "a", at); err != nil {
		t.Fatal(err)
	}
	for _, p := range set.PromptSet().Prompts {
		a := domain.Answer{ID: uuid.New(), RunID: r.ID, Prompt: p, PromptSetVersion: 3, Provider: "openai", Model: "openai/gpt-oss-20b", Mode: domain.ModeWeb, At: at}
		if err := repo.AddAnswer(ctx, a); err != nil {
			t.Fatalf("%s: %v", p.ID, err)
		}
	}
}

func TestPublishingRollsBackWhenAPartFails(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	twice := domain.ScopeInfo{ID: "twente", Language: "nl", Region: "Twente", Name: "Twente again"}
	if err := repo.PublishPromptSet(ctx, domain.PublishedSet{Version: 3, PublishedAt: at, PublishedBy: "joris", Note: "x"}, []domain.ScopeInfo{twice}); err == nil {
		t.Error("a scope that exists was stored again")
	}
	unknown := domain.PublishedSet{Version: 3, PublishedAt: at, PublishedBy: "joris", Note: "x", Questions: []domain.Question{
		{ID: "q", Family: domain.FamilyOther, Intent: domain.IntentProblem, Scope: domain.ScopeInfo{ID: "mars"}, Text: "?"},
	}}
	if err := repo.PublishPromptSet(ctx, unknown, nil); err == nil {
		t.Error("a question in a scope that does not exist was stored")
	}
	if err := repo.PublishPromptSet(ctx, domain.PublishedSet{Version: 3, PublishedAt: at, PublishedBy: "", Note: "x"}, nil); err == nil {
		t.Error("a version without its publisher was stored")
	}
	sets, err := repo.PromptSets(ctx)
	if err != nil || len(sets) != 2 {
		t.Errorf("sets = %d, %v: a failed publication left a version behind", len(sets), err)
	}
}
