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

func TestVerdictsAndWhatTheyCost(t *testing.T) {
	t.Parallel()
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := t.Context()
	repo := persistence.New(store)
	at := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	accepted := domain.Verdict{
		CandidateID: "00000000000000aa", Phrase: "spiral freezer belts", Accept: true, Sure: true, Reason: "A belt buyers search for.",
		Term:  domain.Acceptance{Text: "spiral freezer belt", Language: "en", Family: "tri_flex_turn_curve", Tier: "application"},
		Model: "claude-haiku-5-5", At: at,
	}
	unsure := domain.Verdict{CandidateID: "00000000000000bb", Phrase: "freezing", Reason: "Could go either way.", Model: "claude-haiku-5-5", At: at.Add(time.Minute)}
	if err := repo.SaveVerdicts(ctx, []domain.Verdict{accepted, unsure}, domain.Usage{InputTokens: 900, OutputTokens: 100, At: at}); err != nil {
		t.Fatal(err)
	}
	// A failed request stores no verdict, but its usage counts; a verdict is never replaced.
	changed := unsure
	changed.Sure = true
	if err := repo.SaveVerdicts(ctx, []domain.Verdict{changed}, domain.Usage{InputTokens: 50, At: at.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Verdicts(ctx)
	if err != nil || len(got) != 2 {
		t.Fatalf("Verdicts = %+v, %v", got, err)
	}
	if got[0].Phrase != accepted.Phrase || got[0].Term.Text != accepted.Term.Text || got[0].Term.Family != accepted.Term.Family || got[0].Term.Tier != accepted.Term.Tier || got[0].Term.Language != "en" || !got[0].At.Equal(at) || got[0].Model != accepted.Model || !got[0].Sure {
		t.Errorf("the accepted verdict = %+v", got[0])
	}
	if got[1].Sure || got[1].Reason != unsure.Reason {
		t.Errorf("the unsure verdict = %+v", got[1])
	}
	if n, err := repo.TokensSince(ctx, at); n != 1050 || err != nil {
		t.Errorf("TokensSince = %d, %v", n, err)
	}
	if n, err := repo.TokensSince(ctx, at.Add(time.Minute)); n != 50 || err != nil {
		t.Errorf("TokensSince later = %d, %v", n, err)
	}
	bad := domain.Verdict{CandidateID: "not-an-id", Phrase: "x", Model: "m", At: at}
	if err := repo.SaveVerdicts(ctx, []domain.Verdict{bad}, domain.Usage{At: at}); err == nil {
		t.Error("a verdict on a bad id was stored")
	}
	if n, _ := repo.TokensSince(ctx, at); n != 1050 {
		t.Errorf("a refused save stored its usage: %d", n)
	}
}
