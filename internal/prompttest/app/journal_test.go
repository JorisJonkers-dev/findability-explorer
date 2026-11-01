package app_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
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

func (j *journal) Beat(context.Context, uuid.UUID, time.Time) error { return nil }

func journalled(r *repo, d *directory, rv app.Reviewers, j *journal, onError func(error)) *app.Service {
	return app.New(r, gateways{gw: newGateway()}, d, func() time.Time { return start }, app.Config{Owner: "a"}, func(uuid.UUID, error) {},
		app.WithSpawn(func(f func()) { f() }), app.WithReviewers(rv, &pages{}, onError), app.WithJournal(j))
}

func companies(n int) [][2]string {
	names := make([][2]string, 0, n)
	for i := range n {
		names = append(names, [2]string{fmt.Sprintf("Company %02d", i), ""})
	}
	return names
}

func TestACandidateReviewIsRecordedWithWhatItCheckedAndUsed(t *testing.T) {
	r, _ := naming(companies(app.CheckBatch + 3)...)
	j := &journal{}
	svc := journalled(r, &directory{}, models{m: &model{verdicts: competitors}}, j, func(err error) { t.Errorf("reported %v", err) })
	svc.StartReview(t.Context())
	// One merge and two batches of checks, each 90 tokens in and 10 out.
	want := work.Outcome{Ending: work.EndingDone, Counts: work.Counts{Done: app.CheckBatch + 3, Total: app.CheckBatch + 3}, InputTokens: 270, OutputTokens: 30}
	if len(j.begun) != 1 || j.begun[0] != work.KindCandidateReview || len(j.ended) != 1 || j.ended[0] != want {
		t.Fatalf("begun %v, ended %+v", j.begun, j.ended)
	}
}

func TestAStoppedCandidateReviewIsRecordedAsStoppedByHand(t *testing.T) {
	r, _ := naming(companies(app.CheckBatch + 3)...)
	j := &journal{}
	m := &model{verdicts: competitors}
	var svc *app.Service
	sm := stopping{model: m, stop: func() { _ = svc.StopReview(t.Context()) }}
	svc = journalled(r, &directory{}, models{m: sm}, j, func(err error) { t.Errorf("reported %v", err) })
	svc.StartReview(t.Context())
	if o := j.ended[0]; o.Ending != work.EndingStoppedByHand || o.Counts != (work.Counts{Done: app.CheckBatch, Total: app.CheckBatch + 3}) {
		t.Errorf("stopped = %+v", o)
	}
}

func TestACompetitorCheckIsRecordedAsItEnded(t *testing.T) {
	r, _ := naming([2]string{"Fenner Dunlop", "https://www.fennerdunlop.com/"})
	j := &journal{}
	svc := journalled(r, onTheList(), models{m: &model{verdicts: rubber}}, j, func(err error) { t.Errorf("reported %v", err) })
	svc.StartListCheck(t.Context())
	want := work.Outcome{Ending: work.EndingDone, Counts: work.Counts{Done: 4, Total: 4}, InputTokens: 90, OutputTokens: 10}
	if len(j.begun) != 1 || j.begun[0] != work.KindCompetitorCheck || j.ended[0] != want {
		t.Fatalf("begun %v, ended %+v", j.begun, j.ended)
	}
	// Spent from the start, failing, and without a key.
	cases := map[string]struct {
		opts []app.Option
		rv   app.Reviewers
		want work.Ending
	}{
		"the budget": {[]app.Option{app.WithBudget(func(context.Context) (bool, error) { return true, nil })}, models{m: &model{verdicts: rubber}}, work.EndingStoppedByBudget},
		"failing":    {nil, models{m: &model{verdicts: rubber, checkErr: errBoom}}, work.EndingFailed},
		"no key":     {nil, models{}, work.EndingFailed},
	}
	for name, c := range cases {
		j := &journal{}
		opts := append([]app.Option{app.WithSpawn(func(f func()) { f() }), app.WithReviewers(c.rv, &pages{}, func(error) {}), app.WithJournal(j)}, c.opts...)
		svc := app.New(newRepo(), gateways{gw: newGateway()}, onTheList(), func() time.Time { return start }, app.Config{Owner: "a"}, func(uuid.UUID, error) {}, opts...)
		svc.StartListCheck(t.Context())
		if len(j.ended) != 1 || j.ended[0].Ending != c.want {
			t.Errorf("%s: ended %+v", name, j.ended)
		}
	}
}

func TestAJournalThatFailsIsReportedAndTheReviewGoesOn(t *testing.T) {
	r, _ := naming([2]string{"Kienzle GmbH", ""})
	j := &journal{beginErr: errBoom}
	var reported []error
	m := &model{verdicts: competitors}
	svc := journalled(r, &directory{}, models{m: m}, j, func(err error) { reported = append(reported, err) })
	svc.StartReview(t.Context())
	if len(reported) != 1 || !errors.Is(reported[0], errBoom) || m.merged != 1 || svc.ReviewState().Problem != app.ReviewProblemNone {
		t.Errorf("reported %v; merged %d; state %+v", reported, m.merged, svc.ReviewState())
	}
	// Without an onError, a failing journal is not reported.
	quiet := journalled(r, &directory{}, models{m: m}, &journal{beginErr: errBoom}, nil)
	quiet.StartReview(t.Context())
}
