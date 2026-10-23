// Package persistence implements the identity Store port on the sqlc-generated queries. Refresh
// tokens are sealed before they reach the database (docs/adr/0001).
package persistence

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/queries"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/seal"
)

// HKDFInfo derives the key that seals refresh tokens from SESSION_KEY.
const HKDFInfo = "findability-explorer-refresh-token-v1"

// bound binds a sealed refresh token to its use, so the blob never opens as anything else.
const bound = "findability-explorer:oidc-refresh-token"

const (
	maxUserAgent = 300
	touchEvery   = time.Minute
)

// DB is what the store needs from the Postgres adapter.
type DB interface {
	Queries() *queries.Queries
	InTx(ctx context.Context, fn func(*queries.Queries) error) error
}

// Store keeps accounts and sessions in Postgres.
type Store struct {
	db     DB
	sealer *seal.Sealer
}

var _ app.Store = (*Store)(nil)

// New returns a Store sealing refresh tokens with sealer.
func New(db DB, sealer *seal.Sealer) *Store {
	return &Store{db: db, sealer: sealer}
}

// SignIn creates or updates the account and opens a session for it.
func (s *Store) SignIn(ctx context.Context, issuer string, id domain.Identity, refreshToken, userAgent string, at, expires time.Time) (string, error) {
	blob, err := s.sealer.Seal(bound, []byte(refreshToken))
	if err != nil {
		return "", fmt.Errorf("seal refresh token: %w", err)
	}
	sid := newSessionID()
	err = s.db.InTx(ctx, func(q *queries.Queries) error {
		account, err := q.UpsertAccount(ctx, queries.UpsertAccountParams{
			Issuer: issuer, Subject: id.Subject, Name: clip(id.Name, 200), Email: clip(id.Email, 320),
			Roles: nonNil(id.Roles), At: at,
		})
		if err != nil {
			return fmt.Errorf("upsert account: %w", err)
		}
		return q.CreateSession(ctx, queries.CreateSessionParams{
			ID: sid, AccountID: account, RefreshToken: blob, KeyID: s.sealer.KeyID(),
			UserAgent: clip(userAgent, maxUserAgent), At: at, ExpiresAt: expires,
		})
	})
	if err != nil {
		return "", err
	}
	return sid, nil
}

// Load returns the live session with its account, and notes that it was seen.
func (s *Store) Load(ctx context.Context, sessionID string, now time.Time) (domain.Session, error) {
	q := s.db.Queries()
	row, err := q.GetSession(ctx, queries.GetSessionParams{ID: sessionID, Now: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Session{}, domain.ErrNoSession
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("select session: %w", err)
	}
	if err := q.TouchSession(ctx, queries.TouchSessionParams{ID: sessionID, At: now, StaleBefore: now.Add(-touchEvery)}); err != nil {
		return domain.Session{}, fmt.Errorf("touch session: %w", err)
	}
	return toSession(row), nil
}

// Refresh re-reads the roles under a row lock, so concurrent requests never spend the same
// rotated refresh token twice.
func (s *Store) Refresh(ctx context.Context, sessionID string, seen, now time.Time, fn app.RefreshFunc) (domain.Session, error) {
	var out domain.Session
	gone := false
	err := s.db.InTx(ctx, func(q *queries.Queries) error {
		locked, err := q.LockSession(ctx, queries.LockSessionParams{ID: sessionID, Now: now})
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNoSession
		}
		if err != nil {
			return fmt.Errorf("lock session: %w", err)
		}
		// Another request re-read the roles while this one waited for the lock.
		if locked.RolesCheckedAt.Equal(seen) {
			if gone, err = s.refreshLocked(ctx, q, locked, now, fn); err != nil || gone {
				return err
			}
		}
		row, err := q.GetSession(ctx, queries.GetSessionParams{ID: sessionID, Now: now})
		if err != nil {
			return fmt.Errorf("select session: %w", err)
		}
		out = toSession(row)
		return nil
	})
	if gone {
		return domain.Session{}, domain.ErrSessionGone
	}
	if err != nil {
		return domain.Session{}, err
	}
	return out, nil
}

// refreshLocked spends the session's refresh token; gone means it deleted the session instead,
// which the transaction commits.
func (s *Store) refreshLocked(ctx context.Context, q *queries.Queries, locked queries.LockSessionRow, now time.Time, fn app.RefreshFunc) (bool, error) {
	plain, err := s.sealer.Open(bound, locked.KeyID, locked.RefreshToken)
	if err != nil { // SESSION_KEY was rotated: the token is lost, and so is the session
		_, err := q.DeleteSession(ctx, locked.ID)
		return true, err
	}
	next, id, err := fn(ctx, string(plain))
	if errors.Is(err, domain.ErrSessionGone) {
		_, err := q.DeleteSession(ctx, locked.ID)
		return true, err
	}
	if err != nil {
		return false, err
	}
	blob, err := s.sealer.Seal(bound, []byte(next))
	if err != nil {
		return false, fmt.Errorf("seal refresh token: %w", err)
	}
	if err := q.SaveSessionRefresh(ctx, queries.SaveSessionRefreshParams{ID: locked.ID, RefreshToken: blob, KeyID: s.sealer.KeyID(), At: now}); err != nil {
		return false, fmt.Errorf("save refresh token: %w", err)
	}
	if err := q.UpdateAccountIdentity(ctx, queries.UpdateAccountIdentityParams{
		ID: locked.AccountID, Name: clip(id.Name, 200), Email: clip(id.Email, 320), Roles: nonNil(id.Roles),
	}); err != nil {
		return false, fmt.Errorf("update account: %w", err)
	}
	return false, nil
}

// SignOut deletes the session and returns its refresh token for revocation; empty when the token
// can no longer be opened, which leaves nothing to revoke.
func (s *Store) SignOut(ctx context.Context, sessionID string) (string, error) {
	row, err := s.db.Queries().DeleteSession(ctx, sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNoSession
	}
	if err != nil {
		return "", fmt.Errorf("delete session: %w", err)
	}
	plain, err := s.sealer.Open(bound, row.KeyID, row.RefreshToken)
	if err != nil {
		return "", nil //nolint:nilerr // an unreadable token cannot be revoked; the session is gone regardless
	}
	return string(plain), nil
}

// Prune deletes every expired session.
func (s *Store) Prune(ctx context.Context, now time.Time) error {
	if err := s.db.Queries().DeleteExpiredSessions(ctx, now); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}

func toSession(r queries.GetSessionRow) domain.Session {
	return domain.Session{
		ID: r.ID, UserAgent: r.UserAgent, CreatedAt: r.CreatedAt.UTC(), LastSeenAt: r.LastSeenAt.UTC(),
		RolesCheckedAt: r.RolesCheckedAt, ExpiresAt: r.ExpiresAt.UTC(),
		Account: domain.Account{
			ID: r.AccountID, Issuer: r.Issuer,
			Identity: domain.Identity{Subject: r.Subject, Name: r.Name, Email: r.Email, Roles: r.Roles},
		},
	}
}

// newSessionID returns a random 256-bit session id.
func newSessionID() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // crypto/rand.Read never fails
	return base64.RawURLEncoding.EncodeToString(b)
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
