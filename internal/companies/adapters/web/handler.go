// Package web is the companies context's inbound HTTP adapter: it implements the competitor
// operations of the generated ogen Handler and maps between the contract's types and the domain's.
package web

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
)

// UseCases is what the adapter needs from the application layer.
type UseCases interface {
	List(ctx context.Context) ([]domain.Listed, error)
	Get(ctx context.Context, id string) (domain.Listed, []domain.Scan, error)
	Add(ctx context.Context, user, name, rawURL string) (domain.Company, error)
	Update(ctx context.Context, user, id string, ch domain.Change) (domain.Listed, error)
	Delete(ctx context.Context, user, id string) error
}

// Handler implements the competitor operations. Business rules stay in the use cases.
type Handler struct {
	uc UseCases
}

// New returns a Handler calling uc.
func New(uc UseCases) *Handler {
	return &Handler{uc: uc}
}

// ListCompetitors implements listCompetitors.
func (h *Handler) ListCompetitors(ctx context.Context) (oas.ListCompetitorsRes, error) {
	listed, err := h.uc.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]oas.Competitor, 0, len(listed))
	for _, l := range listed {
		items = append(items, toCompetitor(l))
	}
	return &oas.CompetitorList{Items: items}, nil
}

// GetCompetitor implements getCompetitor.
func (h *Handler) GetCompetitor(ctx context.Context, params oas.GetCompetitorParams) (oas.GetCompetitorRes, error) {
	listed, scans, err := h.uc.Get(ctx, string(params.CompetitorId))
	if errors.Is(err, domain.ErrNotFound) {
		return &oas.GetCompetitorNotFound{
			StatusCode: http.StatusNotFound,
			Response:   httpx.Problem(http.StatusNotFound, "No company has this id."),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]oas.AnalysisSummary, 0, len(scans))
	for _, s := range scans {
		out = append(out, toSummary(listed.ID, s))
	}
	return &oas.CompetitorDetail{Competitor: toCompetitor(listed), Scans: out}, nil
}

// AddCompetitor implements addCompetitor.
func (h *Handler) AddCompetitor(ctx context.Context, req *oas.NewCompetitor) (oas.AddCompetitorRes, error) {
	c, err := h.uc.Add(ctx, httpx.Subject(ctx), req.Name, req.URL.String())
	var known domain.AlreadyKnownError
	switch {
	case errors.As(err, &known):
		return &oas.AddCompetitorConflict{
			StatusCode: http.StatusConflict,
			Response:   httpx.Problem(http.StatusConflict, known.Error()+"."),
		}, nil
	case errors.Is(err, domain.ErrInvalidName), errors.Is(err, domain.ErrInvalidWebsite):
		detail := domain.ErrInvalidWebsite.Error()
		if errors.Is(err, domain.ErrInvalidName) {
			detail = domain.ErrInvalidName.Error()
		}
		return &oas.AddCompetitorUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity,
			Response:   httpx.Problem(http.StatusUnprocessableEntity, detail+"."),
		}, nil
	case err != nil:
		return nil, err
	}
	out := toCompetitor(domain.Listed{Company: c})
	return &out, nil
}

// UpdateCompetitor implements updateCompetitor.
func (h *Handler) UpdateCompetitor(ctx context.Context, req *oas.CompetitorUpdate, params oas.UpdateCompetitorParams) (oas.UpdateCompetitorRes, error) {
	var ch domain.Change
	if v, ok := req.Name.Get(); ok {
		ch.Name = &v
	}
	if v, ok := req.URL.Get(); ok {
		raw := v.String()
		ch.URL = &raw
	}
	if v, ok := req.Archived.Get(); ok {
		ch.Archived = &v
	}
	l, err := h.uc.Update(ctx, httpx.Subject(ctx), string(params.CompetitorId), ch)
	var known domain.AlreadyKnownError
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.UpdateCompetitorNotFound{StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "No company has this id.")}, nil
	case errors.As(err, &known):
		return &oas.UpdateCompetitorConflict{StatusCode: http.StatusConflict, Response: httpx.Problem(http.StatusConflict, known.Error()+".")}, nil
	case errors.Is(err, domain.ErrInvalidName), errors.Is(err, domain.ErrInvalidWebsite), errors.Is(err, domain.ErrNotArchivable):
		return &oas.UpdateCompetitorUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, sentence(err)),
		}, nil
	case err != nil:
		return nil, err
	}
	out := toCompetitor(l)
	return &out, nil
}

// DeleteCompetitor implements deleteCompetitor.
func (h *Handler) DeleteCompetitor(ctx context.Context, params oas.DeleteCompetitorParams) (oas.DeleteCompetitorRes, error) {
	err := h.uc.Delete(ctx, httpx.Subject(ctx), string(params.CompetitorId))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.DeleteCompetitorNotFound{StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "No company has this id.")}, nil
	case errors.Is(err, app.ErrScanActive):
		return &oas.DeleteCompetitorConflict{StatusCode: http.StatusConflict, Response: httpx.Problem(http.StatusConflict, sentence(err))}, nil
	case err != nil:
		return nil, err
	}
	return &oas.DeleteCompetitorNoContent{}, nil
}

// sentence is the first rule err breaks, as a sentence: the domain's errors are lower case.
func sentence(err error) string {
	for _, known := range []error{domain.ErrInvalidName, domain.ErrInvalidWebsite, domain.ErrNotArchivable, app.ErrScanActive} {
		if errors.Is(err, known) {
			msg := known.Error()
			return strings.ToUpper(msg[:1]) + msg[1:] + "."
		}
	}
	return err.Error()
}

func toCompetitor(l domain.Listed) oas.Competitor {
	names := make([]oas.BeltName, 0, len(l.BeltNames))
	for _, b := range l.BeltNames {
		names = append(names, oas.BeltName{Name: b.Name, Family: oas.BeltFamily(b.Family)})
	}
	languages := l.Languages
	if languages == nil {
		languages = []string{}
	}
	c := oas.Competitor{
		ID: oas.CompetitorId(l.ID), Source: oas.CompetitorSource(l.Source), Name: l.Name, Country: l.Country,
		Website: l.Website, Languages: languages, GroupCompany: l.GroupCompany, BeltNames: names,
		HomepageTitle: l.HomepageTitle, Archived: l.Archived,
		Scans: int32(l.Stats.Count), //nolint:gosec // scans of one company
	}
	if l.URL != nil {
		c.WebsiteUrl = oas.NewOptURI(*l.URL)
	}
	if l.Host != "" {
		c.Host = oas.NewOptString(l.Host)
	}
	if l.Stats.Last != nil {
		c.LastScan = oas.NewOptAnalysisSummary(toSummary(l.ID, *l.Stats.Last))
	}
	if !l.AddedAt.IsZero() {
		c.AddedAt = oas.NewOptDateTime(l.AddedAt)
	}
	return c
}

func toSummary(competitorID string, s domain.Scan) oas.AnalysisSummary {
	out := oas.AnalysisSummary{
		CompetitorId: oas.NewOptCompetitorId(oas.CompetitorId(competitorID)),
		ID:           s.ID, URL: s.URL, Status: oas.AnalysisStatus(s.Status),
		MaxPages: int32(s.MaxPages), PagesCrawled: int32(s.PagesCrawled), //nolint:gosec // at most 100
		CreatedAt: s.CreatedAt,
	}
	if s.Number > 0 {
		out.Number = oas.NewOptInt32(int32(s.Number)) //nolint:gosec // scans of one company
	}
	if s.MatchingModel != "" {
		out.MatchingModel = oas.NewOptString(s.MatchingModel)
	}
	if s.Failure != "" {
		out.Failure = oas.NewOptString(s.Failure)
	}
	if s.Blocked {
		out.Blocked = oas.NewOptBool(true)
	}
	if !s.FinishedAt.IsZero() {
		out.FinishedAt = oas.NewOptDateTime(s.FinishedAt)
	}
	return out
}
