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
	case errors.Is(err, app.ErrRateLimited), errors.Is(err, app.ErrBusy):
		return &oas.StartAnalysisTooManyRequests{
			StatusCode: http.StatusTooManyRequests,
			Response:   httpx.Problem(http.StatusTooManyRequests, capitalise(err.Error())),
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
		MaxPages: int32(a.MaxPages), PagesCrawled: int32(a.PagesCrawled), //nolint:gosec // at most 30
		CreatedAt: a.CreatedAt,
	}
	if a.Failure != "" {
		s.Failure = oas.NewOptString(a.Failure)
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
		pages = append(pages, oas.CrawledPage{
			URL: p.URL, Title: p.Title, Description: p.Description, Language: p.Language,
			WordCount: int32(p.WordCount), //nolint:gosec // a page is at most 2 MiB
		})
	}
	keywords := make([]oas.Keyword, 0, len(a.Keywords))
	for _, k := range a.Keywords {
		kw := oas.Keyword{
			Phrase: k.Phrase, Language: k.Language, Score: k.Score, Pages: int32(k.Pages), //nolint:gosec // at most 30
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
	return oas.Analysis{
		Summary: toSummary(a), Pages: pages, Keywords: keywords, KnownTerms: known,
		Matching: oas.MatchingStatus(a.Matching), Equivalents: eqs,
	}
}
