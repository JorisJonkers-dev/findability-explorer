package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

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
	begin := func(run domain.RecRun) {
		t.Helper()
		if ok, err := repo.BeginRecRun(ctx, run, alive(run.StartedAt)); !ok || err != nil {
			t.Fatalf("BeginRecRun = %v, %v", ok, err)
		}
	}
	first := domain.RecRun{ID: uuid.NewString(), StartedAt: at, By: "joris", Model: "claude-haiku-5-5"}
	begin(first)
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
	begin(second)
	c := rec(uuid.NewString(), "term:nl:c")
	c.RunID = second.ID
	second.FinishedAt, second.Ending, second.Added, second.Replaced, second.Kept = at.Add(3*time.Hour), "done", 1, 1, 1
	if err := repo.FinishRecRun(ctx, second, &domain.Replacement{Replaced: []domain.Recommendation{b}, Kept: []domain.Recommendation{a}, Added: []domain.Recommendation{c}}); err != nil {
		t.Fatal(err)
	}
	third := domain.RecRun{ID: uuid.NewString(), StartedAt: at.Add(4 * time.Hour), By: "ada", Model: "m"}
	begin(third)
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
	begin(fourth)
	bad := rec("Not An Id", "x")
	fourth.FinishedAt, fourth.Ending = at.Add(7*time.Hour), "done"
	if err := repo.FinishRecRun(ctx, fourth, &domain.Replacement{Replaced: []domain.Recommendation{c}, Added: []domain.Recommendation{bad}}); err == nil {
		t.Error("a bad recommendation was stored")
	}
	if got, _ := repo.Recommendations(ctx); len(got) != 2 {
		t.Errorf("after the refused run = %+v", got)
	}
	if _, err := repo.BeginRecRun(ctx, domain.RecRun{ID: "not a uuid"}, alive(at)); err == nil {
		t.Error("a run without a uuid was begun")
	}
	if err := repo.FinishRecRun(ctx, domain.RecRun{ID: "not a uuid"}, nil); err == nil {
		t.Error("a run without a uuid was finished")
	}
}

// alive is the window a run begun at counts as live in: heard of in the last two minutes, or,
// without a heartbeat, begun in the last hour.
func alive(at time.Time) domain.Alive {
	return domain.Alive{HeardSince: at.Add(-2 * time.Minute), BegunSince: at.Add(-time.Hour)}
}

// A run's running state and a stop request are stored, so every instance sees them: one runs at a
// time, a stop reaches it wherever it was asked, and one a stopped server left is ended as failed.
func TestARunsStateIsStoredForEveryInstance(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	url := pgtest.URL(t)
	store, err := pg.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	repo := persistence.New(store)
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	if running, err := repo.RecRunRunning(ctx, alive(at)); running || err != nil {
		t.Fatalf("running before any = %v, %v", running, err)
	}
	if ok, err := repo.RequestRecRunStop(ctx, at, alive(at)); ok || err != nil {
		t.Fatalf("a stop with none running = %v, %v", ok, err)
	}
	run := domain.RecRun{ID: uuid.NewString(), StartedAt: at, By: "joris", Model: "m"}
	if ok, err := repo.BeginRecRun(ctx, run, alive(at)); !ok || err != nil {
		t.Fatalf("begin = %v, %v", ok, err)
	}
	other := domain.RecRun{ID: uuid.NewString(), StartedAt: at.Add(time.Minute), By: "ada", Model: "m"}
	if ok, err := repo.BeginRecRun(ctx, other, alive(at.Add(time.Minute))); ok || err != nil {
		t.Fatalf("a second begin while one runs = %v, %v", ok, err)
	}
	// It beats, so a sweep ten minutes on ends nothing, and it shows as running.
	if err := repo.BeatRecRun(ctx, run.ID, at.Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	later := at.Add(11 * time.Minute)
	if n, err := repo.EndAbandonedRecRuns(ctx, later, alive(later)); n != 0 || err != nil {
		t.Fatalf("a sweep of a live run = %d, %v", n, err)
	}
	if running, err := repo.RecRunRunning(ctx, alive(later)); !running || err != nil {
		t.Fatalf("running = %v, %v", running, err)
	}
	if asked, err := repo.RecRunStopRequested(ctx, run.ID); asked || err != nil {
		t.Fatalf("stop asked before any = %v, %v", asked, err)
	}
	if ok, err := repo.RequestRecRunStop(ctx, later, alive(later)); !ok || err != nil {
		t.Fatalf("stop = %v, %v", ok, err)
	}
	if asked, err := repo.RecRunStopRequested(ctx, run.ID); !asked || err != nil {
		t.Fatalf("stop asked = %v, %v", asked, err)
	}
	// Unheard of for long, it was left by a stopped server: ended as failed, and a new run may begin.
	gone := at.Add(time.Hour)
	if running, _ := repo.RecRunRunning(ctx, alive(gone)); running {
		t.Error("a run unheard of shows as running")
	}
	if n, err := repo.EndAbandonedRecRuns(ctx, gone, alive(gone)); n != 1 || err != nil {
		t.Fatalf("a sweep of a left run = %d, %v", n, err)
	}
	if runs, _ := repo.RecRuns(ctx, 5); len(runs) != 1 || runs[0].Ending != "failed" || !runs[0].FinishedAt.Equal(gone) {
		t.Errorf("runs = %+v", runs)
	}
	if err := repo.BeatRecRun(ctx, run.ID, gone); err == nil {
		t.Error("an ended run beat")
	}
	for _, bad := range []error{
		repo.BeatRecRun(ctx, "not a uuid", gone),
		func() error { _, err := repo.RecRunStopRequested(ctx, "not a uuid"); return err }(),
		func() error { _, err := repo.RecRunStopRequested(ctx, uuid.NewString()); return err }(),
	} {
		if bad == nil {
			t.Error("a bad or unknown run id was taken")
		}
	}
	// A run begun by a server before runs beat counts as live for an hour from when it began.
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(context.Background()) }()
	old := domain.RecRun{ID: uuid.NewString(), StartedAt: gone, By: "joris", Model: "m"}
	if ok, err := repo.BeginRecRun(ctx, old, alive(gone)); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if _, err := conn.Exec(ctx, "UPDATE recommendation_runs SET heartbeat_at = NULL WHERE id = $1", old.ID); err != nil {
		t.Fatal(err)
	}
	if running, _ := repo.RecRunRunning(ctx, alive(gone.Add(30*time.Minute))); !running {
		t.Error("an unbeaten run within its hour does not run")
	}
	if n, _ := repo.EndAbandonedRecRuns(ctx, gone.Add(2*time.Hour), alive(gone.Add(2*time.Hour))); n != 1 {
		t.Errorf("an unbeaten run after its hour was not ended: %d", n)
	}
}
