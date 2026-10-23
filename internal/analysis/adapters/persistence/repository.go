// Package persistence implements the analysis Repository port on the sqlc-generated queries.
package persistence

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/queries"
)

// Store is what the repository needs from the Postgres adapter: queries outside a transaction,
// and a way to run several writes in one.
type Store interface {
	Queries() *queries.Queries
	InTx(ctx context.Context, fn func(*queries.Queries) error) error
}

// Repository maps rows to domain analyses, so the domain never sees a generated type.
type Repository struct {
	store Store
}

var _ domain.Repository = (*Repository)(nil)

// New returns a Repository over store.
func New(store Store) *Repository {
	return &Repository{store: store}
}

// Create stores a queued analysis and returns its number among its company's scans.
func (r *Repository) Create(ctx context.Context, a domain.Analysis) (int, error) {
	n, err := r.store.Queries().CreateAnalysis(ctx, queries.CreateAnalysisParams{
		ID: a.ID, Url: a.URL, MaxPages: int64(a.MaxPages), CreatedBy: a.CreatedBy, CreatedAt: a.CreatedAt,
		CompetitorID: optionalText(a.CompetitorID), Rescan: a.Rescan,
	})
	if err != nil {
		return 0, fmt.Errorf("insert analysis: %w", err)
	}
	return int(n), nil
}

// CountActive counts a company's queued and running analyses.
func (r *Repository) CountActive(ctx context.Context, competitorID string) (int, error) {
	n, err := r.store.Queries().CountActiveAnalysesOfCompetitor(ctx, optionalText(competitorID))
	if err != nil {
		return 0, fmt.Errorf("count active analyses: %w", err)
	}
	return int(n), nil
}

// DeleteByCompetitor deletes every analysis of a company, with its results.
func (r *Repository) DeleteByCompetitor(ctx context.Context, competitorID string) error {
	if err := r.store.Queries().DeleteAnalysesOfCompetitor(ctx, optionalText(competitorID)); err != nil {
		return fmt.Errorf("delete analyses of %s: %w", competitorID, err)
	}
	return nil
}

// FirstScans returns, per phrase, the number of the company's first finished scan that found it.
func (r *Repository) FirstScans(ctx context.Context, competitorID string) (map[domain.PhraseKey]int, error) {
	rows, err := r.store.Queries().KeywordFirstScans(ctx, optionalText(competitorID))
	if err != nil {
		return nil, fmt.Errorf("first scans of %s: %w", competitorID, err)
	}
	out := make(map[domain.PhraseKey]int, len(rows))
	for _, row := range rows {
		out[domain.PhraseKey{Phrase: row.Phrase, Language: row.Language}] = int(row.FirstScan)
	}
	return out, nil
}

// UnmappedFindings returns every keyword a finished scan of a company found outside the
// terminology map, oldest scan first.
func (r *Repository) UnmappedFindings(ctx context.Context) ([]domain.Finding, error) {
	rows, err := r.store.Queries().CandidateEvidence(ctx)
	if err != nil {
		return nil, fmt.Errorf("select unmapped findings: %w", err)
	}
	out := make([]domain.Finding, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Finding{
			AnalysisID: row.AnalysisID, CompetitorID: row.CompetitorID, URL: row.Url, StartedAt: row.CreatedAt.UTC(),
			Keyword: domain.Keyword{
				Phrase: row.Phrase, Language: row.Language, Score: row.Score, Pages: int(row.Pages),
				Classification: domain.Classification{Tier: row.Tier, Family: row.Family, Closest: row.ClosestTerm},
			},
		})
	}
	return out, nil
}

// Get returns the analysis with every result.
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (domain.Analysis, error) {
	q := r.store.Queries()
	row, err := q.GetAnalysis(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Analysis{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Analysis{}, fmt.Errorf("select analysis: %w", err)
	}
	a := toAnalysis(queries.ListAnalysesRow(row))
	if a.Pages, err = pages(ctx, q, id); err != nil {
		return domain.Analysis{}, err
	}
	if a.Keywords, err = keywords(ctx, q, id); err != nil {
		return domain.Analysis{}, err
	}
	if a.KnownTerms, err = knownTerms(ctx, q, id); err != nil {
		return domain.Analysis{}, err
	}
	if a.Equivalents, err = equivalents(ctx, q, id); err != nil {
		return domain.Analysis{}, err
	}
	return a, nil
}

// List returns at most limit analyses without results, newest first.
func (r *Repository) List(ctx context.Context, limit int) ([]domain.Analysis, error) {
	if limit < 0 || limit > math.MaxInt32 {
		return nil, fmt.Errorf("select analyses: limit %d out of range", limit)
	}
	rows, err := r.store.Queries().ListAnalyses(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("select analyses: %w", err)
	}
	out := make([]domain.Analysis, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAnalysis(row))
	}
	return out, nil
}

// ListByCompetitor returns at most limit analyses of one company, newest first.
func (r *Repository) ListByCompetitor(ctx context.Context, competitorID string, limit int) ([]domain.Analysis, error) {
	if limit < 0 || limit > math.MaxInt32 {
		return nil, fmt.Errorf("select analyses: limit %d out of range", limit)
	}
	rows, err := r.store.Queries().ListAnalysesByCompetitor(ctx, queries.ListAnalysesByCompetitorParams{
		CompetitorID: optionalText(competitorID), Limit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("select analyses of %s: %w", competitorID, err)
	}
	out := make([]domain.Analysis, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAnalysis(queries.ListAnalysesRow(row)))
	}
	return out, nil
}

// CompetitorStats counts each company's analyses and returns its newest one.
func (r *Repository) CompetitorStats(ctx context.Context) (map[string]domain.CompetitorStats, error) {
	q := r.store.Queries()
	counts, err := q.AnalysisStatsByCompetitor(ctx)
	if err != nil {
		return nil, fmt.Errorf("count analyses per company: %w", err)
	}
	latest, err := q.LatestAnalysisPerCompetitor(ctx)
	if err != nil {
		return nil, fmt.Errorf("latest analysis per company: %w", err)
	}
	out := make(map[string]domain.CompetitorStats, len(counts))
	for _, c := range counts {
		out[c.CompetitorID] = domain.CompetitorStats{Count: int(c.Scans)}
	}
	for _, l := range latest {
		s := out[l.CompetitorID]
		s.Last = toAnalysis(queries.ListAnalysesRow{
			ID: l.ID, Url: l.Url, MaxPages: l.MaxPages, Status: l.Status, Failure: l.Failure,
			PagesCrawled: l.PagesCrawled, Matching: l.Matching, CreatedBy: l.CreatedBy, CreatedAt: l.CreatedAt,
			FinishedAt: l.FinishedAt, CompetitorID: optionalText(l.CompetitorID), MatchingModel: l.MatchingModel,
			ScanNumber: l.ScanNumber,
		})
		out[l.CompetitorID] = s
	}
	return out, nil
}

// CountCreatedSince counts the analyses user started at or after since.
func (r *Repository) CountCreatedSince(ctx context.Context, user string, since time.Time) (int, error) {
	n, err := r.store.Queries().CountAnalysesCreatedSince(ctx, queries.CountAnalysesCreatedSinceParams{
		CreatedBy: user, CreatedAt: since,
	})
	if err != nil {
		return 0, fmt.Errorf("count analyses: %w", err)
	}
	return int(n), nil
}

// TokensSince sums the language-model tokens of analyses created at or after since.
func (r *Repository) TokensSince(ctx context.Context, since time.Time) (int64, error) {
	n, err := r.store.Queries().TokensSince(ctx, since)
	if err != nil {
		return 0, fmt.Errorf("sum tokens: %w", err)
	}
	return n, nil
}

// Queued returns the ids of every queued analysis, oldest first.
func (r *Repository) Queued(ctx context.Context) ([]uuid.UUID, error) {
	ids, err := r.store.Queries().QueuedAnalyses(ctx)
	if err != nil {
		return nil, fmt.Errorf("select queued analyses: %w", err)
	}
	return ids, nil
}

// FailRunning marks every running analysis failed.
func (r *Repository) FailRunning(ctx context.Context, reason string, at time.Time) error {
	if err := r.store.Queries().FailRunningAnalyses(ctx, queries.FailRunningAnalysesParams{
		Failure: text(reason), FinishedAt: timestamp(at),
	}); err != nil {
		return fmt.Errorf("fail running analyses: %w", err)
	}
	return nil
}

// MarkRunning moves a queued analysis to running.
func (r *Repository) MarkRunning(ctx context.Context, id uuid.UUID) error {
	if err := r.store.Queries().MarkAnalysisRunning(ctx, id); err != nil {
		return fmt.Errorf("mark analysis running: %w", err)
	}
	return nil
}

// AddPage stores the next crawled page and raises the crawled count, in one statement.
func (r *Repository) AddPage(ctx context.Context, id uuid.UUID, p domain.PageSummary) error {
	if err := r.store.Queries().AddAnalysisPage(ctx, queries.AddAnalysisPageParams{
		ID: id, Url: clip(p.URL, 2000), Title: clip(p.Title, 500), Description: clip(p.Description, 1000),
		Language: clip(p.Language, 16), WordCount: int64(p.WordCount), Family: optionalText(p.Family),
	}); err != nil {
		return fmt.Errorf("insert page: %w", err)
	}
	return nil
}

// SaveFindings stores the keywords and known terms in one transaction.
func (r *Repository) SaveFindings(ctx context.Context, id uuid.UUID, kws []domain.Keyword, known []domain.KnownTerm) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error {
		for i, k := range kws {
			if err := q.AddAnalysisKeyword(ctx, queries.AddAnalysisKeywordParams{
				AnalysisID: id, Position: int64(i), Phrase: clip(k.Phrase, 200), Language: clip(k.Language, 16),
				Score: k.Score, Pages: int64(k.Pages), Tier: optionalText(k.Tier), Family: optionalText(k.Family),
				ClosestTerm: optionalText(clip(k.Closest, 200)), InMap: pgtype.Bool{Bool: k.InMap, Valid: true},
			}); err != nil {
				return fmt.Errorf("insert keyword: %w", err)
			}
		}
		for i, k := range known {
			if err := q.AddAnalysisKnownTerm(ctx, queries.AddAnalysisKnownTermParams{
				AnalysisID: id, Position: int64(i), Term: clip(k.Text, 200), Language: k.Language,
				Family: k.Family, Tier: k.Tier, Occurrences: int64(k.Occurrences), UsedByTribelt: k.UsedByTribelt,
			}); err != nil {
				return fmt.Errorf("insert known term: %w", err)
			}
		}
		return nil
	})
}

// SaveMatching stores the matching outcome, the model asked, its token usage and the equivalents
// in one transaction.
func (r *Repository) SaveMatching(ctx context.Context, id uuid.UUID, m domain.Matching, model string, eqs []domain.Equivalent, u domain.Usage) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error {
		if err := q.SetAnalysisMatching(ctx, queries.SetAnalysisMatchingParams{
			ID: id, Matching: string(m), LlmInputTokens: u.InputTokens, LlmOutputTokens: u.OutputTokens,
			MatchingModel: optionalText(clip(model, 300)),
		}); err != nil {
			return fmt.Errorf("update matching: %w", err)
		}
		for i, e := range eqs[:min(len(eqs), 50)] {
			if err := q.AddAnalysisEquivalent(ctx, queries.AddAnalysisEquivalentParams{
				AnalysisID: id, Position: int64(i), Product: clip(e.Product, 300), Family: e.Family,
				Reason: clip(e.Reason, 1000),
			}); err != nil {
				return fmt.Errorf("insert equivalent: %w", err)
			}
			for j, s := range e.SearchTerms[:min(len(e.SearchTerms), 30)] {
				if err := q.AddAnalysisEquivalentTerm(ctx, queries.AddAnalysisEquivalentTermParams{
					AnalysisID: id, EquivalentPosition: int64(i), Position: int64(j),
					Term: clip(s.Term, 200), Language: clip(s.Language, 16),
				}); err != nil {
					return fmt.Errorf("insert equivalent term: %w", err)
				}
			}
		}
		return nil
	})
}

// Finish marks a running analysis done.
func (r *Repository) Finish(ctx context.Context, id uuid.UUID, at time.Time) error {
	if err := r.store.Queries().FinishAnalysis(ctx, queries.FinishAnalysisParams{ID: id, FinishedAt: timestamp(at)}); err != nil {
		return fmt.Errorf("finish analysis: %w", err)
	}
	return nil
}

// Fail marks a queued or running analysis failed.
func (r *Repository) Fail(ctx context.Context, id uuid.UUID, reason string, at time.Time) error {
	if err := r.store.Queries().FailAnalysis(ctx, queries.FailAnalysisParams{
		ID: id, Failure: text(clip(reason, 500)), FinishedAt: timestamp(at),
	}); err != nil {
		return fmt.Errorf("fail analysis: %w", err)
	}
	return nil
}

func toAnalysis(row queries.ListAnalysesRow) domain.Analysis {
	a := domain.Analysis{
		ID: row.ID, URL: row.Url, MaxPages: int(row.MaxPages), Status: domain.Status(row.Status),
		Failure: row.Failure.String, PagesCrawled: int(row.PagesCrawled), Matching: domain.Matching(row.Matching),
		CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt.UTC(), CompetitorID: row.CompetitorID.String,
		MatchingModel: row.MatchingModel.String, Number: int(row.ScanNumber),
	}
	if row.FinishedAt.Valid {
		a.FinishedAt = row.FinishedAt.Time.UTC()
	}
	return a
}

func pages(ctx context.Context, q *queries.Queries, id uuid.UUID) ([]domain.PageSummary, error) {
	rows, err := q.ListAnalysisPages(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("select pages: %w", err)
	}
	out := make([]domain.PageSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.PageSummary{
			URL: r.Url, Title: r.Title, Description: r.Description, Language: r.Language, WordCount: int(r.WordCount),
			Family: r.Family,
		})
	}
	return out, nil
}

func keywords(ctx context.Context, q *queries.Queries, id uuid.UUID) ([]domain.Keyword, error) {
	rows, err := q.ListAnalysisKeywords(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("select keywords: %w", err)
	}
	out := make([]domain.Keyword, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Keyword{
			Phrase: r.Phrase, Language: r.Language, Score: r.Score, Pages: int(r.Pages),
			Classification: domain.Classification{Tier: r.Tier, Family: r.Family, Closest: r.ClosestTerm, InMap: r.InMap},
		})
	}
	return out, nil
}

func knownTerms(ctx context.Context, q *queries.Queries, id uuid.UUID) ([]domain.KnownTerm, error) {
	rows, err := q.ListAnalysisKnownTerms(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("select known terms: %w", err)
	}
	out := make([]domain.KnownTerm, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.KnownTerm{
			GlossaryTerm: domain.GlossaryTerm{Text: r.Term, Language: r.Language, Family: r.Family, Tier: r.Tier, UsedByTribelt: r.UsedByTribelt},
			Occurrences:  int(r.Occurrences),
		})
	}
	return out, nil
}

func equivalents(ctx context.Context, q *queries.Queries, id uuid.UUID) ([]domain.Equivalent, error) {
	rows, err := q.ListAnalysisEquivalents(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("select equivalents: %w", err)
	}
	terms, err := q.ListAnalysisEquivalentTerms(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("select equivalent terms: %w", err)
	}
	byPosition := map[int64][]domain.SearchTerm{}
	for _, t := range terms {
		byPosition[t.EquivalentPosition] = append(byPosition[t.EquivalentPosition], domain.SearchTerm{Term: t.Term, Language: t.Language})
	}
	out := make([]domain.Equivalent, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Equivalent{Product: r.Product, Family: r.Family, Reason: r.Reason, SearchTerms: byPosition[r.Position]})
	}
	return out, nil
}

func text(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }

// optionalText is s, or NULL when s is empty.
func optionalText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

func timestamp(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

// clip shortens s to at most n characters, so text from a crawled site always fits its column.
func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
