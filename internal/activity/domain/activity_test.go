package domain_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/activity/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
)

var at = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

// haiku is what in and out tokens cost on Claude Haiku: $0.10 and $0.50 a million.
func haiku(in, out int64) llm.Micros { return llm.ListPrice(llm.DefaultModel).Of(in, out) }

func TestAPromptRunEndsAsItsStatusAndStopSay(t *testing.T) {
	run := domain.PromptRunRow{
		ID: uuid.New(), StartedAt: at, FinishedAt: at.Add(time.Hour), Status: "done", Tasks: 360, Answered: 360,
		AnswerCost: 2_000_000, OpenRouterReadingCost: 100_000, HaikuInput: 1_000_000, HaikuOutput: 200_000,
	}
	w := run.Work()
	if w.Kind != work.KindPromptRun || w.ID != run.ID.String() || w.Ending != work.EndingDone || w.Running() ||
		w.Counts != (work.Counts{Done: 360, Total: 360}) || w.OpenRouter != 2_100_000 || w.Anthropic != haiku(1_000_000, 200_000) {
		t.Errorf("a finished run = %+v", w)
	}
	cases := map[string]struct {
		status string
		hand   bool
		failed int
		want   work.Ending
	}{
		"failed responses":  {"done", false, 7, work.EndingDoneWithFailures},
		"stopped by hand":   {"stopped", true, 0, work.EndingStoppedByHand},
		"stopped by budget": {"stopped", false, 0, work.EndingStoppedByBudget},
		"failed":            {"failed", false, 0, work.EndingFailed},
		"running":           {"running", false, 0, ""},
		"an unknown status": {"lost", false, 0, work.EndingFailed},
	}
	for name, c := range cases {
		r := run
		r.Status, r.StoppedByHand, r.Failed = c.status, c.hand, c.failed
		if c.status == "running" {
			r.FinishedAt = time.Time{}
		}
		if got := r.Work(); got.Ending != c.want || got.Counts.Failed != c.failed {
			t.Errorf("%s: ending %q, counts %+v", name, got.Ending, got.Counts)
		}
	}
}

func scan(id string, rescan bool, created time.Duration, status string) domain.ScanRow {
	r := domain.ScanRow{
		ID: uuid.NewSHA1(uuid.Nil, []byte(id)), Rescan: rescan, CreatedBy: "joris", CreatedAt: at.Add(created), Status: status,
		Pages: 10, MaxPages: 30, InputTokens: 1000, OutputTokens: 100,
	}
	if status == "done" || status == "failed" {
		r.FinishedAt = r.CreatedAt.Add(time.Minute)
	}
	return r
}

func TestAScanIsOnePieceOfWorkAndARescanItsScansTogether(t *testing.T) {
	single := scan("one", false, 0, "done")
	stopped := scan("stopped", false, time.Second, "failed")
	stopped.StopRequested = true
	failed := scan("failed", false, 2*time.Second, "failed")
	queued := scan("queued", false, 3*time.Second, "queued")
	// One re-scan of three companies, queued within a second, then another an hour later.
	first := []domain.ScanRow{
		scan("r1", true, 10*time.Second, "done"), scan("r2", true, 10*time.Second+time.Millisecond, "failed"),
		scan("r3", true, 11*time.Second, "running"),
	}
	later := scan("r4", true, time.Hour, "done")
	other := scan("r5", true, time.Hour+time.Second, "done")
	other.CreatedBy = "anna" // someone else's re-scan, at about the same time
	rows := append([]domain.ScanRow{later, single, stopped, failed, queued, other}, first...)

	got := map[string]domain.Work{}
	for _, w := range domain.ScanWork(rows) {
		got[w.ID] = w
	}
	if len(got) != 7 {
		t.Fatalf("work = %+v", got)
	}
	if w := got[single.ID.String()]; w.Kind != work.KindScan || w.Ending != work.EndingDone || w.Counts != (work.Counts{Done: 10, Total: 30}) ||
		w.Anthropic != haiku(1000, 100) || w.OpenRouter != 0 || !w.StartedAt.Equal(at) {
		t.Errorf("a scan = %+v", w)
	}
	if w := got[stopped.ID.String()]; w.Ending != work.EndingStoppedByHand {
		t.Errorf("a stopped scan = %+v", w)
	}
	if w := got[failed.ID.String()]; w.Ending != work.EndingFailed {
		t.Errorf("a failed scan = %+v", w)
	}
	if w := got[queued.ID.String()]; !w.Running() || w.Ending != "" {
		t.Errorf("a queued scan = %+v", w)
	}
	re := got[first[0].ID.String()]
	if re.Kind != work.KindRescan || !re.Running() || re.Counts != (work.Counts{Done: 2, Total: 3, Failed: 1}) || re.Anthropic != haiku(3000, 300) ||
		!re.StartedAt.Equal(first[0].CreatedAt) {
		t.Errorf("a running re-scan = %+v", re)
	}
	if w := got[later.ID.String()]; w.Kind != work.KindRescan || w.Ending != work.EndingDone || w.Counts != (work.Counts{Done: 1, Total: 1}) ||
		!w.FinishedAt.Equal(later.FinishedAt) {
		t.Errorf("a finished re-scan = %+v", w)
	}
	if w := got[other.ID.String()]; w.Kind != work.KindRescan {
		t.Errorf("another's re-scan = %+v", w)
	}
}

func TestARescanEndsAsItsScansDid(t *testing.T) {
	cases := map[string]struct {
		statuses []string
		hand     bool
		want     work.Ending
	}{
		"all done":        {[]string{"done", "done"}, false, work.EndingDone},
		"some failed":     {[]string{"done", "failed"}, false, work.EndingDoneWithFailures},
		"all failed":      {[]string{"failed", "failed"}, false, work.EndingFailed},
		"stopped by hand": {[]string{"done", "failed"}, true, work.EndingStoppedByHand},
	}
	for name, c := range cases {
		var rows []domain.ScanRow
		for i, s := range c.statuses {
			r := scan(fmt.Sprint(name, i), true, time.Duration(i)*time.Second, s)
			r.FinishedAt = at.Add(time.Duration(10-i) * time.Minute)
			r.StopRequested = c.hand && s == "failed"
			rows = append(rows, r)
		}
		w := domain.ScanWork(rows)
		if len(w) != 1 || w[0].Ending != c.want || !w[0].FinishedAt.Equal(at.Add(10*time.Minute)) {
			t.Errorf("%s: %+v", name, w)
		}
	}
}

func TestRecordedWorkIsPricedOnTheAnthropicKey(t *testing.T) {
	id := uuid.New()
	r := domain.Recorded{
		ID: id, Kind: work.KindTermReview, StartedAt: at, FinishedAt: at.Add(time.Minute),
		Outcome: work.Outcome{Ending: work.EndingStoppedByHand, Counts: work.Counts{Done: 40, Total: 85}, InputTokens: 2000, OutputTokens: 200},
	}
	w := r.Work()
	if w.ID != id.String() || w.Kind != work.KindTermReview || w.Ending != work.EndingStoppedByHand || w.Counts.Done != 40 ||
		w.Anthropic != haiku(2000, 200) || w.Running() {
		t.Errorf("recorded = %+v", w)
	}
	r.FinishedAt, r.Outcome.Ending = time.Time{}, ""
	if w := r.Work(); !w.Running() {
		t.Errorf("a running one = %+v", w)
	}
}

func TestActivityShowsWhatRunsAndTheLastTwentyEnded(t *testing.T) {
	var all []domain.Work
	for i := range 25 {
		all = append(all, domain.Work{
			ID: fmt.Sprint("done", i), Kind: work.KindScan, StartedAt: at.Add(time.Duration(i) * time.Minute),
			FinishedAt: at.Add(time.Duration(i)*time.Minute + time.Second), Ending: work.EndingDone,
		})
	}
	all = append(all,
		domain.Work{ID: "old", Kind: work.KindPromptRun, StartedAt: at.Add(-time.Hour)},
		domain.Work{ID: "new", Kind: work.KindTermReview, StartedAt: at.Add(time.Hour)},
	)
	a := domain.Arrange(all)
	if len(a.Running) != 2 || a.Running[0].ID != "new" || a.Running[1].ID != "old" {
		t.Errorf("running = %+v", a.Running)
	}
	if len(a.Recent) != domain.RecentLimit || a.Recent[0].ID != "done24" || a.Recent[19].ID != "done5" {
		t.Errorf("recent = %d, first %s, last %s", len(a.Recent), a.Recent[0].ID, a.Recent[len(a.Recent)-1].ID)
	}
	if a := domain.Arrange(nil); a.Running == nil || a.Recent == nil {
		t.Errorf("nothing = %+v", a)
	}
}

func TestThisMonthsSpendingByKindOfWorkWithTheKeyThatPaid(t *testing.T) {
	// Each kind's tokens are priced at the model that used them; one with no model recorded, as Haiku.
	sonnet := llm.TokenUse{Model: "anthropic: claude-sonnet-5-5", Input: 100_000}
	u := domain.MonthUsage{
		PromptAnswers: 3_000_000, OpenRouterReadings: 50_000,
		HaikuReadings: []llm.TokenUse{{Input: 1_000_000, Output: 100_000}}, Scans: []llm.TokenUse{{Input: 2_000_000}, sonnet},
		TermReview: []llm.TokenUse{{Output: 1_000_000}}, CandidateChecks: []llm.TokenUse{{Model: "claude-sonnet-5-5", Input: 500_000}},
		RecommendationRuns: []llm.TokenUse{{Model: "claude-opus-5-5", Output: 1_000}},
	}
	got := domain.SpendingOf(u)
	want := []domain.SpendLine{
		{Kind: domain.SpendPromptRuns, Key: domain.KeyOpenRouter, Cost: 3_000_000},
		{Kind: domain.SpendScans, Key: domain.KeyAnthropic, Cost: haiku(2_000_000, 0) + sonnet.Cost()},
		{Kind: domain.SpendReadingResponses, Key: domain.KeyOpenRouter, Cost: 50_000},
		{Kind: domain.SpendReadingResponses, Key: domain.KeyAnthropic, Cost: haiku(1_000_000, 100_000)},
		{Kind: domain.SpendTermReview, Key: domain.KeyAnthropic, Cost: haiku(0, 1_000_000)},
		{Kind: domain.SpendCandidateChecks, Key: domain.KeyAnthropic, Cost: llm.ListPrice("claude-sonnet-5-5").Of(500_000, 0)},
		{Kind: domain.SpendRecommendationRuns, Key: domain.KeyAnthropic, Cost: llm.ListPrice("claude-opus-5-5").Of(0, 1_000)},
	}
	if len(got) != len(want) {
		t.Fatalf("spending = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestEachPieceOfWorkIsPricedAtTheModelThatDidIt(t *testing.T) {
	const fable = "claude-fable-5-1"
	fablePrice := func(in, out int64) llm.Micros { return llm.ListPrice(fable).Of(in, out) }
	run := domain.PromptRunRow{ID: uuid.New(), StartedAt: at, Status: "running", HaikuInput: 1_000, HaikuOutput: 100, ReadingModel: "anthropic: " + fable}
	if got := run.Work().Anthropic; got != fablePrice(1_000, 100) {
		t.Errorf("a Prompt run read by Fable = %s", got)
	}
	scan := domain.ScanRow{ID: uuid.New(), CreatedAt: at, Status: "done", InputTokens: 2_000, OutputTokens: 200, Model: "anthropic: " + fable}
	older := domain.ScanRow{ID: uuid.New(), CreatedAt: at, Status: "done", InputTokens: 2_000, OutputTokens: 200}
	if works := domain.ScanWork([]domain.ScanRow{scan}); len(works) != 1 || works[0].Anthropic != fablePrice(2_000, 200) {
		t.Errorf("a scan matched by Fable = %+v", works)
	}
	// A re-scan adds each scan at its own model: one by Fable, one recorded before scans named theirs.
	rescan := []domain.ScanRow{
		{ID: uuid.New(), Rescan: true, CreatedBy: "joris", CreatedAt: at, FinishedAt: at, Status: "done", InputTokens: 2_000, OutputTokens: 200, Model: fable},
		{ID: uuid.New(), Rescan: true, CreatedBy: "joris", CreatedAt: at, FinishedAt: at, Status: "done", InputTokens: older.InputTokens, OutputTokens: older.OutputTokens},
	}
	if w := domain.ScanWork(rescan); len(w) != 1 || w[0].Anthropic != fablePrice(2_000, 200)+haiku(2_000, 200) {
		t.Errorf("a re-scan = %+v", w)
	}
	review := domain.Recorded{
		ID: uuid.New(), Kind: work.KindTermReview, StartedAt: at, FinishedAt: at,
		Outcome: work.Outcome{Ending: work.EndingDone, InputTokens: 3_000, OutputTokens: 300, Model: fable},
	}
	if got := review.Work().Anthropic; got != fablePrice(3_000, 300) {
		t.Errorf("a term review on Fable = %s", got)
	}
	if got := (domain.Recorded{ID: uuid.New(), Kind: work.KindTermReview, Outcome: work.Outcome{InputTokens: 3_000}}).Work().Anthropic; got != haiku(3_000, 0) {
		t.Errorf("a review that recorded no model = %s, want Haiku's price", got)
	}
}
