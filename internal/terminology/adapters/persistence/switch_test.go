package persistence_test

import (
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

func TestTheReviewIsOnUntilSwitchedOffAndCountsItsUsage(t *testing.T) {
	t.Parallel()
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := t.Context()
	repo := persistence.New(store)
	at := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	if on, err := repo.ReviewOn(ctx); !on || err != nil {
		t.Fatalf("a review nobody switched = %v, %v", on, err)
	}
	if err := repo.SwitchReview(ctx, false, "joris", at); err != nil {
		t.Fatal(err)
	}
	if on, err := repo.ReviewOn(ctx); on || err != nil {
		t.Fatalf("switched off = %v, %v", on, err)
	}
	if err := repo.SwitchReview(ctx, true, "anna", at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if on, err := repo.ReviewOn(ctx); !on || err != nil {
		t.Fatalf("switched on again = %v, %v", on, err)
	}
	if err := repo.SwitchReview(ctx, false, "", at); err == nil {
		t.Error("a switch by nobody was stored")
	}

	if u, err := repo.UsageSince(ctx, at); u.Tokens() != 0 || err != nil {
		t.Fatalf("no usage = %+v, %v", u, err)
	}
	if err := repo.SaveReview(ctx, nil, nil, []domain.Usage{
		{InputTokens: 1000, OutputTokens: 100, At: at.Add(-time.Hour)},
		{InputTokens: 400, OutputTokens: 40, At: at},
		{InputTokens: 300, OutputTokens: 30, At: at.Add(time.Hour)},
	}); err != nil {
		t.Fatal(err)
	}
	if u, err := repo.UsageSince(ctx, at); u.InputTokens != 700 || u.OutputTokens != 70 || err != nil {
		t.Errorf("UsageSince = %+v, %v", u, err)
	}
}
