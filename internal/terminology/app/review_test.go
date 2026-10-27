package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// store is an in-memory Verdicts that records decisions in reviews; the err fields make a method fail.
type store struct {
	mu               sync.Mutex
	reviews          *reviews
	verdicts         []domain.Verdict
	usage            []domain.Usage
	listErr, saveErr error
}

func (s *store) Verdicts(context.Context) ([]domain.Verdict, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.verdicts, s.listErr
}

func (s *store) SaveReview(ctx context.Context, v []domain.Verdict, ds []domain.Decision, u []domain.Usage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.saveErr != nil {
		return s.saveErr
	}
	s.verdicts = append(s.verdicts, v...)
	s.usage = append(s.usage, u...)
	for _, d := range ds {
		decided, _ := s.reviews.Decisions(ctx)
		if !slices.ContainsFunc(decided, func(x domain.Decision) bool { return x.CandidateID == d.CandidateID }) {
			_ = s.reviews.Decide(ctx, d)
		}
	}
	return nil
}

// judge answers each phrase by the rule given, and counts the requests.
type judge struct {
	mu       sync.Mutex
	rule     func(domain.Candidate) (domain.Verdict, bool)
	err      error
	fail     func([]domain.Candidate) error // fails a request when set and non-nil
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
	if j.fail != nil {
		if err := j.fail(cs); err != nil {
			return nil, domain.Usage{InputTokens: 100}, err
		}
	}
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
	s.reviews = r
	svc.WithReviewer(s, j)
	return svc
}

// state is the review's state, with the waiting candidates counted.
func state(t *testing.T, svc *app.Service) app.ReviewState {
	t.Helper()
	_, counts, err := svc.Candidates(t.Context(), domain.StatusPending)
	if err != nil {
		t.Fatal(err)
	}
	return svc.ReviewState(t.Context(), counts)
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
	if st := state(t, svc); st != (app.ReviewState{}) {
		t.Errorf("ReviewState = %+v", st)
	}
}

func TestTheModelIsAskedInBatches(t *testing.T) {
	names := make([]string, 0, domain.ReviewBatch+5)
	for i := range domain.ReviewBatch + 5 {
		names = append(names, fmt.Sprintf("phrase %02d", i))
	}
	j := &judge{rule: accepting}
	svc := reviewing(phrases(names...), &reviews{}, &store{}, judges{judge: j})
	if st := state(t, svc); st.Waiting != domain.ReviewBatch+5 {
		t.Errorf("waiting = %d", st.Waiting)
	}
	if n, err := svc.Review(t.Context()); n != domain.ReviewBatch+5 || err != nil || len(j.asked) != 2 || len(j.asked[1]) != 5 {
		t.Errorf("Review = %d, %v; asked %d times", n, err, len(j.asked))
	}
}

func TestAPersonsDecisionMeanwhileStands(t *testing.T) {
	r := &reviews{}
	j := &judge{rule: accepting}
	svc := reviewing(phrases("spiral freezer belts"), r, &store{}, judges{judge: j})
	// While the model thinks, a person rejects the phrase.
	j.fail = func(cs []domain.Candidate) error {
		return r.Decide(t.Context(), domain.Decision{CandidateID: cs[0].ID, Phrase: cs[0].Phrase, Status: domain.StatusRejected, By: "joris", At: now})
	}
	if n, err := svc.Review(t.Context()); n != 1 || err != nil {
		t.Fatalf("Review = %d, %v", n, err)
	}
	if len(r.decisions) != 1 || r.decisions[0].By != "joris" {
		t.Errorf("decisions = %+v", r.decisions)
	}
}

func TestAnUnusableAnswerIsAskedAgainInHalves(t *testing.T) {
	j := &judge{rule: accepting, fail: func(cs []domain.Candidate) error {
		if slices.ContainsFunc(cs, func(c domain.Candidate) bool { return c.Phrase == "poison" }) {
			return fmt.Errorf("%w: cut off", app.ErrBadAnswer)
		}
		return nil
	}}
	s := &store{}
	svc := reviewing(phrases("contact", "poison", "spiral freezer belts", "freezing"), &reviews{}, s, judges{judge: j})
	if n, err := svc.Review(t.Context()); n != 4 || err != nil {
		t.Fatalf("Review = %d, %v", n, err)
	}
	// All four, then each half, then each phrase of the half with the poison.
	if len(j.asked) != 5 || len(s.usage) != 5 || len(s.verdicts) != 4 {
		t.Fatalf("asked %v; usage %d, verdicts %d", j.asked, len(s.usage), len(s.verdicts))
	}
	for _, v := range s.verdicts {
		if v.Phrase == "poison" && (v.Sure || v.Reason != "") {
			t.Errorf("the phrase it could not answer = %+v", v)
		}
	}
}

func TestAFailingProviderStopsTheReviewAndKeepsThePhrasesForIt(t *testing.T) {
	j := &judge{rule: accepting, fail: func([]domain.Candidate) error { return errBoom }}
	s := &store{}
	svc := reviewing(phrases("contact", "freezing"), &reviews{}, s, judges{judge: j})
	if _, err := svc.Review(t.Context()); !errors.Is(err, errBoom) {
		t.Fatalf("Review = %v", err)
	}
	if len(j.asked) != 1 || len(s.usage) != 1 || len(s.verdicts) != 0 || state(t, svc).Waiting != 2 {
		t.Errorf("asked %v; usage %d, verdicts %d", j.asked, len(s.usage), len(s.verdicts))
	}
}

func TestReviewWithoutAModel(t *testing.T) {
	plain := newService(catalogue{})
	if _, err := plain.Review(t.Context()); !errors.Is(err, app.ErrNoModel) {
		t.Errorf("without a reviewer = %v", err)
	}
	if st := plain.ReviewState(t.Context(), domain.Counts{Unreviewed: 2}); st.Problem != app.ProblemNoModel || st.Waiting != 2 {
		t.Errorf("ReviewState without a reviewer = %+v", st)
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
		"saving the usage":  {&reviews{}, &store{saveErr: errBoom}, judges{judge: &judge{rule: accepting, err: errBoom}}, evidence(phrases("contact"), nil)},
		"saving the review": {&reviews{}, &store{saveErr: errBoom}, judges{judge: &judge{rule: accepting}}, evidence(phrases("contact"), nil)},
		"reading verdicts":  {&reviews{}, &store{listErr: errBoom}, judges{judge: &judge{rule: accepting}}, evidence(phrases("contact"), nil)},
		"reading the scans": {&reviews{}, &store{}, judges{judge: &judge{rule: accepting}}, evidence(nil, errBoom)},
		"loading the map":   {&reviews{listErrAfter: 3}, &store{}, judges{judge: &judge{rule: accepting}}, evidence(phrases("spiral freezer belts"), nil)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			svc := app.New(catalogue{}, tc.r, tc.e, sightings(nil, nil), func() time.Time { return now })
			tc.s.reviews = tc.r
			svc.WithReviewer(tc.s, tc.j)
			if _, err := svc.Review(t.Context()); !errors.Is(err, errBoom) {
				t.Errorf("Review = %v", err)
			}
		})
	}
	// A second half that fails stops it too.
	calls := 0
	j := &judge{rule: accepting, fail: func([]domain.Candidate) error {
		calls++
		switch calls {
		case 1:
			return app.ErrBadAnswer
		case 3:
			return errBoom
		}
		return nil
	}}
	svc := reviewing(phrases("contact", "freezing"), &reviews{}, &store{}, judges{judge: j})
	if _, err := svc.Review(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("a failing second half = %v", err)
	}
	first := &judge{rule: accepting, fail: func(cs []domain.Candidate) error {
		if len(cs) == 2 {
			return app.ErrBadAnswer
		}
		return errBoom
	}}
	svc = reviewing(phrases("contact", "freezing"), &reviews{}, &store{}, judges{judge: first})
	if _, err := svc.Review(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("a failing first half = %v", err)
	}
}

// A key added after a review found none shows at once.
type switchable struct {
	mu sync.Mutex
	j  app.Judge
}

func (s *switchable) Current(context.Context) (app.Judge, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.j, "m", nil
}

func TestAKeyAddedSinceShowsAtOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	sw := &switchable{}
	svc := reviewing(phrases("contact"), &reviews{}, &store{}, sw)
	go svc.ReviewLoop(ctx, time.Hour, func(err error) { t.Errorf("reported %v", err) })
	waitFor(t, func() bool { return state(t, svc).Problem == app.ProblemNoModel })
	sw.mu.Lock()
	sw.j = &judge{rule: accepting}
	sw.mu.Unlock()
	if st := state(t, svc); st.Problem != app.ProblemNone || st.Waiting != 1 {
		t.Errorf("after adding a key = %+v", st)
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
	waitFor(t, func() bool { st := state(t, svc); return st.Problem == app.ProblemFailed && !st.Running })

	f.mu.Lock()
	f.fails = false
	f.mu.Unlock()
	svc.StartReview()
	svc.StartReview() // asked for already
	waitFor(t, func() bool {
		st := state(t, svc)
		return st.Problem == app.ProblemNone && st.Waiting == 0 && !st.Running
	})

	noModel := reviewing(phrases("contact"), &reviews{}, &store{}, judges{})
	go noModel.ReviewLoop(ctx, time.Hour, func(err error) { t.Errorf("no model reported as %v", err) })
	waitFor(t, func() bool { return state(t, noModel).Problem == app.ProblemNoModel })

	cancel()
	<-done
}

// A review cut short by shutdown is no problem to show.
func TestAReviewStoppedByShutdownIsNoProblem(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	j := &judge{rule: func(domain.Candidate) (domain.Verdict, bool) { cancel(); return domain.Verdict{}, false }, err: context.Canceled}
	svc := reviewing(phrases("contact"), &reviews{}, &store{}, judges{judge: j})
	svc.ReviewLoop(ctx, time.Hour, func(err error) { t.Errorf("reported %v", err) })
	if st := state(t, svc); st.Problem != app.ProblemNone {
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
