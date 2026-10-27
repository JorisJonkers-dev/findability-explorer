// Package persistence implements the terminology Reviews port on the sqlc-generated queries: the
// administrators' decisions on the review queue's candidates and the terms accepted ones became.
package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/queries"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// Store is what the repository needs from the Postgres adapter.
type Store interface {
	Queries() *queries.Queries
	InTx(ctx context.Context, fn func(*queries.Queries) error) error
}

// Repository maps rows to decisions and terms.
type Repository struct {
	store Store
}

var _ app.Reviews = (*Repository)(nil)

// New returns a Repository over store.
func New(store Store) *Repository { return &Repository{store: store} }

// Decisions returns every decision, oldest first, an accepted one with its term.
func (r *Repository) Decisions(ctx context.Context) ([]domain.Decision, error) {
	q := r.store.Queries()
	rows, err := q.ListTermDecisions(ctx)
	if err != nil {
		return nil, fmt.Errorf("select decisions: %w", err)
	}
	terms, err := q.ListAcceptedTerms(ctx)
	if err != nil {
		return nil, fmt.Errorf("select accepted terms: %w", err)
	}
	byCandidate := make(map[string]domain.Term, len(terms))
	for _, t := range terms {
		byCandidate[t.CandidateID] = domain.Term{
			Text: t.Term, Language: domain.Language(t.Language), Family: domain.Family(t.Family), Tier: domain.Tier(t.Tier),
			UsedBy: t.UsedBy, Source: t.Source, Origin: domain.OriginReview,
		}
	}
	out := make([]domain.Decision, 0, len(rows))
	for _, row := range rows {
		d := domain.Decision{
			CandidateID: row.CandidateID, Phrase: row.Phrase, Status: domain.Status(row.Status), By: row.DecidedBy,
			At: row.DecidedAt.UTC(),
		}
		if t, ok := byCandidate[row.CandidateID]; ok {
			d.Term = &t
		}
		out = append(out, d)
	}
	return out, nil
}

// Decide stores d and, for an accepted candidate, its term, in one transaction.
func (r *Repository) Decide(ctx context.Context, d domain.Decision) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error {
		n, err := q.InsertTermDecision(ctx, queries.InsertTermDecisionParams{
			CandidateID: d.CandidateID, Phrase: d.Phrase, Status: string(d.Status), DecidedBy: d.By, DecidedAt: d.At,
		})
		if err != nil {
			return fmt.Errorf("insert decision: %w", err)
		}
		if n == 0 {
			return domain.ErrDecided
		}
		if d.Term == nil {
			return nil
		}
		t := d.Term
		usedBy := t.UsedBy
		if usedBy == nil {
			usedBy = []string{}
		}
		if err := q.InsertAcceptedTerm(ctx, queries.InsertAcceptedTermParams{
			CandidateID: d.CandidateID, Term: t.Text, Language: string(t.Language), Family: string(t.Family),
			Tier: string(t.Tier), UsedBy: usedBy, Source: t.Source,
		}); err != nil {
			return fmt.Errorf("insert accepted term: %w", err)
		}
		return nil
	})
}

// Undo deletes the decision on a candidate and, with it, its term.
func (r *Repository) Undo(ctx context.Context, candidateID string) (bool, error) {
	n, err := r.store.Queries().DeleteTermDecision(ctx, candidateID)
	if err != nil {
		return false, fmt.Errorf("delete decision: %w", err)
	}
	return n > 0, nil
}

var _ app.Verdicts = (*Repository)(nil)

// Verdicts returns the language model's verdicts, oldest first.
func (r *Repository) Verdicts(ctx context.Context) ([]domain.Verdict, error) {
	rows, err := r.store.Queries().ListTermVerdicts(ctx)
	if err != nil {
		return nil, fmt.Errorf("select verdicts: %w", err)
	}
	out := make([]domain.Verdict, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Verdict{
			CandidateID: row.CandidateID, Phrase: row.Phrase, Accept: row.Accept, Sure: row.Sure, Reason: row.Reason,
			Term:  domain.Acceptance{Text: row.Term, Language: row.Language, Family: row.Family, Tier: row.Tier},
			Model: row.Model, At: row.ReviewedAt.UTC(),
		})
	}
	return out, nil
}

// SaveVerdicts stores the verdicts of one request and what it consumed, in one transaction. A
// candidate with a verdict already keeps it.
func (r *Repository) SaveVerdicts(ctx context.Context, verdicts []domain.Verdict, usage domain.Usage) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error {
		for _, v := range verdicts {
			if err := q.InsertTermVerdict(ctx, queries.InsertTermVerdictParams{
				CandidateID: v.CandidateID, Phrase: v.Phrase, Accept: v.Accept, Sure: v.Sure, Reason: v.Reason,
				Term: v.Term.Text, Language: v.Term.Language, Family: v.Term.Family, Tier: v.Term.Tier, Model: v.Model, ReviewedAt: v.At,
			}); err != nil {
				return fmt.Errorf("insert verdict: %w", err)
			}
		}
		if err := q.InsertTermReviewUsage(ctx, queries.InsertTermReviewUsageParams{
			InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens, UsedAt: usage.At,
		}); err != nil {
			return fmt.Errorf("insert usage: %w", err)
		}
		return nil
	})
}

// TokensSince is how many tokens asking the language model for verdicts used since t.
func (r *Repository) TokensSince(ctx context.Context, t time.Time) (int64, error) {
	n, err := r.store.Queries().TermReviewTokensSince(ctx, t)
	if err != nil {
		return 0, fmt.Errorf("sum the review's tokens: %w", err)
	}
	return n, nil
}
