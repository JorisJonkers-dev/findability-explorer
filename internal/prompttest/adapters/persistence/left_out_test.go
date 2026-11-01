package persistence_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain/domaintest"
)

func TestAReadingStoredBeforeFactsLeftOutWereToldApartLeavesNothingOut(t *testing.T) {
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
	a := domain.Answer{
		ID: uuid.New(), RunID: r.ID, Prompt: domaintest.Version2().PromptSet().Prompts[0], PromptSetVersion: 1, Provider: "openai",
		Model: "openai/gpt-oss-20b", Mode: domain.ModeKnowledge, Reply: domain.Reply{Text: "Tribelt is in Hengelo."}, At: at,
	}
	if err := repo.AddAnswer(ctx, a); err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	// A reading written as before: no left-out list.
	if _, err := conn.Exec(ctx, `INSERT INTO prompt_extractions (answer_id, failed, addresses, names_tribelt, cites_tribelt, correctness,
		wrong_claims, input_tokens, output_tokens, cost_micros, extracted_at) VALUES ($1, false, '{}', true, false, 'wrong',
		'{"Tribelt is in Hengelo."}', 1, 1, 1, now())`, a.ID); err != nil {
		t.Fatal(err)
	}
	answers, err := repo.Answers(ctx, r.ID, false)
	if err != nil || len(answers) != 1 || answers[0].Extraction == nil || len(answers[0].Extraction.LeftOut) != 0 ||
		len(answers[0].Extraction.WrongClaims) != 1 {
		t.Fatalf("an older reading = %+v, %v", answers, err)
	}
	if _, err := conn.Exec(ctx, `UPDATE prompt_extractions SET left_out = array_fill('x'::text, ARRAY[11]) WHERE answer_id = $1`, a.ID); err == nil {
		t.Error("a reading kept more than ten facts left out")
	}
}
