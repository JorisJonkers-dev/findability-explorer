// Package app holds the analysis context's use cases: starting an analysis, running it in the
// background, and reading it back. They are the only entry points the inbound adapters call.
package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

const (
	// DefaultListLimit is how many analyses List returns when the caller names no limit.
	DefaultListLimit = 20
	// MaxListLimit is the most analyses one List returns.
	MaxListLimit = 50
	// maxFailure is the longest failure reason stored; the contract and the database agree.
	maxFailure = 500
)

var (
	// ErrRateLimited is returned when a user has started too many analyses in the past hour.
	ErrRateLimited = errors.New("too many analyses started in the past hour; try again later")
	// ErrBusy is returned when the queue is full.
	ErrBusy = errors.New("the analysis queue is full; try again in a few minutes")
)

// Limits bound what analyses may cost.
type Limits struct {
	// PerUserPerHour is how many analyses one user may start in an hour.
	PerUserPerHour int
	// MonthlyTokens is how many language-model tokens all analyses of a calendar month may use.
	MonthlyTokens int64
	// SiteTextChars is how much site text the language model reads per analysis.
	SiteTextChars int
	// QueueSize is how many analyses may wait to run.
	QueueSize int
}

// Glossary returns the terminology map's terms; the terminology context provides it.
type Glossary func() []domain.GlossaryTerm

// Service runs the analysis use cases.
type Service struct {
	repo     domain.Repository
	crawler  domain.Crawler
	matcher  domain.Matcher // nil when no language model is configured
	glossary Glossary
	now      func() time.Time
	limits   Limits
	queue    chan uuid.UUID
}

// New returns a Service. matcher may be nil, which disables product matching.
func New(repo domain.Repository, crawler domain.Crawler, matcher domain.Matcher, glossary Glossary,
	now func() time.Time, limits Limits,
) *Service {
	return &Service{
		repo: repo, crawler: crawler, matcher: matcher, glossary: glossary, now: now, limits: limits,
		queue: make(chan uuid.UUID, max(limits.QueueSize, 1)),
	}
}

// Start validates the request, checks the user's hourly limit, stores a queued analysis and puts
// it on the queue.
func (s *Service) Start(ctx context.Context, user, rawURL string, maxPages int) (domain.Analysis, error) {
	target, err := domain.NewTarget(rawURL)
	if err != nil {
		return domain.Analysis{}, err
	}
	pages, err := domain.CheckMaxPages(maxPages)
	if err != nil {
		return domain.Analysis{}, err
	}
	now := s.now()
	started, err := s.repo.CountCreatedSince(ctx, user, now.Add(-time.Hour))
	if err != nil {
		return domain.Analysis{}, fmt.Errorf("count analyses: %w", err)
	}
	if started >= s.limits.PerUserPerHour {
		return domain.Analysis{}, ErrRateLimited
	}
	a := domain.Analysis{
		ID: uuid.New(), URL: target.String(), MaxPages: pages, Status: domain.StatusQueued,
		CreatedBy: user, CreatedAt: now, Matching: domain.MatchingPending,
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return domain.Analysis{}, fmt.Errorf("create analysis: %w", err)
	}
	select {
	case s.queue <- a.ID:
		return a, nil
	default:
		if err := s.repo.Fail(ctx, a.ID, ErrBusy.Error(), now); err != nil {
			return domain.Analysis{}, fmt.Errorf("fail analysis: %w", err)
		}
		return domain.Analysis{}, ErrBusy
	}
}

// Get returns one analysis with its results, or domain.ErrNotFound.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.Analysis, error) {
	return s.repo.Get(ctx, id)
}

// List returns up to limit analyses, newest first. A limit below 1 means DefaultListLimit.
func (s *Service) List(ctx context.Context, limit int) ([]domain.Analysis, error) {
	switch {
	case limit < 1:
		limit = DefaultListLimit
	case limit > MaxListLimit:
		limit = MaxListLimit
	}
	return s.repo.List(ctx, limit)
}

// Recover fails every analysis a restart interrupted and queues every analysis still waiting.
// Call it once at startup, before Work.
func (s *Service) Recover(ctx context.Context) error {
	if err := s.repo.FailRunning(ctx, "The service restarted while this analysis ran. Start it again.", s.now()); err != nil {
		return fmt.Errorf("fail interrupted analyses: %w", err)
	}
	ids, err := s.repo.Queued(ctx)
	if err != nil {
		return fmt.Errorf("list queued analyses: %w", err)
	}
	for _, id := range ids {
		select {
		case s.queue <- id:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// Work runs queued analyses one at a time until ctx ends. onError hears about every analysis
// that could not be recorded, which only the logs can report.
func (s *Service) Work(ctx context.Context, onError func(uuid.UUID, error)) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-s.queue:
			if err := s.Run(ctx, id); err != nil {
				onError(id, err)
			}
		}
	}
}

// Run crawls the analysis's site, extracts its keywords and known terms, matches its products,
// and records each step. A failure of the site itself is recorded on the analysis; Run returns
// an error only when the analysis could not be recorded.
func (s *Service) Run(ctx context.Context, id uuid.UUID) error {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("load analysis: %w", err)
	}
	if a.Status != domain.StatusQueued {
		return nil
	}
	if err := s.repo.MarkRunning(ctx, id); err != nil {
		return fmt.Errorf("mark running: %w", err)
	}
	target, err := domain.NewTarget(a.URL)
	if err != nil {
		return s.fail(ctx, id, err.Error())
	}
	var pages []domain.Page
	var recordErr error
	crawlErr := s.crawler.Crawl(ctx, target, a.MaxPages, func(p domain.Page) error {
		if err := s.repo.AddPage(ctx, id, domain.Summarise(p)); err != nil {
			recordErr = fmt.Errorf("add page: %w", err)
			return recordErr
		}
		pages = append(pages, p)
		return nil
	})
	if recordErr != nil {
		return recordErr
	}
	if len(pages) == 0 {
		reason := "No page could be fetched."
		if crawlErr != nil {
			reason = "No page could be fetched: " + crawlErr.Error()
		}
		return s.fail(ctx, id, reason)
	}
	if err := s.repo.SaveFindings(ctx, id, domain.Keywords(pages), domain.FindKnownTerms(pages, s.glossary())); err != nil {
		return fmt.Errorf("save findings: %w", err)
	}
	matching, eqs, usage := s.match(ctx, pages)
	if err := s.repo.SaveMatching(ctx, id, matching, eqs, usage); err != nil {
		return fmt.Errorf("save matching: %w", err)
	}
	if err := s.repo.Finish(ctx, id, s.now()); err != nil {
		return fmt.Errorf("finish analysis: %w", err)
	}
	return nil
}

// match asks the language model for the site's equivalents of Tribelt's families, within the
// monthly budget. It never fails the analysis: the crawl results stand on their own.
func (s *Service) match(ctx context.Context, pages []domain.Page) (domain.Matching, []domain.Equivalent, domain.Usage) {
	if s.matcher == nil {
		return domain.MatchingDisabled, nil, domain.Usage{}
	}
	now := s.now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	spent, err := s.repo.TokensSince(ctx, monthStart)
	if err != nil || spent >= s.limits.MonthlyTokens {
		return domain.MatchingSkipped, nil, domain.Usage{}
	}
	text, truncated := domain.SiteText(pages, s.limits.SiteTextChars)
	if strings.TrimSpace(text) == "" {
		return domain.MatchingSkipped, nil, domain.Usage{}
	}
	found, usage, err := s.matcher.Match(ctx, text, truncated, domain.TribeltFamilies())
	if err != nil {
		return domain.MatchingFailed, nil, usage
	}
	eqs := make([]domain.Equivalent, 0, len(found))
	for _, e := range found {
		if checked, err := domain.CheckEquivalent(e); err == nil {
			eqs = append(eqs, checked)
		}
	}
	return domain.MatchingDone, eqs, usage
}

func (s *Service) fail(ctx context.Context, id uuid.UUID, reason string) error {
	if r := []rune(reason); len(r) > maxFailure {
		reason = string(r[:maxFailure-1]) + "…"
	}
	if err := s.repo.Fail(ctx, id, reason, s.now()); err != nil {
		return fmt.Errorf("fail analysis: %w", err)
	}
	return nil
}
