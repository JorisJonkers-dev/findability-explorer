package web

import (
	"context"
	"errors"
	"net/http"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
)

// stopped maps a stop's outcome: done, nothing to stop (with detail), or an error to report.
func stopped[R any](err error, done R, notFound func(oas.ProblemStatusCode) R, detail string) (R, error) {
	switch {
	case errors.Is(err, app.ErrNothingToStop):
		return notFound(oas.ProblemStatusCode{
			StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, detail),
		}), nil
	case err != nil:
		var none R
		return none, err
	}
	return done, nil
}

// StopAnalysis implements stopAnalysis.
func (h *Handler) StopAnalysis(ctx context.Context, params oas.StopAnalysisParams) (oas.StopAnalysisRes, error) {
	return stopped[oas.StopAnalysisRes](h.uc.StopScan(ctx, params.AnalysisId), &oas.StopAnalysisNoContent{},
		func(p oas.ProblemStatusCode) oas.StopAnalysisRes { r := oas.StopAnalysisNotFound(p); return &r },
		"No waiting or running scan has this id.")
}

// RemoveQueuedAnalyses implements removeQueuedAnalyses.
func (h *Handler) RemoveQueuedAnalyses(ctx context.Context) (oas.RemoveQueuedAnalysesRes, error) {
	return stopped[oas.RemoveQueuedAnalysesRes](h.uc.RemoveWaitingScans(ctx), &oas.RemoveQueuedAnalysesNoContent{},
		func(p oas.ProblemStatusCode) oas.RemoveQueuedAnalysesRes {
			r := oas.RemoveQueuedAnalysesNotFound(p)
			return &r
		},
		"No scan waits.")
}

// StopRescan implements stopRescan.
func (h *Handler) StopRescan(ctx context.Context) (oas.StopRescanRes, error) {
	return stopped[oas.StopRescanRes](h.uc.StopRescan(ctx), &oas.StopRescanNoContent{},
		func(p oas.ProblemStatusCode) oas.StopRescanRes { r := oas.StopRescanNotFound(p); return &r },
		"No scan of a re-scan waits or runs.")
}

// StopReadinessCheck implements stopReadinessCheck.
func (h *Handler) StopReadinessCheck(ctx context.Context) (oas.StopReadinessCheckRes, error) {
	return stopped[oas.StopReadinessCheckRes](h.uc.StopReadinessCheck(ctx), &oas.StopReadinessCheckNoContent{},
		func(p oas.ProblemStatusCode) oas.StopReadinessCheckRes {
			r := oas.StopReadinessCheckNotFound(p)
			return &r
		},
		"No AI readiness check runs.")
}
