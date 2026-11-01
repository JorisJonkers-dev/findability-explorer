// Package app holds the Activity use cases: what runs and what ended, and this month's spending by
// kind of work. Their only entry points are the ones the inbound adapters call.
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/activity/domain"
)

// ScanRows is how many of the newest scans Activity reads: enough for the last pieces of work when
// re-scans of many companies are among them.
const ScanRows = 500

// Source reads the work the app did from where it is recorded.
type Source interface {
	// PromptRuns returns the running Prompt run and the newest limit ended ones.
	PromptRuns(ctx context.Context, limit int) ([]domain.PromptRunRow, error)
	// Scans returns the waiting and running scans and the newest limit others.
	Scans(ctx context.Context, limit int) ([]domain.ScanRow, error)
	// Recorded returns the journal's running work and the newest limit ended pieces.
	Recorded(ctx context.Context, limit int) ([]domain.Recorded, error)
	// MonthUsage is what each kind of work used since since.
	MonthUsage(ctx context.Context, since time.Time) (domain.MonthUsage, error)
	// EndAbandoned ends, as failed at at, the journal's work still running, and counts it.
	EndAbandoned(ctx context.Context, at time.Time) (int, error)
}

// Service runs the Activity use cases.
type Service struct {
	source Source
	now    func() time.Time
}

// New returns a Service reading source.
func New(source Source, now func() time.Time) *Service { return &Service{source: source, now: now} }

// Activity is the work running now and the last pieces ended, of every kind.
func (s *Service) Activity(ctx context.Context) (domain.Activity, error) {
	runs, err := s.source.PromptRuns(ctx, domain.RecentLimit)
	if err != nil {
		return domain.Activity{}, fmt.Errorf("the Prompt runs: %w", err)
	}
	scans, err := s.source.Scans(ctx, ScanRows)
	if err != nil {
		return domain.Activity{}, fmt.Errorf("the scans: %w", err)
	}
	recorded, err := s.source.Recorded(ctx, domain.RecentLimit)
	if err != nil {
		return domain.Activity{}, fmt.Errorf("the recorded work: %w", err)
	}
	all := domain.ScanWork(scans)
	for _, r := range runs {
		all = append(all, r.Work())
	}
	for _, r := range recorded {
		all = append(all, r.Work())
	}
	return domain.Arrange(all), nil
}

// Spending is this month's spending by kind of work with the key that paid, and when the month
// began.
func (s *Service) Spending(ctx context.Context) ([]domain.SpendLine, time.Time, error) {
	now := s.now().UTC()
	since := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	u, err := s.source.MonthUsage(ctx, since)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("this month's usage: %w", err)
	}
	return domain.SpendingOf(u), since, nil
}

// EndAbandoned ends, as failed, the recorded work a stopped server left running: it runs in memory
// and stopped with the server. Call it at startup, before any work begins.
func (s *Service) EndAbandoned(ctx context.Context) (int, error) {
	n, err := s.source.EndAbandoned(ctx, s.now())
	if err != nil {
		return 0, fmt.Errorf("end the work left running: %w", err)
	}
	return n, nil
}
