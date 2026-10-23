// Package oidctest is a minimal OpenID Connect provider for tests, behaving like auth-api:
// discovery, keys, an authorization code flow that checks PKCE, refresh tokens that rotate on
// every use (reuseRefreshTokens=false), and token revocation.
package oidctest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// Code is the one authorization code the issuer accepts.
const Code = "good-code"

// Issuer is the fake provider.
type Issuer struct {
	URL      string
	ClientID string
	srv      *httptest.Server
	key      *rsa.PrivateKey

	mu        sync.Mutex
	claims    map[string]any
	challenge string
	nonce     string
	refresh   string
	rotations int
	refreshes int
	down      bool
	revoked   []string
}

// New starts an issuer for clientID; it stops with the test.
func New(t testing.TB, clientID string) *Issuer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &Issuer{key: key, ClientID: clientID, claims: map[string]any{}}
	mux := http.NewServeMux()
	f.srv = httptest.NewServer(mux)
	f.URL = f.srv.URL
	t.Cleanup(f.srv.Close)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": f.URL, "authorization_endpoint": f.URL + "/authorize", "token_endpoint": f.URL + "/token",
			"jwks_uri": f.URL + "/jwks", "revocation_endpoint": f.URL + "/revoke",
			"id_token_signing_alg_values_supported": []string{"RS256"}, "response_types_supported": []string{"code"},
			"subject_types_supported": []string{"public"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "k1", Algorithm: "RS256", Use: "sig"}}})
	})
	mux.HandleFunc("/revoke", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.down {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		f.revoked = append(f.revoked, r.PostForm.Get("token"))
	})
	mux.HandleFunc("/token", f.token(t))
	return f
}

func (f *Issuer) token(t testing.TB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		defer f.mu.Unlock()
		invalid := func() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
		}
		claims := map[string]any{
			"iss": f.URL, "aud": f.ClientID, "sub": "user-1", "exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(), "preferred_username": "joris", "email": "joris@example.com",
		}
		switch r.PostForm.Get("grant_type") {
		case "refresh_token":
			f.refreshes++
			if f.down {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			if r.PostForm.Get("refresh_token") != f.refresh {
				invalid()
				return
			}
		default:
			sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
			if r.PostForm.Get("code") != Code || base64.RawURLEncoding.EncodeToString(sum[:]) != f.challenge {
				invalid()
				return
			}
			claims["nonce"] = f.nonce
		}
		maps.Copy(claims, f.claims)
		f.rotations++
		f.refresh = fmt.Sprintf("rt-%d", f.rotations)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "at", "token_type": "Bearer", "expires_in": 900, "refresh_token": f.refresh, "id_token": f.sign(t, claims),
		})
	}
}

func (f *Issuer) sign(t testing.TB, claims map[string]any) string {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: f.key}, (&jose.SignerOptions{}).WithHeader("kid", "k1").WithType("JWT"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// Authorize records what the authorize endpoint would see at the address a sign-in redirected to,
// and returns the state to send back to the callback.
func (f *Issuer) Authorize(t testing.TB, location string) string {
	t.Helper()
	u, err := url.Parse(location)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Scheme+"://"+u.Host != f.URL || u.Path != "/authorize" || q.Get("code_challenge_method") != "S256" ||
		q.Get("client_id") != f.ClientID || q.Get("nonce") == "" || q.Get("state") == "" {
		t.Fatalf("authorize redirect %s", location)
	}
	f.mu.Lock()
	f.challenge, f.nonce = q.Get("code_challenge"), q.Get("nonce")
	f.mu.Unlock()
	return q.Get("state")
}

// SetClaims makes every later ID token carry claims, at sign-in and at every refresh.
func (f *Issuer) SetClaims(claims map[string]any) {
	f.mu.Lock()
	f.claims = claims
	f.mu.Unlock()
}

// SetNonce replaces the nonce the next sign-in's ID token carries.
func (f *Issuer) SetNonce(nonce string) {
	f.mu.Lock()
	f.nonce = nonce
	f.mu.Unlock()
}

// SetDown makes the token and revocation endpoints fail with 503.
func (f *Issuer) SetDown(down bool) {
	f.mu.Lock()
	f.down = down
	f.mu.Unlock()
}

// RefreshToken is the one refresh token currently valid.
func (f *Issuer) RefreshToken() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refresh
}

// Refreshes counts the refresh grants asked for.
func (f *Issuer) Refreshes() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.refreshes
}

// Revoked reports whether token was revoked.
func (f *Issuer) Revoked(token string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.revoked {
		if r == token {
			return true
		}
	}
	return false
}
