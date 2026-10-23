package pg_test

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/queries"
)

const unreachable = "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1"

func TestStoreAgainstMigratedDatabase(t *testing.T) {
	t.Parallel()
	url := pgtest.URL(t)
	if err := pg.Migrate(t.Context(), url); err != nil {
		t.Fatalf("migrating an up-to-date database again: %v", err)
	}
	store, err := pg.Open(t.Context(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Ping(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Queries().ListAnalyses(t.Context(), 1); err != nil {
		t.Fatalf("the migrated schema has no analyses table: %v", err)
	}
}

func TestInTxCommitsOrRollsBack(t *testing.T) {
	t.Parallel()
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	create := func(q *queries.Queries) error {
		_, err := q.CreateAnalysis(t.Context(), queries.CreateAnalysisParams{
			ID: uuid.New(), Url: "https://example.com/", MaxPages: 1, CreatedBy: "tester", CreatedAt: time.Now(),
		})
		return err
	}
	boom := errors.New("boom")
	if err := store.InTx(t.Context(), func(q *queries.Queries) error {
		if err := create(q); err != nil {
			return err
		}
		return boom
	}); !errors.Is(err, boom) {
		t.Fatalf("InTx error = %v, want boom", err)
	}
	if err := store.InTx(t.Context(), create); err != nil {
		t.Fatal(err)
	}
	rows, err := store.Queries().ListAnalyses(t.Context(), 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("after one rollback and one commit: %d rows, %v; want 1", len(rows), err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if err := store.InTx(cancelled, create); err == nil {
		t.Fatal("InTx on a cancelled context succeeded")
	}
}

func TestOpenFails(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	for _, url := range []string{unreachable, "::not a url::"} {
		if _, err := pg.Open(ctx, url); err == nil {
			t.Fatalf("Open(%q) succeeded", url)
		}
	}
}

func TestMigrateFailsOnUnreachableDatabase(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	if err := pg.Migrate(ctx, unreachable); err == nil {
		t.Fatal("Migrate against an unreachable database succeeded")
	}
}

func TestMigrateRefusesAnEmptyMigrationSet(t *testing.T) {
	t.Parallel()
	if err := pg.MigrateFS(t.Context(), pgtest.URL(t), fstest.MapFS{"migrations/.keep": {}}); err == nil {
		t.Fatal("an empty migration set was accepted")
	}
}

func TestMigrateRefusesAMissingMigrationsDir(t *testing.T) {
	t.Parallel()
	if err := pg.MigrateFS(t.Context(), unreachable, fstest.MapFS{}); err == nil {
		t.Fatal("a filesystem without migrations/ was accepted")
	}
}
