package app_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// journal records the work begun and how it ended; beginErr makes Begin fail.
type journal struct {
	mu       sync.Mutex
	begun    []work.Kind
	ended    []work.Outcome
	beginErr error
}

func (j *journal) Begin(_ context.Context, kind work.Kind, _ string, _ time.Time) (uuid.UUID, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.begun = append(j.begun, kind)
	return uuid.New(), j.beginErr
}

func (j *journal) End(_ context.Context, _ uuid.UUID, o work.Outcome, _ time.Time) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.ended = append(j.ended, o)
	return nil
}

func (j *journal) outcomes() []work.Outcome {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]work.Outcome(nil), j.ended...)
}

func many45() []string {
	names := make([]string, 0, domain.ReviewBatch+5)
	for i := range domain.ReviewBatch + 5 {
		names = append(names, fmt.Sprintf("phrase %02d", i))
	}
	return names
}

func TestAReviewIsRecordedWithWhatItDidAndUsed(t *testing.T) {
	j := &journal{}
	jd := &judge{rule: accepting}
	svc := reviewing(phrases(many45()...), &reviews{}, &store{}, judges{judge: jd})
	svc.WithJournal(j, func(err error) { t.Errorf("reported %v", err) })
	if _, err := svc.Review(t.Context()); err != nil {
		t.Fatal(err)
	}
	want := work.Outcome{
		Ending: work.EndingDone, Counts: work.Counts{Done: domain.ReviewBatch + 5, Total: domain.ReviewBatch + 5}, InputTokens: 200, OutputTokens: 20,
	}
	if len(j.begun) != 1 || j.begun[0] != work.KindTermReview || len(j.ended) != 1 || j.ended[0] != want {
		t.Fatalf("begun %v, ended %+v", j.begun, j.ended)
	}
	// With nothing waiting, nothing ran, and nothing is recorded.
	if _, err := svc.Review(t.Context()); err != nil || len(j.begun) != 1 {
		t.Errorf("a review of nothing = %v; begun %v", err, j.begun)
	}
}

func TestAReviewCutShortIsRecordedAsItEnded(t *testing.T) {
	// The budget is spent after the first batch.
	j := &journal{}
	jd := &judge{rule: accepting}
	svc := reviewing(phrases(many45()...), &reviews{}, &store{}, judges{judge: jd})
	b := &budget{j: jd, after: 1}
	svc.WithBudget(b.spent)
	svc.WithJournal(j, func(err error) { t.Errorf("reported %v", err) })
	if _, err := svc.Review(t.Context()); !errors.Is(err, app.ErrBudgetSpent) {
		t.Fatal(err)
	}
	if o := j.ended[0]; o.Ending != work.EndingStoppedByBudget || o.Counts != (work.Counts{Done: domain.ReviewBatch, Total: domain.ReviewBatch + 5}) {
		t.Errorf("stopped by the budget = %+v", o)
	}
	// A review the budget allows no batch of is not recorded: it asked nothing.
	if _, err := svc.Review(t.Context()); !errors.Is(err, app.ErrBudgetSpent) || len(j.begun) != 1 {
		t.Errorf("a spent budget = %v; begun %v", err, j.begun)
	}
	// The model fails.
	j = &journal{}
	failing := reviewing(phrases("contact"), &reviews{}, &store{}, judges{judge: &judge{rule: accepting, fail: func([]domain.Candidate) error { return errBoom }}})
	failing.WithJournal(j, func(err error) { t.Errorf("reported %v", err) })
	if _, err := failing.Review(t.Context()); !errors.Is(err, errBoom) {
		t.Fatal(err)
	}
	if o := j.ended[0]; o.Ending != work.EndingFailed || o.InputTokens != 100 {
		t.Errorf("failed = %+v", o)
	}
}

func TestAReviewStoppedByHandIsRecordedSo(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	j := &journal{}
	jd := &judge{rule: accepting}
	svc := reviewing(phrases(many45()...), &reviews{}, &store{}, judges{judge: jd})
	svc.WithJournal(j, func(err error) { t.Errorf("reported %v", err) })
	jd.fail = func([]domain.Candidate) error {
		if len(jd.asked) == 1 {
			_ = svc.StopReview(t.Context())
		}
		return nil
	}
	go svc.ReviewLoop(ctx, time.Hour, func(err error) { t.Errorf("reported %v", err) })
	waitFor(t, func() bool { return len(j.outcomes()) == 1 })
	if o := j.outcomes()[0]; o.Ending != work.EndingStoppedByHand || o.Counts.Done != domain.ReviewBatch {
		t.Errorf("stopped by hand = %+v", o)
	}
}

func TestAJournalThatFailsLeavesTheReviewAlone(t *testing.T) {
	j := &journal{beginErr: errBoom}
	var reported []error
	svc := reviewing(phrases("contact"), &reviews{}, &store{}, judges{judge: &judge{rule: accepting}})
	svc.WithJournal(j, func(err error) { reported = append(reported, err) })
	if n, err := svc.Review(t.Context()); n != 1 || err != nil || len(reported) != 1 || !errors.Is(reported[0], errBoom) {
		t.Errorf("Review = %d, %v; reported %v", n, err, reported)
	}
}
