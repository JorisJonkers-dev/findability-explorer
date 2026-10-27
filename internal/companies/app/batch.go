package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/site"
)

// MaxBatch is the most companies one batch adds.
const MaxBatch = 100

// ErrBatchSize is returned for a batch with no company, or more than MaxBatch.
var ErrBatchSize = fmt.Errorf("add 1 to %d companies at once", MaxBatch)

// errNoBatchPorts is returned when the composition root never handed over the batch's ports.
var errNoBatchPorts = errors.New("adding several companies is not wired: call WithBatch")

// Candidates is the port to the prompt-test context: a competitor candidate the answers named,
// recorded as added once its company is.
type Candidates interface {
	RecordAdded(ctx context.Context, user, candidateID, companyID string) error
}

// Scanner is the port to the analysis context: scans of the companies just added.
type Scanner interface {
	Scan(ctx context.Context, user string, ids []string) (int, error)
}

// NewItem is one company of a batch: its name, its website, and the competitor candidate it
// comes from, if any.
type NewItem struct {
	Name        string
	Website     string
	CandidateID string
}

// Outcome is what became of one item of a batch.
type Outcome string

const (
	OutcomeAdded     Outcome = "added"
	OutcomeDuplicate Outcome = "duplicate"
	OutcomeRejected  Outcome = "rejected"
)

// ItemResult is one item's outcome: the company added or repeated, or why it was refused.
type ItemResult struct {
	Name      string
	Outcome   Outcome
	CompanyID string
	Reason    string
}

// Batch is what a batch did: each item's outcome, and how many scans it started.
type Batch struct {
	Items []ItemResult
	Scans int
}

// AddMany adds several companies in one go, in order: an item whose website or name a known
// company (or an earlier item) has is skipped as a duplicate, and one without a valid name or
// website is rejected. An item from a competitor candidate records the candidate as added. With
// scan, the added companies are scanned straight away.
func (s *Service) AddMany(ctx context.Context, user string, items []NewItem, scan bool) (Batch, error) {
	if len(items) == 0 || len(items) > MaxBatch {
		return Batch{}, ErrBatchSize
	}
	if s.candidates == nil || s.scanner == nil {
		return Batch{}, errNoBatchPorts
	}
	d, err := s.directory(ctx)
	if err != nil {
		return Batch{}, err
	}
	out := Batch{Items: make([]ItemResult, 0, len(items))}
	var added []string
	for _, item := range items {
		r, c, err := s.addOne(ctx, user, d, item)
		if err != nil {
			return Batch{}, err
		}
		if r.Outcome == OutcomeAdded {
			d = d.With(c)
			added = append(added, c.ID)
		}
		out.Items = append(out.Items, r)
	}
	if scan && len(added) > 0 {
		if out.Scans, err = s.scanner.Scan(ctx, user, added); err != nil {
			return Batch{}, fmt.Errorf("scan the added companies: %w", err)
		}
	}
	return out, nil
}

// addOne adds one item of a batch, or says why not; err is for a failure to record it.
func (s *Service) addOne(ctx context.Context, user string, d domain.Directory, item NewItem) (ItemResult, domain.Company, error) {
	r := ItemResult{Name: item.Name, Outcome: OutcomeRejected}
	name, err := domain.CheckName(item.Name)
	if err != nil {
		r.Reason = "A company needs a name of at most 200 characters."
		return r, domain.Company{}, nil //nolint:nilerr // the reason is the outcome
	}
	r.Name = name
	if item.Website == "" {
		r.Reason = "Add its website first."
		return r, domain.Company{}, nil
	}
	website, err := domain.NewWebsite(item.Website)
	if err != nil {
		r.Reason = "Its website is no http or https address on a public host."
		return r, domain.Company{}, nil //nolint:nilerr // the reason is the outcome
	}
	if known, ok := d.Duplicate(name, site.Host(website)); ok {
		return ItemResult{Name: name, Outcome: OutcomeDuplicate, CompanyID: known.ID, Reason: known.Name + " is a competitor already."}, domain.Company{}, nil
	}
	c := d.Fresh(name, website, user, s.now())
	if err := s.repo.Add(ctx, c); err != nil {
		return ItemResult{}, domain.Company{}, fmt.Errorf("add company: %w", err)
	}
	if item.CandidateID != "" {
		if err := s.candidates.RecordAdded(ctx, user, item.CandidateID, c.ID); err != nil && !errors.Is(err, ErrCandidateGone) {
			return ItemResult{}, domain.Company{}, fmt.Errorf("record the candidate: %w", err)
		}
	}
	return ItemResult{Name: name, Outcome: OutcomeAdded, CompanyID: c.ID}, c, nil
}

// ErrCandidateGone is what Candidates returns for a candidate that was decided or is unknown:
// the company is added all the same.
var ErrCandidateGone = errors.New("the candidate is no longer pending")
