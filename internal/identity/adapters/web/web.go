// Package web is the identity context's inbound HTTP adapter: the sign-in routes under /auth/,
// the gate in front of the web app, the generated API's security handler, and getMe.
package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/seal"
)

const (
	// SessionCookie holds the sealed session id; the contract's sessionCookie scheme names it.
	SessionCookie = "fe_session"
	// flowCookie holds the sealed state, nonce and PKCE verifier of a sign-in in progress.
	flowCookie = "fe_oidc"
	flowTTL    = 10 * time.Minute
	// CodecInfo derives the cookie key from SESSION_KEY.
	CodecInfo = "findability-explorer-cookie-v1"
	// SignedOutPath is where signing out lands: a page, not the app, which would sign the
	// browser straight back in through auth-api.
	SignedOutPath = "/auth/signed-out"
	maxReturn     = 2000
)

// UseCases is what the adapter needs from the application layer.
type UseCases interface {
	SignIn(ctx context.Context, id domain.Identity, refreshToken, userAgent string) (string, error)
	Principal(ctx context.Context, sessionID string, need domain.Need) (domain.Principal, error)
	SignOut(ctx context.Context, sessionID string) error
}

// Flow is the sign-in flow at auth-api.
type Flow interface {
	AuthCodeURL(state, nonce, verifier string) string
	Exchange(ctx context.Context, code, verifier, nonce string) (domain.Identity, string, error)
}

// Authenticator resolves the value of the session cookie to the principal a request runs as.
type Authenticator interface {
	Authenticate(ctx context.Context, cookie string, need domain.Need) (domain.Principal, error)
}

// Sessions authenticates with server-side sessions.
type Sessions struct {
	uc    UseCases
	codec *seal.Codec
	now   func() time.Time
}

// NewSessions returns an Authenticator opening session cookies with codec.
func NewSessions(uc UseCases, codec *seal.Codec, now func() time.Time) *Sessions {
	return &Sessions{uc: uc, codec: codec, now: now}
}

// Authenticate opens the cookie and resolves its session.
func (s *Sessions) Authenticate(ctx context.Context, cookie string, need domain.Need) (domain.Principal, error) {
	var id string
	if cookie == "" || s.codec.Open(SessionCookie, cookie, s.now(), &id) != nil {
		return domain.Principal{}, domain.ErrNoSession
	}
	return s.uc.Principal(ctx, id, need)
}

// Dev stands in for a signed-in account on a local run (DEV_USER, DEV_ROLES); never in a deployment.
type Dev struct {
	Principal domain.Principal
}

// Authenticate returns the stand-in, if its roles meet need.
func (d Dev) Authenticate(_ context.Context, _ string, need domain.Need) (domain.Principal, error) {
	return need.Authorise(d.Principal)
}

// WithDevCookie gives every request without a session cookie a placeholder one, so the generated
// API calls the security handler, where Dev answers. Wire it only with Dev.
func WithDevCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie(SessionCookie); err != nil {
			r.AddCookie(&http.Cookie{Name: SessionCookie, Value: "dev"}) //nolint:gosec // a request cookie no browser ever sees
		}
		next.ServeHTTP(w, r)
	})
}

type principalKey struct{}

// WithPrincipal returns ctx carrying p.
func WithPrincipal(ctx context.Context, p domain.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalOf returns the principal the request runs as, if the API authenticated it.
func PrincipalOf(ctx context.Context) (domain.Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(domain.Principal)
	return p, ok
}

// Security is the generated API's security handler.
type Security struct {
	auth Authenticator
}

// NewSecurity returns the security handler authenticating with auth.
func NewSecurity(auth Authenticator) *Security { return &Security{auth: auth} }

var _ oas.SecurityHandler = (*Security)(nil)

// needs are the operations that need more than a signed-in account.
var needs = map[string]domain.Need{ //nolint:gochecknoglobals // a constant table
	oas.GetLlmSettingsOperation:      domain.NeedAdminView,
	oas.ListLlmModelsOperation:       domain.NeedAdminView,
	oas.GetLlmKeyUsageOperation:      domain.NeedAdminView,
	oas.AddLlmCredentialOperation:    domain.NeedAdminChange,
	oas.DeleteLlmCredentialOperation: domain.NeedAdminChange,
	oas.TestLlmCredentialOperation:   domain.NeedAdminChange,
	oas.SetActiveLlmOperation:        domain.NeedAdminChange,
	oas.ClearActiveLlmOperation:      domain.NeedAdminChange,
	// Scans and companies: everyone signed in reads them, only administrators change them.
	oas.StartAnalysisOperation:       domain.NeedAdminChange,
	oas.StartRescansOperation:        domain.NeedAdminChange,
	oas.StartReadinessCheckOperation: domain.NeedAdminChange,
	oas.AddCompetitorOperation:       domain.NeedAdminChange,
	oas.AddCompetitorsOperation:      domain.NeedAdminChange,
	oas.UpdateCompetitorOperation:    domain.NeedAdminChange,
	oas.DeleteCompetitorOperation:    domain.NeedAdminChange,
	// The Prompt test: everyone signed in reads the results, only administrators run and set it.
	oas.StartPromptRunOperation:             domain.NeedAdminChange,
	oas.RetryPromptRunOperation:             domain.NeedAdminChange,
	oas.GetPromptTestSettingsOperation:      domain.NeedAdminView,
	oas.UpdatePromptTestSettingsOperation:   domain.NeedAdminChange,
	oas.ListCompetitorCandidatesOperation:   domain.NeedAdminView,
	oas.AddCompetitorCandidateOperation:     domain.NeedAdminChange,
	oas.DismissCompetitorCandidateOperation: domain.NeedAdminChange,
	oas.UndoCompetitorCandidateOperation:    domain.NeedAdminChange,
	oas.DecideCompetitorCandidatesOperation: domain.NeedAdminChange,
	oas.SplitCompetitorCandidateOperation:   domain.NeedAdminChange,
	oas.StartCandidateReviewOperation:       domain.NeedAdminChange,
	oas.ListCompetitorChecksOperation:       domain.NeedAdminView,
	oas.StartCompetitorCheckOperation:       domain.NeedAdminChange,
	// The plan: everyone signed in reads it, only administrators tick it off and set its targets.
	oas.TickImprovementOperation: domain.NeedAdminChange,
	oas.SetPlanTargetsOperation:  domain.NeedAdminChange,
	// The review queue of the terminology map.
	oas.ListTermCandidatesOperation:  domain.NeedAdminView,
	oas.AcceptTermCandidateOperation: domain.NeedAdminChange,
	oas.RejectTermCandidateOperation: domain.NeedAdminChange,
	oas.UndoTermDecisionOperation:    domain.NeedAdminChange,
	oas.StartTermReviewOperation:     domain.NeedAdminChange,
}

// HandleSessionCookie resolves the session and puts the principal, and its subject, on ctx. The
// subject feeds the per-user limit on analyses.
func (s *Security) HandleSessionCookie(ctx context.Context, op oas.OperationName, t oas.SessionCookie) (context.Context, error) {
	p, err := s.auth.Authenticate(ctx, t.APIKey, needs[op])
	if err != nil {
		return nil, err
	}
	return httpx.WithSubject(WithPrincipal(ctx, p), p.Subject), nil
}

// Handler implements getMe.
type Handler struct{}

// New returns a Handler.
func New() *Handler { return &Handler{} }

// GetMe implements getMe.
func (*Handler) GetMe(ctx context.Context) (oas.GetMeRes, error) {
	p, ok := PrincipalOf(ctx)
	if !ok {
		return nil, domain.ErrNoSession
	}
	me := &oas.Me{Subject: p.Subject, Name: p.Name, IsAdmin: p.IsAdmin()}
	if p.Email != "" {
		me.Email = oas.NewOptString(p.Email)
	}
	return me, nil
}

// Status is the HTTP status for an error Authenticate returned.
func Status(err error) int {
	switch {
	case errors.Is(err, domain.ErrNoSession):
		return http.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, domain.ErrRolesUnconfirmed):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Gate lets a request for the web app through only with a live session. Without one, a browser
// asking for a page is sent to sign in and back; anything else gets 401.
func Gate(auth Authenticator, logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie := ""
		if c, err := r.Cookie(SessionCookie); err == nil {
			cookie = c.Value
		}
		_, err := auth.Authenticate(r.Context(), cookie, domain.NeedView)
		switch status := Status(err); {
		case err == nil:
			next.ServeHTTP(w, r)
		case status == http.StatusUnauthorized && wantsPage(r):
			http.Redirect(w, r, LoginPath(r.URL.RequestURI()), http.StatusFound)
		case status == http.StatusInternalServerError:
			logger.ErrorContext(r.Context(), "resolve session", "error", err)
			httpx.WriteProblem(w, status, "")
		default:
			httpx.WriteProblem(w, status, Detail(err))
		}
	})
}

// LoginPath is the sign-in route that returns to path afterwards.
func LoginPath(path string) string {
	return "/auth/login?" + url.Values{"return": {path}}.Encode()
}

// wantsPage reports whether r is a browser navigating to a page, rather than fetching an asset or data.
func wantsPage(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if dest := r.Header.Get("Sec-Fetch-Dest"); dest != "" {
		return dest == "document"
	}
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}

// Detail is the sentence a caller reads for an error Authenticate returned.
func Detail(err error) string {
	s := err.Error()
	if i := strings.Index(s, ": "); i > 0 {
		s = s[:i]
	}
	return strings.ToUpper(s[:1]) + s[1:] + "."
}

// Routes serves /auth/login, /auth/callback, /auth/logout and the signed-out page.
type Routes struct {
	flow   Flow
	uc     UseCases
	codec  *seal.Codec
	secure bool
	now    func() time.Time
	logger *slog.Logger
}

// NewRoutes returns the sign-in routes. secure marks cookies Secure; only a local run on plain
// http goes without.
func NewRoutes(flow Flow, uc UseCases, codec *seal.Codec, secure bool, now func() time.Time, logger *slog.Logger) *Routes {
	return &Routes{flow: flow, uc: uc, codec: codec, secure: secure, now: now, logger: logger}
}

// Handler returns the routes.
func (rt *Routes) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/login", rt.login)
	mux.HandleFunc("GET /auth/callback", rt.callback)
	mux.HandleFunc("POST /auth/logout", rt.logout)
	mux.HandleFunc("GET "+SignedOutPath, signedOut)
	return mux
}

type flow struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	Return   string `json:"r"`
}

func returnOf(r *http.Request) string {
	ret := r.URL.Query().Get("return")
	if len(ret) > maxReturn {
		return "/"
	}
	return domain.SafeReturn(ret)
}

func (rt *Routes) login(w http.ResponseWriter, r *http.Request) {
	f := flow{State: random(24), Nonce: random(24), Verifier: random(32), Return: returnOf(r)}
	sealed, err := rt.codec.Seal(flowCookie, f, rt.now(), flowTTL)
	if err != nil {
		rt.fail(w, r, http.StatusInternalServerError, "seal the sign-in cookie", err)
		return
	}
	rt.setCookie(w, flowCookie, sealed, flowTTL)
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, rt.flow.AuthCodeURL(f.State, f.Nonce, f.Verifier), http.StatusFound)
}

func (rt *Routes) callback(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(flowCookie)
	if err != nil {
		rt.fail(w, r, http.StatusBadRequest, "callback without a sign-in cookie", err)
		return
	}
	rt.clearCookie(w, flowCookie)
	var f flow
	if err := rt.codec.Open(flowCookie, c.Value, rt.now(), &f); err != nil {
		rt.fail(w, r, http.StatusBadRequest, "open the sign-in cookie", err)
		return
	}
	q := r.URL.Query()
	if e := q.Get("error"); e != "" {
		rt.fail(w, r, http.StatusForbidden, "auth-api refused the sign-in", fmt.Errorf("%s: %s", e, q.Get("error_description")))
		return
	}
	if q.Get("state") == "" || q.Get("state") != f.State {
		rt.fail(w, r, http.StatusBadRequest, "state mismatch", errors.New("the state is not the one sent"))
		return
	}
	id, refreshToken, err := rt.flow.Exchange(r.Context(), q.Get("code"), f.Verifier, f.Nonce)
	if err != nil {
		rt.fail(w, r, http.StatusBadGateway, "complete the sign-in", err)
		return
	}
	sid, err := rt.uc.SignIn(r.Context(), id, refreshToken, r.UserAgent())
	if err != nil {
		rt.fail(w, r, http.StatusInternalServerError, "open a session", err)
		return
	}
	sealed, err := rt.codec.Seal(SessionCookie, sid, rt.now(), domain.SessionTTL)
	if err != nil {
		rt.fail(w, r, http.StatusInternalServerError, "seal the session cookie", err)
		return
	}
	rt.setCookie(w, SessionCookie, sealed, domain.SessionTTL)
	rt.logger.InfoContext(r.Context(), "signed in", "subject", id.Subject, "admin", domain.Principal{Roles: id.Roles}.IsAdmin())
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, f.Return, http.StatusFound)
}

// logout ends this service's session only; the auth-api session stays.
func (rt *Routes) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(SessionCookie); err == nil {
		var id string
		if rt.codec.Open(SessionCookie, c.Value, rt.now(), &id) == nil {
			if err := rt.uc.SignOut(r.Context(), id); err != nil {
				rt.logger.WarnContext(r.Context(), "sign out", "error", err)
			}
		}
	}
	rt.clearCookie(w, SessionCookie)
	http.Redirect(w, r, SignedOutPath, http.StatusSeeOther)
}

func (rt *Routes) fail(w http.ResponseWriter, r *http.Request, status int, what string, err error) {
	rt.logger.WarnContext(r.Context(), "sign-in: "+what, "error", err)
	page(w, status, "Sign-in failed", "Signing in did not work. Try again.", "/auth/login", "Sign in")
}

func signedOut(w http.ResponseWriter, _ *http.Request) {
	page(w, http.StatusOK, "Signed out", "You are signed out of the Findability Explorer. You stay signed in to your account elsewhere.", "/auth/login", "Sign in again")
}

func page(w http.ResponseWriter, status int, title, msg, link, linkText string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, "<!doctype html><html lang=en><meta charset=utf-8><meta name=viewport content=\"width=device-width\">"+
		"<meta name=robots content=noindex><title>%s · Findability Explorer</title><main><h1>%s</h1><p>%s</p><p><a href=\"%s\">%s</a></p></main></html>",
		html.EscapeString(title), html.EscapeString(title), html.EscapeString(msg), html.EscapeString(link), html.EscapeString(linkText))
}

func (rt *Routes) setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure is off only on a local run over plain http
		Name: name, Value: value, Path: "/", MaxAge: int(ttl.Seconds()),
		Secure: rt.secure, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

func (rt *Routes) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // Secure is off only on a local run over plain http
		Name: name, Value: "", Path: "/", MaxAge: -1, Secure: rt.secure, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

// DevRoutes are the sign-in routes on a local run with DEV_USER: signing in returns at once.
func DevRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/login", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, returnOf(r), http.StatusFound) //nolint:gosec // returnOf only returns paths on this site
	})
	mux.HandleFunc("POST /auth/logout", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, SignedOutPath, http.StatusSeeOther)
	})
	mux.HandleFunc("GET "+SignedOutPath, signedOut)
	return mux
}

func random(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b) // crypto/rand.Read never fails
	return base64.RawURLEncoding.EncodeToString(b)
}
