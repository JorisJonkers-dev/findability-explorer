// Package web is the terminology context's inbound HTTP adapter: it implements the terminology
// operations of the generated ogen Handler and maps between the contract's types and the domain's.
package web

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// UseCases is what the adapter needs from the application layer.
type UseCases interface {
	Search(q domain.Query) domain.Result
	FoundBy(ctx context.Context) (map[domain.TermKey][]string, error)
	Candidates(ctx context.Context, status domain.Status) ([]domain.Candidate, domain.Counts, error)
	Accept(ctx context.Context, user, id string, a domain.Acceptance) (domain.Candidate, error)
	Reject(ctx context.Context, user, id string) (domain.Candidate, error)
	Undo(ctx context.Context, id string) (domain.Candidate, error)
}

// Handler implements the terminology operations. Business rules stay in the use cases.
type Handler struct {
	uc    UseCases
	actor func(context.Context) string
}

// New returns a Handler calling uc; actor names the administrator a decision records.
func New(uc UseCases, actor func(context.Context) string) *Handler {
	return &Handler{uc: uc, actor: actor}
}

// maxCandidates is the most candidates one list returns, as the contract allows.
const maxCandidates = 1000

// ListTermCandidates implements listTermCandidates.
func (h *Handler) ListTermCandidates(ctx context.Context, params oas.ListTermCandidatesParams) (oas.ListTermCandidatesRes, error) {
	status := domain.StatusPending
	if v, ok := params.Status.Get(); ok {
		status = domain.Status(v)
	}
	list, counts, err := h.uc.Candidates(ctx, status)
	if err != nil {
		return nil, err
	}
	list = list[:min(len(list), maxCandidates)]
	items := make([]oas.TermCandidate, 0, len(list))
	for _, c := range list {
		items = append(items, toCandidate(c))
	}
	return &oas.TermCandidateList{Items: items, Counts: oas.TermCandidateCounts{
		Pending: int32(counts.Pending), Accepted: int32(counts.Accepted), Rejected: int32(counts.Rejected), //nolint:gosec // small counts
	}}, nil
}

// AcceptTermCandidate implements acceptTermCandidate.
func (h *Handler) AcceptTermCandidate(ctx context.Context, req *oas.TermAcceptance, params oas.AcceptTermCandidateParams) (oas.AcceptTermCandidateRes, error) {
	c, err := h.uc.Accept(ctx, h.actor(ctx), string(params.CandidateId), domain.Acceptance{
		Text: req.Term, Language: string(req.Language), Family: string(req.Family), Tier: string(req.Tier), UsedBy: req.UsedBy,
	})
	switch {
	case errors.Is(err, domain.ErrCandidateNotFound):
		return &oas.AcceptTermCandidateNotFound{StatusCode: http.StatusNotFound, Response: problem(http.StatusNotFound, err)}, nil
	case errors.Is(err, domain.ErrDecided), errors.Is(err, domain.ErrTermExists):
		return &oas.AcceptTermCandidateConflict{StatusCode: http.StatusConflict, Response: problem(http.StatusConflict, err)}, nil
	case errors.Is(err, domain.ErrInvalidTerm), errors.Is(err, domain.ErrUnknownValue):
		return &oas.AcceptTermCandidateUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: problem(http.StatusUnprocessableEntity, err),
		}, nil
	case err != nil:
		return nil, err
	}
	out := toCandidate(c)
	return &out, nil
}

// RejectTermCandidate implements rejectTermCandidate.
func (h *Handler) RejectTermCandidate(ctx context.Context, params oas.RejectTermCandidateParams) (oas.RejectTermCandidateRes, error) {
	c, err := h.uc.Reject(ctx, h.actor(ctx), string(params.CandidateId))
	switch {
	case errors.Is(err, domain.ErrCandidateNotFound):
		return &oas.RejectTermCandidateNotFound{StatusCode: http.StatusNotFound, Response: problem(http.StatusNotFound, err)}, nil
	case errors.Is(err, domain.ErrDecided):
		return &oas.RejectTermCandidateConflict{StatusCode: http.StatusConflict, Response: problem(http.StatusConflict, err)}, nil
	case err != nil:
		return nil, err
	}
	out := toCandidate(c)
	return &out, nil
}

// UndoTermDecision implements undoTermDecision.
func (h *Handler) UndoTermDecision(ctx context.Context, params oas.UndoTermDecisionParams) (oas.UndoTermDecisionRes, error) {
	c, err := h.uc.Undo(ctx, string(params.CandidateId))
	switch {
	case errors.Is(err, domain.ErrCandidateNotFound):
		return &oas.UndoTermDecisionNotFound{StatusCode: http.StatusNotFound, Response: problem(http.StatusNotFound, err)}, nil
	case errors.Is(err, domain.ErrPending):
		return &oas.UndoTermDecisionConflict{StatusCode: http.StatusConflict, Response: problem(http.StatusConflict, err)}, nil
	case err != nil:
		return nil, err
	}
	out := toCandidate(c)
	return &out, nil
}

// problem is a problem whose detail is the first sentence of err, capitalised.
func problem(status int, err error) oas.Problem {
	for _, known := range []error{
		domain.ErrCandidateNotFound, domain.ErrDecided, domain.ErrPending, domain.ErrTermExists, domain.ErrInvalidTerm,
	} {
		if errors.Is(err, known) {
			err = known
		}
	}
	msg := err.Error()
	return httpx.Problem(status, strings.ToUpper(msg[:1])+msg[1:]+".")
}

func toCandidate(c domain.Candidate) oas.TermCandidate {
	companies := make([]oas.CandidateCompany, 0, len(c.Companies))
	for _, x := range c.Companies {
		companies = append(companies, oas.CandidateCompany{CompetitorId: oas.CompetitorId(x.ID), Name: x.Name})
	}
	out := oas.TermCandidate{
		ID: oas.TermCandidateId(c.ID), Phrase: c.Phrase, Status: oas.TermCandidateStatus(c.Status), Languages: nonNil(c.Languages),
		Companies: companies, Scans: int32(c.Scans), BestScore: c.BestScore, MaxPages: int32(c.MaxPages), //nolint:gosec // small counts
		SuggestedTier: oas.IntentTier(c.SuggestedTier), LastSeenAt: c.LastSeenAt,
	}
	if c.SuggestedFamily != "" {
		out.SuggestedFamily = oas.NewOptBeltFamily(oas.BeltFamily(c.SuggestedFamily))
	}
	if c.Closest != "" {
		out.ClosestTerm = oas.NewOptString(c.Closest)
	}
	if d := c.Decision; d != nil {
		decision := oas.TermDecision{Status: oas.TermCandidateStatus(d.Status), By: d.By, At: d.At}
		if d.Term != nil {
			decision.Term = oas.NewOptTerm(ToTerm(*d.Term))
		}
		out.Decision = oas.NewOptTermDecision(decision)
	}
	return out
}

// SearchTerms implements searchTerms. ogen has already checked every enum against the contract,
// whose values are the domain's.
func (h *Handler) SearchTerms(ctx context.Context, params oas.SearchTermsParams) (oas.SearchTermsRes, error) {
	foundBy, err := h.uc.FoundBy(ctx)
	if err != nil {
		return nil, err
	}
	result := h.uc.Search(domain.Query{
		Text:     params.Q.Or(""),
		Language: domain.Language(params.Language.Or("")),
		Tier:     domain.Tier(params.Tier.Or("")),
		Family:   domain.Family(params.Family.Or("")),
		User:     params.User.Or(""),
		Limit:    int(params.Limit.Or(0)),
	})
	items := make([]oas.Term, 0, len(result.Items))
	for _, t := range result.Items {
		term := ToTerm(t)
		for _, id := range foundBy[t.Key()] {
			term.FoundBy = append(term.FoundBy, oas.CompetitorId(id))
		}
		items = append(items, term)
	}
	return &oas.TermSearch{
		Items: items,
		Total: int32(result.Total), //nolint:gosec // the map holds a few hundred terms
		Facets: oas.Facets{
			Languages: toCounts(result.Facets.Languages),
			Tiers:     toCounts(result.Facets.Tiers),
			Families:  toCounts(result.Facets.Families),
		},
	}, nil
}

// ToTerm maps a domain term to the contract's; the analysis context reuses it for known terms.
func ToTerm(t domain.Term) oas.Term {
	return oas.Term{
		Term: t.Text, Language: oas.Language(t.Language), Family: oas.BeltFamily(t.Family),
		Tier: oas.IntentTier(t.Tier), UsedBy: nonNil(t.UsedBy), Source: t.Source,
		Origin: oas.TermOrigin(t.Origin),
	}
}

func toCounts(counts []domain.FacetCount) []oas.FacetCount {
	out := make([]oas.FacetCount, 0, len(counts))
	for _, c := range counts {
		out = append(out, oas.FacetCount{Value: c.Value, Count: int32(c.Count)}) //nolint:gosec // small counts
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
