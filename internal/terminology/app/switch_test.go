package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

func TestTheReviewIsOnUntilAnAdministratorSwitchesItOff(t *testing.T) {
	s := &store{}
	svc := reviewing(phrases("contact"), &reviews{}, s, judges{judge: &judge{rule: accepting}})
	if st := state(t, svc); !st.On {
		t.Fatalf("a review nobody switched = %+v", st)
	}
	if err := svc.SwitchReview(t.Context(), "joris", false); err != nil {
		t.Fatal(err)
	}
	if st := state(t, svc); st.On {
		t.Errorf("switched off = %+v", st)
	}
	if err := svc.SwitchReview(t.Context(), "anna", true); err != nil {
		t.Fatal(err)
	}
	want := []string{"false by joris at 2026-10-23T09:00:00Z", "true by anna at 2026-10-23T09:00:00Z"}
	if st := state(t, svc); !st.On || !slices.Equal(s.switched, want) {
		t.Errorf("switched on again = %+v; switched %v", st, s.switched)
	}
	s.switchErr = errBoom
	if err := svc.SwitchReview(t.Context(), "joris", false); !errors.Is(err, errBoom) {
		t.Errorf("a switch that cannot be stored = %v", err)
	}
	if err := newService(catalogue{}).SwitchReview(t.Context(), "joris", false); !errors.Is(err, app.ErrNoModel) {
		t.Errorf("a switch without a reviewer = %v", err)
	}
}

func TestTheReviewStateCountsThisMonthsUsage(t *testing.T) {
	s := &store{usage: []domain.Usage{
		{InputTokens: 1_000_000, OutputTokens: 100_000, At: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
		{InputTokens: 1_000_000, OutputTokens: 300_000, At: now},
		{InputTokens: 9_000_000, OutputTokens: 9_000_000, At: time.Date(2026, 9, 30, 23, 59, 0, 0, time.UTC)},
	}}
	svc := reviewing(phrases("contact"), &reviews{}, s, judges{judge: &judge{rule: accepting}})
	st := state(t, svc)
	if st.Month.Tokens() != 2_400_000 || st.Month.CostUSD() != 0.4 || !s.usedSince.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("this month = %+v ($%v), since %s", st.Month, st.Month.CostUSD(), s.usedSince)
	}
	for name, fail := range map[string]func(*store){
		"the switch": func(s *store) { s.onErr = errBoom },
		"the usage":  func(s *store) { s.usageErr = errBoom },
	} {
		broken := &store{}
		fail(broken)
		svc := reviewing(phrases("contact"), &reviews{}, broken, judges{})
		if _, err := svc.ReviewState(t.Context(), domain.Counts{}); !errors.Is(err, errBoom) {
			t.Errorf("%s unreadable: ReviewState = %v", name, err)
		}
	}
}

func TestWithTheReviewOffNothingIsReviewedByItselfButReviewNowRunsOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	j := &judge{rule: accepting}
	s := &store{off: true}
	svc := reviewing(phrases("contact"), &reviews{}, s, judges{judge: j})
	done := make(chan struct{})
	go func() {
		svc.ReviewLoop(ctx, time.Millisecond, func(err error) { t.Errorf("reported %v", err) })
		close(done)
	}()
	// The loop looks at the switch at the start and on every tick, and asks nothing.
	waitFor(t, func() bool { return s.reads() >= 5 })
	if asked := asks(j); asked != 0 {
		t.Fatalf("asked %d times with the review off", asked)
	}
	svc.StartReview()
	waitFor(t, func() bool { st := state(t, svc); return st.Waiting == 0 && !st.Running })
	if asked := asks(j); asked != 1 {
		t.Errorf("Review now asked %d times", asked)
	}
	cancel()
	<-done
}

func TestASwitchThatCannotBeReadReviewsNothingAndIsReported(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	j := &judge{rule: accepting}
	svc := reviewing(phrases("contact"), &reviews{}, &store{onErr: errBoom}, judges{judge: j})
	errs := make(chan error, 10)
	go svc.ReviewLoop(ctx, time.Hour, func(err error) { errs <- err })
	if err := <-errs; !errors.Is(err, errBoom) {
		t.Errorf("reported %v", err)
	}
	if asked := asks(j); asked != 0 {
		t.Errorf("asked %d times", asked)
	}
}

func TestAStoppedReviewKeepsWhatItDecidedAndIsRecordedAsStoppedByHand(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	names := make([]string, 0, 2*domain.ReviewBatch+5)
	for i := range 2*domain.ReviewBatch + 5 {
		names = append(names, fmt.Sprintf("phrase %03d", i))
	}
	j := &judge{rule: accepting}
	s := &store{}
	svc := reviewing(phrases(names...), &reviews{}, s, judges{judge: j})
	stopped := make(chan error, 1)
	j.fail = func([]domain.Candidate) error {
		if len(j.asked) == 1 {
			stopped <- svc.StopReview(t.Context())
		}
		return nil
	}
	go svc.ReviewLoop(ctx, time.Hour, func(err error) { t.Errorf("reported %v", err) })
	if err := <-stopped; err != nil {
		t.Fatalf("the stop while it ran = %v", err)
	}
	waitFor(t, func() bool { return !state(t, svc).Running })
	// The batch under way when the stop came is kept; no batch is asked after it.
	st := state(t, svc)
	if st.Problem != app.ProblemStopped || st.Waiting != domain.ReviewBatch+5 || asks(j) != 1 || len(s.verdicts) != domain.ReviewBatch {
		t.Errorf("state %+v; asked %d times, %d verdicts", st, asks(j), len(s.verdicts))
	}
	if err := svc.StopReview(t.Context()); !errors.Is(err, app.ErrNothingToStop) {
		t.Errorf("a stop with nothing running = %v", err)
	}
	// Review now goes on from there.
	svc.StartReview()
	waitFor(t, func() bool {
		st := state(t, svc)
		return st.Waiting == 0 && !st.Running && st.Problem == app.ProblemNone
	})
}

func TestAReviewAskedForAndStoppedBeforeItStartsDoesNotRun(t *testing.T) {
	j := &judge{rule: accepting}
	svc := reviewing(phrases("contact"), &reviews{}, &store{}, judges{judge: j})
	if err := svc.StopReview(t.Context()); !errors.Is(err, app.ErrNothingToStop) {
		t.Fatalf("a stop with nothing asked = %v", err)
	}
	svc.StartReview()
	if !state(t, svc).Running {
		t.Fatal("a review asked for does not show")
	}
	if err := svc.StopReview(t.Context()); err != nil {
		t.Fatal(err)
	}
	if st := state(t, svc); st.Running || st.Problem != app.ProblemStopped {
		t.Errorf("state = %+v", st)
	}
	if err := newService(catalogue{}).StopReview(t.Context()); !errors.Is(err, app.ErrNothingToStop) {
		t.Errorf("a stop without a reviewer = %v", err)
	}
}

// asks counts the requests j had.
func asks(j *judge) int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return len(j.asked)
}
