package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/activity/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/activity/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
)

var (
	now     = time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)
	errBoom = errors.New("boom")
)

// source is an in-memory Source; failOn names the method that fails.
type source struct {
	runs      []domain.PromptRunRow
	scans     []domain.ScanRow
	recorded  []domain.Recorded
	usage     domain.MonthUsage
	since     time.Time
	limits    []int
	failOn    string
	abandoned, staleBefore, unbeatenBefore time.Time
}

func (s *source) err(op string) error {
	if s.failOn == op {
		return errBoom
	}
	return nil
}

func (s *source) PromptRuns(_ context.Context, limit int) ([]domain.PromptRunRow, error) {
	s.limits = append(s.limits, limit)
	return s.runs, s.err("PromptRuns")
}

func (s *source) Scans(_ context.Context, limit int) ([]domain.ScanRow, error) {
	s.limits = append(s.limits, limit)
	return s.scans, s.err("Scans")
}

func (s *source) Recorded(_ context.Context, limit int) ([]domain.Recorded, error) {
	s.limits = append(s.limits, limit)
	return s.recorded, s.err("Recorded")
}

func (s *source) MonthUsage(_ context.Context, since time.Time) (domain.MonthUsage, error) {
	s.since = since
	return s.usage, s.err("MonthUsage")
}

func (s *source) EndAbandoned(_ context.Context, at, staleBefore, unbeatenBefore time.Time) (int, error) {
	s.abandoned, s.staleBefore, s.unbeatenBefore = at, staleBefore, unbeatenBefore
	return 2, s.err("EndAbandoned")
}

func TestActivityGathersEveryKindOfWork(t *testing.T) {
	src := &source{
		runs:  []domain.PromptRunRow{{ID: uuid.New(), StartedAt: now.Add(-time.Hour), Status: "running", Tasks: 10, Answered: 3}},
		scans: []domain.ScanRow{{ID: uuid.New(), CreatedAt: now.Add(-2 * time.Hour), FinishedAt: now.Add(-time.Hour), Status: "done", Pages: 5, MaxPages: 30}},
		recorded: []domain.Recorded{{
			ID: uuid.New(), Kind: work.KindTermReview, StartedAt: now.Add(-3 * time.Hour), FinishedAt: now.Add(-2 * time.Hour),
			Outcome: work.Outcome{Ending: work.EndingStoppedByHand},
		}},
	}
	a, err := app.New(src, func() time.Time { return now }).Activity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Running) != 1 || a.Running[0].Kind != work.KindPromptRun || len(a.Recent) != 2 || a.Recent[0].Kind != work.KindScan ||
		a.Recent[1].Ending != work.EndingStoppedByHand {
		t.Errorf("activity = %+v", a)
	}
	if len(src.limits) != 3 || src.limits[0] != domain.RecentLimit || src.limits[1] != app.ScanRows || src.limits[2] != domain.RecentLimit {
		t.Errorf("limits = %v", src.limits)
	}
	for _, op := range []string{"PromptRuns", "Scans", "Recorded"} {
		src.failOn = op
		if _, err := app.New(src, func() time.Time { return now }).Activity(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("%s failing: %v", op, err)
		}
	}
}

func TestSpendingIsThisMonths(t *testing.T) {
	src := &source{usage: domain.MonthUsage{PromptAnswers: 2_000_000, TermReview: llm.TokenUse{Input: 1_000_000}}}
	svc := app.New(src, func() time.Time { return now })
	lines, since, err := svc.Spending(t.Context())
	if err != nil || len(lines) != 7 || lines[0].Cost != 2_000_000 || !since.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) ||
		!src.since.Equal(since) {
		t.Errorf("spending = %+v since %s, %v", lines, since, err)
	}
	src.failOn = "MonthUsage"
	if _, _, err := svc.Spending(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("usage failing: %v", err)
	}
}

func TestWorkLeftRunningWhenTheServerStoppedIsEndedAsFailed(t *testing.T) {
	src := &source{}
	svc := app.New(src, func() time.Time { return now })
	if n, err := svc.EndAbandoned(t.Context()); n != 2 || err != nil || !src.abandoned.Equal(now) ||
		!src.staleBefore.Equal(now.Add(-work.Stale)) || !src.unbeatenBefore.Equal(now.Add(-time.Hour)) {
		t.Errorf("EndAbandoned = %d, %v at %s, stale before %s, unbeaten before %s", n, err, src.abandoned, src.staleBefore, src.unbeatenBefore)
	}
	src.failOn = "EndAbandoned"
	if _, err := svc.EndAbandoned(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("failing: %v", err)
	}
}
