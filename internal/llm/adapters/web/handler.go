// Package web is the language-model context's inbound HTTP adapter: it implements the admin
// operations of the generated ogen Handler and maps between the contract's types and the domain's.
// The security handler has already refused everyone but administrators.
package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
)

// UseCases is what the adapter needs from the application layer.
type UseCases interface {
	Settings(ctx context.Context) (app.Settings, error)
	Add(ctx context.Context, actor, provider, label, key string) (domain.Credential, error)
	Delete(ctx context.Context, actor, id string) error
	Test(ctx context.Context, actor, id string) (domain.Test, error)
	Models(ctx context.Context, id string) ([]domain.Model, error)
	SetActive(ctx context.Context, actor, credentialID, model string) (app.Settings, error)
	ClearActive(ctx context.Context, actor string) (app.Settings, error)
	Usage(ctx context.Context, id string) (domain.Usage, error)
}

// Actor names who makes a request, for the events.
type Actor func(ctx context.Context) string

// Handler implements the admin operations. Business rules stay in the use cases.
type Handler struct {
	uc    UseCases
	actor Actor
}

// New returns a Handler calling uc, naming the caller with actor.
func New(uc UseCases, actor Actor) *Handler {
	return &Handler{uc: uc, actor: actor}
}

// GetLlmSettings implements getLlmSettings.
func (h *Handler) GetLlmSettings(ctx context.Context) (oas.GetLlmSettingsRes, error) {
	s, err := h.uc.Settings(ctx)
	if err != nil {
		return nil, err
	}
	out := toSettings(s)
	return &out, nil
}

// AddLlmCredential implements addLlmCredential.
func (h *Handler) AddLlmCredential(ctx context.Context, req *oas.NewLlmCredential) (oas.AddLlmCredentialRes, error) {
	c, err := h.uc.Add(ctx, h.actor(ctx), string(req.Provider), req.Label, req.ApiKey)
	if app.IsInputError(err) {
		return &oas.AddLlmCredentialUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, detail(err)),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	out := toCredential(c)
	return &out, nil
}

// DeleteLlmCredential implements deleteLlmCredential.
func (h *Handler) DeleteLlmCredential(ctx context.Context, params oas.DeleteLlmCredentialParams) (oas.DeleteLlmCredentialRes, error) {
	err := h.uc.Delete(ctx, h.actor(ctx), string(params.CredentialId))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.DeleteLlmCredentialNotFound{StatusCode: http.StatusNotFound, Response: notFound()}, nil
	case app.IsInputError(err):
		return &oas.DeleteLlmCredentialUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, detail(err)),
		}, nil
	case err != nil:
		return nil, err
	}
	return &oas.DeleteLlmCredentialNoContent{}, nil
}

// TestLlmCredential implements testLlmCredential.
func (h *Handler) TestLlmCredential(ctx context.Context, params oas.TestLlmCredentialParams) (oas.TestLlmCredentialRes, error) {
	t, err := h.uc.Test(ctx, h.actor(ctx), string(params.CredentialId))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.TestLlmCredentialNotFound{StatusCode: http.StatusNotFound, Response: notFound()}, nil
	case app.IsInputError(err):
		return &oas.TestLlmCredentialUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, detail(err)),
		}, nil
	case err != nil:
		return nil, err
	}
	out := toTest(t)
	return &out, nil
}

// ListLlmModels implements listLlmModels.
func (h *Handler) ListLlmModels(ctx context.Context, params oas.ListLlmModelsParams) (oas.ListLlmModelsRes, error) {
	models, err := h.uc.Models(ctx, string(params.CredentialId))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.ListLlmModelsNotFound{StatusCode: http.StatusNotFound, Response: notFound()}, nil
	case app.IsInputError(err):
		return &oas.ListLlmModelsUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, detail(err)),
		}, nil
	case errors.Is(err, domain.ErrProvider):
		return &oas.ListLlmModelsBadGateway{
			StatusCode: http.StatusBadGateway, Response: httpx.Problem(http.StatusBadGateway, detail(err)),
		}, nil
	case err != nil:
		return nil, err
	}
	items := make([]oas.LlmModel, 0, len(models))
	for _, m := range models[:min(len(models), 2000)] {
		items = append(items, oas.LlmModel{ID: clip(m.ID, 200), Name: clip(m.Name, 300)})
	}
	return &oas.LlmModelList{Items: items}, nil
}

// GetLlmKeyUsage implements getLlmKeyUsage.
func (h *Handler) GetLlmKeyUsage(ctx context.Context, params oas.GetLlmKeyUsageParams) (oas.GetLlmKeyUsageRes, error) {
	u, err := h.uc.Usage(ctx, string(params.CredentialId))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.GetLlmKeyUsageNotFound{StatusCode: http.StatusNotFound, Response: notFound()}, nil
	case app.IsInputError(err):
		return &oas.GetLlmKeyUsageUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, detail(err)),
		}, nil
	case errors.Is(err, domain.ErrProvider):
		return &oas.GetLlmKeyUsageBadGateway{
			StatusCode: http.StatusBadGateway, Response: httpx.Problem(http.StatusBadGateway, detail(err)),
		}, nil
	case err != nil:
		return nil, err
	}
	out := &oas.LlmKeyUsage{
		Source: oas.LlmUsageSource(u.Source), Message: clip(u.Message, 500), Models: make([]oas.LlmModelTokens, 0, len(u.Models)),
		TotalUsd: optFloat(u.Total), DailyUsd: optFloat(u.Daily), WeeklyUsd: optFloat(u.Weekly), MonthlyUsd: optFloat(u.Monthly),
		LimitUsd: optFloat(u.Limit), LimitRemainingUsd: optFloat(u.LimitRemaining), OrganisationCostUsd: optFloat(u.OrganisationCost),
	}
	if u.LimitReset != "" {
		out.LimitReset = oas.NewOptString(clip(u.LimitReset, 20))
	}
	if f := u.Free; f != nil {
		out.FreeRequests = oas.NewOptLlmFreeRequests(oas.LlmFreeRequests{
			Used: int32(f.Used), Limit: int32(f.Limit), Remaining: int32(f.Remaining), //nolint:gosec // small counts
		})
	}
	if !u.Since.IsZero() {
		out.Since = oas.NewOptDateTime(u.Since)
	}
	for _, m := range u.Models[:min(len(u.Models), 1000)] {
		out.Models = append(out.Models, oas.LlmModelTokens{
			Model: clip(m.Model, 200), InputTokens: m.InputTokens, OutputTokens: m.OutputTokens, CacheReadTokens: m.CacheRead,
		})
	}
	return out, nil
}

func optFloat(v *float64) oas.OptFloat64 {
	if v == nil {
		return oas.OptFloat64{}
	}
	return oas.NewOptFloat64(*v)
}

// SetActiveLlm implements setActiveLlm.
func (h *Handler) SetActiveLlm(ctx context.Context, req *oas.ActiveLlm) (oas.SetActiveLlmRes, error) {
	s, err := h.uc.SetActive(ctx, h.actor(ctx), string(req.CredentialId), req.Model)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.SetActiveLlmNotFound{StatusCode: http.StatusNotFound, Response: notFound()}, nil
	case app.IsInputError(err):
		return &oas.SetActiveLlmUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, detail(err)),
		}, nil
	case err != nil:
		return nil, err
	}
	out := toSettings(s)
	return &out, nil
}

// ClearActiveLlm implements clearActiveLlm.
func (h *Handler) ClearActiveLlm(ctx context.Context) (oas.ClearActiveLlmRes, error) {
	s, err := h.uc.ClearActive(ctx, h.actor(ctx))
	if err != nil {
		return nil, err
	}
	out := toSettings(s)
	return &out, nil
}

func notFound() oas.Problem {
	return httpx.Problem(http.StatusNotFound, "No credential has this id.")
}

// detail is the sentence an admin reads for err: the provider's reason when it refused a key.
func detail(err error) string {
	msg := err.Error()
	for _, prefix := range []string{domain.ErrKeyRefused.Error() + ": ", domain.ErrProvider.Error() + ": ", domain.ErrInvalid.Error() + ": "} {
		msg = strings.TrimPrefix(msg, prefix)
	}
	r := []rune(strings.TrimSuffix(clip(msg, 1000), ".") + ".")
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func toSettings(s app.Settings) oas.LlmSettings {
	creds := make([]oas.LlmCredential, 0, len(s.Credentials))
	for _, c := range s.Credentials {
		creds = append(creds, toCredential(c))
	}
	events := make([]oas.LlmEvent, 0, len(s.Events))
	for _, e := range s.Events {
		events = append(events, toEvent(e))
	}
	out := oas.LlmSettings{
		Credentials: creds, Choice: oas.LlmChoice(s.Choice), MatchingEnabled: s.MatchingEnabled, Events: events,
		Budget: oas.LlmBudget{
			MonthlyTokens: s.Budget.MonthlyTokens, UsedThisMonth: s.Budget.UsedThisMonth,
			AverageTokensPerScan: s.Budget.AveragePerScan, ScansMatched: int64(s.Budget.ScansMatched),
		},
	}
	if s.Active != nil {
		out.Active = oas.NewOptActiveLlm(oas.ActiveLlm{CredentialId: oas.LlmCredentialId(s.Active.CredentialID), Model: s.Active.Model})
	}
	return out
}

func toCredential(c domain.Credential) oas.LlmCredential {
	out := oas.LlmCredential{
		ID: oas.LlmCredentialId(c.ID), Provider: oas.LlmProvider(c.Provider), Label: c.Label,
		Source: oas.LlmCredentialSource(c.Source), Fingerprint: c.Fingerprint, Readable: c.Readable,
	}
	if c.CreatedBy != "" {
		out.CreatedBy = oas.NewOptString(c.CreatedBy)
	}
	if !c.CreatedAt.IsZero() {
		out.CreatedAt = oas.NewOptDateTime(c.CreatedAt)
	}
	if c.LastTest != nil {
		out.LastTest = oas.NewOptLlmTest(toTest(*c.LastTest))
	}
	return out
}

func toTest(t domain.Test) oas.LlmTest {
	return oas.LlmTest{Ok: t.OK, At: t.At, Message: t.Message}
}

func toEvent(e domain.Event) oas.LlmEvent {
	out := oas.LlmEvent{At: e.At, Actor: e.Actor, Action: oas.LlmEventAction(e.Action), Message: e.Message}
	if e.CredentialID != "" {
		out.CredentialId = oas.NewOptLlmCredentialId(oas.LlmCredentialId(e.CredentialID))
	}
	if e.Provider != "" {
		out.Provider = oas.NewOptLlmProvider(oas.LlmProvider(e.Provider))
	}
	if e.Model != "" {
		out.Model = oas.NewOptString(e.Model)
	}
	if e.OK != nil {
		out.Ok = oas.NewOptBool(*e.OK)
	}
	return out
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
