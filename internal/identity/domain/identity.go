// Package domain is the identity context's pure core: local accounts of auth-api users, their
// server-side sessions, and the rules for how old the roles a request relies on may be
// (docs/adr/0001). It imports no transport, persistence or generated code (enforced by depguard).
package domain

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode"
)

// AdminRole is auth-api's administrator role; only it may use the admin settings.
const AdminRole = "ROLE_ADMIN"

const (
	// SessionTTL is how long a session lasts after signing in.
	SessionTTL = 7 * 24 * time.Hour
	// FreshFor is how old a session's roles may be before a request re-reads them from auth-api.
	FreshFor = 15 * time.Second
	// Grace is how long a request that changes nothing may go on with the last known roles while
	// auth-api cannot be reached. A change always needs roles read for it.
	Grace = 5 * time.Minute
)

var (
	// ErrNoSession means there is no live session: never signed in, signed out, expired, or ended
	// by auth-api.
	ErrNoSession = errors.New("not signed in")
	// ErrSessionGone means auth-api no longer lets the account in: the refresh token was revoked
	// or the account disabled. The session is deleted.
	ErrSessionGone = errors.New("the session was ended by auth-api")
	// ErrForbidden means the account is signed in but lacks the administrator role.
	ErrForbidden = errors.New("only an administrator may do this")
	// ErrRolesUnconfirmed means auth-api could not be asked for the account's roles, and the
	// request may not go on without them.
	ErrRolesUnconfirmed = errors.New("auth-api could not confirm the account's roles; try again shortly")
)

// Identity is what auth-api says about an account in a verified token.
type Identity struct {
	Subject string
	Name    string
	Email   string
	Roles   []string
}

// Account is the local record of an auth-api account that signed in.
type Account struct {
	ID       int64
	Issuer   string
	Identity Identity
}

// Session is a server-side session of one account.
type Session struct {
	ID             string
	Account        Account
	UserAgent      string
	CreatedAt      time.Time
	LastSeenAt     time.Time
	RolesCheckedAt time.Time
	ExpiresAt      time.Time
}

// Principal is who a request runs as.
type Principal struct {
	Subject string
	Name    string
	Email   string
	Roles   []string
}

// IsAdmin reports whether the principal holds AdminRole.
func (p Principal) IsAdmin() bool { return slices.Contains(p.Roles, AdminRole) }

// Principal is the session's account as a Principal.
func (s Session) Principal() Principal {
	id := s.Account.Identity
	return Principal{Subject: id.Subject, Name: DisplayName(id), Email: id.Email, Roles: id.Roles}
}

// DisplayName is the name to greet an identity with: its name, else its email, else its subject.
func DisplayName(id Identity) string {
	for _, v := range []string{id.Name, id.Email, id.Subject} {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// Need is what a request needs of the roles it runs with.
type Need int

const (
	// NeedView is any signed-in account, with roles at most FreshFor old.
	NeedView Need = iota
	// NeedAdminView is an administrator, with roles at most FreshFor old.
	NeedAdminView
	// NeedAdminChange is an administrator, with roles read from auth-api for this request.
	NeedAdminChange
)

// MaxAge is how old the roles may be: FreshFor, or zero for a change.
func (n Need) MaxAge() time.Duration {
	if n == NeedAdminChange {
		return 0
	}
	return FreshFor
}

// MayUseStale reports whether, with auth-api unreachable, a request may go on with roles age old.
func (n Need) MayUseStale(age time.Duration) bool {
	return n != NeedAdminChange && age <= Grace
}

// Authorise returns p when it meets n, or ErrForbidden.
func (n Need) Authorise(p Principal) (Principal, error) {
	if n != NeedView && !p.IsAdmin() {
		return Principal{}, ErrForbidden
	}
	return p, nil
}

// ParseRoles splits a comma-separated role list, as DEV_ROLES holds it, dropping blanks.
func ParseRoles(raw string) []string {
	out := []string{}
	for r := range strings.SplitSeq(raw, ",") {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, r)
		}
	}
	return out
}

// SafeReturn is path when it is a path on this site to return to after signing in, else "/".
// Only an absolute path counts: no scheme, no host, no "//" or backslash that a browser could read
// as another host, and nothing under /auth/.
func SafeReturn(path string) string {
	switch {
	case !strings.HasPrefix(path, "/"),
		strings.HasPrefix(path, "//"),
		strings.Contains(path, "\\"),
		strings.ContainsFunc(path, unicode.IsControl),
		strings.HasPrefix(path, "/auth/"):
		return "/"
	default:
		return path
	}
}
