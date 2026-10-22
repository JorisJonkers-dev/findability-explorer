// Package httpapi assembles the generated ogen server from each context's web adapter, and maps
// every error the contract does not name to an RFC 9457 problem.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"

	analysisweb "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/web"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
	terminologyweb "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/web"
)

// The aliases give each embedded adapter its own field name.
type (
	terminologyHandler = terminologyweb.Handler
	analysisHandler    = analysisweb.Handler
)

// api is the oas.Handler: one embedded web adapter per context, plus the shared error mapping.
type api struct {
	*terminologyHandler
	*analysisHandler
	logger *slog.Logger
}

var _ oas.Handler = (*api)(nil)

// NewError maps every error the contract does not name to a problem: a request without an
// identity to a 401, anything else a handler returned to a 500. A 500's cause is logged, never sent.
func (a *api) NewError(ctx context.Context, err error) *oas.ProblemStatusCode {
	code := ogenerrors.ErrorCode(err)
	if code >= http.StatusInternalServerError {
		a.logger.ErrorContext(ctx, "request failed", "error", err)
	}
	return &oas.ProblemStatusCode{StatusCode: code, Response: httpx.Problem(code, "")}
}

// identity accepts every request the edge put an identity on; ogen refuses one without before
// calling it. The subject goes on ctx, where the analysis context counts a user's analyses.
type identity struct{}

func (identity) HandleForwardAuth(ctx context.Context, _ oas.OperationName, t oas.ForwardAuth) (context.Context, error) {
	return httpx.WithSubject(ctx, t.APIKey), nil
}

// Contexts are the use cases of every context the API serves.
type Contexts struct {
	Terminology terminologyweb.UseCases
	Analysis    analysisweb.UseCases
}

// New returns the API's http.Handler, serving every path the contract declares under /api.
func New(logger *slog.Logger, c Contexts) (http.Handler, error) {
	return oas.NewServer(
		&api{
			terminologyHandler: terminologyweb.New(c.Terminology),
			analysisHandler:    analysisweb.New(c.Analysis),
			logger:             logger,
		},
		identity{},
		oas.WithErrorHandler(func(_ context.Context, w http.ResponseWriter, _ *http.Request, err error) {
			httpx.WriteProblem(w, ogenerrors.ErrorCode(err), "")
		}),
		oas.WithNotFound(func(w http.ResponseWriter, _ *http.Request) {
			httpx.WriteProblem(w, http.StatusNotFound, "")
		}),
		oas.WithMethodNotAllowed(func(w http.ResponseWriter, _ *http.Request, allowed string) {
			w.Header().Set("Allow", allowed)
			httpx.WriteProblem(w, http.StatusMethodNotAllowed, "")
		}),
	)
}
