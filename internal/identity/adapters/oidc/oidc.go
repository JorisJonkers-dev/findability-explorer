// Package oidc is the identity context's client of auth-api: the authorization code flow with
// PKCE, state and nonce; trading a refresh token for the account's current roles; and revoking a
// refresh token at sign-out (docs/adr/0001).
package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/domain"
)

const httpTimeout = 10 * time.Second

// Config is the client's registration at auth-api.
type Config struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// Client talks to auth-api.
type Client struct {
	oauth    oauth2.Config
	verifier *gooidc.IDTokenVerifier
	issuer   string
	revoke   string
	http     *http.Client
}

var _ app.Issuer = (*Client)(nil)

// New discovers the issuer's endpoints and keys.
func New(ctx context.Context, cfg Config) (*Client, error) {
	hc := &http.Client{Timeout: httpTimeout}
	provider, err := gooidc.NewProvider(gooidc.ClientContext(ctx, hc), cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("discover %s: %w", cfg.Issuer, err)
	}
	var meta struct {
		Revocation string `json:"revocation_endpoint"`
	}
	if err := provider.Claims(&meta); err != nil {
		return nil, fmt.Errorf("read %s's discovery document: %w", cfg.Issuer, err)
	}
	return &Client{
		oauth: oauth2.Config{
			ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, RedirectURL: cfg.RedirectURL,
			Endpoint: provider.Endpoint(), Scopes: []string{gooidc.ScopeOpenID, "profile", "email"},
		},
		verifier: provider.VerifierContext(gooidc.ClientContext(context.WithoutCancel(ctx), hc), &gooidc.Config{ClientID: cfg.ClientID}),
		issuer:   cfg.Issuer,
		revoke:   meta.Revocation,
		http:     hc,
	}, nil
}

// Issuer is the issuer's URL, which keys local accounts with the subject.
func (c *Client) Issuer() string { return c.issuer }

// AuthCodeURL is where to send the browser to sign in.
func (c *Client) AuthCodeURL(state, nonce, verifier string) string {
	return c.oauth.AuthCodeURL(state, gooidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier))
}

// claims are the ID token claims the service reads; auth-api puts the roles in "roles".
type claims struct {
	Subject           string   `json:"sub"`
	Nonce             string   `json:"nonce"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	Email             string   `json:"email"`
	Roles             []string `json:"roles"`
}

func (c claims) identity() domain.Identity {
	name := c.PreferredUsername
	if name == "" {
		name = c.Name
	}
	return domain.Identity{Subject: c.Subject, Name: name, Email: c.Email, Roles: c.Roles}
}

// ErrNonce is returned for an ID token whose nonce is not the one the sign-in sent.
var ErrNonce = errors.New("the ID token's nonce does not match")

// Exchange trades the callback's code for tokens, verifies the ID token and its nonce, and
// returns the identity and the refresh token.
func (c *Client) Exchange(ctx context.Context, code, verifier, nonce string) (domain.Identity, string, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.http)
	tok, err := c.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return domain.Identity{}, "", fmt.Errorf("exchange the code: %w", err)
	}
	cl, err := c.verify(ctx, tok)
	if err != nil {
		return domain.Identity{}, "", err
	}
	if cl.Nonce != nonce {
		return domain.Identity{}, "", ErrNonce
	}
	return cl.identity(), tok.RefreshToken, nil
}

func (c *Client) verify(ctx context.Context, tok *oauth2.Token) (claims, error) {
	raw, _ := tok.Extra("id_token").(string)
	idt, err := c.verifier.Verify(ctx, raw)
	if err != nil {
		return claims{}, fmt.Errorf("verify the ID token: %w", err)
	}
	var cl claims
	if err := idt.Claims(&cl); err != nil {
		return claims{}, fmt.Errorf("read the ID token: %w", err)
	}
	return cl, nil
}

// Refresh trades a refresh token for a new one and the account's current identity. A token
// auth-api refuses is domain.ErrSessionGone.
func (c *Client) Refresh(ctx context.Context, refreshToken string) (string, domain.Identity, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.http)
	tok, err := c.oauth.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken, Expiry: time.Unix(1, 0)}).Token()
	if err != nil {
		var re *oauth2.RetrieveError
		if errors.As(err, &re) && (re.ErrorCode == "invalid_grant" || re.Response != nil && re.Response.StatusCode == http.StatusUnauthorized) {
			return "", domain.Identity{}, domain.ErrSessionGone
		}
		return "", domain.Identity{}, fmt.Errorf("refresh: %w", err)
	}
	cl, err := c.verify(ctx, tok)
	if err != nil {
		return "", domain.Identity{}, err
	}
	next := tok.RefreshToken
	if next == "" {
		next = refreshToken
	}
	return next, cl.identity(), nil
}

// Revoke tells auth-api the refresh token is done. Without a revocation endpoint it does nothing.
func (c *Client) Revoke(ctx context.Context, refreshToken string) error {
	if c.revoke == "" {
		return nil
	}
	form := url.Values{"token": {refreshToken}, "token_type_hint": {"refresh_token"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.revoke, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(url.QueryEscape(c.oauth.ClientID), url.QueryEscape(c.oauth.ClientSecret))
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("revoke: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("revoke: auth-api answered %d", resp.StatusCode)
	}
	return nil
}
