package app

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// ErrNothingToStop is returned by a stop that finds no queued or running work.
var ErrNothingToStop = errors.New("there is nothing queued or running to stop")

// errStopped ends the context of a scan an administrator stopped.
var errStopped = errors.New("stopped by hand")

// runs holds a cancel per scan this instance runs, so a stop asked here ends it at once. A scan
// running on another instance sees the stop in the database between pages (Service.stopped).
type runs struct {
	mu     sync.Mutex
	cancel map[uuid.UUID]context.CancelCauseFunc
}

func (r *runs) add(id uuid.UUID, cancel context.CancelCauseFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel == nil {
		r.cancel = map[uuid.UUID]context.CancelCauseFunc{}
	}
	r.cancel[id] = cancel
}

func (r *runs) drop(id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cancel, id)
}

// stop ends the scan with id when this instance runs it.
func (r *runs) stop(id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if cancel, ok := r.cancel[id]; ok {
		cancel(errStopped)
	}
}

// StopScan stops one scan: a waiting one is taken off the queue, a running one ends after the
// page it reads, keeping the pages it read so far. Either is recorded as stopped by hand
// (domain.StoppedByHandReason). ErrNothingToStop when the scan neither waits nor runs.
func (s *Service) StopScan(ctx context.Context, id uuid.UUID) error {
	return s.stop(ctx, domain.StopScope{ID: id})
}

// StopRescan stops every scan of the re-scan: the waiting ones are taken off the queue, the
// running ones end. A scan an administrator started alone runs on. ErrNothingToStop when no
// scan of a re-scan waits or runs.
func (s *Service) StopRescan(ctx context.Context) error {
	return s.stop(ctx, domain.StopScope{RescanOnly: true})
}

// RemoveWaitingScans takes every waiting scan off the queue, recorded as stopped by hand; the
// running ones run on. ErrNothingToStop when none waits.
func (s *Service) RemoveWaitingScans(ctx context.Context) error {
	n, err := s.stopQueued(ctx, domain.StopScope{})
	if err == nil && n == 0 {
		return ErrNothingToStop
	}
	return err
}

// stop stops the waiting and the running scans scope picks. The waiting ones come first, so a
// scan that starts meanwhile is asked to stop as a running one.
func (s *Service) stop(ctx context.Context, scope domain.StopScope) error {
	queued, err := s.stopQueued(ctx, scope)
	if err != nil {
		return err
	}
	running, err := s.repo.RequestStop(ctx, scope, s.now())
	if err != nil {
		return fmt.Errorf("ask the running scans to stop: %w", err)
	}
	for _, id := range running {
		s.runs.stop(id)
	}
	if queued+len(running) == 0 {
		return ErrNothingToStop
	}
	return nil
}

// stopQueued records the waiting scans scope picks as stopped by hand, takes them off the queue
// and counts them.
func (s *Service) stopQueued(ctx context.Context, scope domain.StopScope) (int, error) {
	ids, err := s.repo.StopQueued(ctx, scope, domain.StoppedByHandReason, s.now())
	if err != nil {
		return 0, fmt.Errorf("take the waiting scans off the queue: %w", err)
	}
	s.queue.remove(ids)
	return len(ids), nil
}

// stopped reports whether the scan with id was stopped: here, which ended run, or on any
// instance, which the database holds. A stop that cannot be read is no stop.
func (s *Service) stopped(ctx, run context.Context, id uuid.UUID) bool {
	if errors.Is(context.Cause(run), errStopped) {
		return true
	}
	asked, err := s.repo.StopRequested(ctx, id)
	return err == nil && asked
}
