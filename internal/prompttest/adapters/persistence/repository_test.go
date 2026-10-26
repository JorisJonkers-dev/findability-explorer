package persistence_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func repository(t *testing.T) *persistence.Repository {
	t.Helper()
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	return persistence.New(store)
}

var at = time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)

func run(trigger domain.Trigger) domain.Run {
	return domain.Run{
		ID: uuid.New(), PromptSetVersion: 1, FactSheetVersion: 2, FactSheet: "Tribelt is in Haaksbergen.", Trigger: trigger,
		StartedBy: "joris", StartedAt: at, Status: domain.RunRunning, Tasks: 360,
		Models: []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}, {Provider: "google", Model: "google/gemma:free", Free: true, Fallback: "google/gemini"}},
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	if _, ok, err := repo.Settings(ctx); ok || err != nil {
		t.Fatalf("settings before any = %v, %v", ok, err)
	}
	s := domain.DefaultSettings(at)
	s.CredentialID = "3f2a9c4e-7d1b-4c6a-9e8f-1a2b3c4d5e6f"
	s.FactSheet = domain.FactSheet{Text: "Tribelt makes belts.", Version: 2, UpdatedBy: "joris", UpdatedAt: at}
	if err := repo.SaveSettings(ctx, s, "joris", at); err != nil {
		t.Fatal(err)
	}
	got, ok, err := repo.Settings(ctx)
	if err != nil || !ok || got.CredentialID != s.CredentialID || !slices.Equal(got.Providers, s.Providers) || got.MonthlyBudget != s.MonthlyBudget ||
		got.Cadence != s.Cadence || !got.NextDue.Equal(s.NextDue) || got.FactSheet != s.FactSheet {
		t.Errorf("settings = %+v, %v, %v", got, ok, err)
	}
	s.CredentialID, s.FactSheet.UpdatedBy, s.FactSheet.UpdatedAt = "", "", time.Time{}
	if err := repo.SaveSettings(ctx, s, "joris", at); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := repo.Settings(ctx); got.CredentialID != "" || !got.FactSheet.UpdatedAt.IsZero() {
		t.Errorf("cleared settings = %+v", got)
	}
}

func TestOneRunAtATimeAndTheLease(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	first := run(domain.TriggerManual)
	if err := repo.CreateRun(ctx, first, "a", at); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRun(ctx, run(domain.TriggerManual), "b", at); !errors.Is(err, domain.ErrRunActive) {
		t.Fatalf("a second running run = %v", err)
	}
	got, err := repo.Run(ctx, first.ID)
	if err != nil || got.Status != domain.RunRunning || !slices.Equal(got.Models, first.Models) || got.FactSheet != first.FactSheet || got.Tasks != 360 {
		t.Fatalf("run = %+v, %v", got, err)
	}
	if held, err := repo.Heartbeat(ctx, first.ID, "b", at); held || err != nil {
		t.Errorf("another owner's heartbeat = %v, %v", held, err)
	}
	if held, _ := repo.Heartbeat(ctx, first.ID, "a", at.Add(time.Minute)); !held {
		t.Error("the owner's heartbeat failed")
	}
	if stale, err := repo.AdoptStale(ctx, "b", at, at.Add(2*time.Minute)); stale != nil || err != nil {
		t.Errorf("adopted a live run: %+v, %v", stale, err)
	}
	stale, err := repo.AdoptStale(ctx, "b", at.Add(2*time.Minute), at.Add(3*time.Minute))
	if err != nil || stale == nil || stale.ID != first.ID {
		t.Fatalf("adopt = %+v, %v", stale, err)
	}
	if held, _ := repo.Heartbeat(ctx, first.ID, "a", at.Add(4*time.Minute)); held {
		t.Error("the old owner still holds the run")
	}
	if err := repo.FinishRun(ctx, first.ID, domain.RunStopped, "budget", at.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.Run(ctx, first.ID); got.Status != domain.RunStopped || got.Failure != "budget" || !got.FinishedAt.Equal(at.Add(5*time.Minute)) {
		t.Errorf("finished run = %+v", got)
	}
	if _, err := repo.Run(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("no run = %v", err)
	}
	second := run(domain.TriggerManual)
	second.StartedAt = at.Add(time.Hour)
	if err := repo.CreateRun(ctx, second, "a", at); err != nil {
		t.Fatal(err)
	}
	runs, err := repo.Runs(ctx, 10)
	if err != nil || len(runs) != 2 || runs[0].ID != second.ID {
		t.Errorf("runs = %d, %v", len(runs), err)
	}
}

func TestTheScheduleIsClaimedOnce(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	if claimed, err := repo.ClaimDue(ctx, at, at, run(domain.TriggerSchedule), "a"); claimed || err != nil {
		t.Fatalf("claimed without settings: %v, %v", claimed, err)
	}
	s := domain.DefaultSettings(at) // due 1 November, 06:00
	if err := repo.SaveSettings(ctx, s, "joris", at); err != nil {
		t.Fatal(err)
	}
	if claimed, _ := repo.ClaimDue(ctx, at, at, run(domain.TriggerSchedule), "a"); claimed {
		t.Fatal("claimed before it was due")
	}
	due := s.NextDue.Add(time.Minute)
	next := domain.NextDue(due, domain.CadenceMonthly)
	if claimed, err := repo.ClaimDue(ctx, due, next, run(domain.TriggerSchedule), "a"); !claimed || err != nil {
		t.Fatalf("claim = %v, %v", claimed, err)
	}
	if claimed, _ := repo.ClaimDue(ctx, due, next, run(domain.TriggerSchedule), "b"); claimed {
		t.Error("claimed twice")
	}
	if got, _, _ := repo.Settings(ctx); !got.NextDue.Equal(next) {
		t.Errorf("next due = %v", got.NextDue)
	}
	// Due again while a run is running: it waits, still due.
	if claimed, _ := repo.ClaimDue(ctx, next.Add(time.Minute), domain.NextDue(next, domain.CadenceMonthly), run(domain.TriggerSchedule), "b"); claimed {
		t.Error("claimed beside a running run")
	}
	if got, _, _ := repo.Settings(ctx); !got.NextDue.Equal(next) {
		t.Errorf("a waiting due run moved the schedule to %v", got.NextDue)
	}
}

func TestAnswersExtractionsSpendAndCandidates(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	r := run(domain.TriggerManual)
	if err := repo.CreateRun(ctx, r, "a", at); err != nil {
		t.Fatal(err)
	}
	prompt := domain.CurrentPromptSet().Prompts[0]
	read := domain.Answer{
		ID: uuid.New(), RunID: r.ID, Prompt: prompt, PromptSetVersion: 1, Provider: "openai", Model: "openai/gpt-oss-20b", Mode: domain.ModeWeb,
		Reply: domain.Reply{Text: "Twentebelt and Ashworth.\x00", Citations: []string{"https://twentebelt.com"}, InputTokens: 10, OutputTokens: 20, Cost: 1500},
		At:    at,
		Extraction: &domain.Extraction{
			Companies:    []domain.Company{{Name: "Twentebelt", CompetitorID: "twentebelt", Position: 1}, {Name: "Ashworth", URL: "https://ashworth.com", Position: 2}},
			Addresses:    []string{"https://twentebelt.com"},
			NamesTribelt: true, CitesTribelt: true, Correctness: domain.CorrectnessWrong, WrongClaims: []string{"Tribelt is in Hengelo."},
			InputTokens: 5, OutputTokens: 6, Cost: 200,
		},
	}
	unread := domain.Answer{ID: uuid.New(), RunID: r.ID, Prompt: prompt, PromptSetVersion: 1, Provider: "google", Model: "google/gemma:free", Mode: domain.ModeKnowledge, Reply: domain.Reply{Text: "No idea."}, At: at.Add(time.Millisecond)}
	failed := domain.Answer{ID: uuid.New(), RunID: r.ID, Prompt: prompt, PromptSetVersion: 1, Provider: "google", Model: "google/gemma:free", Mode: domain.ModeWeb, Failure: "Rate limited.", At: at.Add(time.Second)}
	for _, a := range []domain.Answer{read, unread, failed} {
		if err := repo.AddAnswer(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.AddAnswer(ctx, read); err == nil {
		t.Error("the same task answered twice")
	}
	if err := repo.SetExtraction(ctx, unread.ID, domain.Extraction{Failed: true, Cost: 7}, at); err != nil {
		t.Fatal(err)
	}
	answers, err := repo.Answers(ctx, r.ID, true)
	if err != nil || len(answers) != 3 {
		t.Fatalf("answers = %d, %v", len(answers), err)
	}
	got := answers[0]
	if got.Text != "Twentebelt and Ashworth." || got.Prompt != prompt || got.Cost != 1500 || !slices.Equal(got.Citations, read.Citations) ||
		got.Extraction == nil || !slices.Equal(got.Extraction.Companies, read.Extraction.Companies) || got.Extraction.Correctness != domain.CorrectnessWrong ||
		!slices.Equal(got.Extraction.WrongClaims, read.Extraction.WrongClaims) || got.Extraction.Cost != 200 {
		t.Errorf("answer = %+v, extraction %+v", got, got.Extraction)
	}
	if answers[1].Extraction == nil || !answers[1].Extraction.Failed || answers[2].Failure != "Rate limited." || answers[2].Extraction != nil {
		t.Errorf("unread %+v, failed %+v", answers[1].Extraction, answers[2])
	}
	if bare, _ := repo.Answers(ctx, r.ID, false); bare[0].Text != "" || bare[0].Extraction == nil {
		t.Errorf("answers without text = %+v", bare[0])
	}
	if spent, err := repo.SpentSince(ctx, at); err != nil || spent != 1500+200+7 {
		t.Errorf("spent = %d, %v", spent, err)
	}
	if spent, _ := repo.SpentSince(ctx, at.Add(time.Hour)); spent != 0 {
		t.Errorf("spent later = %d", spent)
	}
	sightings, err := repo.Sightings(ctx)
	if err != nil || len(sightings) != 1 || sightings[0] != (domain.Sighting{Name: "Ashworth", URL: "https://ashworth.com", Provider: "openai", At: at}) {
		t.Errorf("sightings = %+v, %v", sightings, err)
	}

	d := domain.CandidateDecision{ID: domain.CandidateID("ashworth"), Name: "Ashworth", Status: domain.CandidateAdded, CompetitorID: "ashworth", By: "joris", At: at}
	if err := repo.Decide(ctx, d); err != nil {
		t.Fatal(err)
	}
	if decisions, err := repo.CandidateDecisions(ctx); err != nil || len(decisions) != 1 || decisions[0] != d {
		t.Errorf("decisions = %+v, %v", decisions, err)
	}
	if err := repo.Undecide(ctx, d.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.Undecide(ctx, d.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("undecide of nothing = %v", err)
	}
}
