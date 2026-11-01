package persistence_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/activity/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/activity/domain"
	analysispersistence "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/persistence"
	analysisdomain "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
	promptpersistence "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/persistence"
	promptdomain "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain/domaintest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
	termpersistence "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/persistence"
	termdomain "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
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

func TestTheJournalRecordsWorkAsItBeginsAndEnds(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	repo := persistence.New(open(t))
	review, err := repo.Begin(ctx, work.KindTermReview, "language model", at)
	if err != nil {
		t.Fatal(err)
	}
	check, err := repo.Begin(ctx, work.KindReadinessCheck, "joris", at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	o := work.Outcome{Ending: work.EndingStoppedByHand, Counts: work.Counts{Done: 40, Total: 85, Failed: 1}, InputTokens: 2000, OutputTokens: 200}
	if err := repo.End(ctx, review, o, at.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repo.End(ctx, review, o, at.Add(3*time.Minute)); err == nil {
		t.Error("ended twice")
	}
	got, err := repo.Recorded(ctx, 10)
	if err != nil || len(got) != 2 {
		t.Fatalf("Recorded = %+v, %v", got, err)
	}
	byID := map[uuid.UUID]domain.Recorded{}
	for _, r := range got {
		byID[r.ID] = r
	}
	if r := byID[review]; r.Kind != work.KindTermReview || r.Outcome != o || !r.StartedAt.Equal(at) || !r.FinishedAt.Equal(at.Add(2*time.Minute)) {
		t.Errorf("the review = %+v", r)
	}
	if r := byID[check]; !r.FinishedAt.IsZero() || r.Outcome.Ending != "" {
		t.Errorf("the running check = %+v", r)
	}
	// The newest ended ones only, and every running one.
	if got, _ := repo.Recorded(ctx, 0); len(got) != 1 || got[0].ID != check {
		t.Errorf("with no room for ended work = %+v", got)
	}
	// A restart ends what was running, as failed.
	if n, err := repo.EndAbandoned(ctx, at.Add(time.Hour)); n != 1 || err != nil {
		t.Fatalf("EndAbandoned = %d, %v", n, err)
	}
	if got, _ := repo.Recorded(ctx, 10); len(got) != 2 {
		t.Fatalf("after the restart = %+v", got)
	} else {
		for _, r := range got {
			if r.ID == check && (r.Outcome.Ending != work.EndingFailed || !r.FinishedAt.Equal(at.Add(time.Hour))) {
				t.Errorf("the abandoned check = %+v", r)
			}
		}
	}
	if _, err := repo.Begin(ctx, work.Kind("nap"), "joris", at); err == nil {
		t.Error("an unknown kind was recorded")
	}
}

func TestPromptRunsScansAndThisMonthsUsageAreReadWithTheirCost(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	store := open(t)
	repo := persistence.New(store)
	prompts := promptpersistence.New(store)
	run := promptdomain.Run{
		ID: uuid.New(), PromptSetVersion: 1, FactSheetVersion: 1, FactSheet: "Tribelt makes belts.", Trigger: promptdomain.TriggerManual,
		StartedBy: "joris", StartedAt: at, Status: promptdomain.RunRunning, Tasks: 3,
	}
	if err := prompts.CreateRun(ctx, run, "a", at); err != nil {
		t.Fatal(err)
	}
	set := domaintest.Version2().PromptSet()
	answer := func(i int, failure string, cost promptdomain.Micros) promptdomain.Answer {
		return promptdomain.Answer{
			ID: uuid.New(), RunID: run.ID, Prompt: set.Prompts[i], PromptSetVersion: 1, Provider: "openai", Model: "openai/gpt-oss-20b",
			Mode: promptdomain.ModeKnowledge, Reply: promptdomain.Reply{Text: "Tribelt.", Cost: cost}, Failure: failure, At: at,
		}
	}
	read, haiku, failed := answer(0, "", 1000), answer(1, "", 2000), answer(2, "Rate limited.", 0)
	for _, a := range []promptdomain.Answer{read, haiku, failed} {
		if err := prompts.AddAnswer(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	if err := prompts.SetExtraction(ctx, read.ID, promptdomain.Extraction{Cost: 50, InputTokens: 10, OutputTokens: 1}, at); err != nil {
		t.Fatal(err)
	}
	if err := prompts.SetExtraction(ctx, haiku.ID, promptdomain.Extraction{Cost: 70, InputTokens: 1_000_000, OutputTokens: 100_000, HaikuTokens: 1_100_000}, at); err != nil {
		t.Fatal(err)
	}
	if ok, err := prompts.RequestStop(ctx, run.ID, at); !ok || err != nil {
		t.Fatal(ok, err)
	}
	if err := prompts.FinishRun(ctx, run.ID, promptdomain.RunStopped, promptdomain.StoppedByHand, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	runs, err := repo.PromptRuns(ctx, 5)
	if err != nil || len(runs) != 1 {
		t.Fatalf("PromptRuns = %+v, %v", runs, err)
	}
	r := runs[0]
	if r.ID != run.ID || r.Status != "stopped" || !r.StoppedByHand || r.Tasks != 3 || r.Answered != 2 || r.Failed != 1 ||
		r.AnswerCost != 3000 || r.OpenRouterReadingCost != 50 || r.HaikuInput != 1_000_000 || r.HaikuOutput != 100_000 ||
		!r.FinishedAt.Equal(at.Add(time.Hour)) || !r.StartedAt.Equal(at) {
		t.Errorf("the run = %+v", r)
	}

	scans := analysispersistence.New(store)
	alone := analysisdomain.Analysis{ID: uuid.New(), URL: "https://www.twentebelt.com/", MaxPages: 30, CreatedBy: "joris", CreatedAt: at}
	rescanned := analysisdomain.Analysis{ID: uuid.New(), URL: "https://www.pelger.de/", MaxPages: 20, CreatedBy: "joris", CreatedAt: at.Add(time.Second), Rescan: true}
	for _, a := range []analysisdomain.Analysis{alone, rescanned} {
		if _, err := scans.Create(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	if err := scans.MarkRunning(ctx, alone.ID); err != nil {
		t.Fatal(err)
	}
	if err := scans.Finish(ctx, alone.ID, at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := scans.StopQueued(ctx, analysisdomain.StopScope{ID: rescanned.ID}, analysisdomain.StoppedByHandReason, at.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	rows, err := repo.Scans(ctx, 5)
	if err != nil || len(rows) != 2 {
		t.Fatalf("Scans = %+v, %v", rows, err)
	}
	for _, s := range rows {
		switch s.ID {
		case alone.ID:
			if s.Rescan || s.Status != "done" || s.MaxPages != 30 || s.CreatedBy != "joris" || !s.FinishedAt.Equal(at.Add(time.Minute)) || s.StopRequested {
				t.Errorf("the scan = %+v", s)
			}
		case rescanned.ID:
			if !s.Rescan || s.Status != "failed" || !s.StopRequested || !s.CreatedAt.Equal(at.Add(time.Second)) {
				t.Errorf("the re-scanned one = %+v", s)
			}
		}
	}
	queued := analysisdomain.Analysis{ID: uuid.New(), URL: "https://www.wirebelt.com/", MaxPages: 10, CreatedBy: "joris", CreatedAt: at.Add(-time.Hour)}
	if _, err := scans.Create(ctx, queued); err != nil {
		t.Fatal(err)
	}
	// A waiting scan is read however old it is.
	if rows, _ := repo.Scans(ctx, 1); len(rows) != 2 {
		t.Errorf("the newest and the waiting scan = %+v", rows)
	}

	terms := termpersistence.New(store)
	if err := terms.SaveReview(ctx, nil, nil, []termdomain.Usage{{InputTokens: 300, OutputTokens: 30, At: at}, {InputTokens: 9, At: at.Add(-time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	if err := prompts.SaveReviewUsage(ctx, []promptdomain.ReviewUsage{{InputTokens: 400, OutputTokens: 40, At: at}}); err != nil {
		t.Fatal(err)
	}
	u, err := repo.MonthUsage(ctx, at)
	if err != nil {
		t.Fatal(err)
	}
	if u.PromptAnswers != 3000 || u.OpenRouterReadings != 50 || u.HaikuReadings.Input != 1_000_000 || u.HaikuReadings.Output != 100_000 ||
		u.TermReview.Input != 300 || u.TermReview.Output != 30 || u.CandidateChecks.Input != 400 || u.CandidateChecks.Output != 40 ||
		u.Scans.Input != 0 {
		t.Errorf("this month = %+v", u)
	}
}
