package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/adapters/oidc/oidctest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
)

// browser sends requests the way a browser on the app's origin would, keeping the cookies.
type browser struct {
	t       *testing.T
	base    string
	cookies map[string]*http.Cookie
}

// answer is what a test reads of a response.
type answer struct {
	StatusCode int
	Location   string
}

func (b *browser) do(method, target string, headers map[string]string) answer {
	b.t.Helper()
	if strings.HasPrefix(target, "/") {
		target = b.base + target
	}
	req, err := http.NewRequestWithContext(b.t.Context(), method, target, nil)
	if err != nil {
		b.t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, c := range b.cookies {
		req.AddCookie(c)
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	for _, c := range resp.Cookies() {
		if c.MaxAge < 0 {
			delete(b.cookies, c.Name)
		} else {
			b.cookies[c.Name] = &http.Cookie{Name: c.Name, Value: c.Value} //nolint:gosec // a request cookie
		}
	}
	return answer{StatusCode: resp.StatusCode, Location: resp.Header.Get("Location")}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

func TestSignInEndToEnd(t *testing.T) {
	issuer := oidctest.New(t, "findability-explorer")
	addr := freeAddr(t)
	base := "http://" + addr
	vars := map[string]string{
		"DATABASE_URL": pgtest.URL(t), "ADDR": addr, "SESSION_KEY": sessionKeyValue, "OIDC_ISSUER": issuer.URL,
		"OIDC_CLIENT_ID": "findability-explorer", "OIDC_CLIENT_SECRET": "secret", "OIDC_REDIRECT_URL": base + "/auth/callback",
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- run(ctx, slog.New(slog.DiscardHandler), env(vars)) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("run: %v", err)
		}
	})
	b := &browser{t: t, base: base, cookies: map[string]*http.Cookie{}}
	for range 100 {
		if conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", addr); err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	page := map[string]string{"Sec-Fetch-Dest": "document"}
	if r := b.do(http.MethodGet, "/competitors", page); r.StatusCode != http.StatusFound || r.Location != "/auth/login?return=%2Fcompetitors" {
		t.Fatalf("a page without a session: %d %q", r.StatusCode, r.Location)
	}
	if r := b.do(http.MethodGet, "/api/v1/me", nil); r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("the API without a session: %d", r.StatusCode)
	}
	login := b.do(http.MethodGet, "/auth/login?return=%2Fcompetitors", nil)
	state := issuer.Authorize(t, login.Location)
	callback := b.do(http.MethodGet, "/auth/callback?"+url.Values{"code": {oidctest.Code}, "state": {state}}.Encode(), nil)
	if callback.StatusCode != http.StatusFound || callback.Location != "/competitors" || b.cookies["fe_session"] == nil {
		t.Fatalf("callback: %d %q", callback.StatusCode, callback.Location)
	}
	if r := b.do(http.MethodGet, "/api/v1/me", nil); r.StatusCode != http.StatusOK {
		t.Fatalf("me: %d", r.StatusCode)
	}
	if r := b.do(http.MethodGet, "/api/v1/admin/llm", nil); r.StatusCode != http.StatusForbidden {
		t.Fatalf("admin settings as a user: %d", r.StatusCode)
	}
	// auth-api grants the admin role; a change re-reads the roles at once.
	issuer.SetClaims(map[string]any{"roles": []string{domain.AdminRole}})
	same := map[string]string{"Sec-Fetch-Site": "same-origin"}
	if r := b.do(http.MethodDelete, "/api/v1/admin/llm/active", same); r.StatusCode != http.StatusOK {
		t.Fatalf("an admin change after the grant: %d", r.StatusCode)
	}
	if r := b.do(http.MethodDelete, "/api/v1/admin/llm/active", map[string]string{"Sec-Fetch-Site": "cross-site"}); r.StatusCode != http.StatusForbidden {
		t.Fatalf("a cross-site change: %d", r.StatusCode)
	}
	token := issuer.RefreshToken()
	if r := b.do(http.MethodPost, "/auth/logout", same); r.StatusCode != http.StatusSeeOther || b.cookies["fe_session"] != nil {
		t.Fatalf("logout: %d", r.StatusCode)
	}
	for range 200 {
		if issuer.Revoked(token) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !issuer.Revoked(token) {
		t.Fatal("signing out did not revoke the refresh token")
	}
}
