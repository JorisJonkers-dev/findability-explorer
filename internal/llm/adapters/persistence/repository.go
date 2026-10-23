// Package persistence implements the language-model Repository port on the sqlc-generated
// queries. Keys are sealed before they reach the database (docs/adr/0002).
package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/queries"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/seal"
)

// HKDFInfo derives the key that seals API keys from SESSION_KEY.
const HKDFInfo = "findability-explorer-llm-v1"

// DB is what the repository needs from the Postgres adapter.
type DB interface {
	Queries() *queries.Queries
	InTx(ctx context.Context, fn func(*queries.Queries) error) error
}

// Repository keeps credentials, the active model and the events in Postgres.
type Repository struct {
	db     DB
	sealer *seal.Sealer
}

var _ app.Repository = (*Repository)(nil)

// New returns a Repository sealing keys with sealer.
func New(db DB, sealer *seal.Sealer) *Repository {
	return &Repository{db: db, sealer: sealer}
}

// bound ties a sealed key to its row and provider, so a blob cannot move to another row.
func bound(id uuid.UUID, p domain.Provider) string {
	return "findability-explorer:llm-credential:" + id.String() + ":" + string(p)
}

// Add seals key and stores the credential.
func (r *Repository) Add(ctx context.Context, c domain.Credential, key string) error {
	id, err := uuid.Parse(c.ID)
	if err != nil {
		return fmt.Errorf("credential id: %w", err)
	}
	blob, err := r.sealer.Seal(bound(id, c.Provider), []byte(key))
	if err != nil {
		return fmt.Errorf("seal key: %w", err)
	}
	if err := r.db.Queries().AddLlmCredential(ctx, queries.AddLlmCredentialParams{
		ID: id, Provider: string(c.Provider), Label: c.Label, Secret: blob, KeyID: r.sealer.KeyID(),
		Fingerprint: c.Fingerprint, CreatedBy: c.CreatedBy, CreatedAt: c.CreatedAt,
	}); err != nil {
		return fmt.Errorf("insert credential: %w", err)
	}
	return nil
}

// List returns every stored credential, oldest first. A key sealed under another key id is not
// readable.
func (r *Repository) List(ctx context.Context) ([]domain.Credential, error) {
	rows, err := r.db.Queries().ListLlmCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("select credentials: %w", err)
	}
	out := make([]domain.Credential, 0, len(rows))
	for _, row := range rows {
		out = append(out, r.credential(row.ID, row.Provider, row.Label, row.KeyID, row.Fingerprint, row.CreatedBy, row.CreatedAt))
	}
	return out, nil
}

func (r *Repository) credential(id uuid.UUID, provider, label, keyID, fingerprint, by string, at time.Time) domain.Credential {
	return domain.Credential{
		ID: id.String(), Provider: domain.Provider(provider), Label: label, Source: domain.SourceStored,
		Fingerprint: fingerprint, Readable: keyID == r.sealer.KeyID(), CreatedBy: by, CreatedAt: at.UTC(),
	}
}

func (r *Repository) get(ctx context.Context, id string) (queries.LlmCredential, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return queries.LlmCredential{}, domain.ErrNotFound
	}
	row, err := r.db.Queries().GetLlmCredential(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return queries.LlmCredential{}, domain.ErrNotFound
	}
	if err != nil {
		return queries.LlmCredential{}, fmt.Errorf("select credential: %w", err)
	}
	return row, nil
}

// Get returns one stored credential without its secret.
func (r *Repository) Get(ctx context.Context, id string) (domain.Credential, error) {
	row, err := r.get(ctx, id)
	if err != nil {
		return domain.Credential{}, err
	}
	return r.credential(row.ID, row.Provider, row.Label, row.KeyID, row.Fingerprint, row.CreatedBy, row.CreatedAt), nil
}

// Secret opens a stored key.
func (r *Repository) Secret(ctx context.Context, id string) (domain.Provider, string, error) {
	row, err := r.get(ctx, id)
	if err != nil {
		return "", "", err
	}
	p := domain.Provider(row.Provider)
	plain, err := r.sealer.Open(bound(row.ID, p), row.KeyID, row.Secret)
	if err != nil {
		return "", "", domain.ErrUnreadable
	}
	return p, string(plain), nil
}

// Delete removes a stored credential and, when it was the active one, the decision about the
// active model, in one transaction.
func (r *Repository) Delete(ctx context.Context, id string) (bool, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return false, domain.ErrNotFound
	}
	var wasActive bool
	err = r.db.InTx(ctx, func(q *queries.Queries) error {
		n, err := q.DeleteLlmCredential(ctx, uid)
		if err != nil {
			return fmt.Errorf("delete credential: %w", err)
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		reset, err := q.ResetLlmActiveFor(ctx, pgtype.Text{String: id, Valid: true})
		if err != nil {
			return fmt.Errorf("reset the active model: %w", err)
		}
		wasActive = reset > 0
		return nil
	})
	return wasActive, err
}

// Setting returns the decision about the active model; no row is ModeUnset.
func (r *Repository) Setting(ctx context.Context) (domain.Setting, error) {
	row, err := r.db.Queries().GetLlmActive(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Setting{Mode: domain.ModeUnset}, nil
	}
	if err != nil {
		return domain.Setting{}, fmt.Errorf("select the active model: %w", err)
	}
	return domain.Setting{Mode: domain.Mode(row.Mode), Active: domain.Active{CredentialID: row.CredentialID, Model: row.Model}}, nil
}

// SaveSetting stores the decision about the active model.
func (r *Repository) SaveSetting(ctx context.Context, s domain.Setting, actor string, at time.Time) error {
	chosen := s.Mode == domain.ModeChosen
	if err := r.db.Queries().SetLlmActive(ctx, queries.SetLlmActiveParams{
		Mode:         string(s.Mode),
		CredentialID: pgtype.Text{String: s.Active.CredentialID, Valid: chosen},
		Model:        pgtype.Text{String: s.Active.Model, Valid: chosen},
		UpdatedBy:    actor, UpdatedAt: at,
	}); err != nil {
		return fmt.Errorf("upsert the active model: %w", err)
	}
	return nil
}

// AddEvent appends an event.
func (r *Repository) AddEvent(ctx context.Context, e domain.Event) error {
	ok := pgtype.Bool{}
	if e.OK != nil {
		ok = pgtype.Bool{Bool: *e.OK, Valid: true}
	}
	if err := r.db.Queries().AddLlmEvent(ctx, queries.AddLlmEventParams{
		At: e.At, Actor: e.Actor, Action: string(e.Action), CredentialID: optional(e.CredentialID),
		Provider: optional(string(e.Provider)), Model: optional(e.Model), Ok: ok, Message: e.Message,
	}); err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

// Events returns the latest limit events, newest first.
func (r *Repository) Events(ctx context.Context, limit int) ([]domain.Event, error) {
	rows, err := r.db.Queries().ListLlmEvents(ctx, int32(min(max(limit, 0), domain.MaxEvents))) //nolint:gosec // at most MaxEvents
	if err != nil {
		return nil, fmt.Errorf("select events: %w", err)
	}
	out := make([]domain.Event, 0, len(rows))
	for _, row := range rows {
		e := domain.Event{
			At: row.At.UTC(), Actor: row.Actor, Action: domain.Action(row.Action), CredentialID: row.CredentialID.String,
			Provider: domain.Provider(row.Provider.String), Model: row.Model.String, Message: row.Message,
		}
		if row.Ok.Valid {
			ok := row.Ok.Bool
			e.OK = &ok
		}
		out = append(out, e)
	}
	return out, nil
}

// LastTests returns the latest test of each credential.
func (r *Repository) LastTests(ctx context.Context) (map[string]domain.Test, error) {
	rows, err := r.db.Queries().LatestLlmTests(ctx)
	if err != nil {
		return nil, fmt.Errorf("select tests: %w", err)
	}
	out := make(map[string]domain.Test, len(rows))
	for _, row := range rows {
		out[row.CredentialID] = domain.Test{OK: row.Ok, At: row.At.UTC(), Message: row.Message}
	}
	return out, nil
}

func optional(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }
