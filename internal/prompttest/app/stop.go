package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// StopRun asks a running Prompt run to stop, or returns domain.ErrNotFound when no run of id is
// running. The request is stored, so the instance running the run sees it at its next heartbeat,
// within a minute; the run keeps the responses so far and is recorded as stopped by hand, and a
// retry finishes it. user is who asked.
func (s *Service) StopRun(ctx context.Context, _ string, id uuid.UUID) error {
	ok, err := s.repo.RequestStop(ctx, id, s.now())
	if err != nil {
		return fmt.Errorf("ask the run to stop: %w", err)
	}
	if !ok {
		return domain.ErrNotFound
	}
	return nil
}
