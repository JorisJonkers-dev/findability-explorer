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

func TestAReviewIsStoredWhole(t *testing.T) {
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
		Model: "anthropic: claude-haiku-5-5", At: at,
	}
	unsure := domain.Verdict{CandidateID: "00000000000000bb", Phrase: "freezing", Reason: "Could go either way.", Model: "anthropic: claude-haiku-5-5", At: at.Add(time.Minute)}
	rejected := domain.Verdict{CandidateID: "00000000000000cc", Phrase: "contact", Sure: true, Model: "anthropic: claude-haiku-5-5", At: at}
	term := domain.Term{Text: "spiral freezer belt", Language: "en", Family: "tri_flex_turn_curve", Tier: "application", Origin: domain.OriginReview}
	// A person decided "contact" while the model thought; that decision stands.
	if err := repo.Decide(ctx, domain.Decision{CandidateID: "00000000000000cc", Phrase: "contact", Status: domain.StatusAccepted, By: "joris", At: at, Term: &domain.Term{
		Text: "contact", Language: "en", Family: "other", Tier: "broad",
	}}); err != nil {
		t.Fatal(err)
	}
	decisions := []domain.Decision{
		{CandidateID: accepted.CandidateID, Phrase: accepted.Phrase, Status: domain.StatusAccepted, By: domain.ModelReviewer, At: at, Term: &term},
		{CandidateID: rejected.CandidateID, Phrase: rejected.Phrase, Status: domain.StatusRejected, By: domain.ModelReviewer, At: at},
	}
	usage := []domain.Usage{{InputTokens: 900, OutputTokens: 100, At: at}, {InputTokens: 50, At: at.Add(time.Hour)}}
	if err := repo.SaveReview(ctx, []domain.Verdict{accepted, unsure, rejected}, decisions, usage); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Verdicts(ctx)
	if err != nil || len(got) != 3 {
		t.Fatalf("Verdicts = %+v, %v", got, err)
	}
	if got[0].Phrase != accepted.Phrase || got[0].Term.Text != accepted.Term.Text || got[0].Term.Family != accepted.Term.Family ||
		got[0].Term.Tier != accepted.Term.Tier || got[0].Term.Language != "en" || !got[0].At.Equal(at) || got[0].Model != accepted.Model || !got[0].Sure {
		t.Errorf("the accepted verdict = %+v", got[0])
	}
	if got[2].Sure || got[2].Reason != unsure.Reason {
		t.Errorf("the unsure verdict = %+v", got[2])
	}
	ds, err := repo.Decisions(ctx)
	if err != nil || len(ds) != 2 {
		t.Fatalf("Decisions = %+v, %v", ds, err)
	}
	for _, d := range ds {
		if d.CandidateID == "00000000000000cc" && d.By != "joris" {
			t.Errorf("the person's decision was replaced: %+v", d)
		}
		if d.CandidateID == "00000000000000aa" && (d.By != domain.ModelReviewer || d.Term == nil) {
			t.Errorf("the model's decision = %+v", d)
		}
	}
	// A verdict is never replaced.
	changed := unsure
	changed.Sure = true
	if err := repo.SaveReview(ctx, []domain.Verdict{changed}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.Verdicts(ctx); got[2].Sure {
		t.Error("a verdict was replaced")
	}
	if n, err := repo.TokensSince(ctx, at); n != 1050 || err != nil {
		t.Errorf("TokensSince = %d, %v", n, err)
	}
	if n, err := repo.TokensSince(ctx, at.Add(time.Minute)); n != 50 || err != nil {
		t.Errorf("TokensSince later = %d, %v", n, err)
	}
	// A bad row rolls the whole review back.
	bad := domain.Verdict{CandidateID: "not-an-id", Phrase: "x", Model: "m", At: at}
	if err := repo.SaveReview(ctx, []domain.Verdict{bad}, nil, []domain.Usage{{InputTokens: 7, At: at}}); err == nil {
		t.Error("a verdict on a bad id was stored")
	}
	badTerm := domain.Decision{CandidateID: "00000000000000dd", Phrase: "y", Status: domain.StatusAccepted, By: "m", At: at, Term: &domain.Term{Text: "y", Language: "klingon"}}
	if err := repo.SaveReview(ctx, nil, []domain.Decision{badTerm}, nil); err == nil {
		t.Error("a term in an unknown language was stored")
	}
	if n, _ := repo.TokensSince(ctx, at); n != 1050 {
		t.Errorf("a refused review stored its usage: %d", n)
	}
	if err := repo.SaveReview(ctx, nil, nil, []domain.Usage{{InputTokens: -1, At: at}}); err == nil {
		t.Error("negative tokens were stored")
	}
}
