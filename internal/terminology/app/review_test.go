package app_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// store is an in-memory Verdicts; the err fields make a method fail.
type store struct {
	mu               sync.Mutex
	verdicts         []domain.Verdict
	usage            []domain.Usage
	listErr, saveErr error
}

func (s *store) Verdicts(context.Context) ([]domain.Verdict, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.verdicts, s.listErr
}

func (s *store) SaveVerdicts(_ context.Context, v []domain.Verdict, u domain.Usage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.saveErr != nil {
		return s.saveErr
	}
	s.verdicts = append(s.verdicts, v...)
	s.usage = append(s.usage, u)
	return nil
}

// judge answers each phrase by the rule given, and counts the requests.
type judge struct {
	mu       sync.Mutex
	rule     func(domain.Candidate) (domain.Verdict, bool)
	err      error
	asked    [][]string
	examples int
}

func (j *judge) Judge(_ context.Context, cs []domain.Candidate, examples []domain.Term) ([]domain.Verdict, domain.Usage, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	var phrases []string
	var out []domain.Verdict
	for _, c := range cs {
		phrases = append(phrases, c.Phrase)
		if v, ok := j.rule(c); ok {
			v.CandidateID = c.ID
			out = append(out, v)
		}
	}
	j.asked = append(j.asked, phrases)
	j.examples = len(examples)
	return out, domain.Usage{InputTokens: 100, OutputTokens: 10}, j.err
}

type judges struct {
	judge app.Judge
	err   error
}

func (j judges) Current(context.Context) (app.Judge, string, error) {
	return j.judge, "claude-haiku-5-5", j.err
}

// accepting accepts "spiral freezer belts", rejects "contact", and is unsure about the rest.
func accepting(c domain.Candidate) (domain.Verdict, bool) {
	switch c.Phrase {
	case "spiral freezer belts":
		return domain.Verdict{Accept: true, Sure: true, Reason: "A belt buyers search for.", Term: domain.Acceptance{
			Text: "spiral freezer belt", Language: "en", Family: "tri_flex_turn_curve", Tier: "application",
		}}, true
	case "contact":
		return domain.Verdict{Sure: true, Reason: "Navigation."}, true
	default:
		return domain.Verdict{Accept: true, Reason: "Could go either way."}, true
	}
}

func phrases(names ...string) []domain.Observation {
	var out []domain.Observation
	for i, p := range names {
		out = append(out, domain.Observation{
			ScanID: fmt.Sprintf("s%d", i), CompanyID: "twentebelt", CompanyName: "Twentebelt", ScanURL: "https://www.twentebelt.com/",
			StartedAt: now, Phrase: p, Language: "en", Score: 10, Pages: 1, Tier: "product",
		})
	}
	return out
}

func reviewing(obs []domain.Observation, r *reviews, s *store, j app.Judges) *app.Service {
	svc := app.New(catalogue{terms: many(3)}, r, evidence(obs, nil), sightings(nil, nil), func() time.Time { return now })
	svc.WithReviewer(s, j)
	return svc
}

func TestTheModelDecidesWhatItIsSureOf(t *testing.T) {
	r, s, j := &reviews{}, &store{}, &judge{rule: accepting}
	svc := reviewing(phrases("spiral freezer belts", "contact", "freezing"), r, s, judges{judge: j})

	n, err := svc.Review(t.Context())
	if err != nil || n != 3 || len(j.asked) != 1 || j.examples != 3 {
		t.Fatalf("Review = %d, %v; asked %v with %d examples", n, err, j.asked, j.examples)
	}
	if len(s.verdicts) != 3 || len(s.usage) != 1 || s.usage[0].InputTokens != 100 {
		t.Errorf("stored %+v, usage %+v", s.verdicts, s.usage)
	}
	if len(r.decisions) != 2 {
		t.Fatalf("decisions = %+v", r.decisions)
	}
	byPhrase := map[string]domain.Decision{}
	for _, d := range r.decisions {
		byPhrase[d.Phrase] = d
	}
	if d := byPhrase["spiral freezer belts"]; d.Status != domain.StatusAccepted || d.By != domain.ModelReviewer || d.Term == nil || d.Term.Text != "spiral freezer belt" {
		t.Errorf("the accepted one = %+v", d)
	}
	if d := byPhrase["contact"]; d.Status != domain.StatusRejected || d.By != domain.ModelReviewer {
		t.Errorf("the rejected one = %+v", d)
	}
	// The accepted term joins the map at once.
	if len(svc.Terms()) != 4 {
		t.Errorf("the map has %d terms", len(svc.Terms()))
	}
	// The unsure one waits for a person, with the model's verdict; it is not asked about again.
	pending, _, err := svc.Candidates(t.Context(), domain.StatusPending)
	if err != nil || len(pending) != 1 || pending[0].Verdict == nil || pending[0].Verdict.Reason != "Could go either way." {
		t.Fatalf("pending = %+v, %v", pending, err)
	}
	if n, err := svc.Review(t.Context()); n != 0 || err != nil || len(j.asked) != 1 {
		t.Errorf("a second Review = %d, %v; asked %v", n, err, j.asked)
	}
	state, err := svc.ReviewState(t.Context())
	if err != nil || state != (app.ReviewState{}) {
		t.Errorf("ReviewState = %+v, %v", state, err)
	}
}

func TestTheModelIsAskedInBatches(t *testing.T) {
	names := make([]string, 0, domain.ReviewBatch+5)
	for i := range domain.ReviewBatch + 5 {
		names = append(names, fmt.Sprintf("phrase %02d", i))
	}
	j := &judge{rule: accepting}
	svc := reviewing(phrases(names...), &reviews{}, &store{}, judges{judge: j})
	if state, _ := svc.ReviewState(t.Context()); state.Waiting != domain.ReviewBatch+5 {
		t.Errorf("waiting = %d", state.Waiting)
	}
	if n, err := svc.Review(t.Context()); n != domain.ReviewBatch+5 || err != nil || len(j.asked) != 2 || len(j.asked[1]) != 5 {
		t.Errorf("Review = %d, %v; asked %d times", n, err, len(j.asked))
	}
}

func TestAPersonsDecisionMeanwhileStands(t *testing.T) {
	r := &reviews{decideErr: domain.ErrDecided}
	svc := reviewing(phrases("spiral freezer belts"), r, &store{}, judges{judge: &judge{rule: accepting}})
	if n, err := svc.Review(t.Context()); n != 1 || err != nil {
		t.Errorf("Review = %d, %v", n, err)
	}
}

func TestReviewWithoutAModel(t *testing.T) {
	plain := newService(catalogue{})
	if _, err := plain.Review(t.Context()); !errors.Is(err, app.ErrNoModel) {
		t.Errorf("without a reviewer = %v", err)
	}
	if state, err := plain.ReviewState(t.Context()); err != nil || state.Problem != app.ProblemNoModel {
		t.Errorf("ReviewState without a reviewer = %+v, %v", state, err)
	}
	plain.StartReview()                                                // nothing to start
	plain.ReviewLoop(t.Context(), time.Hour, func(error) { t.Fail() }) // returns at once
	svc := reviewing(phrases("contact"), &reviews{}, &store{}, judges{})
	if _, err := svc.Review(t.Context()); !errors.Is(err, app.ErrNoModel) {
		t.Errorf("without a model = %v", err)
	}
}

func TestReviewFailuresStopIt(t *testing.T) {
	cases := map[string]struct {
		r *reviews
		s *store
		j app.Judges
		e app.Evidence
	}{
		"the settings":      {&reviews{}, &store{}, judges{err: errBoom}, evidence(phrases("contact"), nil)},
		"the model":         {&reviews{}, &store{}, judges{judge: &judge{rule: accepting, err: errBoom}}, evidence(phrases("contact"), nil)},
		"saving the usage":  {&reviews{}, &store{saveErr: errBoom}, judges{judge: &judge{rule: accepting, err: errBoom}}, evidence(phrases("contact"), nil)},
		"saving verdicts":   {&reviews{}, &store{saveErr: errBoom}, judges{judge: &judge{rule: accepting}}, evidence(phrases("contact"), nil)},
		"a decision":        {&reviews{decideErr: errBoom}, &store{}, judges{judge: &judge{rule: accepting}}, evidence(phrases("contact"), nil)},
		"reading verdicts":  {&reviews{}, &store{listErr: errBoom}, judges{judge: &judge{rule: accepting}}, evidence(phrases("contact"), nil)},
		"reading the scans": {&reviews{}, &store{}, judges{judge: &judge{rule: accepting}}, evidence(nil, errBoom)},
		"loading the map":   {&reviews{listErrAfter: 3}, &store{}, judges{judge: &judge{rule: accepting}}, evidence(phrases("contact"), nil)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			svc := app.New(catalogue{}, tc.r, tc.e, sightings(nil, nil), func() time.Time { return now })
			svc.WithReviewer(tc.s, tc.j)
			if _, err := svc.Review(t.Context()); !errors.Is(err, errBoom) {
				t.Errorf("Review = %v", err)
			}
		})
	}
	// A failed request's usage still counts.
	s := &store{}
	svc := reviewing(phrases("contact"), &reviews{}, s, judges{judge: &judge{rule: accepting, err: errBoom}})
	_, _ = svc.Review(t.Context())
	if len(s.usage) != 1 || len(s.verdicts) != 0 {
		t.Errorf("after a failed request: usage %+v, verdicts %+v", s.usage, s.verdicts)
	}
	broken := reviewing(phrases("contact"), &reviews{}, &store{listErr: errBoom}, judges{})
	if _, err := broken.ReviewState(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("ReviewState reading verdicts = %v", err)
	}
	unread := app.New(catalogue{}, &reviews{}, evidence(nil, errBoom), sightings(nil, nil), func() time.Time { return now })
	unread.WithReviewer(&store{}, judges{})
	if _, err := unread.ReviewState(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("ReviewState reading the scans = %v", err)
	}
}

// A judge that fails until it is fixed, so the loop shows the problem and then clears it.
type flaky struct {
	mu    sync.Mutex
	fails bool
	j     *judge
}

func (f *flaky) Current(context.Context) (app.Judge, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fails {
		return nil, "", errBoom
	}
	return f.j, "claude-haiku-5-5", nil
}

func TestTheLoopReviewsWhenAskedAndSaysWhyItStopped(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f := &flaky{fails: true, j: &judge{rule: accepting}}
	svc := reviewing(phrases("contact"), &reviews{}, &store{}, f)
	errs := make(chan error, 10)
	done := make(chan struct{})
	go func() {
		svc.ReviewLoop(ctx, time.Hour, func(err error) { errs <- err })
		close(done)
	}()
	if err := <-errs; !errors.Is(err, errBoom) {
		t.Fatalf("the loop's error = %v", err)
	}
	waitFor(t, func() bool { st, _ := svc.ReviewState(ctx); return st.Problem == app.ProblemFailed && !st.Running })

	f.mu.Lock()
	f.fails = false
	f.mu.Unlock()
	svc.StartReview()
	svc.StartReview() // asked for already
	waitFor(t, func() bool {
		st, _ := svc.ReviewState(ctx)
		return st.Problem == app.ProblemNone && st.Waiting == 0 && !st.Running
	})

	noModel := reviewing(phrases("contact"), &reviews{}, &store{}, judges{})
	go noModel.ReviewLoop(ctx, time.Hour, func(err error) { t.Errorf("no model reported as %v", err) })
	waitFor(t, func() bool { st, _ := noModel.ReviewState(ctx); return st.Problem == app.ProblemNoModel })

	cancel()
	<-done
}

// A review cut short by shutdown is no problem to show.
func TestAReviewStoppedByShutdownIsNoProblem(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	j := &judge{rule: func(domain.Candidate) (domain.Verdict, bool) { cancel(); return domain.Verdict{}, false }, err: context.Canceled}
	svc := reviewing(phrases("contact"), &reviews{}, &store{}, judges{judge: j})
	svc.ReviewLoop(ctx, time.Hour, func(err error) { t.Errorf("reported %v", err) })
	if st, _ := svc.ReviewState(t.Context()); st.Problem != app.ProblemNone {
		t.Errorf("problem = %q", st.Problem)
	}
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
