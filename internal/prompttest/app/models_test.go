package app_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func TestASavedModelListIsWhatRunsAsk(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	view, err := e.svc.Update(t.Context(), "joris", app.Change{Models: []string{" google/gemini-flash-lite ", "openai/gpt-oss-20b", "google/gemma:free"}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(view.Settings.Models, []string{"google/gemini-flash-lite", "openai/gpt-oss-20b", "google/gemma:free"}) {
		t.Fatalf("models = %v", view.Settings.Models)
	}
	// Before any run, the list is priced by OpenRouter's catalogue, one row per model.
	if view.Prices.Basis != domain.BasisCatalogue || len(view.Prices.Providers) != 3 || view.Prices.Providers[0].Model != "google/gemini-flash-lite" {
		t.Errorf("prices = %+v", view.Prices)
	}
	run, err := e.svc.Start(t.Context(), "joris")
	if err != nil {
		t.Fatal(err)
	}
	// Two models of Google are a column each; OpenAI's one keeps its provider as its column.
	want := []domain.ModelChoice{
		{Provider: "google/gemini-flash-lite", Model: "google/gemini-flash-lite"},
		{Provider: "openai", Model: "openai/gpt-oss-20b"},
		{Provider: "google/gemma:free", Model: "google/gemma:free", Free: true, Fallback: "google/gemini-flash-lite"},
	}
	if !slices.Equal(run.Models, want) {
		t.Fatalf("run models = %+v", run.Models)
	}
	if e.gw.asked["google/gemini-flash-lite"] == 0 || e.gw.asked["google/gemma:free"] == 0 {
		t.Errorf("asked = %v", e.gw.asked)
	}
	// After the run, the asked models are priced by their own responses.
	after, err := e.svc.Settings(t.Context())
	if err != nil || after.Prices.Basis != domain.BasisRun || len(after.Prices.Providers) != 3 || after.Prices.Providers[0].Knowledge != 10 {
		t.Errorf("after a run = %+v, %v", after.Prices, err)
	}
	// A model added since is estimated beside them.
	e.repo.settings.Models = append(e.repo.settings.Models, "openai/gpt-new")
	e.gw.catalogue = append(e.gw.catalogue, domain.CatalogueModel{ID: "openai/gpt-new", PromptPrice: 0.000001, CompletionPrice: 0.000001, InputModalities: []string{"text"}, OutputModalities: []string{"text"}})
	added, err := e.svc.Settings(t.Context())
	if err != nil || len(added.Prices.Providers) != 4 || added.Prices.Providers[3].Model != "openai/gpt-new" || added.Prices.Providers[3].Provider != "openai/gpt-new" {
		t.Errorf("with a model added = %+v, %v", added.Prices, err)
	}
}

func TestAModelListWithNothingOnTheCatalogueStartsNoRun(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Models = []string{"gone/model"}
	if _, err := e.svc.Start(t.Context(), "joris"); !errors.Is(err, domain.ErrNoModels) {
		t.Errorf("Start = %v", err)
	}
	if _, err := e.svc.Update(t.Context(), "joris", app.Change{Models: []string{"no model"}}); !errors.Is(err, domain.ErrInvalid) || !app.IsInputError(err) {
		t.Errorf("Update = %v", err)
	}
}

func TestTheCatalogueIsReadOnceADayAndKeptWhenOpenRouterFails(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	got, err := e.svc.Catalogue(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !got.ReadAt.Equal(start) || got.Questions != 20 || len(got.Entries) != 3 || !got.Entries[0].Recommended || got.Entries[0].Model.ID != "openai/gpt-oss-20b" {
		t.Fatalf("catalogue = %+v", got)
	}
	// Within the day it is not read again.
	e.clock.set(start.Add(23 * time.Hour))
	if again, err := e.svc.Catalogue(t.Context()); err != nil || !again.ReadAt.Equal(start) || e.gw.modelCalls != 1 {
		t.Errorf("within a day = %v, %v, %d reads", again.ReadAt, err, e.gw.modelCalls)
	}
	// A day later it is, and a failing read keeps the day-old list.
	e.clock.set(start.Add(25 * time.Hour))
	e.gw.modelsErr = errBoom
	if kept, err := e.svc.Catalogue(t.Context()); err != nil || !kept.ReadAt.Equal(start) || e.gw.modelCalls != 2 {
		t.Errorf("a failing read = %v, %v, %d reads", kept.ReadAt, err, e.gw.modelCalls)
	}
	e.gw.modelsErr = nil
	if fresh, err := e.svc.Catalogue(t.Context()); err != nil || !fresh.ReadAt.Equal(start.Add(25*time.Hour)) {
		t.Errorf("read again = %v, %v", fresh.ReadAt, err)
	}
	// After a run, a model it asked is priced by its responses.
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	priced, err := e.svc.Catalogue(t.Context())
	if err != nil || priced.Entries[0].Basis != domain.BasisRun || priced.Entries[0].Knowledge != 10 {
		t.Errorf("after a run = %+v, %v", priced.Entries[0], err)
	}
}

func TestTheCatalogueNeedsAWorkingKey(t *testing.T) {
	r := newRepo()
	s := domain.DefaultSettings(start)
	r.settings = &s
	e := newEnv(t, r, "a")
	if _, err := e.svc.Catalogue(t.Context()); !errors.Is(err, domain.ErrNoKey) {
		t.Errorf("no key = %v", err)
	}
	e = newEnv(t, newRepo(), "a")
	e.gw.modelsErr = errBoom
	if _, err := e.svc.Catalogue(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("never read = %v", err)
	}
	e = newEnv(t, newRepo(), "a")
	e.repo.failOn["Runs"] = true
	if _, err := e.svc.Catalogue(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("runs failing = %v", err)
	}
	e = newEnv(t, newRepo(), "a")
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	e.repo.failOn["Answers"] = true
	if _, err := e.svc.Catalogue(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("answers failing = %v", err)
	}
	e = newEnv(t, newRepo(), "a")
	e.repo.failOn["PromptSets"] = true
	if _, err := e.svc.Catalogue(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("prompt sets failing = %v", err)
	}
	e = newEnv(t, newRepo(), "a")
	e.repo.failOn["Settings"] = true
	if _, err := e.svc.Catalogue(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("settings failing = %v", err)
	}
	e = newEnv(t, newRepo(), "a")
	svc := app.New(e.repo, gateways{gw: e.gw, gatewayErr: errBoom}, e.dir, e.clock.Now, app.Config{}, func(uuid.UUID, error) {})
	if _, err := svc.Catalogue(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("no gateway = %v", err)
	}
}
