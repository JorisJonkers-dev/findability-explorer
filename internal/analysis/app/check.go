package app

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
)

// checkTimeout bounds one check of every company's AI readiness.
const checkTimeout = 2 * time.Hour

// ReadinessCheck is how far checking every scanned company's AI readiness again has come.
type ReadinessCheck struct {
	Running bool
	// Checked counts the sites done so far, of Total; Failed those of them that could not be read
	// or recorded, which keep what their scan recorded.
	Checked int
	Failed  int
	Total   int
	// Stopped is true for a check an administrator stopped; the sites it checked keep what it
	// recorded, the others what their scan recorded.
	Stopped bool
}

// readinessCheck holds the state of the one check that may run at a time, and ends it.
type readinessCheck struct {
	mu     sync.Mutex
	state  ReadinessCheck
	cancel context.CancelFunc
}

// ReadinessCheckState reports the check that runs, or how the last one ended.
func (s *Service) ReadinessCheckState() ReadinessCheck {
	s.check.mu.Lock()
	defer s.check.mu.Unlock()
	return s.check.state
}

// StartReadinessCheck checks every scanned company's AI readiness again in the background, unless
// a check runs already, and returns the check's state.
func (s *Service) StartReadinessCheck(ctx context.Context) (ReadinessCheck, error) {
	scans, err := s.checkable(ctx)
	if err != nil {
		return ReadinessCheck{}, err
	}
	s.check.mu.Lock()
	defer s.check.mu.Unlock()
	if s.check.state.Running {
		return s.check.state, nil
	}
	s.check.state = ReadinessCheck{Running: true, Total: len(scans)}
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), checkTimeout)
	s.check.cancel = cancel
	go func() {
		defer cancel()
		entry := work.Start(bg, s.journal, work.KindReadinessCheck, checkBy, s.now(), s.report)
		s.CheckReadiness(bg, scans)
		s.check.mu.Lock()
		defer s.check.mu.Unlock()
		entry.Finish(context.WithoutCancel(bg), checkOutcome(s.check.state), s.now(), s.report)
		s.check.state.Running = false
	}()
	return s.check.state, nil
}

// checkBy is who the work journal names as having started a readiness check.
const checkBy = "administrator"

// checkOutcome is how a check ended, for the work journal.
func checkOutcome(c ReadinessCheck) work.Outcome {
	o := work.Outcome{Ending: work.EndingDone, Counts: work.Counts{Done: c.Checked, Total: c.Total, Failed: c.Failed}}
	switch {
	case c.Stopped:
		o.Ending = work.EndingStoppedByHand
	case c.Failed > 0:
		o.Ending = work.EndingDoneWithFailures
	}
	return o
}

// WithJournal records each readiness check in j, as Activity lists it; report hears that j
// failed, which leaves the check alone.
func (s *Service) WithJournal(j work.Journal, report func(error)) {
	s.journal, s.report = j, report
}

// StopReadinessCheck stops the check that runs: it reads no further site, and the site it reads
// is not counted. ErrNothingToStop when no check runs.
func (s *Service) StopReadinessCheck(context.Context) error {
	s.check.mu.Lock()
	defer s.check.mu.Unlock()
	if !s.check.state.Running {
		return ErrNothingToStop
	}
	s.check.state.Stopped = true
	s.check.cancel()
	return nil
}

// checkable are each company's newest scan, by company id, leaving out one still queued or
// running: that scan reads the readiness itself.
func (s *Service) checkable(ctx context.Context) ([]domain.Analysis, error) {
	stats, err := s.repo.CompetitorStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the scans: %w", err)
	}
	var out []domain.Analysis
	for _, id := range slices.Sorted(maps.Keys(stats)) {
		if last := stats[id].Last; last.Status != domain.StatusQueued && last.Status != domain.StatusRunning {
			out = append(out, last)
		}
	}
	return out, nil
}

// CheckReadiness reads each scan's site's AI readiness files again, a few sites at a time, and
// records what they say on that scan, dated now, with the sitemaps' coverage of the pages the scan
// read. It crawls no pages and asks no language model,
// so it costs nothing. A site that cannot be read keeps what its scan recorded.
func (s *Service) CheckReadiness(ctx context.Context, scans []domain.Analysis) {
	work := make(chan domain.Analysis)
	var wg sync.WaitGroup
	for range max(1, s.limits.Workers) {
		wg.Go(func() {
			for a := range work {
				ok := s.checkOne(ctx, a)
				if ctx.Err() != nil {
					continue // a site the check was reading when it ended does not count
				}
				s.check.mu.Lock()
				s.check.state.Checked++
				if !ok {
					s.check.state.Failed++
				}
				s.check.mu.Unlock()
			}
		})
	}
	for _, a := range scans {
		if ctx.Err() != nil {
			break
		}
		work <- a
	}
	close(work)
	wg.Wait()
}

// checkOne checks one scan's site again and reports whether its readiness was recorded.
func (s *Service) checkOne(ctx context.Context, a domain.Analysis) bool {
	target, err := domain.NewTarget(a.URL)
	if err != nil {
		return false
	}
	files, err := s.crawler.Readiness(ctx, target)
	if err != nil {
		return false
	}
	// The coverage is measured again against the pages the scan read: a check crawls none.
	scan, err := s.repo.Get(ctx, a.ID)
	if err != nil {
		return false
	}
	read := make([]string, 0, len(scan.Pages))
	for _, p := range scan.Pages {
		read = append(read, p.URL)
	}
	r := domain.AssessReadiness(files)
	r.Sitemap.Coverage = domain.CoverageOf(files.Sitemaps, read)
	r.CheckedAt = s.now()
	return s.repo.SaveReadiness(ctx, a.ID, r, domain.RawFilesOf(files)) == nil
}
