package persistence_test

import (
	"io/fs"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/JorisJonkers-dev/findability-explorer/db"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain/domaintest"
)

// The migration runs on a database migrated past it, so the test holds when later ones are added;
// it is written to run again without harm.
func TestTheMigrationMarksStoredEmptyResponsesFailedAndRemovesTheirReadings(t *testing.T) {
	t.Parallel()
	url := pgtest.URL(t)
	ctx := t.Context()
	store, err := pg.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	repo := persistence.New(store)
	r := run(domain.TriggerManual)
	if err := repo.CreateRun(ctx, r, "a", at); err != nil {
		t.Fatal(err)
	}
	prompts := domaintest.Version2().PromptSet().Prompts
	answer := func(p domain.Prompt, text, failure string) domain.Answer {
		return domain.Answer{
			ID: uuid.New(), RunID: r.ID, Prompt: p, PromptSetVersion: 1, Provider: "openai", Model: "openai/gpt-oss-20b", Mode: domain.ModeKnowledge,
			Reply: domain.Reply{Text: text, InputTokens: 20, OutputTokens: 2000, Cost: 10}, Failure: failure, At: at,
		}
	}
	// Stored before an empty response was a failed one: read, and naming a company.
	empty := answer(prompts[0], " \n ", "")
	full := answer(prompts[1], "Twentebelt makes these belts.", "")
	failed := answer(prompts[2], "", "Rate limited.")
	reading := domain.Extraction{Companies: []domain.Company{{Name: "Twentebelt", CompetitorID: "twentebelt", Position: 1}}, Cost: 2}
	for _, a := range []domain.Answer{empty, full, failed} {
		if err := repo.AddAnswer(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	for _, a := range []domain.Answer{empty, full} {
		if err := repo.SetExtraction(ctx, a.ID, reading, at); err != nil {
			t.Fatal(err)
		}
	}

	migration, err := fs.ReadFile(db.Migrations, "migrations/00027_empty_responses_fail.sql")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	for range 2 {
		if _, err := conn.Exec(ctx, string(migration)); err != nil {
			t.Fatal(err)
		}
	}

	answers, err := repo.Answers(ctx, r.ID, true)
	if err != nil || len(answers) != 3 {
		t.Fatalf("answers = %d, %v", len(answers), err)
	}
	byID := map[uuid.UUID]domain.Answer{}
	for _, a := range answers {
		byID[a.ID] = a
	}
	if a := byID[empty.ID]; a.Failure != "The model gave no text." || a.Extraction != nil || a.Counts() || a.Cost != 10 {
		t.Errorf("the empty response = %+v: failed, unread, its cost kept", a)
	}
	if a := byID[full.ID]; a.Failure != "" || a.Extraction == nil || !a.Counts() {
		t.Errorf("a response with text = %+v: untouched", a)
	}
	if a := byID[failed.ID]; a.Failure != "Rate limited." {
		t.Errorf("a failed response = %+v: keeps its own failure", a)
	}
	var names int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM prompt_answer_companies WHERE answer_id = $1", empty.ID).Scan(&names); err != nil || names != 0 {
		t.Errorf("the empty response still names %d companies, %v", names, err)
	}
}
