package oidc_test

import (
	"errors"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/adapters/oidc"
	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/adapters/oidc/oidctest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/domain"
)

const verifier = "a-verifier-of-at-least-forty-three-characters-long"

func client(t *testing.T, f *oidctest.Issuer) *oidc.Client {
	t.Helper()
	c, err := oidc.New(t.Context(), oidc.Config{
		Issuer: f.URL, ClientID: f.ClientID, ClientSecret: "secret", RedirectURL: "https://app.test/auth/callback",
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.Issuer() != f.URL {
		t.Fatalf("issuer = %q", c.Issuer())
	}
	return c
}

// signIn runs the code flow up to the exchange and returns its outcome.
func signIn(t *testing.T, f *oidctest.Issuer, c *oidc.Client, nonce string) (domain.Identity, string, error) {
	t.Helper()
	f.Authorize(t, c.AuthCodeURL("state-1", nonce, verifier))
	return c.Exchange(t.Context(), oidctest.Code, verifier, nonce)
}

func TestSignInRefreshAndRevoke(t *testing.T) {
	f := oidctest.New(t, "findability-explorer")
	f.SetClaims(map[string]any{"roles": []string{"ROLE_USER", domain.AdminRole}})
	c := client(t, f)

	id, rt, err := signIn(t, f, c, "nonce-1")
	if err != nil {
		t.Fatal(err)
	}
	if id.Subject != "user-1" || id.Name != "joris" || id.Email != "joris@example.com" || len(id.Roles) != 2 || rt != f.RefreshToken() {
		t.Fatalf("identity = %+v, refresh token %q", id, rt)
	}

	f.SetClaims(map[string]any{"preferred_username": "", "name": "Joris Jonkers"})
	next, id, err := c.Refresh(t.Context(), rt)
	if err != nil || next == rt || next != f.RefreshToken() || id.Name != "Joris Jonkers" || len(id.Roles) != 0 {
		t.Fatalf("refresh = %q %+v, %v", next, id, err)
	}
	if _, _, err := c.Refresh(t.Context(), rt); !errors.Is(err, domain.ErrSessionGone) {
		t.Fatalf("a spent refresh token: %v", err)
	}
	f.SetDown(true)
	if _, _, err := c.Refresh(t.Context(), next); err == nil || errors.Is(err, domain.ErrSessionGone) {
		t.Fatalf("auth-api down: %v", err)
	}
	if err := c.Revoke(t.Context(), next); err == nil {
		t.Fatal("revoking while auth-api is down succeeded")
	}
	f.SetDown(false)
	if err := c.Revoke(t.Context(), next); err != nil || !f.Revoked(next) {
		t.Fatalf("revoke: %v", err)
	}
}

func TestSignInRefusals(t *testing.T) {
	f := oidctest.New(t, "findability-explorer")
	c := client(t, f)
	f.Authorize(t, c.AuthCodeURL("s", "n", verifier))
	if _, _, err := c.Exchange(t.Context(), "bad-code", verifier, "n"); err == nil {
		t.Fatal("a bad code was accepted")
	}
	if _, _, err := c.Exchange(t.Context(), oidctest.Code, "another-verifier-of-at-least-forty-three-chars", "n"); err == nil {
		t.Fatal("a wrong PKCE verifier was accepted")
	}
	f.Authorize(t, c.AuthCodeURL("s", "n", verifier))
	f.SetNonce("replayed")
	if _, _, err := c.Exchange(t.Context(), oidctest.Code, verifier, "n"); !errors.Is(err, oidc.ErrNonce) {
		t.Fatalf("nonce mismatch: %v", err)
	}
	f.SetClaims(map[string]any{"aud": "someone-else"})
	if _, _, err := signIn(t, f, c, "n2"); err == nil {
		t.Fatal("an ID token for another client was accepted")
	}
	if _, _, err := c.Refresh(t.Context(), f.RefreshToken()); err == nil {
		t.Fatal("a refreshed ID token for another client was accepted")
	}
}

func TestNewFailsWithoutDiscovery(t *testing.T) {
	if _, err := oidc.New(t.Context(), oidc.Config{Issuer: "http://127.0.0.1:1"}); err == nil {
		t.Fatal("an unreachable issuer was accepted")
	}
}
