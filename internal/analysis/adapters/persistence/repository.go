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

// Findings returns every keyword a finished scan of a company found, oldest scan first. Whether
// the terminology map holds it is for the review to say: a scan made before a phrase was paired
// only with terms in its own language stored it as in the map through another language's term.
func (r *Repository) Findings(ctx context.Context) ([]domain.Finding, error) {
	rows, err := r.store.Queries().CandidateEvidence(ctx)
	if err != nil {
		return nil, fmt.Errorf("select findings: %w", err)
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
	if a.Readiness, err = readiness(ctx, q, id); err != nil {
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

// Sightings returns the map terms each company's newest finished scan found, once per company.
func (r *Repository) Sightings(ctx context.Context) ([]domain.Sighting, error) {
	rows, err := r.store.Queries().KnownTermSightings(ctx)
	if err != nil {
		return nil, fmt.Errorf("known-term sightings: %w", err)
	}
	out := make([]domain.Sighting, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Sighting{CompetitorID: row.CompetitorID, Term: row.Term, Language: row.Language, PageLanguages: row.PageLanguages})
	}
	return domain.SightedInTheirLanguage(out), nil
}

// MatchedProducts returns the products matched on each company's newest finished scan whose
// matching ran, in the order the model named them.
func (r *Repository) MatchedProducts(ctx context.Context) ([]domain.MatchedProduct, error) {
	rows, err := r.store.Queries().LatestMatchedProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("matched products: %w", err)
	}
	out := make([]domain.MatchedProduct, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.MatchedProduct{CompetitorID: row.CompetitorID, Product: row.Product, Family: row.Family})
	}
	return out, nil
}

// TokensSince sums the language-model tokens of analyses created at or after since.
func (r *Repository) TokensSince(ctx context.Context, since time.Time) (int64, error) {
	n, err := r.store.Queries().TokensSince(ctx, since)
	if err != nil {
		return 0, fmt.Errorf("sum tokens: %w", err)
	}
	return n, nil
}

// TokensPerScan returns how many scans the language model answered for, and the tokens one of them
// used on average; scans with matching off or skipped used none and are left out.
func (r *Repository) TokensPerScan(ctx context.Context) (scans int, average int64, err error) {
	row, err := r.store.Queries().TokensPerMatchedScan(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("average tokens per scan: %w", err)
	}
	return int(row.Scans), row.Average, nil
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

// SaveReadiness stores the AI readiness, its crawler rules and its raw files in one transaction.
func (r *Repository) SaveReadiness(ctx context.Context, id uuid.UUID, rd domain.Readiness, raw domain.RawFiles) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error {
		// A check again replaces what the scan, or an earlier check, recorded.
		if err := q.DeleteAnalysisReadiness(ctx, id); err != nil {
			return fmt.Errorf("delete readiness: %w", err)
		}
		if err := q.AddAnalysisReadiness(ctx, queries.AddAnalysisReadinessParams{
			AnalysisID: id, Robots: string(rd.Robots), SitemapDeclared: rd.Sitemap.Declared,
			SitemapFiles: int64(rd.Sitemap.Files), SitemapAddresses: int64(rd.Sitemap.Addresses),
			SitemapLanguages: clipAll(rd.Sitemap.Languages, 3, 16), LlmsPresent: rd.LLMs.Present,
			LlmsBytes: int64(rd.LLMs.Bytes), LlmsLinks: int64(rd.LLMs.Links), LlmsFullPresent: rd.LLMsFull,
			Markdown: rd.Markdown, HomeRead: rd.Home, StructuredData: clipAll(rd.StructuredData, 50, 100),
			Languages: clipAll(rd.Languages, 50, 16), Cms: clip(rd.Stack.CMS, 60), SeoPlugin: clip(rd.Stack.SEOPlugin, 60),
			BotProtection: clip(rd.Stack.BotProtection, 60), CheckedAt: optionalTime(rd.CheckedAt),
			SitemapPagesRead: coverageRead(rd.Sitemap.Coverage), SitemapPagesListed: coverageListed(rd.Sitemap.Coverage),
		}); err != nil {
			return fmt.Errorf("insert readiness: %w", err)
		}
		for i, b := range rd.Bots[:min(len(rd.Bots), 20)] {
			if err := q.AddAnalysisBotRule(ctx, queries.AddAnalysisBotRuleParams{
				AnalysisID: id, Position: int64(i), Bot: b.Bot, Access: string(b.Access), Named: b.Named,
			}); err != nil {
				return fmt.Errorf("insert crawler rule: %w", err)
			}
		}
		if err := q.AddAnalysisReadinessFiles(ctx, queries.AddAnalysisReadinessFilesParams{
			AnalysisID: id, RobotsTxt: raw.Robots, RobotsTruncated: raw.RobotsTruncated, LlmsTxt: raw.LLMs,
			LlmsTruncated: raw.LLMsTruncated, LlmsFullTxt: raw.LLMsFull, LlmsFullTruncated: raw.LLMsFullTruncated,
		}); err != nil {
			return fmt.Errorf("insert readiness files: %w", err)
		}
		return nil
	})
}

// SaveSitemapCoverage records the sitemaps' coverage on the readiness the scan recorded, or
// returns domain.ErrNoReadiness when it recorded none.
func (r *Repository) SaveSitemapCoverage(ctx context.Context, id uuid.UUID, c domain.SitemapCoverage) error {
	n, err := r.store.Queries().SetSitemapCoverage(ctx, queries.SetSitemapCoverageParams{
		AnalysisID: id, SitemapPagesRead: coverageRead(&c), SitemapPagesListed: coverageListed(&c),
	})
	if err != nil {
		return fmt.Errorf("update sitemap coverage: %w", err)
	}
	if n == 0 {
		return domain.ErrNoReadiness
	}
	return nil
}

// coverageRead and coverageListed are a coverage's counts, or null when it was not measured.
func coverageRead(c *domain.SitemapCoverage) pgtype.Int8 {
	if c == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: int64(c.Pages), Valid: true}
}

func coverageListed(c *domain.SitemapCoverage) pgtype.Int8 {
	if c == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: int64(c.Listed), Valid: true}
}

// ReadinessFiles returns the raw files of an analysis's readiness, or domain.ErrNoReadiness.
func (r *Repository) ReadinessFiles(ctx context.Context, id uuid.UUID) (domain.RawFiles, error) {
	row, err := r.store.Queries().GetAnalysisReadinessFiles(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RawFiles{}, domain.ErrNoReadiness
	}
	if err != nil {
		return domain.RawFiles{}, fmt.Errorf("select readiness files: %w", err)
	}
	return domain.RawFiles{
		Robots: row.RobotsTxt, RobotsTruncated: row.RobotsTruncated, LLMs: row.LlmsTxt, LLMsTruncated: row.LlmsTruncated,
		LLMsFull: row.LlmsFullTxt, LLMsFullTruncated: row.LlmsFullTruncated,
	}, nil
}

// LatestReadiness returns, per company, its newest analysis that recorded its readiness.
func (r *Repository) LatestReadiness(ctx context.Context) (map[string]domain.Analysis, error) {
	q := r.store.Queries()
	rows, err := q.LatestReadinessPerCompetitor(ctx)
	if err != nil {
		return nil, fmt.Errorf("latest readiness per company: %w", err)
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	bots, err := botRules(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.Analysis, len(rows))
	for _, row := range rows {
		a := toAnalysis(queries.ListAnalysesRow{
			ID: row.ID, Url: row.Url, MaxPages: row.MaxPages, Status: row.Status, Failure: row.Failure,
			PagesCrawled: row.PagesCrawled, Matching: row.Matching, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt,
			FinishedAt: row.FinishedAt, CompetitorID: optionalText(row.CompetitorID), MatchingModel: row.MatchingModel,
			ScanNumber: row.ScanNumber,
		})
		rd := toReadiness(queries.AnalysisReadiness{
			Robots: row.Robots, SitemapDeclared: row.SitemapDeclared, SitemapFiles: row.SitemapFiles,
			SitemapAddresses: row.SitemapAddresses, SitemapLanguages: row.SitemapLanguages, LlmsPresent: row.LlmsPresent,
			LlmsBytes: row.LlmsBytes, LlmsLinks: row.LlmsLinks, LlmsFullPresent: row.LlmsFullPresent, Markdown: row.Markdown,
			HomeRead: row.HomeRead, StructuredData: row.StructuredData, Languages: row.Languages, Cms: row.Cms,
			SeoPlugin: row.SeoPlugin, BotProtection: row.BotProtection, CheckedAt: row.CheckedAt,
			SitemapPagesRead: row.SitemapPagesRead, SitemapPagesListed: row.SitemapPagesListed,
		}, bots[row.ID])
		a.Readiness = &rd
		out[row.CompetitorID] = a
	}
	return out, nil
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

// readiness reads an analysis's readiness, or nil when it has none.
func readiness(ctx context.Context, q *queries.Queries, id uuid.UUID) (*domain.Readiness, error) {
	rows, err := q.GetAnalysisReadiness(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("select readiness: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil //nolint:nilnil // no readiness is a valid answer
	}
	bots, err := botRules(ctx, q, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	rd := toReadiness(rows[0], bots[id])
	return &rd, nil
}

func botRules(ctx context.Context, q *queries.Queries, ids []uuid.UUID) (map[uuid.UUID][]domain.BotRule, error) {
	rows, err := q.ListAnalysisBotRules(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("select crawler rules: %w", err)
	}
	out := map[uuid.UUID][]domain.BotRule{}
	for _, r := range rows {
		out[r.AnalysisID] = append(out[r.AnalysisID], domain.BotRule{Bot: r.Bot, Access: domain.Access(r.Access), Named: r.Named})
	}
	return out, nil
}

func toReadiness(r queries.AnalysisReadiness, bots []domain.BotRule) domain.Readiness {
	return domain.Readiness{
		Robots: domain.RobotsStatus(r.Robots), Bots: bots,
		Sitemap: domain.SitemapSummary{
			Declared: r.SitemapDeclared, Files: int(r.SitemapFiles), Addresses: int(r.SitemapAddresses), Languages: r.SitemapLanguages,
			Coverage: toCoverage(r.SitemapPagesRead, r.SitemapPagesListed),
		},
		LLMs:     domain.LLMsFacts{Present: r.LlmsPresent, Bytes: int(r.LlmsBytes), Links: int(r.LlmsLinks)},
		LLMsFull: r.LlmsFullPresent, Markdown: r.Markdown, Home: r.HomeRead, StructuredData: r.StructuredData,
		Languages: r.Languages, Stack: domain.Stack{CMS: r.Cms, SEOPlugin: r.SeoPlugin, BotProtection: r.BotProtection},
		CheckedAt: r.CheckedAt.Time,
	}
}

// toCoverage is the stored coverage, or nil when it was not measured.
func toCoverage(read, listed pgtype.Int8) *domain.SitemapCoverage {
	if !read.Valid || !listed.Valid {
		return nil
	}
	return &domain.SitemapCoverage{Pages: int(read.Int64), Listed: int(listed.Int64)}
}

// optionalTime is t, or null when it is zero.
func optionalTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

// clipAll keeps at most n items, each at most chars characters.
func clipAll(items []string, n, chars int) []string {
	out := make([]string, 0, min(len(items), n))
	for _, s := range items[:min(len(items), n)] {
		out = append(out, clip(s, chars))
	}
	return out
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
