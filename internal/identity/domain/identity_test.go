package domain_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/domain"
)

func TestSessionPrincipal(t *testing.T) {
	s := domain.Session{Account: domain.Account{Identity: domain.Identity{
		Subject: "sub-1", Name: " ", Email: "j@example.com", Roles: []string{"ROLE_USER", domain.AdminRole},
	}}}
	p := s.Principal()
	if p.Subject != "sub-1" || p.Name != "j@example.com" || p.Email != "j@example.com" || !p.IsAdmin() {
		t.Fatalf("principal = %+v", p)
	}
	if (domain.Principal{Roles: []string{"ROLE_USER"}}).IsAdmin() {
		t.Fatal("a user is an admin")
	}
}

func TestDisplayName(t *testing.T) {
	cases := map[string]domain.Identity{
		"joris": {Name: "joris", Email: "e", Subject: "s"},
		"e":     {Email: "e", Subject: "s"},
		"s":     {Subject: "s"},
		"":      {},
	}
	for want, id := range cases {
		if got := domain.DisplayName(id); got != want {
			t.Errorf("DisplayName(%+v) = %q, want %q", id, got, want)
		}
	}
}

func TestNeeds(t *testing.T) {
	user := domain.Principal{Subject: "u"}
	admin := domain.Principal{Subject: "a", Roles: []string{domain.AdminRole}}
	cases := []struct {
		need          domain.Need
		maxAge        time.Duration
		staleOK       bool
		userForbidden bool
	}{
		{domain.NeedView, domain.FreshFor, true, false},
		{domain.NeedAdminView, domain.FreshFor, true, true},
		{domain.NeedAdminChange, 0, false, true},
	}
	for _, c := range cases {
		if got := c.need.MaxAge(); got != c.maxAge {
			t.Errorf("need %d: MaxAge %v", c.need, got)
		}
		if got := c.need.MayUseStale(time.Minute); got != c.staleOK {
			t.Errorf("need %d: MayUseStale(1m) = %v", c.need, got)
		}
		if c.need.MayUseStale(domain.Grace + time.Second) {
			t.Errorf("need %d: roles past the grace period will do", c.need)
		}
		if _, err := c.need.Authorise(user); errors.Is(err, domain.ErrForbidden) != c.userForbidden {
			t.Errorf("need %d: user gets %v", c.need, err)
		}
		if p, err := c.need.Authorise(admin); err != nil || p.Subject != "a" {
			t.Errorf("need %d: admin gets %+v, %v", c.need, p, err)
		}
	}
}

func TestParseRoles(t *testing.T) {
	if got := domain.ParseRoles(" ROLE_ADMIN, ,SERVICE_X "); !slices.Equal(got, []string{"ROLE_ADMIN", "SERVICE_X"}) {
		t.Fatalf("roles = %v", got)
	}
	if got := domain.ParseRoles(""); got == nil || len(got) != 0 {
		t.Fatalf("no roles = %#v", got)
	}
}

func TestSafeReturn(t *testing.T) {
	cases := map[string]string{
		"/competitors?x=1#y":   "/competitors?x=1#y",
		"/":                    "/",
		"":                     "/",
		"https://evil.test/":   "/",
		"//evil.test/":         "/",
		"/\\evil.test":         "/",
		"/a\nb":                "/",
		"/auth/login?return=/": "/",
		"competitors":          "/",
	}
	for in, want := range cases {
		if got := domain.SafeReturn(in); got != want {
			t.Errorf("SafeReturn(%q) = %q, want %q", in, got, want)
		}
	}
}
