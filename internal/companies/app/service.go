// Package app holds the companies context's use cases: listing and adding companies, and
// matching a scanned website to the company it belongs to.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/site"
)

// MaxScans is how many of a company's scans Get returns.
const MaxScans = 20

// Research returns the companies the desk research studied; the terminology seed provides them.
type Research func() []domain.Company

// Repository stores the companies users and scans added.
type Repository interface {
	List(ctx context.Context) ([]domain.Company, error)
	// Add stores c, or fails when its id or host is taken meanwhile.
	Add(ctx context.Context, c domain.Company) error
}

// Scans is what the companies context reads of the analyses: how often each company was scanned,
// and a company's newest scans.
type Scans interface {
	Stats(ctx context.Context) (map[string]domain.Stats, error)
	ByCompany(ctx context.Context, id string, limit int) ([]domain.Scan, error)
}

// Service runs the companies use cases.
type Service struct {
	research Research
	repo     Repository
	scans    Scans
	now      func() time.Time
}

// New returns a Service.
func New(research Research, repo Repository, scans Scans, now func() time.Time) *Service {
	return &Service{research: research, repo: repo, scans: scans, now: now}
}

func (s *Service) directory(ctx context.Context) (domain.Directory, error) {
	added, err := s.repo.List(ctx)
	if err != nil {
		return domain.Directory{}, fmt.Errorf("list added companies: %w", err)
	}
	return domain.NewDirectory(s.research(), added), nil
}

// List returns every known company with its scan statistics.
func (s *Service) List(ctx context.Context) ([]domain.Listed, error) {
	d, err := s.directory(ctx)
	if err != nil {
		return nil, err
	}
	stats, err := s.scans.Stats(ctx)
	if err != nil {
		return nil, fmt.Errorf("scan statistics: %w", err)
	}
	return d.Sorted(stats), nil
}

// Get returns one company with its statistics and its newest scans, or domain.ErrNotFound.
func (s *Service) Get(ctx context.Context, id string) (domain.Listed, []domain.Scan, error) {
	d, err := s.directory(ctx)
	if err != nil {
		return domain.Listed{}, nil, err
	}
	c, ok := d.ByID(id)
	if !ok {
		return domain.Listed{}, nil, domain.ErrNotFound
	}
	stats, err := s.scans.Stats(ctx)
	if err != nil {
		return domain.Listed{}, nil, fmt.Errorf("scan statistics: %w", err)
	}
	scans, err := s.scans.ByCompany(ctx, id, MaxScans)
	if err != nil {
		return domain.Listed{}, nil, fmt.Errorf("scans of %s: %w", id, err)
	}
	return domain.Listed{Company: c, Stats: stats[id]}, scans, nil
}

// Add validates and stores a company. A website whose host a known company has is refused with
// domain.AlreadyKnownError.
func (s *Service) Add(ctx context.Context, user, rawName, rawURL string) (domain.Company, error) {
	name, err := domain.CheckName(rawName)
	if err != nil {
		return domain.Company{}, err
	}
	website, err := domain.NewWebsite(rawURL)
	if err != nil {
		return domain.Company{}, err
	}
	return s.add(ctx, user, name, website)
}

func (s *Service) add(ctx context.Context, user, name string, website *url.URL) (domain.Company, error) {
	d, err := s.directory(ctx)
	if err != nil {
		return domain.Company{}, err
	}
	c, err := d.NewCompany(name, website, user, s.now())
	if err != nil {
		return domain.Company{}, err
	}
	if err := s.repo.Add(ctx, c); err != nil {
		return domain.Company{}, fmt.Errorf("add company: %w", err)
	}
	return c, nil
}

// Resolve returns the id of the company whose website has site's host, adding the company under
// name (or, without one, its host) when nobody knows the host yet. The analysis context calls it
// when a scan starts.
func (s *Service) Resolve(ctx context.Context, user string, website *url.URL, name string) (string, error) {
	d, err := s.directory(ctx)
	if err != nil {
		return "", err
	}
	host := site.Host(website)
	if c, ok := d.ByHost(host); ok {
		return c.ID, nil
	}
	if name == "" {
		name = host
	}
	if name, err = domain.CheckName(name); err != nil {
		return "", err
	}
	root := &url.URL{Scheme: website.Scheme, Host: website.Host, Path: "/"}
	c, err := s.add(ctx, user, name, root)
	var known domain.AlreadyKnownError
	if errors.As(err, &known) {
		return known.ID, nil // added by a scan that started at the same moment
	}
	if err != nil {
		return "", err
	}
	return c.ID, nil
}
