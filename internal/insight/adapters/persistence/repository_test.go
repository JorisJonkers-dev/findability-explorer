package persistence_test

import (
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
)

func TestTicksAndTargetsRoundTrip(t *testing.T) {
	t.Parallel()
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := t.Context()
	repo := persistence.New(store.Queries())
	at := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	if err := repo.Tick(ctx, domain.Tick{ImprovementID: "listings", Version: 1, By: "joris", At: at}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Tick(ctx, domain.Tick{ImprovementID: "listings", Version: 1, By: "ada", At: at.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	ticks, err := repo.Ticks(ctx)
	if err != nil || len(ticks) != 1 || ticks[0].By != "ada" || !ticks[0].At.Equal(at.Add(time.Hour)) || ticks[0].Version != 1 {
		t.Fatalf("ticks = %+v, %v", ticks, err)
	}
	if err := repo.Untick(ctx, "listings"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Untick(ctx, "listings"); err != nil {
		t.Errorf("undoing twice = %v", err)
	}
	if ticks, _ := repo.Ticks(ctx); len(ticks) != 0 {
		t.Errorf("after undo = %+v", ticks)
	}
	if err := repo.SetTarget(ctx, domain.TargetTermsUsed, 40, "joris", at); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetTarget(ctx, domain.TargetTermsUsed, 45, "joris", at); err != nil {
		t.Fatal(err)
	}
	set, err := repo.Targets(ctx)
	if err != nil || len(set) != 1 || set[domain.TargetTermsUsed] != 45 {
		t.Errorf("targets = %v, %v", set, err)
	}
	store.Close()
	if _, err := repo.Ticks(ctx); err == nil {
		t.Error("a closed store answered ticks")
	}
	if _, err := repo.Targets(ctx); err == nil {
		t.Error("a closed store answered targets")
	}
}
