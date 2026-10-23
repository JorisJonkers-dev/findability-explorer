package persistence_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/seal"
)

var (
	errBoom = errors.New("boom")
	at      = time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
)

func sealer(t *testing.T, secret string) *seal.Sealer {
	t.Helper()
	s, err := seal.NewSealer(secret, persistence.HKDFInfo)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func open(t *testing.T) (*pg.Store, *persistence.Store) {
	t.Helper()
	db, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db, persistence.New(db, sealer(t, strings.Repeat("k", 32)))
}

func signIn(t *testing.T, s *persistence.Store, roles ...string) string {
	t.Helper()
	sid, err := s.SignIn(t.Context(), "https://auth.test", domain.Identity{Subject: "u1", Name: "joris", Email: "j@example.com", Roles: roles},
		"rt-1", strings.Repeat("u", 400), at, at.Add(domain.SessionTTL))
	if err != nil {
		t.Fatal(err)
	}
	return sid
}

func TestSignInAndLoad(t *testing.T) {
	t.Parallel()
	_, s := open(t)
	sid := signIn(t, s, domain.AdminRole)
	if len(sid) != 43 {
		t.Fatalf("session id %q", sid)
	}
	got, err := s.Load(t.Context(), sid, at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	id := got.Account.Identity
	if got.ID != sid || got.Account.Issuer != "https://auth.test" || id.Subject != "u1" || id.Email != "j@example.com" ||
		len(id.Roles) != 1 || len(got.UserAgent) != 300 || !got.RolesCheckedAt.Equal(at) || !got.ExpiresAt.Equal(at.Add(domain.SessionTTL)) {
		t.Fatalf("session = %+v", got)
	}
	if got, _ := s.Load(t.Context(), sid, at.Add(2*time.Hour)); !got.LastSeenAt.Equal(at.Add(time.Hour)) {
		t.Fatalf("last seen = %v", got.LastSeenAt)
	}
	// A second sign-in of the same account updates it and opens another session.
	sid2, err := s.SignIn(t.Context(), "https://auth.test", domain.Identity{Subject: "u1", Name: "Joris"}, "rt-2", "ua", at, at.Add(time.Hour))
	if err != nil || sid2 == sid {
		t.Fatalf("second sign-in = %q, %v", sid2, err)
	}
	if got, _ := s.Load(t.Context(), sid, at); got.Account.Identity.Name != "Joris" || len(got.Account.Identity.Roles) != 0 {
		t.Fatalf("account after the second sign-in = %+v", got.Account)
	}
	if _, err := s.Load(t.Context(), sid2, at.Add(time.Hour)); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("expired session: %v", err)
	}
	if _, err := s.Load(t.Context(), "nobody", at); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("unknown session: %v", err)
	}
}

func TestRefresh(t *testing.T) {
	t.Parallel()
	_, s := open(t)
	sid := signIn(t, s, domain.AdminRole)
	later := at.Add(time.Minute)
	var spent string
	got, err := s.Refresh(t.Context(), sid, at, later, func(_ context.Context, token string) (string, domain.Identity, error) {
		spent = token
		return "rt-2", domain.Identity{Subject: "u1", Name: "joris", Roles: nil}, nil
	})
	if err != nil || spent != "rt-1" || !got.RolesCheckedAt.Equal(later) || len(got.Account.Identity.Roles) != 0 {
		t.Fatalf("refresh = %+v, %v (spent %q)", got, err, spent)
	}
	// Another request already refreshed: nothing is spent.
	called := false
	got, err = s.Refresh(t.Context(), sid, at, later, func(context.Context, string) (string, domain.Identity, error) {
		called = true
		return "", domain.Identity{}, nil
	})
	if err != nil || called || !got.RolesCheckedAt.Equal(later) {
		t.Fatalf("stale refresh = %+v, %v, called %v", got, err, called)
	}
	if _, err := s.Refresh(t.Context(), sid, later, later, func(context.Context, string) (string, domain.Identity, error) {
		return "", domain.Identity{}, errBoom
	}); !errors.Is(err, errBoom) {
		t.Fatalf("failing refresh: %v", err)
	}
	if token, err := s.SignOut(t.Context(), sid); err != nil || token != "rt-2" {
		t.Fatalf("the rotated token was not kept: %q, %v", token, err)
	}
}

func TestRefreshEndsARevokedSession(t *testing.T) {
	t.Parallel()
	_, s := open(t)
	sid := signIn(t, s)
	if _, err := s.Refresh(t.Context(), sid, at, at, func(context.Context, string) (string, domain.Identity, error) {
		return "", domain.Identity{}, domain.ErrSessionGone
	}); !errors.Is(err, domain.ErrSessionGone) {
		t.Fatalf("revoked: %v", err)
	}
	if _, err := s.Load(t.Context(), sid, at); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("the session outlived its revocation: %v", err)
	}
	if _, err := s.Refresh(t.Context(), sid, at, at, nil); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("refresh of a deleted session: %v", err)
	}
}

func TestARotatedSessionKeySignsEveryoneOut(t *testing.T) {
	t.Parallel()
	db, s := open(t)
	sid := signIn(t, s)
	rotated := persistence.New(db, sealer(t, strings.Repeat("r", 32)))
	if _, err := rotated.Refresh(t.Context(), sid, at, at, nil); !errors.Is(err, domain.ErrSessionGone) {
		t.Fatalf("refresh under a new key: %v", err)
	}
	sid = signIn(t, s)
	if token, err := rotated.SignOut(t.Context(), sid); err != nil || token != "" {
		t.Fatalf("sign out under a new key = %q, %v", token, err)
	}
}

func TestConcurrentRefreshesSpendTheTokenOnce(t *testing.T) {
	t.Parallel()
	_, s := open(t)
	sid := signIn(t, s)
	var mu sync.Mutex
	spends := 0
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			_, err := s.Refresh(context.Background(), sid, at, at.Add(time.Second), func(context.Context, string) (string, domain.Identity, error) {
				mu.Lock()
				spends++
				mu.Unlock()
				return "rt-2", domain.Identity{Subject: "u1"}, nil
			})
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if spends != 1 {
		t.Fatalf("the refresh token was spent %d times", spends)
	}
}

func TestSignOutAndPrune(t *testing.T) {
	t.Parallel()
	_, s := open(t)
	sid := signIn(t, s)
	if token, err := s.SignOut(t.Context(), sid); err != nil || token != "rt-1" {
		t.Fatalf("sign out = %q, %v", token, err)
	}
	if _, err := s.SignOut(t.Context(), sid); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("second sign out: %v", err)
	}
	sid = signIn(t, s)
	if err := s.Prune(t.Context(), at.Add(domain.SessionTTL)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SignOut(t.Context(), sid); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("an expired session survived pruning: %v", err)
	}
}

func TestDatabaseFailures(t *testing.T) {
	t.Parallel()
	db, s := open(t)
	sid := signIn(t, s)
	db.Close()
	ctx := t.Context()
	if _, err := s.SignIn(ctx, "i", domain.Identity{Subject: "u"}, "rt", "", at, at); err == nil {
		t.Error("SignIn on a closed pool")
	}
	if _, err := s.Load(ctx, sid, at); err == nil || errors.Is(err, domain.ErrNoSession) {
		t.Errorf("Load on a closed pool: %v", err)
	}
	if _, err := s.Refresh(ctx, sid, at, at, nil); err == nil {
		t.Error("Refresh on a closed pool")
	}
	if _, err := s.SignOut(ctx, sid); err == nil || errors.Is(err, domain.ErrNoSession) {
		t.Errorf("SignOut on a closed pool: %v", err)
	}
	if err := s.Prune(ctx, at); err == nil {
		t.Error("Prune on a closed pool")
	}
}
