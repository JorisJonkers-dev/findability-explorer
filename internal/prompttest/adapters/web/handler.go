// Package web is the prompt-test context's inbound HTTP adapter: it implements the prompt-test
// operations of the generated ogen Handler and maps between the contract's types and the domain's.
package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// UseCases is what the adapter needs from the application layer.
type UseCases interface {
	Results(ctx context.Context) (app.Results, error)
	Answers(ctx context.Context, runID uuid.UUID, filter app.AnswerFilter) (app.RunResult, []domain.Answer, error)
	Start(ctx context.Context, actor string) (domain.Run, error)
	Retry(ctx context.Context, runID uuid.UUID, task *domain.TaskKey) (app.Retry, error)
	Settings(ctx context.Context) (app.View, error)
	Update(ctx context.Context, actor string, ch app.Change) (app.View, error)
	Candidates(ctx context.Context, status domain.CandidateStatus) ([]domain.Candidate, map[domain.CandidateStatus]int, error)
	AddCandidate(ctx context.Context, actor, id, name, website string) (domain.Candidate, error)
	DismissCandidate(ctx context.Context, actor, id string) (domain.Candidate, error)
	UndoCandidate(ctx context.Context, actor, id string) error
}

// Handler implements the prompt-test operations. Business rules stay in the use cases.
type Handler struct {
	uc    UseCases
	actor func(context.Context) string
}

// New returns a Handler calling uc; actor names the administrator a change records.
func New(uc UseCases, actor func(context.Context) string) *Handler {
	return &Handler{uc: uc, actor: actor}
}

// The most mentions, answers and candidates one response lists, as the contract allows.
const (
	maxMentions   = 1000
	maxAnswers    = 5000
	maxCandidates = 1000
)

// GetPromptTestResults implements getPromptTestResults.
func (h *Handler) GetPromptTestResults(ctx context.Context) (oas.GetPromptTestResultsRes, error) {
	res, err := h.uc.Results(ctx)
	if err != nil {
		return nil, err
	}
	out := &oas.PromptTestResults{
		PromptSetVersion: domain.PromptSetVersion, Runs: make([]oas.PromptRun, 0, len(res.Runs)),
		Mentions: make([]oas.PromptMention, 0, len(res.Mentions)), WrongClaims: make([]oas.PromptWrongClaim, 0, len(res.WrongClaims)),
	}
	for _, r := range res.Runs {
		out.Runs = append(out.Runs, toRun(r.Run, r.Progress, r.Cells))
	}
	for _, m := range res.Mentions[:min(len(res.Mentions), maxMentions)] {
		out.Mentions = append(out.Mentions, oas.PromptMention{
			CompetitorId: oas.CompetitorId(m.CompetitorID), Name: m.Name, Role: oas.CompanyRole(m.Role),
			Answers: int32(m.Answers), First: int32(m.First), AveragePosition: m.AveragePosition(), //nolint:gosec // answers of one run
		})
	}
	for _, w := range res.WrongClaims {
		claim := oas.PromptWrongClaim{
			AnswerId: w.AnswerID, RunId: w.RunID, At: w.At, Provider: w.Provider, Model: w.Model, PromptId: w.PromptID,
			Mode: oas.PromptMode(w.Mode), Claim: w.Claim,
		}
		if w.Correctness != domain.CorrectnessNone {
			claim.Correctness = oas.NewOptPromptCorrectness(oas.PromptCorrectness(w.Correctness))
		}
		out.WrongClaims = append(out.WrongClaims, claim)
	}
	return out, nil
}

// ListPromptAnswers implements listPromptAnswers.
func (h *Handler) ListPromptAnswers(ctx context.Context, params oas.ListPromptAnswersParams) (oas.ListPromptAnswersRes, error) {
	filter := app.AnswerFilter{
		Provider: params.Provider.Or(""), Scope: domain.Scope(params.Scope.Or("")), Family: domain.Family(params.Family.Or("")),
		Mode: domain.Mode(params.Mode.Or("")),
	}
	run, answers, err := h.uc.Answers(ctx, params.RunId, filter)
	if errors.Is(err, domain.ErrNotFound) {
		return &oas.ListPromptAnswersNotFound{StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "No Prompt run has this id.")}, nil
	}
	if err != nil {
		return nil, err
	}
	out := &oas.PromptAnswerList{Run: toRun(run.Run, run.Progress, run.Cells), Items: make([]oas.PromptAnswer, 0, min(len(answers), maxAnswers))}
	for _, a := range answers[:min(len(answers), maxAnswers)] {
		out.Items = append(out.Items, toAnswer(a))
	}
	return out, nil
}

// StartPromptRun implements startPromptRun.
func (h *Handler) StartPromptRun(ctx context.Context) (oas.StartPromptRunRes, error) {
	run, err := h.uc.Start(ctx, h.actor(ctx))
	switch {
	case errors.Is(err, domain.ErrRunActive):
		return &oas.StartPromptRunConflict{StatusCode: http.StatusConflict, Response: httpx.Problem(http.StatusConflict, sentence(err))}, nil
	case app.IsInputError(err):
		return &oas.StartPromptRunUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, sentence(err)),
		}, nil
	case err != nil:
		return nil, err
	}
	out := toRun(run, domain.Progress{}, nil)
	return &out, nil
}

// RetryPromptRun implements retryPromptRun.
func (h *Handler) RetryPromptRun(ctx context.Context, req *oas.PromptRetryRequest, params oas.RetryPromptRunParams) (oas.RetryPromptRunRes, error) {
	prompt, hasPrompt := req.PromptId.Get()
	provider, hasProvider := req.Provider.Get()
	mode, hasMode := req.Mode.Get()
	var task *domain.TaskKey
	switch {
	case hasPrompt && hasProvider && hasMode:
		task = &domain.TaskKey{PromptID: prompt, Provider: provider, Mode: domain.Mode(mode)}
	case hasPrompt || hasProvider || hasMode:
		return &oas.RetryPromptRunUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity,
			Response:   httpx.Problem(http.StatusUnprocessableEntity, "Name the question, the provider and the mode, or none of them."),
		}, nil
	}
	r, err := h.uc.Retry(ctx, params.RunId, task)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.RetryPromptRunNotFound{StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "No Prompt run has this id.")}, nil
	case errors.Is(err, domain.ErrRunActive), errors.Is(err, domain.ErrBudgetReached):
		return &oas.RetryPromptRunConflict{StatusCode: http.StatusConflict, Response: httpx.Problem(http.StatusConflict, sentence(err))}, nil
	case app.IsInputError(err):
		return &oas.RetryPromptRunUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, sentence(err)),
		}, nil
	case err != nil:
		return nil, err
	}
	return &oas.PromptRetry{
		Run: toRun(r.Run, domain.Progress{}, nil), Tasks: int32(r.Tasks), EstimatedCostUsd: oas.Usd(r.Estimate.Dollars()), //nolint:gosec // at most a run's tasks
	}, nil
}

// GetPromptTestSettings implements getPromptTestSettings.
func (h *Handler) GetPromptTestSettings(ctx context.Context) (oas.GetPromptTestSettingsRes, error) {
	v, err := h.uc.Settings(ctx)
	if err != nil {
		return nil, err
	}
	out := toSettings(v)
	return &out, nil
}

// UpdatePromptTestSettings implements updatePromptTestSettings.
func (h *Handler) UpdatePromptTestSettings(ctx context.Context, req *oas.PromptTestSettingsChange) (oas.UpdatePromptTestSettingsRes, error) {
	ch := app.Change{Providers: req.Providers}
	if v, ok := req.CredentialId.Get(); ok {
		id := string(v)
		ch.CredentialID = &id
	}
	if v, ok := req.MonthlyBudgetUsd.Get(); ok {
		m := domain.MicrosOf(v)
		ch.Budget = &m
	}
	if v, ok := req.Cadence.Get(); ok {
		c := string(v)
		ch.Cadence = &c
	}
	if v, ok := req.FactSheet.Get(); ok {
		ch.FactSheet = &v
	}
	v, err := h.uc.Update(ctx, h.actor(ctx), ch)
	switch {
	case app.IsInputError(err):
		return &oas.UpdatePromptTestSettingsUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, sentence(err)),
		}, nil
	case err != nil:
		return nil, err
	}
	out := toSettings(v)
	return &out, nil
}

// ListCompetitorCandidates implements listCompetitorCandidates.
func (h *Handler) ListCompetitorCandidates(ctx context.Context, params oas.ListCompetitorCandidatesParams) (oas.ListCompetitorCandidatesRes, error) {
	status := domain.CandidatePending
	if v, ok := params.Status.Get(); ok {
		status = domain.CandidateStatus(v)
	}
	list, counts, err := h.uc.Candidates(ctx, status)
	if err != nil {
		return nil, err
	}
	items := make([]oas.CompetitorCandidate, 0, min(len(list), maxCandidates))
	for _, c := range list[:min(len(list), maxCandidates)] {
		items = append(items, toCandidate(c))
	}
	return &oas.CompetitorCandidateList{Items: items, Counts: oas.CompetitorCandidateCounts{
		Pending: int32(counts[domain.CandidatePending]), Added: int32(counts[domain.CandidateAdded]), //nolint:gosec // small counts
		Dismissed: int32(counts[domain.CandidateDismissed]), //nolint:gosec // small counts
	}}, nil
}

// AddCompetitorCandidate implements addCompetitorCandidate.
func (h *Handler) AddCompetitorCandidate(ctx context.Context, req *oas.NewCompetitor, params oas.AddCompetitorCandidateParams) (oas.AddCompetitorCandidateRes, error) {
	c, err := h.uc.AddCandidate(ctx, h.actor(ctx), string(params.CompetitorCandidateId), req.Name, req.URL.String())
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.AddCompetitorCandidateNotFound{StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "No competitor candidate has this id.")}, nil
	case errors.Is(err, app.ErrDecided), errors.Is(err, domain.ErrCompanyKnown):
		return &oas.AddCompetitorCandidateConflict{StatusCode: http.StatusConflict, Response: httpx.Problem(http.StatusConflict, sentence(err))}, nil
	case errors.Is(err, domain.ErrInvalid):
		return &oas.AddCompetitorCandidateUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, sentence(err)),
		}, nil
	case err != nil:
		return nil, err
	}
	out := toCandidate(c)
	return &out, nil
}

// DismissCompetitorCandidate implements dismissCompetitorCandidate.
func (h *Handler) DismissCompetitorCandidate(ctx context.Context, params oas.DismissCompetitorCandidateParams) (oas.DismissCompetitorCandidateRes, error) {
	c, err := h.uc.DismissCandidate(ctx, h.actor(ctx), string(params.CompetitorCandidateId))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.DismissCompetitorCandidateNotFound{StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "No competitor candidate has this id.")}, nil
	case errors.Is(err, app.ErrDecided):
		return &oas.DismissCompetitorCandidateConflict{StatusCode: http.StatusConflict, Response: httpx.Problem(http.StatusConflict, sentence(err))}, nil
	case err != nil:
		return nil, err
	}
	out := toCandidate(c)
	return &out, nil
}

// UndoCompetitorCandidate implements undoCompetitorCandidate.
func (h *Handler) UndoCompetitorCandidate(ctx context.Context, params oas.UndoCompetitorCandidateParams) (oas.UndoCompetitorCandidateRes, error) {
	err := h.uc.UndoCandidate(ctx, h.actor(ctx), string(params.CompetitorCandidateId))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.UndoCompetitorCandidateNotFound{StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "The candidate has no decision to undo.")}, nil
	case errors.Is(err, domain.ErrInvalid):
		return &oas.UndoCompetitorCandidateConflict{StatusCode: http.StatusConflict, Response: httpx.Problem(http.StatusConflict, sentence(err))}, nil
	case err != nil:
		return nil, err
	}
	return &oas.UndoCompetitorCandidateNoContent{}, nil
}

func toRun(r domain.Run, p domain.Progress, cells []domain.Cell) oas.PromptRun {
	out := oas.PromptRun{
		ID: r.ID, PromptSetVersion: int32(r.PromptSetVersion), FactSheetVersion: int32(max(r.FactSheetVersion, 1)), //nolint:gosec // small versions
		Trigger: oas.PromptTrigger(r.Trigger), StartedBy: r.StartedBy, StartedAt: r.StartedAt, Status: oas.PromptRunStatus(r.Status),
		Tasks:    int32(r.Tasks),                                                                                       //nolint:gosec // at most a few thousand
		Answered: int32(p.Answered), Failed: int32(p.Failed), InputTokens: p.InputTokens, OutputTokens: p.OutputTokens, //nolint:gosec // at most a few thousand
		CostUsd: oas.Usd(p.Cost.Dollars()), ByProvider: make([]oas.PromptProviderProgress, 0, len(p.ByProvider)),
		Cells: make([]oas.PromptCell, 0, len(cells)),
	}
	if !r.FinishedAt.IsZero() {
		out.FinishedAt = oas.NewOptDateTime(r.FinishedAt)
	}
	if r.Failure != "" {
		out.Failure = oas.NewOptString(r.Failure)
	}
	out.Models = toModels(r.Models)
	for _, pp := range p.ByProvider {
		out.ByProvider = append(out.ByProvider, oas.PromptProviderProgress{
			Provider: pp.Provider, Answered: int32(pp.Answered), Failed: int32(pp.Failed), //nolint:gosec // at most a few thousand
			CostUsd: oas.Usd(pp.Cost.Dollars()), Models: pp.Models[:min(len(pp.Models), 2)],
		})
	}
	for _, c := range cells {
		out.Cells = append(out.Cells, oas.PromptCell{
			Provider: c.Provider, Scope: oas.PromptScope(c.Scope), Family: oas.PromptFamily(c.Family), Mode: oas.PromptMode(c.Mode),
			Answers: int32(c.Answers), Named: int32(c.Named), Cited: int32(c.Cited), //nolint:gosec // answers of one run
		})
	}
	return out
}

func toAnswer(a domain.Answer) oas.PromptAnswer {
	cost := a.Cost
	out := oas.PromptAnswer{
		ID: a.ID, PromptId: a.Prompt.ID, Family: oas.PromptFamily(a.Prompt.Family), Kind: oas.PromptKind(a.Prompt.Kind),
		Scope: oas.PromptScope(a.Prompt.Scope), Language: a.Prompt.Language, Prompt: a.Prompt.Text,
		PromptSetVersion: int32(a.PromptSetVersion), Provider: a.Provider, Model: a.Model, Mode: oas.PromptMode(a.Mode), //nolint:gosec // small versions
		Text: a.Text, Citations: orEmpty(a.Citations), At: a.At, InputTokens: a.InputTokens, OutputTokens: a.OutputTokens,
	}
	if a.Failure != "" {
		out.Failure = oas.NewOptString(a.Failure)
	}
	if x := a.Extraction; x != nil {
		cost += x.Cost
		out.InputTokens += x.InputTokens
		out.OutputTokens += x.OutputTokens
		e := oas.PromptExtraction{
			Failed: x.Failed, Companies: make([]oas.PromptNamedCompany, 0, len(x.Companies)), Addresses: orEmpty(x.Addresses),
			NamesTribelt: x.NamesTribelt, CitesTribelt: x.CitesTribelt, WrongClaims: orEmpty(x.WrongClaims), CostUsd: oas.Usd(x.Cost.Dollars()),
		}
		if x.Correctness != domain.CorrectnessNone {
			e.Correctness = oas.NewOptPromptCorrectness(oas.PromptCorrectness(x.Correctness))
		}
		for _, c := range x.Companies {
			n := oas.PromptNamedCompany{Name: c.Name, Position: int32(c.Position)} //nolint:gosec // at most 30
			if c.URL != "" {
				n.URL = oas.NewOptString(c.URL)
			}
			if c.CompetitorID != "" {
				n.CompetitorId = oas.NewOptCompetitorId(oas.CompetitorId(c.CompetitorID))
			}
			e.Companies = append(e.Companies, n)
		}
		out.Extraction = oas.NewOptPromptExtraction(e)
	}
	out.CostUsd = oas.Usd(cost.Dollars())
	return out
}

func toSettings(v app.View) oas.PromptTestSettings {
	s := v.Settings
	out := oas.PromptTestSettings{
		Providers: s.Providers, MonthlyBudgetUsd: oas.Usd(s.MonthlyBudget.Dollars()), SpentThisMonthUsd: oas.Usd(v.SpentThisMonth.Dollars()),
		Cadence: oas.PromptCadence(s.Cadence), NextDueAt: s.NextDue, ExtractorModel: v.ExtractorModel,
		FactSheet: oas.PromptFactSheet{Text: s.FactSheet.Text, Version: int32(s.FactSheet.Version)},                                     //nolint:gosec // small versions
		PromptSet: oas.PromptSet{Version: int32(v.PromptSet.Version), Prompts: make([]oas.PromptQuestion, 0, len(v.PromptSet.Prompts))}, //nolint:gosec // small versions
	}
	if s.CredentialID != "" {
		out.CredentialId = oas.NewOptLlmCredentialId(oas.LlmCredentialId(s.CredentialID))
	}
	out.KeyWorks, out.Models, out.Prices = v.KeyWorks, toModels(v.Models), toPrices(v.Prices)
	if s.FactSheet.UpdatedBy != "" {
		out.FactSheet.UpdatedBy = oas.NewOptString(s.FactSheet.UpdatedBy)
		out.FactSheet.UpdatedAt = oas.NewOptDateTime(s.FactSheet.UpdatedAt)
	}
	for _, p := range v.PromptSet.Prompts {
		out.PromptSet.Prompts = append(out.PromptSet.Prompts, oas.PromptQuestion{
			ID: p.ID, Family: oas.PromptFamily(p.Family), Kind: oas.PromptKind(p.Kind), Scope: oas.PromptScope(p.Scope),
			Language: p.Language, Text: p.Text,
		})
	}
	return out
}

func toCandidate(c domain.Candidate) oas.CompetitorCandidate {
	out := oas.CompetitorCandidate{
		ID: oas.CompetitorCandidateId(c.ID), Name: c.Name, Status: oas.CompetitorCandidateStatus(c.Status()),
		Answers: int32(c.Answers), Providers: c.Providers, Models: c.Models[:min(len(c.Models), 50)], //nolint:gosec // answers
		Addresses:  c.Addresses[:min(len(c.Addresses), 50)],
		LastSeenAt: c.LastSeen,
	}
	if d := c.Decision; d != nil {
		dec := oas.CompetitorCandidateDecision{Status: oas.CompetitorCandidateStatus(d.Status), By: d.By, At: d.At}
		if d.CompetitorID != "" {
			dec.CompetitorId = oas.NewOptCompetitorId(oas.CompetitorId(d.CompetitorID))
		}
		out.Decision = oas.NewOptCompetitorCandidateDecision(dec)
	}
	return out
}

func orEmpty(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}

// sentence is an error's message as a sentence, without the wrapped "invalid: ".
func sentence(err error) string {
	msg := strings.TrimPrefix(err.Error(), domain.ErrInvalid.Error()+": ")
	r := []rune(strings.TrimSuffix(msg, ".") + ".")
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func toPrices(p domain.Prices) oas.PromptPrices {
	out := oas.PromptPrices{
		Basis: oas.PromptPriceBasis(p.Basis), Providers: make([]oas.PromptProviderPrice, 0, len(p.Providers)),
		ReadingUsd: oas.Usd(p.Reading.Dollars()), RunUsd: oas.Usd(p.Run.Dollars()), RunsPerMonth: p.RunsPerMonth,
		MonthUsd: oas.Usd(p.Month.Dollars()), BudgetUsd: oas.Usd(p.Budget.Dollars()), OverBudget: p.OverBudget(),
	}
	for _, pp := range p.Providers {
		o := oas.PromptProviderPrice{
			Provider: pp.Provider, Model: pp.Model, KnowledgeUsd: oas.Usd(pp.Knowledge.Dollars()), WebUsd: oas.Usd(pp.Web.Dollars()),
			RunUsd: oas.Usd(pp.PerRun.Dollars()),
		}
		if pp.Fallback != "" {
			o.Fallback = oas.NewOptString(pp.Fallback)
		}
		out.Providers = append(out.Providers, o)
	}
	return out
}

func toModels(models []domain.ModelChoice) []oas.PromptModelChoice {
	out := make([]oas.PromptModelChoice, 0, len(models))
	for _, m := range models {
		c := oas.PromptModelChoice{Provider: m.Provider, Model: m.Model, Free: m.Free}
		if m.Fallback != "" {
			c.Fallback = oas.NewOptString(m.Fallback)
		}
		out = append(out, c)
	}
	return out
}
