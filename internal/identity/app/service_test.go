package app_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/domain"
)

var errBoom = errors.New("boom")

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }

func (c *clock) add(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

// store is an in-memory Store; failOn names the methods that fail.
type store struct {
	mu       sync.Mutex
	clock    *clock
	sessions map[string]*entry
	failOn   map[string]bool
	loads    int
	next     int
	issuers  []string
}

type entry struct {
	session domain.Session
	token   string
}

func newStore(c *clock) *store {
	return &store{clock: c, sessions: map[string]*entry{}, failOn: map[string]bool{}}
}

func (s *store) SignIn(_ context.Context, issuer string, id domain.Identity, token, ua string, at, expires time.Time) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failOn["SignIn"] {
		return "", errBoom
	}
	s.next++
	sid := fmt.Sprintf("s%d", s.next)
	s.issuers = append(s.issuers, issuer)
	s.sessions[sid] = &entry{token: token, session: domain.Session{
		ID: sid, Account: domain.Account{ID: 1, Issuer: issuer, Identity: id}, UserAgent: ua,
		CreatedAt: at, RolesCheckedAt: at, ExpiresAt: expires,
	}}
	return sid, nil
}

func (s *store) Load(_ context.Context, id string, now time.Time) (domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loads++
	if s.failOn["Load"] {
		return domain.Session{}, errBoom
	}
	e, ok := s.sessions[id]
	if !ok || !now.Before(e.session.ExpiresAt) {
		return domain.Session{}, domain.ErrNoSession
	}
	return e.session, nil
}

func (s *store) Refresh(ctx context.Context, id string, seen, now time.Time, fn app.RefreshFunc) (domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.sessions[id]
	if !ok {
		return domain.Session{}, domain.ErrNoSession
	}
	if !e.session.RolesCheckedAt.Equal(seen) {
		return e.session, nil
	}
	next, ident, err := fn(ctx, e.token)
	if errors.Is(err, domain.ErrSessionGone) {
		delete(s.sessions, id)
		return domain.Session{}, err
	}
	if err != nil {
		return domain.Session{}, err
	}
	e.token, e.session.RolesCheckedAt, e.session.Account.Identity = next, now, ident
	return e.session, nil
}

func (s *store) SignOut(_ context.Context, id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failOn["SignOut"] {
		return "", errBoom
	}
	e, ok := s.sessions[id]
	if !ok {
		return "", domain.ErrNoSession
	}
	delete(s.sessions, id)
	return e.token, nil
}

func (s *store) Prune(context.Context, time.Time) error {
	if s.failOn["Prune"] {
		return errBoom
	}
	return nil
}

// issuer hands out the next token and identity, or fails with err; revoked hears every revocation.
type issuer struct {
	mu        sync.Mutex
	identity  domain.Identity
	err       error
	revokeErr error
	calls     int
	revoked   chan string
}

func (i *issuer) Refresh(_ context.Context, token string) (string, domain.Identity, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.calls++
	if i.err != nil {
		return "", domain.Identity{}, i.err
	}
	return token + "+", i.identity, nil
}

func (i *issuer) Revoke(_ context.Context, token string) error {
	i.mu.Lock()
	err := i.revokeErr
	i.mu.Unlock()
	i.revoked <- token
	return err
}

type warnings struct {
	mu   sync.Mutex
	msgs []string
}

func (w *warnings) warn(msg string, _ error) {
	w.mu.Lock()
	w.msgs = append(w.msgs, msg)
	w.mu.Unlock()
}

func (w *warnings) count() int { w.mu.Lock(); defer w.mu.Unlock(); return len(w.msgs) }

type fixture struct {
	svc    *app.Service
	store  *store
	issuer *issuer
	clock  *clock
	warn   *warnings
}

func setup() fixture {
	c := &clock{t: time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)}
	st := newStore(c)
	is := &issuer{identity: domain.Identity{Subject: "u1", Name: "joris"}, revoked: make(chan string, 4)}
	w := &warnings{}
	return fixture{svc: app.New(st, is, "https://auth.test", c.now, w.warn), store: st, issuer: is, clock: c, warn: w}
}

func (f fixture) signIn(t *testing.T, roles ...string) string {
	t.Helper()
	sid, err := f.svc.SignIn(t.Context(), domain.Identity{Subject: "u1", Name: "joris", Roles: roles}, "rt-1", "Firefox")
	if err != nil {
		t.Fatal(err)
	}
	return sid
}

func TestSignIn(t *testing.T) {
	f := setup()
	if _, err := f.svc.SignIn(t.Context(), domain.Identity{Subject: "u1"}, "", "ua"); !errors.Is(err, app.ErrNoRefreshToken) {
		t.Fatalf("no refresh token: %v", err)
	}
	sid := f.signIn(t)
	if e := f.store.sessions[sid]; e.session.ExpiresAt != f.clock.now().Add(domain.SessionTTL) || f.store.issuers[0] != "https://auth.test" {
		t.Fatalf("session = %+v", e.session)
	}
	f.store.failOn["SignIn"] = true
	if _, err := f.svc.SignIn(t.Context(), domain.Identity{Subject: "u1"}, "rt", "ua"); !errors.Is(err, errBoom) {
		t.Fatalf("store failure: %v", err)
	}
}

func TestPrincipalUsesFreshRolesAndCachesTheSession(t *testing.T) {
	f := setup()
	sid := f.signIn(t, domain.AdminRole)
	for range 3 {
		p, err := f.svc.Principal(t.Context(), sid, domain.NeedAdminView)
		if err != nil || p.Subject != "u1" || !p.IsAdmin() {
			t.Fatalf("principal = %+v, %v", p, err)
		}
	}
	if f.store.loads != 1 || f.issuer.calls != 0 {
		t.Fatalf("%d loads, %d refreshes for fresh roles", f.store.loads, f.issuer.calls)
	}
	f.clock.add(5 * time.Second)
	if _, err := f.svc.Principal(t.Context(), sid, domain.NeedView); err != nil || f.store.loads != 2 {
		t.Fatalf("after the cache expired: %v, %d loads", err, f.store.loads)
	}
}

func TestPrincipalRereadsOldRoles(t *testing.T) {
	f := setup()
	sid := f.signIn(t, domain.AdminRole)
	f.clock.add(domain.FreshFor)
	// auth-api withdrew the admin role meanwhile.
	p, err := f.svc.Principal(t.Context(), sid, domain.NeedView)
	if err != nil || p.IsAdmin() || f.issuer.calls != 1 || f.store.sessions[sid].token != "rt-1+" {
		t.Fatalf("principal = %+v, %v after %d refreshes", p, err, f.issuer.calls)
	}
	if _, err := f.svc.Principal(t.Context(), sid, domain.NeedAdminView); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("a former admin: %v", err)
	}
	// A change always re-reads.
	f.issuer.identity.Roles = []string{domain.AdminRole}
	if p, err := f.svc.Principal(t.Context(), sid, domain.NeedAdminChange); err != nil || !p.IsAdmin() || f.issuer.calls != 2 {
		t.Fatalf("change = %+v, %v after %d refreshes", p, err, f.issuer.calls)
	}
}

func TestPrincipalWhenAuthAPIEndsTheSession(t *testing.T) {
	f := setup()
	sid := f.signIn(t)
	f.clock.add(time.Minute)
	f.issuer.err = domain.ErrSessionGone
	if _, err := f.svc.Principal(t.Context(), sid, domain.NeedView); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("revoked: %v", err)
	}
	if _, ok := f.store.sessions[sid]; ok {
		t.Fatal("the session outlived its revocation")
	}
	if _, err := f.svc.Principal(t.Context(), sid, domain.NeedView); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("after revocation: %v", err)
	}
}

func TestPrincipalWhenAuthAPIIsDown(t *testing.T) {
	f := setup()
	sid := f.signIn(t, domain.AdminRole)
	f.issuer.err = errBoom
	f.clock.add(time.Minute)
	if p, err := f.svc.Principal(t.Context(), sid, domain.NeedAdminView); err != nil || !p.IsAdmin() || f.warn.count() != 1 {
		t.Fatalf("read within grace: %+v, %v, %d warnings", p, err, f.warn.count())
	}
	if _, err := f.svc.Principal(t.Context(), sid, domain.NeedAdminChange); !errors.Is(err, domain.ErrRolesUnconfirmed) || !errors.Is(err, errBoom) {
		t.Fatalf("change without auth-api: %v", err)
	}
	f.clock.add(domain.Grace)
	if _, err := f.svc.Principal(t.Context(), sid, domain.NeedView); !errors.Is(err, domain.ErrRolesUnconfirmed) {
		t.Fatalf("read past grace: %v", err)
	}
}

func TestPrincipalWithoutASession(t *testing.T) {
	f := setup()
	if _, err := f.svc.Principal(t.Context(), "", domain.NeedView); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("empty id: %v", err)
	}
	if _, err := f.svc.Principal(t.Context(), "nobody", domain.NeedView); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("unknown id: %v", err)
	}
	f.store.failOn["Load"] = true
	if _, err := f.svc.Principal(t.Context(), "nobody", domain.NeedView); !errors.Is(err, errBoom) {
		t.Fatalf("store down: %v", err)
	}
}

func TestPrincipalWhenTheSessionVanishesDuringARefresh(t *testing.T) {
	f := setup()
	sid := f.signIn(t)
	if _, err := f.svc.Principal(t.Context(), sid, domain.NeedView); err != nil {
		t.Fatal(err)
	}
	delete(f.store.sessions, sid) // signed out elsewhere while cached
	if _, err := f.svc.Principal(t.Context(), sid, domain.NeedAdminChange); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("vanished: %v", err)
	}
}

func TestTheCacheDropsStaleSessions(t *testing.T) {
	f := setup()
	for i := range 1100 {
		sid := f.signIn(t)
		if _, err := f.svc.Principal(t.Context(), sid, domain.NeedView); err != nil {
			t.Fatal(err)
		}
		if i == 1000 {
			f.clock.add(5 * time.Second)
		}
	}
}

func TestSignOut(t *testing.T) {
	f := setup()
	sid := f.signIn(t)
	if _, err := f.svc.Principal(t.Context(), sid, domain.NeedView); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.SignOut(t.Context(), sid); err != nil {
		t.Fatal(err)
	}
	if got := <-f.issuer.revoked; got != "rt-1" {
		t.Fatalf("revoked %q", got)
	}
	if _, err := f.svc.Principal(t.Context(), sid, domain.NeedView); !errors.Is(err, domain.ErrNoSession) {
		t.Fatalf("after sign-out: %v", err)
	}
	if err := f.svc.SignOut(t.Context(), sid); err != nil {
		t.Fatalf("signing out twice: %v", err)
	}
	if err := f.svc.SignOut(t.Context(), ""); err != nil {
		t.Fatalf("signing out without a session: %v", err)
	}

	f.issuer.mu.Lock()
	f.issuer.revokeErr = errBoom
	f.issuer.mu.Unlock()
	sid = f.signIn(t)
	before := f.warn.count()
	if err := f.svc.SignOut(t.Context(), sid); err != nil {
		t.Fatal(err)
	}
	<-f.issuer.revoked
	for range 200 {
		if f.warn.count() > before {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if f.warn.count() != before+1 {
		t.Fatal("a failed revocation was not reported")
	}

	sid, _ = f.store.SignIn(t.Context(), "i", domain.Identity{}, "", "", f.clock.now(), f.clock.now().Add(time.Hour))
	if err := f.svc.SignOut(t.Context(), sid); err != nil {
		t.Fatalf("a session whose token cannot be read: %v", err)
	}
	f.store.failOn["SignOut"] = true
	if err := f.svc.SignOut(t.Context(), "any"); !errors.Is(err, errBoom) {
		t.Fatalf("store down: %v", err)
	}
}

func TestPrune(t *testing.T) {
	f := setup()
	if err := f.svc.Prune(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.store.failOn["Prune"] = true
	if err := f.svc.Prune(t.Context()); !errors.Is(err, errBoom) {
		t.Fatalf("prune: %v", err)
	}
}
