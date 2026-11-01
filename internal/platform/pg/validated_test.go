package pg_test

import (
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
)

// Every check a migration added NOT VALID is validated by a later one, so the database holds every
// stored row to it: none is left with convalidated false.
func TestTheMigrationsLeaveNoCheckUnvalidated(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	conn, err := pgx.Connect(ctx, pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	rows, err := conn.Query(ctx, `
		SELECT conrelid::regclass::text || '.' || conname
		FROM pg_constraint
		WHERE NOT convalidated AND connamespace = 'public'::regnamespace
		ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	open, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 0 {
		t.Errorf("checks still NOT VALID: %v", open)
	}
	// The checks 00045 validates are there, and validated.
	var validated int
	err = conn.QueryRow(ctx, `
		SELECT count(*) FROM pg_constraint
		WHERE convalidated AND conname IN (
			'llm_events_action_check', 'prompt_answers_scope_check', 'prompt_answers_language_check', 'prompt_answers_kind_check')`).Scan(&validated)
	if err != nil || validated != 4 {
		t.Errorf("validated = %d, %v; want the four checks 00045 validates", validated, err)
	}
}
