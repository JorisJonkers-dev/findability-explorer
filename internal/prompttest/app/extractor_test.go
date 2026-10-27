package app_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// haiku is an Extractor on Claude Haiku: it reads every answer as naming Twentebelt, and fails the
// first fail readings.
type haiku struct {
	mu   sync.Mutex
	read int
	fail int
}

func (h *haiku) Extract(_ context.Context, in app.ExtractInput) (domain.Extracted, domain.Reply, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.fail > 0 {
		h.fail--
		return domain.Extracted{}, domain.Reply{InputTokens: 7, OutputTokens: 1, Cost: 1}, errBoom
	}
	if in.FactSheet == "" || in.Prompt == "" {
		return domain.Extracted{}, domain.Reply{}, errors.New("the extractor got the wrong input")
	}
	h.read++
	return domain.Extracted{Companies: []domain.Named{{Name: "Twentebelt"}}}, domain.Reply{InputTokens: 60, OutputTokens: 25, Cost: 3}, nil
}

// extractors hands out x, or no extractor, or fails.
type extractors struct {
	x   app.Extractor
	err error
}

func (e extractors) Current(context.Context) (app.Extractor, string, error) {
	if e.err != nil || e.x == nil {
		return nil, "", e.err
	}
	return e.x, "anthropic: claude-haiku-5-5", nil
}

func (e *env) withExtractors(x app.Extractors, opts ...app.Option) *app.Service {
	opts = append([]app.Option{app.WithSleep(e.sleeps.Sleep), app.WithSpawn(func(f func()) { f() }), app.WithExtractors(x)}, opts...)
	return app.New(e.repo, gateways{gw: e.gw}, e.dir, e.clock.Now, app.Config{Owner: "a", Pace: time.Second, RetryWait: 10 * time.Second, Retries: 2},
		func(_ uuid.UUID, err error) {
			e.mu.Lock()
			defer e.mu.Unlock()
			e.failures = append(e.failures, err)
		},
		opts...)
}

func TestHaikuReadsTheAnswersWhileAnAnthropicKeyWorks(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	h := &haiku{fail: 1}
	e.svc = e.withExtractors(extractors{x: h})
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if got := e.run(t); got.Status != domain.RunDone || len(e.repo.answers) != 80 || h.read != 80 || e.gw.extracted != 0 {
		t.Fatalf("status %s, %d answers, %d read by Haiku, %d by OpenRouter", got.Status, len(e.repo.answers), h.read, e.gw.extracted)
	}
	// The first reading failed and was tried again; both count.
	var in, out int64
	var cost domain.Micros
	for _, a := range e.repo.answers {
		if a.Extraction == nil || a.Extraction.Failed {
			t.Fatalf("answer not read: %+v", a)
		}
		in, out, cost = in+a.Extraction.InputTokens, out+a.Extraction.OutputTokens, cost+a.Extraction.Cost
	}
	if in != 80*60+7 || out != 80*25+1 || cost != 80*3+1 {
		t.Errorf("readings used %d in, %d out, cost %d", in, out, cost)
	}
	res, err := e.svc.Results(t.Context())
	if err != nil || res.Runs[0].Progress.Cost != 80*(10+3)+1 {
		t.Errorf("progress = %+v, %v", res.Runs[0].Progress, err)
	}
	view, err := e.svc.Settings(t.Context())
	if err != nil || view.ExtractorModel != "anthropic: claude-haiku-5-5" {
		t.Errorf("extractor shown = %q, %v", view.ExtractorModel, err)
	}
}

func TestOpenRouterReadsTheAnswersWithoutAnAnthropicKey(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.svc = e.withExtractors(extractors{})
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if got := e.run(t); got.Status != domain.RunDone || e.gw.extracted != 80 {
		t.Fatalf("status %s, %d read by OpenRouter", got.Status, e.gw.extracted)
	}
	view, err := e.svc.Settings(t.Context())
	if err != nil || view.ExtractorModel != domain.ExtractorModel {
		t.Errorf("extractor shown = %q, %v", view.ExtractorModel, err)
	}
}

func TestARunFailsWhenTheExtractorCannotBeHad(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.svc = e.withExtractors(extractors{err: errBoom})
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if got := e.run(t); got.Status != domain.RunFailed || len(e.repo.answers) != 0 {
		t.Fatalf("status %s, %d answers", got.Status, len(e.repo.answers))
	}
	if _, err := e.svc.Settings(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("settings = %v", err)
	}
}

func (h *haiku) reads() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.read
}

func TestHaikuHandsTheReadingToOpenRouterOnceTheMonthsTokensAreSpent(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	h := &haiku{}
	// The month's tokens are spent after Haiku's tenth reading.
	e.svc = e.withExtractors(extractors{x: h}, app.WithBudget(func(context.Context) (bool, error) { return h.reads() >= 10, nil }))
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if got := e.run(t); got.Status != domain.RunDone || h.read < 10 || h.read+e.gw.extracted != 80 || e.gw.extracted == 0 {
		t.Fatalf("status %s, %d read by Haiku, %d by OpenRouter", got.Status, h.read, e.gw.extracted)
	}
	var tokens int64
	for _, a := range e.repo.answers {
		tokens += a.Extraction.HaikuTokens
	}
	if tokens != int64(h.read)*(60+25) {
		t.Errorf("Haiku's tokens = %d for %d readings", tokens, h.read)
	}
	view, err := e.svc.Settings(t.Context())
	if err != nil || view.ExtractorModel != domain.ExtractorModel {
		t.Errorf("extractor shown once the tokens are spent = %q, %v", view.ExtractorModel, err)
	}
}

func TestHaikuKeepsReadingWhenTheBudgetCannotBeReadDuringARun(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	h := &haiku{fail: 2} // the first answer cannot be read: its tokens still count
	var mu sync.Mutex
	calls := 0
	e.svc = e.withExtractors(extractors{x: h}, app.WithBudget(func(context.Context) (bool, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls > 1 {
			return false, errBoom
		}
		return false, nil
	}))
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if got := e.run(t); got.Status != domain.RunDone || h.read != 79 || e.gw.extracted != 0 {
		t.Fatalf("status %s, %d read by Haiku, %d by OpenRouter", got.Status, h.read, e.gw.extracted)
	}
	failed := 0
	for _, a := range e.repo.answers {
		if a.Extraction.Failed {
			failed++
			if a.Extraction.HaikuTokens != 2*(7+1) {
				t.Errorf("a failed reading's Haiku tokens = %d", a.Extraction.HaikuTokens)
			}
		}
	}
	if failed != 1 {
		t.Errorf("%d readings failed", failed)
	}
}

func TestARunFailsWhenTheBudgetCannotBeReadAtItsStart(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.svc = e.withExtractors(extractors{x: &haiku{}}, app.WithBudget(func(context.Context) (bool, error) { return false, errBoom }))
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if got := e.run(t); got.Status != domain.RunFailed || len(e.repo.answers) != 0 {
		t.Fatalf("status %s, %d answers", got.Status, len(e.repo.answers))
	}
}
