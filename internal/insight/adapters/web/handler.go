// Package web is the insight context's inbound HTTP adapter: it implements the standings,
// products and plan operations of the generated ogen Handler and maps between the contract's
// types and the domain's.
package web

import (
	"context"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
)

// UseCases is what the adapter needs from the application layer.
type UseCases interface {
	Standings(ctx context.Context) (app.Standings, error)
}

// Handler implements the insight operations. Business rules stay in the use cases.
type Handler struct {
	uc    UseCases
	actor func(context.Context) string
}

// New returns a Handler calling uc; actor names the caller in what it records.
func New(uc UseCases, actor func(context.Context) string) *Handler {
	return &Handler{uc: uc, actor: actor}
}

// GetStandings implements getStandings.
func (h *Handler) GetStandings(ctx context.Context) (oas.GetStandingsRes, error) {
	s, err := h.uc.Standings(ctx)
	if err != nil {
		return nil, err
	}
	out := &oas.Standings{Measures: toStandings(s.Standings), Trailing: int32(s.Trailing)} //nolint:gosec // at most five measures
	if s.Run != nil {
		out.RunStartedAt = oas.NewOptDateTime(s.Run.StartedAt)
	}
	return out, nil
}

func toStandings(in []domain.Standing) []oas.Standing {
	out := make([]oas.Standing, 0, len(in))
	for _, s := range in {
		o := oas.Standing{
			Measure: oas.StandingMeasure(s.Measure), Competitors: int32(s.Competitors), //nolint:gosec // a count of companies
			Trails: s.Trails(), Leads: s.Leads(),
		}
		if s.Tribelt != nil {
			o.Tribelt = oas.NewOptFloat64(*s.Tribelt)
		}
		if s.Median != nil {
			o.Median = oas.NewOptFloat64(*s.Median)
		}
		if s.Leader != nil {
			o.Leader = oas.NewOptMeasureLeader(oas.MeasureLeader{CompetitorId: oas.CompetitorId(s.Leader.CompanyID), Name: s.Leader.Name, Value: s.Leader.Value})
		}
		out = append(out, o)
	}
	return out
}
