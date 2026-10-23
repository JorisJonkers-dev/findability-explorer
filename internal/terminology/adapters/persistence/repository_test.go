package persistence_test

import (
	"errors"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

func TestDecideListAndUndo(t *testing.T) {
	t.Parallel()
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := t.Context()
	repo := persistence.New(store)
	at := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	term := domain.Term{
		Text: "spiral freezer belt", Language: domain.LanguageEn, Family: domain.FamilyTriFlex, Tier: domain.TierApplication,
		Source: "https://www.twentebelt.com/", Origin: domain.OriginReview,
	}
	accepted := domain.Decision{CandidateID: "00000000000000aa", Phrase: "spiral freezer", Status: domain.StatusAccepted, By: "joris", At: at, Term: &term}
	rejected := domain.Decision{CandidateID: "00000000000000bb", Phrase: "belt news", Status: domain.StatusRejected, By: "joris", At: at.Add(time.Minute)}
	for _, d := range []domain.Decision{accepted, rejected} {
		if err := repo.Decide(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Decide(ctx, rejected); !errors.Is(err, domain.ErrDecided) {
		t.Fatalf("deciding twice = %v", err)
	}
	bad := domain.Term{Text: "x", Language: "klingon", Family: domain.FamilyOther, Tier: domain.TierBroad}
	if err := repo.Decide(ctx, domain.Decision{CandidateID: "00000000000000cc", Phrase: "x", Status: domain.StatusAccepted, By: "j", At: at, Term: &bad}); err == nil {
		t.Fatal("a term in an unknown language was stored")
	}
	got, err := repo.Decisions(ctx)
	if err != nil || len(got) != 2 {
		t.Fatalf("Decisions = %+v, %v", got, err)
	}
	if got[0].Term == nil || got[0].Term.Text != "spiral freezer belt" || got[0].Term.UsedBy == nil || got[0].Term.Origin != domain.OriginReview ||
		got[1].Term != nil || got[1].Status != domain.StatusRejected || !got[1].At.Equal(at.Add(time.Minute)) {
		t.Fatalf("decisions %+v", got)
	}
	if ok, err := repo.Undo(ctx, accepted.CandidateID); !ok || err != nil {
		t.Fatalf("Undo = %v, %v", ok, err)
	}
	if ok, err := repo.Undo(ctx, accepted.CandidateID); ok || err != nil {
		t.Fatalf("second Undo = %v, %v", ok, err)
	}
	if got, err := repo.Decisions(ctx); err != nil || len(got) != 1 {
		t.Fatalf("after Undo = %+v, %v", got, err)
	}
}
