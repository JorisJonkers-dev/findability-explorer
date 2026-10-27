// Package persistence implements the prompt-test Repository port on the sqlc-generated queries.
package persistence

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/queries"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// Store is what the repository needs from the Postgres adapter.
type Store interface {
	Queries() *queries.Queries
	InTx(ctx context.Context, fn func(*queries.Queries) error) error
}

// Repository maps rows to the prompt-test domain, so the domain never sees a generated type.
type Repository struct {
	store Store
}

var _ app.Repository = (*Repository)(nil)

// New returns a Repository over store.
func New(store Store) *Repository { return &Repository{store: store} }

// Settings returns the stored settings, or ok false when none are stored yet.
func (r *Repository) Settings(ctx context.Context) (domain.Settings, bool, error) {
	rows, err := r.store.Queries().GetPromptSettings(ctx)
	if err != nil {
		return domain.Settings{}, false, fmt.Errorf("select prompt settings: %w", err)
	}
	if len(rows) == 0 {
		return domain.Settings{}, false, nil
	}
	row := rows[0]
	return domain.Settings{
		CredentialID: row.CredentialID, Providers: row.Providers, MonthlyBudget: domain.Micros(row.MonthlyBudgetMicros),
		Cadence: domain.Cadence(row.Cadence), NextDue: row.NextDue.UTC(),
		FactSheet: domain.FactSheet{
			Text: row.FactSheet, Version: int(row.FactSheetVersion), UpdatedBy: row.FactSheetUpdatedBy, UpdatedAt: timeOf(row.FactSheetUpdatedAt),
		},
	}, true, nil
}

// SaveSettings stores the settings in place of the earlier ones.
func (r *Repository) SaveSettings(ctx context.Context, s domain.Settings, by string, at time.Time) error {
	if err := r.store.Queries().SavePromptSettings(ctx, queries.SavePromptSettingsParams{
		CredentialID: optionalText(s.CredentialID), Providers: s.Providers, MonthlyBudgetMicros: int64(s.MonthlyBudget),
		Cadence: string(s.Cadence), NextDue: s.NextDue, FactSheet: s.FactSheet.Text, FactSheetVersion: int64(s.FactSheet.Version),
		FactSheetUpdatedBy: optionalText(s.FactSheet.UpdatedBy), FactSheetUpdatedAt: optionalTime(s.FactSheet.UpdatedAt),
		UpdatedBy: by, UpdatedAt: at,
	}); err != nil {
		return fmt.Errorf("save prompt settings: %w", err)
	}
	return nil
}

// CreateRun stores a running run held by owner, or returns domain.ErrRunActive.
func (r *Repository) CreateRun(ctx context.Context, run domain.Run, owner string, at time.Time) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error { return createRun(ctx, q, run, owner, at) })
}

func createRun(ctx context.Context, q *queries.Queries, run domain.Run, owner string, at time.Time) error {
	n, err := q.CreatePromptRun(ctx, queries.CreatePromptRunParams{
		ID: run.ID, PromptSetVersion: int64(run.PromptSetVersion), FactSheetVersion: int64(run.FactSheetVersion),
		FactSheet: run.FactSheet, Trigger: string(run.Trigger), StartedBy: run.StartedBy, StartedAt: run.StartedAt,
		Tasks: int64(run.Tasks), Owner: owner, HeartbeatAt: at,
	})
	if err != nil {
		return fmt.Errorf("insert prompt run: %w", err)
	}
	if n == 0 {
		return domain.ErrRunActive
	}
	for i, m := range run.Models {
		if err := q.AddPromptRunModel(ctx, queries.AddPromptRunModelParams{
			RunID: run.ID, Position: int64(i), Provider: m.Provider, Model: m.Model, Free: m.Free, Fallback: m.Fallback,
		}); err != nil {
			return fmt.Errorf("insert prompt run model: %w", err)
		}
	}
	return nil
}

// ClaimDue starts run when the schedule is due at now and no run is running, moving the schedule
// to next, in one transaction that locks the schedule's row: of two instances, one claims it.
func (r *Repository) ClaimDue(ctx context.Context, now, next time.Time, run domain.Run, owner string) (bool, error) {
	claimed := false
	err := r.store.InTx(ctx, func(q *queries.Queries) error {
		due, err := q.LockPromptSchedule(ctx)
		if err != nil {
			return fmt.Errorf("lock the schedule: %w", err)
		}
		if len(due) == 0 || now.Before(due[0]) {
			return nil
		}
		switch err := createRun(ctx, q, run, owner, now); {
		case errors.Is(err, domain.ErrRunActive):
			return nil // still due: it starts once the running one ends
		case err != nil:
			return err
		}
		if err := q.MovePromptSchedule(ctx, next); err != nil {
			return fmt.Errorf("move the schedule: %w", err)
		}
		claimed = true
		return nil
	})
	return claimed, err
}

// AdoptStale takes over the running run whose holder was last heard of before staleBefore.
func (r *Repository) AdoptStale(ctx context.Context, owner string, staleBefore, now time.Time) (*domain.Run, error) {
	ids, err := r.store.Queries().AdoptStalePromptRun(ctx, queries.AdoptStalePromptRunParams{Owner: owner, Now: now, StaleBefore: staleBefore})
	if err != nil {
		return nil, fmt.Errorf("adopt a stale run: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil //nolint:nilnil // no run to take over
	}
	run, err := r.Run(ctx, ids[0])
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// Heartbeat records that owner still runs the run; false when another instance took it.
func (r *Repository) Heartbeat(ctx context.Context, id uuid.UUID, owner string, at time.Time) (bool, error) {
	n, err := r.store.Queries().HeartbeatPromptRun(ctx, queries.HeartbeatPromptRunParams{ID: id, Owner: owner, At: at})
	if err != nil {
		return false, fmt.Errorf("heartbeat: %w", err)
	}
	return n > 0, nil
}

// FinishRun ends a running run.
func (r *Repository) FinishRun(ctx context.Context, id uuid.UUID, status domain.RunStatus, failure string, at time.Time) error {
	if err := r.store.Queries().FinishPromptRun(ctx, queries.FinishPromptRunParams{
		ID: id, Status: string(status), Failure: optionalText(failure), FinishedAt: optionalTime(at),
	}); err != nil {
		return fmt.Errorf("finish prompt run: %w", err)
	}
	return nil
}

// Run returns one run with its models, or domain.ErrNotFound.
func (r *Repository) Run(ctx context.Context, id uuid.UUID) (domain.Run, error) {
	q := r.store.Queries()
	row, err := q.GetPromptRun(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Run{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Run{}, fmt.Errorf("select prompt run: %w", err)
	}
	runs, err := withModels(ctx, q, []queries.ListPromptRunsRow{queries.ListPromptRunsRow(row)})
	if err != nil {
		return domain.Run{}, err
	}
	return runs[0], nil
}

// Runs returns at most limit runs, newest first.
func (r *Repository) Runs(ctx context.Context, limit int) ([]domain.Run, error) {
	q := r.store.Queries()
	rows, err := q.ListPromptRuns(ctx, int32(min(max(limit, 0), math.MaxInt32))) //nolint:gosec // clamped
	if err != nil {
		return nil, fmt.Errorf("select prompt runs: %w", err)
	}
	return withModels(ctx, q, rows)
}

func withModels(ctx context.Context, q *queries.Queries, rows []queries.ListPromptRunsRow) ([]domain.Run, error) {
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	models, err := q.ListPromptRunModels(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("select prompt run models: %w", err)
	}
	byRun := map[uuid.UUID][]domain.ModelChoice{}
	for _, m := range models {
		byRun[m.RunID] = append(byRun[m.RunID], domain.ModelChoice{Provider: m.Provider, Model: m.Model, Free: m.Free, Fallback: m.Fallback})
	}
	out := make([]domain.Run, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Run{
			ID: row.ID, PromptSetVersion: int(row.PromptSetVersion), FactSheetVersion: int(row.FactSheetVersion),
			FactSheet: row.FactSheet, Trigger: domain.Trigger(row.Trigger), StartedBy: row.StartedBy, StartedAt: row.StartedAt.UTC(),
			FinishedAt: timeOf(row.FinishedAt), Status: domain.RunStatus(row.Status), Failure: row.Failure.String,
			Models: byRun[row.ID], Tasks: int(row.Tasks),
		})
	}
	return out, nil
}

// ReopenRun runs a finished run again, held by owner, or returns domain.ErrRunActive when a run
// is running: the same lock that keeps one run running at a time.
func (r *Repository) ReopenRun(ctx context.Context, id uuid.UUID, owner string, at time.Time) error {
	n, err := r.store.Queries().ReopenPromptRun(ctx, queries.ReopenPromptRunParams{ID: id, Owner: owner, At: at})
	if isUniqueViolation(err) || (err == nil && n == 0) {
		return domain.ErrRunActive
	}
	if err != nil {
		return fmt.Errorf("reopen prompt run: %w", err)
	}
	return nil
}

// AddAnswer stores an answer with its extraction, if it has one, in one transaction. An answer to
// a task whose earlier answer failed replaces it, adding to what it cost; a task answered already
// is domain.ErrAnswered.
func (r *Repository) AddAnswer(ctx context.Context, a domain.Answer) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error {
		id, err := q.AddPromptAnswer(ctx, queries.AddPromptAnswerParams{
			ID: a.ID, RunID: a.RunID, PromptID: a.Prompt.ID, Family: string(a.Prompt.Family), Kind: string(a.Prompt.Kind),
			Scope: string(a.Prompt.Scope), Language: a.Prompt.Language, PromptText: clip(a.Prompt.Text, 1000),
			PromptSetVersion: int64(a.PromptSetVersion), Provider: a.Provider, Model: clip(a.Model, 200), Mode: string(a.Mode),
			Answer: clip(clean(a.Text), 100_000), Citations: clipAll(a.Citations, 100, 500), InputTokens: a.InputTokens,
			OutputTokens: a.OutputTokens, CostMicros: max(int64(a.Cost), 0), Failure: optionalText(clip(a.Failure, 300)),
			AnsweredAt: a.At,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAnswered
		}
		if err != nil {
			return fmt.Errorf("insert prompt answer: %w", err)
		}
		if a.Extraction == nil {
			return nil
		}
		return addExtraction(ctx, q, id, *a.Extraction, a.At)
	})
}

// SetExtraction stores the extraction of an answer stored without one, or replaces a failed one,
// adding to what it used; an answer read already is domain.ErrRead.
func (r *Repository) SetExtraction(ctx context.Context, answerID uuid.UUID, x domain.Extraction, at time.Time) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error { return addExtraction(ctx, q, answerID, x, at) })
}

func addExtraction(ctx context.Context, q *queries.Queries, id uuid.UUID, x domain.Extraction, at time.Time) error {
	n, err := q.AddPromptExtraction(ctx, queries.AddPromptExtractionParams{
		AnswerID: id, Failed: x.Failed, Addresses: clipAll(x.Addresses, 200, 500), NamesTribelt: x.NamesTribelt,
		CitesTribelt: x.CitesTribelt, Correctness: string(x.Correctness), WrongClaims: clipAll(x.WrongClaims, 10, 500),
		InputTokens: x.InputTokens, OutputTokens: x.OutputTokens, CostMicros: max(int64(x.Cost), 0),
		HaikuTokens: max(x.HaikuTokens, 0), ExtractedAt: at,
	})
	if err != nil {
		return fmt.Errorf("insert prompt extraction: %w", err)
	}
	if n == 0 {
		return domain.ErrRead
	}
	for _, c := range x.Companies[:min(len(x.Companies), 30)] {
		if err := q.AddPromptAnswerCompany(ctx, queries.AddPromptAnswerCompanyParams{
			AnswerID: id, Position: int64(c.Position), Name: clip(c.Name, 200), Url: clip(c.URL, 500),
			CompetitorID: optionalText(c.CompetitorID),
		}); err != nil {
			return fmt.Errorf("insert named company: %w", err)
		}
	}
	return nil
}

// Answers returns a run's answers, oldest first, with their text only when withText is true.
func (r *Repository) Answers(ctx context.Context, runID uuid.UUID, withText bool) ([]domain.Answer, error) {
	q := r.store.Queries()
	rows, err := q.ListPromptAnswers(ctx, queries.ListPromptAnswersParams{RunID: runID, WithText: withText})
	if err != nil {
		return nil, fmt.Errorf("select prompt answers: %w", err)
	}
	companies, err := q.ListPromptAnswerCompanies(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("select named companies: %w", err)
	}
	named := map[uuid.UUID][]domain.Company{}
	for _, c := range companies {
		named[c.AnswerID] = append(named[c.AnswerID], domain.Company{Name: c.Name, URL: c.Url, CompetitorID: c.CompetitorID, Position: int(c.Position)})
	}
	out := make([]domain.Answer, 0, len(rows))
	for _, row := range rows {
		a := domain.Answer{
			ID: row.ID, RunID: runID, PromptSetVersion: int(row.PromptSetVersion), Provider: row.Provider, Model: row.Model,
			Mode: domain.Mode(row.Mode), Failure: row.Failure, At: row.AnsweredAt.UTC(),
			Prompt: domain.Prompt{
				ID: row.PromptID, Family: domain.Family(row.Family), Kind: domain.Kind(row.Kind), Scope: domain.Scope(row.Scope),
				Language: row.Language, Text: row.PromptText,
			},
			Reply: domain.Reply{
				Text: row.Answer, Citations: row.Citations, InputTokens: row.InputTokens, OutputTokens: row.OutputTokens,
				Cost: domain.Micros(row.CostMicros),
			},
		}
		if row.Extracted {
			a.Extraction = &domain.Extraction{
				Failed: row.ExtractionFailed, Companies: named[row.ID], Addresses: row.Addresses, NamesTribelt: row.NamesTribelt,
				CitesTribelt: row.CitesTribelt, Correctness: domain.Correctness(row.Correctness), WrongClaims: row.WrongClaims,
				InputTokens: row.ExtractionInputTokens, OutputTokens: row.ExtractionOutputTokens, Cost: domain.Micros(row.ExtractionCostMicros),
			}
		}
		out = append(out, a)
	}
	return out, nil
}

// SpentSince sums the cost of every answer and extraction at or after since.
func (r *Repository) SpentSince(ctx context.Context, since time.Time) (domain.Micros, error) {
	n, err := r.store.Queries().PromptSpentSince(ctx, since)
	if err != nil {
		return 0, fmt.Errorf("sum prompt spend: %w", err)
	}
	return domain.Micros(n), nil
}

// Sightings returns every name an answer gave that was not matched to a company.
func (r *Repository) Sightings(ctx context.Context) ([]domain.Sighting, error) {
	rows, err := r.store.Queries().PromptSightings(ctx)
	if err != nil {
		return nil, fmt.Errorf("select unmatched names: %w", err)
	}
	out := make([]domain.Sighting, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Sighting{AnswerID: row.AnswerID, Name: row.Name, URL: row.Url, Provider: row.Provider, Model: row.Model, At: row.AnsweredAt.UTC()})
	}
	return out, nil
}

// CandidateDecisions returns every decision on a competitor candidate.
func (r *Repository) CandidateDecisions(ctx context.Context) ([]domain.CandidateDecision, error) {
	rows, err := r.store.Queries().ListPromptCandidateDecisions(ctx)
	if err != nil {
		return nil, fmt.Errorf("select candidate decisions: %w", err)
	}
	out := make([]domain.CandidateDecision, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CandidateDecision{
			ID: row.ID, Name: row.Name, Status: domain.CandidateStatus(row.Status), CompetitorID: row.CompetitorID,
			By: row.DecidedBy, At: row.DecidedAt.UTC(),
		})
	}
	return out, nil
}

// Decide stores a decision on a candidate.
func (r *Repository) Decide(ctx context.Context, d domain.CandidateDecision) error {
	if err := r.store.Queries().AddPromptCandidateDecision(ctx, queries.AddPromptCandidateDecisionParams{
		ID: d.ID, Name: clip(d.Name, 200), Status: string(d.Status), CompetitorID: optionalText(d.CompetitorID),
		DecidedBy: d.By, DecidedAt: d.At,
	}); err != nil {
		return fmt.Errorf("insert candidate decision: %w", err)
	}
	return nil
}

// Undecide deletes a decision, or returns domain.ErrNotFound.
func (r *Repository) Undecide(ctx context.Context, id string) error {
	n, err := r.store.Queries().DeletePromptCandidateDecision(ctx, id)
	if err != nil {
		return fmt.Errorf("delete candidate decision: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Merges returns every merge of candidates.
func (r *Repository) Merges(ctx context.Context) ([]domain.Merge, error) {
	rows, err := r.store.Queries().ListPromptCandidateMerges(ctx)
	if err != nil {
		return nil, fmt.Errorf("select candidate merges: %w", err)
	}
	out := make([]domain.Merge, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Merge{
			CandidateID: row.CandidateID, IntoID: row.IntoID, CompetitorID: row.CompetitorID, Separate: row.KeptApart,
			By: row.MergedBy, At: row.MergedAt.UTC(),
		})
	}
	return out, nil
}

// SaveMerges stores merges in one transaction, each replacing the one on its candidate.
func (r *Repository) SaveMerges(ctx context.Context, merges []domain.Merge) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error {
		for _, m := range merges {
			if err := q.SavePromptCandidateMerge(ctx, queries.SavePromptCandidateMergeParams{
				CandidateID: m.CandidateID, IntoID: optionalText(m.IntoID), CompetitorID: optionalText(m.CompetitorID),
				KeptApart: m.Separate, MergedBy: clip(m.By, 200), MergedAt: m.At,
			}); err != nil {
				return fmt.Errorf("store candidate merge: %w", err)
			}
		}
		return nil
	})
}

// CompanyVerdicts returns every check of a candidate by the language model.
func (r *Repository) CompanyVerdicts(ctx context.Context) ([]domain.CompanyVerdict, error) {
	rows, err := r.store.Queries().ListPromptCandidateChecks(ctx)
	if err != nil {
		return nil, fmt.Errorf("select candidate checks: %w", err)
	}
	out := make([]domain.CompanyVerdict, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CompanyVerdict{
			CandidateID: row.CandidateID, Competitor: row.Competitor, Sure: row.Sure, Reason: row.Reason, Model: row.Model,
			Version: int(row.CheckVersion), At: row.CheckedAt.UTC(),
		})
	}
	return out, nil
}

// SaveCompanyVerdicts stores verdicts in one transaction, each replacing the one on its candidate.
func (r *Repository) SaveCompanyVerdicts(ctx context.Context, verdicts []domain.CompanyVerdict) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error {
		for _, v := range verdicts {
			if err := q.SavePromptCandidateCheck(ctx, queries.SavePromptCandidateCheckParams{
				CandidateID: v.CandidateID, Competitor: v.Competitor, Sure: v.Sure, Reason: clip(v.Reason, 500),
				Model: clip(v.Model, 200), CheckVersion: int64(max(v.Version, 1)), CheckedAt: v.At,
			}); err != nil {
				return fmt.Errorf("store candidate check: %w", err)
			}
		}
		return nil
	})
}

// ListChecks returns every check of a company on the list by the language model.
func (r *Repository) ListChecks(ctx context.Context) ([]domain.ListCheck, error) {
	rows, err := r.store.Queries().ListPromptCompanyChecks(ctx)
	if err != nil {
		return nil, fmt.Errorf("select company checks: %w", err)
	}
	out := make([]domain.ListCheck, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ListCheck{
			CompanyID: row.CompanyID, Competitor: row.Competitor, Sure: row.Sure, Reason: row.Reason, Model: row.Model,
			Version: int(row.CheckVersion), At: row.CheckedAt.UTC(),
		})
	}
	return out, nil
}

// SaveListChecks stores checks in one transaction, each replacing the one on its company.
func (r *Repository) SaveListChecks(ctx context.Context, checks []domain.ListCheck) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error {
		for _, c := range checks {
			if err := q.SavePromptCompanyCheck(ctx, queries.SavePromptCompanyCheckParams{
				CompanyID: c.CompanyID, Competitor: c.Competitor, Sure: c.Sure, Reason: clip(c.Reason, 500),
				Model: clip(c.Model, 200), CheckVersion: int64(max(c.Version, 1)), CheckedAt: c.At,
			}); err != nil {
				return fmt.Errorf("store company check: %w", err)
			}
		}
		return nil
	})
}

// Namings returns every name an answer gave, matched or not, the newest answer first.
func (r *Repository) Namings(ctx context.Context) ([]domain.Naming, error) {
	rows, err := r.store.Queries().PromptNamings(ctx)
	if err != nil {
		return nil, fmt.Errorf("select the names answers gave: %w", err)
	}
	out := make([]domain.Naming, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Naming{AnswerID: row.AnswerID, Name: row.Name, URL: row.Url, CompetitorID: row.CompetitorID})
	}
	return out, nil
}

// AnswerTexts returns the text of each answer of ids there is, by id.
func (r *Repository) AnswerTexts(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	out := map[uuid.UUID]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.store.Queries().ListPromptAnswerTexts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("select answer texts: %w", err)
	}
	for _, row := range rows {
		out[row.ID] = row.Answer
	}
	return out, nil
}

// SaveReviewUsage stores what requests to the language model for the review consumed.
func (r *Repository) SaveReviewUsage(ctx context.Context, usage []domain.ReviewUsage) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error {
		for _, u := range usage {
			if err := q.InsertPromptReviewUsage(ctx, queries.InsertPromptReviewUsageParams{
				InputTokens: max(u.InputTokens, 0), OutputTokens: max(u.OutputTokens, 0), UsedAt: u.At,
			}); err != nil {
				return fmt.Errorf("store review usage: %w", err)
			}
		}
		return nil
	})
}

// ReviewTokensSince is how many tokens the review of the candidates used since t.
func (r *Repository) ReviewTokensSince(ctx context.Context, t time.Time) (int64, error) {
	n, err := r.store.Queries().PromptReviewTokensSince(ctx, t)
	if err != nil {
		return 0, fmt.Errorf("sum review tokens: %w", err)
	}
	return n, nil
}

// ReadingTokensSince is how many tokens Claude Haiku on an Anthropic key used reading answers since t.
func (r *Repository) ReadingTokensSince(ctx context.Context, t time.Time) (int64, error) {
	n, err := r.store.Queries().PromptReadingTokensSince(ctx, t)
	if err != nil {
		return 0, fmt.Errorf("sum reading tokens: %w", err)
	}
	return n, nil
}

// optionalText is s, or NULL when s is empty.
func optionalText(s string) pgtype.Text { return pgtype.Text{String: s, Valid: s != ""} }

// optionalTime is t, or NULL when t is zero.
func optionalTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

func timeOf(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}

// clip shortens s to at most n characters.
func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// clipAll keeps at most n items, each at most chars characters.
func clipAll(items []string, n, chars int) []string {
	out := make([]string, 0, min(len(items), n))
	for _, s := range items[:min(len(items), n)] {
		out = append(out, clip(s, chars))
	}
	return out
}

// clean drops what Postgres text cannot hold: invalid UTF-8 and NUL characters.
func clean(s string) string { return strings.ReplaceAll(strings.ToValidUTF8(s, ""), "\x00", "") }

// isUniqueViolation reports whether err is Postgres refusing a second row a unique index forbids.
func isUniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}
