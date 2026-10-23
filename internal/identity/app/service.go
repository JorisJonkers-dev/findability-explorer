// Package app holds the identity context's use cases: signing in, resolving a session to the
// principal a request runs as (re-reading the roles from auth-api when they are too old), and
// signing out. They are the only entry points the inbound adapters call.
package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/domain"
)

const (
	// cacheFor is how long a loaded session is reused without asking the store; a page load
	// fetches the app and several API calls within it.
	cacheFor = 3 * time.Second
	// cacheSize is how many sessions the cache holds before it drops the stale ones.
	cacheSize = 1024
	// revokeTimeout bounds the call that tells auth-api a signed-out session's token is done.
	revokeTimeout = 10 * time.Second
)

// ErrNoRefreshToken is returned when auth-api signed someone in without a refresh token, without
// which the roles could never be re-read.
var ErrNoRefreshToken = errors.New("auth-api returned no refresh token")

// RefreshFunc trades a refresh token for a new one and the account's current identity. It
// returns domain.ErrSessionGone when auth-api refuses the token.
type RefreshFunc func(ctx context.Context, refreshToken string) (string, domain.Identity, error)

// Store keeps local accounts and server-side sessions.
type Store interface {
	// SignIn creates or updates the account and opens a session for it, returning its id.
	SignIn(ctx context.Context, issuer string, id domain.Identity, refreshToken, userAgent string, at, expires time.Time) (string, error)
	// Load returns the live session, or domain.ErrNoSession.
	Load(ctx context.Context, sessionID string, now time.Time) (domain.Session, error)
	// Refresh locks the session and, unless another request re-read its roles since seen (the
	// RolesCheckedAt this request loaded), stores what fn returns. fn returning
	// domain.ErrSessionGone deletes the session, and Refresh returns that error.
	Refresh(ctx context.Context, sessionID string, seen, now time.Time, fn RefreshFunc) (domain.Session, error)
	// SignOut deletes the session and returns its refresh token, empty when it cannot be read.
	SignOut(ctx context.Context, sessionID string) (string, error)
	// Prune deletes every expired session.
	Prune(ctx context.Context, now time.Time) error
}

// Issuer is auth-api, as far as a live session needs it.
type Issuer interface {
	// Refresh trades a refresh token for a new one and the account's current identity.
	Refresh(ctx context.Context, refreshToken string) (string, domain.Identity, error)
	// Revoke tells auth-api a refresh token is done.
	Revoke(ctx context.Context, refreshToken string) error
}

// Warn hears about failures a request survives, which only the logs can report.
type Warn func(msg string, err error)

// Service runs the identity use cases.
type Service struct {
	store     Store
	issuer    Issuer
	issuerURL string
	now       func() time.Time
	warn      Warn

	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	session domain.Session
	at      time.Time
}

// New returns a Service for accounts of the issuer at issuerURL.
func New(store Store, issuer Issuer, issuerURL string, now func() time.Time, warn Warn) *Service {
	return &Service{store: store, issuer: issuer, issuerURL: issuerURL, now: now, warn: warn, cache: map[string]cached{}}
}

// SignIn records a sign-in auth-api verified and returns the new session's id.
func (s *Service) SignIn(ctx context.Context, id domain.Identity, refreshToken, userAgent string) (string, error) {
	if refreshToken == "" {
		return "", ErrNoRefreshToken
	}
	now := s.now()
	sid, err := s.store.SignIn(ctx, s.issuerURL, id, refreshToken, userAgent, now, now.Add(domain.SessionTTL))
	if err != nil {
		return "", fmt.Errorf("open session: %w", err)
	}
	return sid, nil
}

// Principal returns who the session runs as, with roles fresh enough for need: it re-reads them
// from auth-api when they are older than need allows. It returns domain.ErrNoSession without a
// live session, domain.ErrForbidden when need asks for an administrator the account is not, and
// domain.ErrRolesUnconfirmed when auth-api cannot be asked and the last known roles will not do.
func (s *Service) Principal(ctx context.Context, sessionID string, need domain.Need) (domain.Principal, error) {
	if sessionID == "" {
		return domain.Principal{}, domain.ErrNoSession
	}
	sess, err := s.load(ctx, sessionID)
	if err != nil {
		return domain.Principal{}, err
	}
	if age := s.now().Sub(sess.RolesCheckedAt); age >= need.MaxAge() {
		fresh, err := s.store.Refresh(ctx, sessionID, sess.RolesCheckedAt, s.now(), s.issuer.Refresh)
		switch {
		case err == nil:
			s.remember(fresh)
			sess = fresh
		case errors.Is(err, domain.ErrSessionGone), errors.Is(err, domain.ErrNoSession):
			s.forget(sessionID)
			return domain.Principal{}, domain.ErrNoSession
		case need.MayUseStale(age):
			s.warn("re-read roles from auth-api; going on with the last known ones", err)
		default:
			return domain.Principal{}, fmt.Errorf("%w: %w", domain.ErrRolesUnconfirmed, err)
		}
	}
	return need.Authorise(sess.Principal())
}

// SignOut ends the session here and tells auth-api its refresh token is done; the account stays
// signed in to auth-api and every other app.
func (s *Service) SignOut(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	s.forget(sessionID)
	token, err := s.store.SignOut(ctx, sessionID)
	if errors.Is(err, domain.ErrNoSession) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("close session: %w", err)
	}
	if token != "" {
		go s.revoke(context.WithoutCancel(ctx), token)
	}
	return nil
}

func (s *Service) revoke(ctx context.Context, token string) {
	ctx, cancel := context.WithTimeout(ctx, revokeTimeout)
	defer cancel()
	if err := s.issuer.Revoke(ctx, token); err != nil {
		s.warn("revoke a signed-out session's refresh token", err)
	}
}

// Prune deletes every expired session.
func (s *Service) Prune(ctx context.Context) error {
	return s.store.Prune(ctx, s.now())
}

func (s *Service) load(ctx context.Context, id string) (domain.Session, error) {
	s.mu.Lock()
	c, ok := s.cache[id]
	s.mu.Unlock()
	if ok && s.now().Sub(c.at) < cacheFor {
		return c.session, nil
	}
	sess, err := s.store.Load(ctx, id, s.now())
	if err != nil {
		return domain.Session{}, err
	}
	s.remember(sess)
	return sess, nil
}

func (s *Service) remember(sess domain.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if len(s.cache) >= cacheSize {
		for k, c := range s.cache {
			if now.Sub(c.at) >= cacheFor {
				delete(s.cache, k)
			}
		}
	}
	s.cache[sess.ID] = cached{session: sess, at: now}
}

func (s *Service) forget(id string) {
	s.mu.Lock()
	delete(s.cache, id)
	s.mu.Unlock()
}
