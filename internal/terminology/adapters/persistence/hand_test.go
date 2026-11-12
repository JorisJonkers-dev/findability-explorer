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

func TestHandTermsAddRemoveAndPutBack(t *testing.T) {
	t.Parallel()
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := t.Context()
	repo := persistence.New(store)
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	oven := domain.HandTerm{
		Term: domain.Term{Text: "oven belt", Language: domain.LanguageEn, Family: domain.FamilyEyelink, Tier: domain.TierApplication, UsedBy: []string{}, Origin: domain.OriginHand},
		By:   "joris", At: at,
	}
	if err := repo.Add(ctx, oven); err != nil {
		t.Fatal(err)
	}
	if err := repo.Add(ctx, oven); !errors.Is(err, domain.ErrTermExists) {
		t.Errorf("adding twice = %v", err)
	}
	added, err := repo.Added(ctx)
	if err != nil || len(added) != 1 || added[0].Term.Text != "oven belt" || added[0].Term.Origin != domain.OriginHand ||
		added[0].Term.Family != domain.FamilyEyelink || added[0].By != "joris" || !added[0].At.Equal(at) || added[0].Term.UsedBy == nil {
		t.Fatalf("Added = %+v, %v", added, err)
	}
	key := domain.TermKey{Text: "spiral belt", Language: domain.LanguageEn}
	for range 2 { // removing twice keeps one removal
		if err := repo.Remove(ctx, domain.Removal{Key: key, By: "joris", At: at}); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := repo.Removed(ctx)
	if err != nil || len(removed) != 1 || removed[0].Key != key || removed[0].By != "joris" || !removed[0].At.Equal(at) {
		t.Fatalf("Removed = %+v, %v", removed, err)
	}
	if found, err := repo.PutBack(ctx, key); err != nil || !found {
		t.Fatalf("PutBack = %v, %v", found, err)
	}
	if found, err := repo.PutBack(ctx, key); err != nil || found {
		t.Errorf("putting back twice = %v, %v", found, err)
	}
	if removed, _ := repo.Removed(ctx); len(removed) != 0 {
		t.Errorf("after putting back: %+v", removed)
	}
	bad := oven
	bad.Term.Language = "klingon"
	if err := repo.Add(ctx, bad); err == nil {
		t.Error("a term in an unknown language was stored")
	}
}
