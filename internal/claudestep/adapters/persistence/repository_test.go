package persistence_test

import (
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
)

var at = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

func open(t *testing.T) *pg.Store {
	t.Helper()
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	return store
}

func TestVersionsAreStoredWithTheOriginalFirstAndReadOldestFirst(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := persistence.New(open(t))
	if vs, err := repo.Versions(ctx, domain.Matching); err != nil || len(vs) != 0 {
		t.Fatalf("before a save: %v %v", vs, err)
	}
	original := domain.Version{Step: domain.Matching, Number: 1, Model: "claude-haiku-5-5", Instructions: "Match.", Note: "The instructions the tool came with.", By: domain.ByTheTool, At: at}
	second := domain.Version{Step: domain.Matching, Number: 2, Model: "claude-sonnet-5-5", Instructions: "Match better.", By: "Joris", At: at.Add(time.Minute)}
	if err := repo.Save(ctx, second, &original); err != nil {
		t.Fatal(err)
	}
	third := domain.Version{Step: domain.Matching, Number: 3, Model: "claude-haiku-5-5", Instructions: "Match.", Note: "Restored from version 1.", By: "Joris", At: at.Add(2 * time.Minute)}
	if err := repo.Save(ctx, third, nil); err != nil {
		t.Fatal(err)
	}
	vs, err := repo.Versions(ctx, domain.Matching)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 3 || vs[0] != original || vs[1] != second || vs[2] != third {
		t.Errorf("versions = %+v", vs)
	}
	// Two saves of the same number cannot both land; the second fails and stores nothing.
	if err := repo.Save(ctx, third, nil); err == nil {
		t.Error("a version was stored twice")
	}
}
