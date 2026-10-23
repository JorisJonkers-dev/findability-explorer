package web_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/adapters/web"
	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/seal"
)

var (
	errBoom = errors.New("boom")
	now     = time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
)

func clock() time.Time { return now }

// cases stands in for the identity use cases: one session "sid-1" of an admin.
type cases struct {
	signInErr  error
	signOutErr error
	signedIn   domain.Identity
	signedOut  string
	err        error
}

func (c *cases) SignIn(_ context.Context, id domain.Identity, refreshToken, _ string) (string, error) {
	c.signedIn = id
	if refreshToken == "" {
		return "", errors.New("no refresh token")
	}
	return "sid-1", c.signInErr
}

func (c *cases) Principal(_ context.Context, sid string, need domain.Need) (domain.Principal, error) {
	if c.err != nil {
		return domain.Principal{}, c.err
	}
	if sid != "sid-1" {
		return domain.Principal{}, domain.ErrNoSession
	}
	return need.Authorise(domain.Principal{Subject: "u1", Name: "joris", Roles: []string{domain.AdminRole}})
}

func (c *cases) SignOut(_ context.Context, sid string) error {
	c.signedOut = sid
	return c.signOutErr
}

// flow stands in for auth-api.
type flow struct {
	err      error
	verifier string
	nonce    string
	token    string
}

func (f *flow) AuthCodeURL(state, nonce, verifier string) string {
	f.nonce, f.verifier = nonce, verifier
	return "https://auth.test/authorize?" + url.Values{"state": {state}}.Encode()
}

func (f *flow) Exchange(_ context.Context, code, verifier, nonce string) (domain.Identity, string, error) {
	if f.err != nil {
		return domain.Identity{}, "", f.err
	}
	if code != "good" || verifier != f.verifier || nonce != f.nonce {
		return domain.Identity{}, "", errors.New("mismatch")
	}
	return domain.Identity{Subject: "u1", Roles: []string{domain.AdminRole}}, f.token, nil
}

func codec(t *testing.T) *seal.Codec {
	t.Helper()
	c, err := seal.NewCodec(strings.Repeat("k", 32), web.CodecInfo)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func sessionCookie(t *testing.T, c *seal.Codec, sid string) *http.Cookie {
	t.Helper()
	v, err := c.Seal(web.SessionCookie, sid, now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: web.SessionCookie, Value: v} //nolint:gosec // a request cookie
}

func do(h http.Handler, method, target string, headers map[string]string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func cookieOf(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func logger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func TestSessionsAuthenticate(t *testing.T) {
	c := codec(t)
	s := web.NewSessions(&cases{}, c, clock)
	if _, err := s.Authenticate(t.Context(), "", domain.NeedView); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("no cookie: %v", err)
	}
	if _, err := s.Authenticate(t.Context(), "forged", domain.NeedView); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("forged cookie: %v", err)
	}
	if p, err := s.Authenticate(t.Context(), sessionCookie(t, c, "sid-1").Value, domain.NeedAdminChange); err != nil || p.Subject != "u1" {
		t.Fatalf("session = %+v, %v", p, err)
	}
}

func TestDev(t *testing.T) {
	user := web.Dev{Principal: domain.Principal{Subject: "dev"}}
	if p, err := user.Authenticate(t.Context(), "", domain.NeedView); err != nil || p.Subject != "dev" {
		t.Fatalf("dev = %+v, %v", p, err)
	}
	if _, err := user.Authenticate(t.Context(), "", domain.NeedAdminView); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("dev without DEV_ROLES as admin: %v", err)
	}
	var seen string
	h := web.WithDevCookie(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		c, _ := r.Cookie(web.SessionCookie)
		seen = c.Value
	}))
	do(h, http.MethodGet, "/api/v1/me", nil)
	if seen != "dev" {
		t.Fatalf("placeholder cookie = %q", seen)
	}
	do(h, http.MethodGet, "/api/v1/me", nil, &http.Cookie{Name: web.SessionCookie, Value: "real"}) //nolint:gosec // a request cookie
	if seen != "real" {
		t.Fatalf("an existing cookie became %q", seen)
	}
}

func TestSecurityAndGetMe(t *testing.T) {
	sec := web.NewSecurity(web.Dev{Principal: domain.Principal{Subject: "dev", Name: "Dev", Email: "d@example.com"}})
	ctx, err := sec.HandleSessionCookie(t.Context(), oas.GetMeOperation, oas.SessionCookie{APIKey: "x"})
	if err != nil || httpx.Subject(ctx) != "dev" {
		t.Fatalf("security = %v, subject %q", err, httpx.Subject(ctx))
	}
	res, err := web.New().GetMe(ctx)
	if me, ok := res.(*oas.Me); err != nil || !ok || me.Name != "Dev" || me.Email.Value != "d@example.com" || me.IsAdmin {
		t.Fatalf("me = %+v, %v", res, err)
	}
	for _, op := range []string{oas.GetLlmSettingsOperation, oas.SetActiveLlmOperation, oas.AddLlmCredentialOperation} {
		if _, err := sec.HandleSessionCookie(t.Context(), op, oas.SessionCookie{APIKey: "x"}); !errors.Is(err, domain.ErrForbidden) {
			t.Errorf("%s without the admin role: %v", op, err)
		}
	}
	if _, err := web.New().GetMe(t.Context()); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("me without a principal: %v", err)
	}
	if _, ok := web.PrincipalOf(t.Context()); ok {
		t.Fatal("a principal from nowhere")
	}
	plain, _ := web.NewSecurity(web.Dev{Principal: domain.Principal{Subject: "dev"}}).HandleSessionCookie(t.Context(), oas.GetMeOperation, oas.SessionCookie{})
	if res, _ := web.New().GetMe(plain); res.(*oas.Me).Email.Set {
		t.Fatal("an email from nowhere")
	}
}

func TestStatusAndDetail(t *testing.T) {
	cases := map[error]int{
		domain.ErrNoSession: http.StatusUnauthorized,
		domain.ErrForbidden: http.StatusForbidden,
		errors.Join(domain.ErrRolesUnconfirmed, errBoom): http.StatusServiceUnavailable,
		errBoom: http.StatusInternalServerError,
	}
	for err, want := range cases {
		if got := web.Status(err); got != want {
			t.Errorf("Status(%v) = %d, want %d", err, got, want)
		}
	}
	if got := web.Detail(errors.New(domain.ErrRolesUnconfirmed.Error() + ": dial tcp: refused")); got != "Auth-api could not confirm the account's roles; try again shortly." {
		t.Fatalf("detail = %q", got)
	}
}

func TestGate(t *testing.T) {
	c := codec(t)
	uc := &cases{}
	h := web.Gate(web.NewSessions(uc, c, clock), logger(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	if rec := do(h, http.MethodGet, "/competitors", nil, sessionCookie(t, c, "sid-1")); rec.Code != http.StatusTeapot {
		t.Fatalf("signed in: %d", rec.Code)
	}
	page := do(h, http.MethodGet, "/competitors/x?tab=scans", map[string]string{"Sec-Fetch-Dest": "document"})
	if page.Code != http.StatusFound || page.Header().Get("Location") != "/auth/login?return=%2Fcompetitors%2Fx%3Ftab%3Dscans" {
		t.Fatalf("page without a session: %d %q", page.Code, page.Header().Get("Location"))
	}
	if rec := do(h, http.MethodGet, "/", map[string]string{"Accept": "text/html,*/*"}); rec.Code != http.StatusFound {
		t.Fatalf("an old browser's page: %d", rec.Code)
	}
	for name, req := range map[string]struct {
		method  string
		headers map[string]string
	}{
		"asset":       {http.MethodGet, map[string]string{"Sec-Fetch-Dest": "script", "Accept": "text/html"}},
		"data":        {http.MethodGet, map[string]string{"Accept": "application/json"}},
		"form submit": {http.MethodPost, map[string]string{"Sec-Fetch-Dest": "document"}},
	} {
		if rec := do(h, req.method, "/assets/app.js", req.headers); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s without a session: %d", name, rec.Code)
		}
	}
	if rec := do(h, http.MethodGet, "/", map[string]string{"Sec-Fetch-Dest": "document"}, sessionCookie(t, c, "signed-out")); rec.Code != http.StatusFound {
		t.Fatalf("a signed-out session's page: %d", rec.Code)
	}
	uc.err = domain.ErrRolesUnconfirmed
	if rec := do(h, http.MethodGet, "/", nil, sessionCookie(t, c, "sid-1")); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("auth-api down: %d", rec.Code)
	}
	uc.err = errBoom
	if rec := do(h, http.MethodGet, "/", nil, sessionCookie(t, c, "sid-1")); rec.Code != http.StatusInternalServerError {
		t.Fatalf("database down: %d", rec.Code)
	}
}

func routes(t *testing.T, uc *cases, f *flow, secure bool) (http.Handler, *seal.Codec) {
	t.Helper()
	c := codec(t)
	return web.NewRoutes(f, uc, c, secure, clock, logger()).Handler(), c
}

// login starts a sign-in and returns the state auth-api would send back, and the flow cookie.
func login(t *testing.T, h http.Handler, ret string) (string, *http.Cookie) {
	t.Helper()
	rec := do(h, http.MethodGet, "/auth/login?return="+url.QueryEscape(ret), nil)
	loc, _ := url.Parse(rec.Header().Get("Location"))
	flowCookie := cookieOf(rec, "fe_oidc")
	if rec.Code != http.StatusFound || loc.Host != "auth.test" || flowCookie == nil || !flowCookie.HttpOnly ||
		flowCookie.SameSite != http.SameSiteLaxMode || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("login = %d %q %+v", rec.Code, loc, flowCookie)
	}
	return loc.Query().Get("state"), flowCookie
}

func TestSignInRoundTrip(t *testing.T) {
	uc := &cases{}
	h, c := routes(t, uc, &flow{token: "rt"}, true)
	state, flowCookie := login(t, h, "/competitors?x=1")
	if !flowCookie.Secure {
		t.Fatal("the flow cookie is not Secure")
	}
	rec := do(h, http.MethodGet, "/auth/callback?code=good&state="+state, nil, flowCookie)
	sess := cookieOf(rec, web.SessionCookie)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/competitors?x=1" || sess == nil || !sess.Secure ||
		!sess.HttpOnly || sess.Path != "/" || sess.MaxAge != int(domain.SessionTTL.Seconds()) || uc.signedIn.Subject != "u1" {
		t.Fatalf("callback = %d %q %+v", rec.Code, rec.Header().Get("Location"), sess)
	}
	if cleared := cookieOf(rec, "fe_oidc"); cleared == nil || cleared.MaxAge >= 0 {
		t.Fatalf("the flow cookie was not cleared: %+v", cleared)
	}
	var sid string
	if err := c.Open(web.SessionCookie, sess.Value, now, &sid); err != nil || sid != "sid-1" {
		t.Fatalf("session cookie holds %q, %v", sid, err)
	}
	// An unsafe return path goes home.
	state, flowCookie = login(t, h, "https://evil.test/")
	if rec := do(h, http.MethodGet, "/auth/callback?code=good&state="+state, nil, flowCookie); rec.Header().Get("Location") != "/" {
		t.Fatalf("unsafe return = %q", rec.Header().Get("Location"))
	}
	state, flowCookie = login(t, h, "/"+strings.Repeat("a", 2001))
	if rec := do(h, http.MethodGet, "/auth/callback?code=good&state="+state, nil, flowCookie); rec.Header().Get("Location") != "/" {
		t.Fatalf("overlong return = %q", rec.Header().Get("Location"))
	}
}

func TestCallbackRefusals(t *testing.T) {
	uc := &cases{}
	f := &flow{token: "rt"}
	h, _ := routes(t, uc, f, false)
	state, flowCookie := login(t, h, "/")
	if flowCookie.Secure {
		t.Fatal("a plain-http run set a Secure cookie")
	}
	forged := &http.Cookie{Name: "fe_oidc", Value: "forged"} //nolint:gosec // a request cookie
	cases := []struct {
		name   string
		query  string
		cookie *http.Cookie
		setup  func()
		want   int
	}{
		{"no flow cookie", "code=good&state=" + state, nil, nil, http.StatusBadRequest},
		{"forged flow cookie", "code=good&state=" + state, forged, nil, http.StatusBadRequest},
		{"refused at auth-api", "error=access_denied&error_description=no", flowCookie, nil, http.StatusForbidden},
		{"wrong state", "code=good&state=other", flowCookie, nil, http.StatusBadRequest},
		{"no state", "code=good", flowCookie, nil, http.StatusBadRequest},
		{"exchange fails", "code=good&state=" + state, flowCookie, func() { f.err = errBoom }, http.StatusBadGateway},
		{"no refresh token", "code=good&state=" + state, flowCookie, func() { f.err, f.token = nil, "" }, http.StatusInternalServerError},
		{"session store fails", "code=good&state=" + state, flowCookie, func() { f.token, uc.signInErr = "rt", errBoom }, http.StatusInternalServerError},
	}
	for _, c := range cases {
		if c.setup != nil {
			c.setup()
		}
		var cookies []*http.Cookie
		if c.cookie != nil {
			cookies = append(cookies, c.cookie)
		}
		rec := do(h, http.MethodGet, "/auth/callback?"+c.query, nil, cookies...)
		if rec.Code != c.want || !strings.Contains(rec.Body.String(), "Sign-in failed") || cookieOf(rec, web.SessionCookie) != nil {
			t.Errorf("%s: %d %q", c.name, rec.Code, rec.Body.String())
		}
	}
}

func TestLogout(t *testing.T) {
	uc := &cases{}
	h, c := routes(t, uc, &flow{}, true)
	rec := do(h, http.MethodPost, "/auth/logout", nil, sessionCookie(t, c, "sid-1"))
	cleared := cookieOf(rec, web.SessionCookie)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != web.SignedOutPath || uc.signedOut != "sid-1" ||
		cleared == nil || cleared.MaxAge >= 0 {
		t.Fatalf("logout = %d %q, signed out %q, cookie %+v", rec.Code, rec.Header().Get("Location"), uc.signedOut, cleared)
	}
	uc.signOutErr, uc.signedOut = errBoom, ""
	if rec := do(h, http.MethodPost, "/auth/logout", nil, sessionCookie(t, c, "sid-1")); rec.Code != http.StatusSeeOther {
		t.Fatalf("logout with a failing store: %d", rec.Code)
	}
	uc.signedOut = ""
	forged := &http.Cookie{Name: web.SessionCookie, Value: "forged"} //nolint:gosec // a request cookie
	for _, cookies := range [][]*http.Cookie{nil, {forged}} {
		if rec := do(h, http.MethodPost, "/auth/logout", nil, cookies...); rec.Code != http.StatusSeeOther || uc.signedOut != "" {
			t.Fatalf("logout without a session: %d, signed out %q", rec.Code, uc.signedOut)
		}
	}
	if rec := do(h, http.MethodGet, "/auth/logout", nil); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET logout: %d", rec.Code)
	}
	page := do(h, http.MethodGet, web.SignedOutPath, nil)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Signed out") || !strings.Contains(page.Body.String(), `href="/auth/login"`) {
		t.Fatalf("signed-out page = %d %s", page.Code, page.Body)
	}
}

func TestDevRoutes(t *testing.T) {
	h := web.DevRoutes()
	if rec := do(h, http.MethodGet, "/auth/login?return=%2Fscan", nil); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/scan" {
		t.Fatalf("dev login = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if rec := do(h, http.MethodPost, "/auth/logout", nil); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != web.SignedOutPath {
		t.Fatalf("dev logout = %d", rec.Code)
	}
	if rec := do(h, http.MethodGet, web.SignedOutPath, nil); rec.Code != http.StatusOK {
		t.Fatalf("dev signed-out page = %d", rec.Code)
	}
	if rec := do(h, http.MethodGet, "/auth/callback", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("dev callback = %d", rec.Code)
	}
}

func TestLoginPath(t *testing.T) {
	if got := web.LoginPath("/a?b=c"); got != "/auth/login?return=%2Fa%3Fb%3Dc" {
		t.Fatalf("LoginPath = %q", got)
	}
}
