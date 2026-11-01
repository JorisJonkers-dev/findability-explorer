package persistence_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
)

func TestARunReplacesTheOpenRecommendationsInOneGo(t *testing.T) {
	t.Parallel()
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := t.Context()
	repo := persistence.New(store)
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	first := domain.RecRun{ID: uuid.NewString(), StartedAt: at, By: "joris", Model: "claude-haiku-5-5"}
	if err := repo.BeginRecRun(ctx, first); err != nil {
		t.Fatal(err)
	}
	rec := func(id, key string) domain.Recommendation {
		return domain.Recommendation{
			ID: id, RunID: first.ID, Gap: domain.Gap{Key: key, Kind: domain.GapTerm, Language: "nl", Family: "eyelink", Specifics: []string{"Two use it."}, Benefit: 4, Effort: 2},
			Title: "Use it", Where: "https://www.tribelt.nl/", Why: "Because.", Steps: []string{"Do it."}, DoneWhen: "Found.", Benefit: 3, Effort: 2,
			ScoreReason: "Few search for it.", CreatedAt: at,
		}
	}
	a, b := rec(uuid.NewString(), "term:nl:a"), rec(uuid.NewString(), "term:nl:b")
	b.Where, b.WhereNote, b.Steps, b.Gap.Specifics = "", domain.ForTheAgency, nil, nil
	first.FinishedAt, first.Ending, first.Gaps, first.Added, first.InputTokens, first.OutputTokens = at.Add(time.Minute), "done", 2, 2, 900, 90
	if err := repo.FinishRecRun(ctx, first, &domain.Replacement{Added: []domain.Recommendation{a, b}}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Recommendations(ctx)
	if err != nil || len(got) != 2 {
		t.Fatalf("Recommendations = %+v, %v", got, err)
	}
	byID := map[string]domain.Recommendation{got[0].ID: got[0], got[1].ID: got[1]}
	if g := byID[a.ID]; g.RunID != first.ID || g.Gap.Key != "term:nl:a" || g.Gap.Benefit != 4 || g.Gap.Specifics[0] != "Two use it." || g.Where != a.Where ||
		g.Benefit != 3 || g.ScoreReason != "Few search for it." || g.Steps[0] != "Do it." || !g.CreatedAt.Equal(at) || !g.DoneAt.IsZero() || !g.EditedAt.IsZero() {
		t.Errorf("a = %+v", g)
	}
	if g := byID[b.ID]; g.Where != "" || g.WhereNote != domain.ForTheAgency || len(g.Steps) != 0 {
		t.Errorf("b = %+v", g)
	}
	// Ticked off, and undone.
	if ok, err := repo.SetRecDone(ctx, a.ID, "joris", at.Add(time.Hour)); !ok || err != nil {
		t.Fatalf("tick = %v, %v", ok, err)
	}
	if got, _ := repo.Recommendations(ctx); got[0].DoneBy != "joris" && got[1].DoneBy != "joris" {
		t.Errorf("not ticked: %+v", got)
	}
	if ok, err := repo.SetRecDone(ctx, "nothing", "joris", at); ok || err != nil {
		t.Errorf("an unknown one = %v, %v", ok, err)
	}
	// An edit is stored with when it was made.
	edited := a
	edited.Title, edited.Steps, edited.Benefit, edited.EditedAt = "Use it on the spiral page", []string{"Add it."}, 5, at.Add(30*time.Minute)
	if err := repo.UpdateRecommendation(ctx, edited); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.Recommendations(ctx); got[0].Title != "Use it on the spiral page" && got[1].Title != "Use it on the spiral page" {
		t.Errorf("not edited: %+v", got)
	}
	if err := repo.UpdateRecommendation(ctx, domain.Recommendation{ID: "nothing", Title: "x", Benefit: 1, Effort: 1}); err == nil {
		t.Error("an unknown recommendation was edited")
	}
	// A second run replaces b and keeps a; a stopped third changes nothing.
	second := domain.RecRun{ID: uuid.NewString(), StartedAt: at.Add(2 * time.Hour), By: "ada", Model: "m"}
	if err := repo.BeginRecRun(ctx, second); err != nil {
		t.Fatal(err)
	}
	c := rec(uuid.NewString(), "term:nl:c")
	c.RunID = second.ID
	second.FinishedAt, second.Ending, second.Added, second.Replaced, second.Kept = at.Add(3*time.Hour), "done", 1, 1, 1
	if err := repo.FinishRecRun(ctx, second, &domain.Replacement{Replaced: []domain.Recommendation{b}, Kept: []domain.Recommendation{a}, Added: []domain.Recommendation{c}}); err != nil {
		t.Fatal(err)
	}
	third := domain.RecRun{ID: uuid.NewString(), StartedAt: at.Add(4 * time.Hour), By: "ada", Model: "m"}
	if err := repo.BeginRecRun(ctx, third); err != nil {
		t.Fatal(err)
	}
	third.FinishedAt, third.Ending = at.Add(5*time.Hour), "stoppedByHand"
	if err := repo.FinishRecRun(ctx, third, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.Recommendations(ctx); len(got) != 2 || (got[0].ID != a.ID && got[1].ID != a.ID) {
		t.Errorf("after the runs = %+v", got)
	}
	runs, err := repo.RecRuns(ctx, 10)
	if err != nil || len(runs) != 3 || runs[0].ID != third.ID || runs[0].Ending != "stoppedByHand" || runs[1].Added != 1 || runs[1].Kept != 1 ||
		runs[2].InputTokens != 900 || runs[2].By != "joris" || !runs[2].FinishedAt.Equal(at.Add(time.Minute)) {
		t.Errorf("runs = %+v, %v", runs, err)
	}
	if in, out, err := repo.RecommendationTokensSince(ctx, at); in != 900 || out != 90 || err != nil {
		t.Errorf("tokens = %d, %d, %v", in, out, err)
	}
	// A bad recommendation rolls the whole run back.
	fourth := domain.RecRun{ID: uuid.NewString(), StartedAt: at.Add(6 * time.Hour), By: "ada", Model: "m"}
	if err := repo.BeginRecRun(ctx, fourth); err != nil {
		t.Fatal(err)
	}
	bad := rec("Not An Id", "x")
	fourth.FinishedAt, fourth.Ending = at.Add(7*time.Hour), "done"
	if err := repo.FinishRecRun(ctx, fourth, &domain.Replacement{Replaced: []domain.Recommendation{c}, Added: []domain.Recommendation{bad}}); err == nil {
		t.Error("a bad recommendation was stored")
	}
	if got, _ := repo.Recommendations(ctx); len(got) != 2 {
		t.Errorf("after the refused run = %+v", got)
	}
	if err := repo.BeginRecRun(ctx, domain.RecRun{ID: "not a uuid"}); err == nil {
		t.Error("a run without a uuid was begun")
	}
	if err := repo.FinishRecRun(ctx, domain.RecRun{ID: "not a uuid"}, nil); err == nil {
		t.Error("a run without a uuid was finished")
	}
}
