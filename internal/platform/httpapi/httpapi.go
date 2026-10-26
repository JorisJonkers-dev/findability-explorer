// Package httpapi assembles the generated ogen server from each context's web adapter, and maps
// every error the contract does not name to an RFC 9457 problem.
package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"

	analysisweb "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/web"
	companiesweb "github.com/JorisJonkers-dev/findability-explorer/internal/companies/adapters/web"
	identityweb "github.com/JorisJonkers-dev/findability-explorer/internal/identity/adapters/web"
	llmweb "github.com/JorisJonkers-dev/findability-explorer/internal/llm/adapters/web"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
	prompttestweb "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/web"
	terminologyweb "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/web"
)

// The aliases give each embedded adapter its own field name.
type (
	terminologyHandler = terminologyweb.Handler
	analysisHandler    = analysisweb.Handler
	companiesHandler   = companiesweb.Handler
	identityHandler    = identityweb.Handler
	llmHandler         = llmweb.Handler
	promptTestHandler  = prompttestweb.Handler
)

// api is the oas.Handler: one embedded web adapter per context, plus the shared error mapping.
type api struct {
	*terminologyHandler
	*analysisHandler
	*companiesHandler
	*identityHandler
	*llmHandler
	*promptTestHandler
	logger *slog.Logger
}

var _ oas.Handler = (*api)(nil)

// NewError maps every error the contract does not name to a problem. The security handler's
// errors become 401 without a live session, 403 for an account that is not an administrator, and
// 503 when auth-api cannot confirm the roles; anything else a handler returned is a 500, whose
// cause is logged, never sent.
func (a *api) NewError(ctx context.Context, err error) *oas.ProblemStatusCode {
	code := ogenerrors.ErrorCode(err)
	detail := ""
	var sec *ogenerrors.SecurityError
	if errors.As(err, &sec) && !errors.Is(err, ogenerrors.ErrSecurityRequirementIsNotSatisfied) {
		code = identityweb.Status(sec.Err)
		if code != http.StatusInternalServerError && code != http.StatusUnauthorized {
			detail = identityweb.Detail(sec.Err)
		}
	}
	if code >= http.StatusInternalServerError && code != http.StatusServiceUnavailable {
		a.logger.ErrorContext(ctx, "request failed", "error", err)
	}
	return &oas.ProblemStatusCode{StatusCode: code, Response: httpx.Problem(code, detail)}
}

// Contexts are the use cases of every context the API serves, and how it authenticates.
type Contexts struct {
	Terminology terminologyweb.UseCases
	Analysis    analysisweb.UseCases
	Companies   companiesweb.UseCases
	LLM         llmweb.UseCases
	PromptTest  prompttestweb.UseCases
	// Auth resolves the session cookie to the principal a request runs as.
	Auth identityweb.Authenticator
}

// actor names the caller in the language-model events: the account's name.
func actor(ctx context.Context) string {
	if p, ok := identityweb.PrincipalOf(ctx); ok && p.Name != "" {
		return p.Name
	}
	return httpx.Subject(ctx)
}

// New returns the API's http.Handler, serving every path the contract declares under /api.
func New(logger *slog.Logger, c Contexts) (http.Handler, error) {
	return oas.NewServer(
		&api{
			terminologyHandler: terminologyweb.New(c.Terminology, actor),
			analysisHandler:    analysisweb.New(c.Analysis),
			companiesHandler:   companiesweb.New(c.Companies),
			identityHandler:    identityweb.New(),
			llmHandler:         llmweb.New(c.LLM, actor),
			promptTestHandler:  prompttestweb.New(c.PromptTest, actor),
			logger:             logger,
		},
		identityweb.NewSecurity(c.Auth),
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
