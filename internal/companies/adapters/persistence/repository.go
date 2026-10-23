// Package persistence implements the companies Repository port on the sqlc-generated queries.
package persistence

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/queries"
)

// Repository maps rows to added companies.
type Repository struct {
	q queries.Querier
}

var _ app.Repository = (*Repository)(nil)

// New returns a Repository over q.
func New(q queries.Querier) *Repository {
	return &Repository{q: q}
}

// List returns every added company, oldest first.
func (r *Repository) List(ctx context.Context) ([]domain.Company, error) {
	rows, err := r.q.ListCompanies(ctx)
	if err != nil {
		return nil, fmt.Errorf("select companies: %w", err)
	}
	out := make([]domain.Company, 0, len(rows))
	for _, row := range rows {
		u, err := url.Parse(row.WebsiteUrl)
		if err != nil {
			return nil, fmt.Errorf("company %s: website: %w", row.ID, err)
		}
		out = append(out, domain.Company{
			ID: row.ID, Source: domain.SourceAdded, Name: row.Name, Website: row.WebsiteUrl, URL: u,
			Host: row.Host, AddedBy: row.AddedBy, AddedAt: row.AddedAt.UTC(),
		})
	}
	return out, nil
}

// Add stores c. The primary key and the unique host refuse a company added meanwhile.
func (r *Repository) Add(ctx context.Context, c domain.Company) error {
	if err := r.q.InsertCompany(ctx, queries.InsertCompanyParams{
		ID: c.ID, Name: c.Name, WebsiteUrl: c.URL.String(), Host: c.Host, AddedBy: c.AddedBy, AddedAt: c.AddedAt,
	}); err != nil {
		return fmt.Errorf("insert company: %w", err)
	}
	return nil
}

// Update stores an added company's new name and website.
func (r *Repository) Update(ctx context.Context, c domain.Company) error {
	n, err := r.q.UpdateCompany(ctx, queries.UpdateCompanyParams{ID: c.ID, Name: c.Name, WebsiteUrl: c.URL.String(), Host: c.Host})
	if err != nil {
		return fmt.Errorf("update company: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete deletes an added company.
func (r *Repository) Delete(ctx context.Context, id string) error {
	n, err := r.q.DeleteCompany(ctx, id)
	if err != nil {
		return fmt.Errorf("delete company: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Overrides returns every change to a researched company.
func (r *Repository) Overrides(ctx context.Context) ([]domain.Override, error) {
	rows, err := r.q.ListCompanyOverrides(ctx)
	if err != nil {
		return nil, fmt.Errorf("select company changes: %w", err)
	}
	out := make([]domain.Override, 0, len(rows))
	for _, row := range rows {
		o := domain.Override{
			ID: row.ID, Name: row.Name.String, Host: row.Host.String, ArchivedBy: row.ArchivedBy.String,
			UpdatedBy: row.UpdatedBy, UpdatedAt: row.UpdatedAt.UTC(),
		}
		if row.WebsiteUrl.Valid {
			if o.Website, err = url.Parse(row.WebsiteUrl.String); err != nil {
				return nil, fmt.Errorf("company change %s: website: %w", row.ID, err)
			}
		}
		if row.ArchivedAt.Valid {
			o.ArchivedAt = row.ArchivedAt.Time.UTC()
		}
		out = append(out, o)
	}
	return out, nil
}

// SaveOverride stores o in place of the researched company's earlier change.
func (r *Repository) SaveOverride(ctx context.Context, o domain.Override) error {
	website := ""
	if o.Website != nil {
		website = o.Website.String()
	}
	if err := r.q.UpsertCompanyOverride(ctx, queries.UpsertCompanyOverrideParams{
		ID: o.ID, Name: optional(o.Name), WebsiteUrl: optional(website), Host: optional(o.Host),
		ArchivedBy: optional(o.ArchivedBy), ArchivedAt: pgtype.Timestamptz{Time: o.ArchivedAt, Valid: !o.ArchivedAt.IsZero()},
		UpdatedBy: o.UpdatedBy, UpdatedAt: o.UpdatedAt,
	}); err != nil {
		return fmt.Errorf("upsert company change: %w", err)
	}
	return nil
}

func optional(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }
