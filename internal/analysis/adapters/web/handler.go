// Package web is the analysis context's inbound HTTP adapter: it implements the analysis
// operations of the generated ogen Handler and maps between the contract's types and the domain's.
package web

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
)

// UseCases is what the adapter needs from the application layer.
type UseCases interface {
	Start(ctx context.Context, user, rawURL string, maxPages int, companyName string) (domain.Analysis, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Analysis, error)
	List(ctx context.Context, limit int) ([]domain.Analysis, error)
	Rescan(ctx context.Context, user string, pick app.Pick, maxPages int) ([]domain.Analysis, []app.Skipped, error)
	Profile(ctx context.Context, competitorID string) (domain.Profile, error)
	Compare(ctx context.Context, competitorID string, from, to uuid.UUID) (domain.Comparison, error)
	ReadinessTable(ctx context.Context) ([]domain.ReadinessRow, error)
	ReadinessFiles(ctx context.Context, id uuid.UUID) (domain.RawFiles, error)
}

// Handler implements the analysis operations. Business rules stay in the use cases.
type Handler struct {
	uc UseCases
}

// New returns a Handler calling uc.
func New(uc UseCases) *Handler {
	return &Handler{uc: uc}
}

// StartAnalysis implements startAnalysis.
func (h *Handler) StartAnalysis(ctx context.Context, req *oas.NewAnalysis) (oas.StartAnalysisRes, error) {
	a, err := h.uc.Start(ctx, httpx.Subject(ctx), req.URL.String(), int(req.MaxPages.Or(0)), req.CompanyName.Or(""))
	switch {
	case errors.Is(err, domain.ErrInvalidTarget):
		return &oas.StartAnalysisUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity,
			Response:   httpx.Problem(http.StatusUnprocessableEntity, domain.ErrInvalidTarget.Error()+"."),
		}, nil
	case errors.Is(err, domain.ErrInvalidMaxPages):
		return &oas.StartAnalysisUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity,
			Response:   httpx.Problem(http.StatusUnprocessableEntity, domain.ErrInvalidMaxPages.Error()+"."),
		}, nil
	case errors.Is(err, app.ErrScanActive):
		return &oas.StartAnalysisConflict{
			StatusCode: http.StatusConflict,
			Response:   httpx.Problem(http.StatusConflict, capitalise(err.Error())),
		}, nil
	case err != nil:
		return nil, err
	}
	out := toAnalysis(a)
	return &out, nil
}

// GetAnalysis implements getAnalysis.
func (h *Handler) GetAnalysis(ctx context.Context, params oas.GetAnalysisParams) (oas.GetAnalysisRes, error) {
	a, err := h.uc.Get(ctx, params.AnalysisId)
	if errors.Is(err, domain.ErrNotFound) {
		return &oas.GetAnalysisNotFound{
			StatusCode: http.StatusNotFound,
			Response:   httpx.Problem(http.StatusNotFound, "No analysis has this id."),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	out := toAnalysis(a)
	return &out, nil
}

// ListAnalyses implements listAnalyses.
func (h *Handler) ListAnalyses(ctx context.Context, params oas.ListAnalysesParams) (oas.ListAnalysesRes, error) {
	list, err := h.uc.List(ctx, int(params.Limit.Or(0)))
	if err != nil {
		return nil, err
	}
	items := make([]oas.AnalysisSummary, 0, len(list))
	for _, a := range list {
		items = append(items, toSummary(a))
	}
	return &oas.AnalysisList{Items: items}, nil
}

// StartRescans implements startRescans.
func (h *Handler) StartRescans(ctx context.Context, req *oas.RescanRequest) (oas.StartRescansRes, error) {
	ids := make([]string, 0, len(req.CompetitorIds))
	for _, id := range req.CompetitorIds {
		ids = append(ids, string(id))
	}
	pick := app.Pick{IDs: ids, All: req.All.Or(false), Unscanned: req.Unscanned.Or(false)}
	queued, skipped, err := h.uc.Rescan(ctx, httpx.Subject(ctx), pick, int(req.MaxPages.Or(0)))
	switch {
	case errors.Is(err, app.ErrInvalidRescan), errors.Is(err, domain.ErrInvalidMaxPages):
		return &oas.StartRescansUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, capitalise(firstRule(err))),
		}, nil
	case err != nil:
		return nil, err
	}
	out := &oas.RescanBatch{Queued: make([]oas.AnalysisSummary, 0, len(queued)), Skipped: make([]oas.RescanSkip, 0, len(skipped))}
	for _, a := range queued {
		out.Queued = append(out.Queued, toSummary(a))
	}
	for _, s := range skipped {
		out.Skipped = append(out.Skipped, oas.RescanSkip{CompetitorId: oas.CompetitorId(s.CompetitorID), Name: s.Name, Reason: s.Reason})
	}
	return out, nil
}

// firstRule is the message of the rule err breaks, without the details wrapped around it.
func firstRule(err error) string {
	for _, known := range []error{app.ErrInvalidRescan, domain.ErrInvalidMaxPages} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	return err.Error()
}

// GetCompetitorProfile implements getCompetitorProfile.
func (h *Handler) GetCompetitorProfile(ctx context.Context, params oas.GetCompetitorProfileParams) (oas.GetCompetitorProfileRes, error) {
	p, err := h.uc.Profile(ctx, string(params.CompetitorId))
	if errors.Is(err, app.ErrUnknownCompany) {
		return &oas.GetCompetitorProfileNotFound{StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "No company has this id.")}, nil
	}
	if err != nil {
		return nil, err
	}
	out := &oas.CompetitorProfile{
		CompetitorId: params.CompetitorId, FinishedScans: int32(p.FinishedScans), //nolint:gosec // scans of one company
		Languages: p.Languages[:min(len(p.Languages), maxLanguages)], Groups: make([]oas.ProfileGroupTerms, 0, len(p.Groups)),
	}
	if p.Scan != nil {
		out.Scan = oas.NewOptAnalysisSummary(toSummary(*p.Scan))
	}
	for _, g := range p.Groups {
		terms := make([]oas.ProfileTerm, 0, len(g.Terms))
		for _, t := range g.Terms {
			terms = append(terms, toProfileTerm(t))
		}
		out.Groups = append(out.Groups, oas.ProfileGroupTerms{Group: oas.ProfileGroup(g.Group), Terms: terms})
	}
	return out, nil
}

// maxLanguages is the most languages a profile lists, as the contract allows.
const maxLanguages = 20

// CompareScans implements compareScans.
func (h *Handler) CompareScans(ctx context.Context, params oas.CompareScansParams) (oas.CompareScansRes, error) {
	c, err := h.uc.Compare(ctx, string(params.CompetitorId), params.From.Or(uuid.Nil), params.To.Or(uuid.Nil))
	switch {
	case errors.Is(err, app.ErrUnknownCompany):
		return &oas.CompareScansNotFound{StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "No company has this id.")}, nil
	case errors.Is(err, domain.ErrNotComparable):
		return &oas.CompareScansUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, capitalise(err.Error())),
		}, nil
	case err != nil:
		return nil, err
	}
	out := &oas.ScanComparison{From: toSummary(c.From), To: toSummary(c.To), Groups: make([]oas.ComparisonGroup, 0, len(c.Groups))}
	if c.Readiness != nil {
		out.Readiness = make([]oas.ReadinessChange, 0, len(c.Readiness))
		for _, r := range c.Readiness {
			out.Readiness = append(out.Readiness, oas.ReadinessChange{Field: r.Field, From: r.From, To: r.To})
		}
	}
	for _, g := range c.Groups {
		out.Groups = append(out.Groups, oas.ComparisonGroup{
			Group: oas.ProfileGroup(g.Group), Added: toChanges(g.Added), Removed: toChanges(g.Removed), Moved: toChanges(g.Moved),
		})
	}
	return out, nil
}

// GetAiReadiness implements getAiReadiness.
func (h *Handler) GetAiReadiness(ctx context.Context) (oas.GetAiReadinessRes, error) {
	rows, err := h.uc.ReadinessTable(ctx)
	if err != nil {
		return nil, err
	}
	out := &oas.AiReadinessTable{Items: make([]oas.AiReadinessRow, 0, len(rows))}
	for _, r := range rows {
		row := oas.AiReadinessRow{CompetitorId: oas.CompetitorId(r.Company.ID), Name: r.Company.Name, Role: oas.CompanyRole(r.Company.Role)}
		if r.Scan != nil {
			row.Scan = oas.NewOptAnalysisSummary(toSummary(*r.Scan))
			if r.Scan.Readiness != nil {
				row.Readiness = oas.NewOptAiReadiness(toReadiness(*r.Scan.Readiness))
			}
		}
		out.Items = append(out.Items, row)
	}
	return out, nil
}

// GetReadinessFiles implements getReadinessFiles.
func (h *Handler) GetReadinessFiles(ctx context.Context, params oas.GetReadinessFilesParams) (oas.GetReadinessFilesRes, error) {
	raw, err := h.uc.ReadinessFiles(ctx, params.AnalysisId)
	if errors.Is(err, domain.ErrNoReadiness) {
		return &oas.GetReadinessFilesNotFound{
			StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, capitalise(err.Error())),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &oas.ReadinessFiles{
		RobotsTxt: raw.Robots, RobotsTruncated: raw.RobotsTruncated, LlmsTxt: raw.LLMs, LlmsTruncated: raw.LLMsTruncated,
		LlmsFullTxt: raw.LLMsFull, LlmsFullTruncated: raw.LLMsFullTruncated,
	}, nil
}

func toReadiness(r domain.Readiness) oas.AiReadiness {
	bots := make([]oas.BotRule, 0, len(r.Bots))
	for _, b := range r.Bots {
		bots = append(bots, oas.BotRule{Bot: b.Bot, Access: oas.CrawlerAccess(b.Access), Named: b.Named, FollowsAdvice: b.FollowsAdvice()})
	}
	advice := make([]oas.ReadinessAdvice, 0, len(r.Advice()))
	for _, a := range r.Advice() {
		advice = append(advice, oas.ReadinessAdvice{Check: oas.ReadinessCheck(a.Check), Ok: a.OK})
	}
	stack := oas.SiteStack{}
	if r.Stack.CMS != "" {
		stack.Cms = oas.NewOptString(r.Stack.CMS)
	}
	if r.Stack.SEOPlugin != "" {
		stack.SeoPlugin = oas.NewOptString(r.Stack.SEOPlugin)
	}
	if r.Stack.BotProtection != "" {
		stack.BotProtection = oas.NewOptString(r.Stack.BotProtection)
	}
	return oas.AiReadiness{
		Robots: oas.RobotsStatus(r.Robots), Bots: bots,
		Sitemap: oas.SitemapSummary{
			Declared: r.Sitemap.Declared, Files: int32(r.Sitemap.Files), //nolint:gosec // at most 10
			Addresses: int32(min(r.Sitemap.Addresses, 1_000_000)), Languages: orEmpty(r.Sitemap.Languages), //nolint:gosec // capped
		},
		LlmsTxt: oas.LlmsFacts{
			Present: r.LLMs.Present, Bytes: int32(min(r.LLMs.Bytes, 100_000_000)), //nolint:gosec // capped
			Links: int32(min(r.LLMs.Links, 10_000_000)), //nolint:gosec // capped
		},
		LlmsFullTxt: r.LLMsFull, Markdown: r.Markdown, HomeRead: r.Home,
		StructuredData: orEmpty(r.StructuredData), Languages: orEmpty(r.Languages), Stack: stack, Advice: advice,
		Gaps: int32(r.Gaps()), //nolint:gosec // at most a few dozen
	}
}

func toBrief(r domain.Readiness) oas.AiReadinessBrief {
	blocked := []string{}
	for _, b := range r.Bots {
		if b.Access == domain.AccessBlocked {
			blocked = append(blocked, b.Bot)
		}
	}
	return oas.AiReadinessBrief{Gaps: int32(r.Gaps()), BlockedBots: blocked} //nolint:gosec // at most a few dozen
}

func orEmpty(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}

func toProfileTerm(t domain.ProfileTerm) oas.ProfileTerm {
	out := oas.ProfileTerm{
		Phrase: t.Phrase, Language: t.Language, Score: t.Score, Pages: int32(t.Pages), //nolint:gosec // at most 100
		Tier: oas.IntentTier(t.Tier), InMap: t.InMap, FirstScan: int32(max(t.FirstScan, 1)), //nolint:gosec // scans of one company
	}
	if t.Family != "" {
		out.Family = oas.NewOptBeltFamily(oas.BeltFamily(t.Family))
	}
	if t.Closest != "" {
		out.ClosestTerm = oas.NewOptString(t.Closest)
	}
	return out
}

// maxChanges is the most changes of one kind a group lists, as the contract allows.
const maxChanges = 10000

func toChanges(changes []domain.TermChange) []oas.TermChange {
	out := make([]oas.TermChange, 0, min(len(changes), maxChanges))
	for _, c := range changes[:min(len(changes), maxChanges)] {
		tc := oas.TermChange{Phrase: c.Phrase, Language: c.Language, Tier: oas.IntentTier(c.Tier), InMap: c.InMap}
		if c.Family != "" {
			tc.Family = oas.NewOptBeltFamily(oas.BeltFamily(c.Family))
		}
		if c.From != nil {
			tc.FromScore, tc.FromPages = oas.NewOptFloat64(c.From.Score), oas.NewOptInt32(int32(c.From.Pages)) //nolint:gosec // at most 100
		}
		if c.To != nil {
			tc.ToScore, tc.ToPages = oas.NewOptFloat64(c.To.Score), oas.NewOptInt32(int32(c.To.Pages)) //nolint:gosec // at most 100
		}
		out = append(out, tc)
	}
	return out
}

func capitalise(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-'a'+'A') + s[1:] + "."
}

// ToSummary maps an analysis to the contract's summary; the companies context reuses it.
func ToSummary(a domain.Analysis) oas.AnalysisSummary { return toSummary(a) }

func toSummary(a domain.Analysis) oas.AnalysisSummary {
	s := oas.AnalysisSummary{
		ID: a.ID, URL: a.URL, Status: oas.AnalysisStatus(a.Status),
		MaxPages: int32(a.MaxPages), PagesCrawled: int32(a.PagesCrawled), //nolint:gosec // at most 100
		CreatedAt: a.CreatedAt,
	}
	if a.Number > 0 {
		s.Number = oas.NewOptInt32(int32(a.Number)) //nolint:gosec // scans of one company
	}
	if a.MatchingModel != "" {
		s.MatchingModel = oas.NewOptString(a.MatchingModel)
	}
	if a.Failure != "" {
		s.Failure = oas.NewOptString(a.Failure)
	}
	if a.Blocked() {
		s.Blocked = oas.NewOptBool(true)
	}
	if a.Readiness != nil {
		s.AiReadiness = oas.NewOptAiReadinessBrief(toBrief(*a.Readiness))
	}
	if a.CompetitorID != "" {
		s.CompetitorId = oas.NewOptCompetitorId(oas.CompetitorId(a.CompetitorID))
	}
	if !a.FinishedAt.Equal(time.Time{}) {
		s.FinishedAt = oas.NewOptDateTime(a.FinishedAt)
	}
	return s
}

func toAnalysis(a domain.Analysis) oas.Analysis {
	pages := make([]oas.CrawledPage, 0, len(a.Pages))
	for _, p := range a.Pages {
		page := oas.CrawledPage{
			URL: p.URL, Title: p.Title, Description: p.Description, Language: p.Language,
			WordCount: int32(p.WordCount), //nolint:gosec // a page is at most 2 MiB
		}
		if p.Family != "" {
			page.Family = oas.NewOptBeltFamily(oas.BeltFamily(p.Family))
		}
		pages = append(pages, page)
	}
	keywords := make([]oas.Keyword, 0, len(a.Keywords))
	for _, k := range a.Keywords {
		kw := oas.Keyword{
			Phrase: k.Phrase, Language: k.Language, Score: k.Score, Pages: int32(k.Pages), //nolint:gosec // at most 100
			Tier: oas.IntentTier(k.Tier), InMap: k.InMap,
		}
		if k.Family != "" {
			kw.Family = oas.NewOptBeltFamily(oas.BeltFamily(k.Family))
		}
		if k.Closest != "" {
			kw.ClosestTerm = oas.NewOptString(k.Closest)
		}
		keywords = append(keywords, kw)
	}
	known := make([]oas.KnownTerm, 0, len(a.KnownTerms))
	for _, k := range a.KnownTerms {
		known = append(known, oas.KnownTerm{
			Term: k.Text, Language: oas.Language(k.Language), Family: oas.BeltFamily(k.Family),
			Tier: oas.IntentTier(k.Tier), Occurrences: int32(k.Occurrences), //nolint:gosec // bounded by page size
			UsedByTribelt: k.UsedByTribelt,
		})
	}
	eqs := make([]oas.Equivalent, 0, len(a.Equivalents))
	for _, e := range a.Equivalents {
		terms := make([]oas.SearchTerm, 0, len(e.SearchTerms))
		for _, s := range e.SearchTerms {
			terms = append(terms, oas.SearchTerm{Term: s.Term, Language: s.Language})
		}
		eqs = append(eqs, oas.Equivalent{Product: e.Product, Family: oas.BeltFamily(e.Family), Reason: e.Reason, SearchTerms: terms})
	}
	out := oas.Analysis{
		Summary: toSummary(a), Pages: pages, Keywords: keywords, KnownTerms: known,
		Matching: oas.MatchingStatus(a.Matching), Equivalents: eqs,
	}
	if a.MatchingModel != "" {
		out.MatchingModel = oas.NewOptString(a.MatchingModel)
	}
	if a.Readiness != nil {
		out.AiReadiness = oas.NewOptAiReadiness(toReadiness(*a.Readiness))
	}
	return out
}
