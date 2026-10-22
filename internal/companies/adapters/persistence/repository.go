// Package persistence implements the companies Repository port on the sqlc-generated queries.
package persistence

import (
	"context"
	"fmt"
	"net/url"

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
