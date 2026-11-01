package app

import (
	"context"
	"fmt"
	"math"
	"slices"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// allRuns asks the store for every run: the history lists every run with a response read, and the
// results pick the newest complete one, however far back.
const allRuns = math.MaxInt32

// MaxWrongClaims is how many wrong claims the results list.
const MaxWrongClaims = 1000

// RunResult is one run with how far it got, its measures, and the reason most of its failed
// answers give ("" when none failed).
type RunResult struct {
	Run           domain.Run
	Progress      domain.Progress
	Cells         []domain.Cell
	CommonFailure string
}

// Results are the Prompt test's results: every run, newest first, each with its measures (the
// trends) and whether it is complete; how often the answers of the newest complete run name each
// company; and the claims about Tribelt the answers got wrong, newest first.
type Results struct {
	// PromptSetVersion is the newest published Prompt set's version, the one a new run asks.
	PromptSetVersion int
	Runs             []RunResult
	Mentions         []NamedMention
	WrongClaims      []domain.WrongClaim
}

// NamedMention is a mention with the name and role of the company it counts, and its share of
// the run's answers that count; a company no longer on the list goes by its id, as a competitor.
type NamedMention struct {
	domain.Mention
	Name  string
	Role  domain.Role
	Share float64
}

// named gives each mention its company's name and role, and its share of the answers that count
// (the AI share of voice every page shows).
func named(mentions []domain.Mention, answers []domain.Answer, companies []domain.DirectoryCompany) []NamedMention {
	counted := 0
	for _, a := range answers {
		if a.Counts() {
			counted++
		}
	}
	out := make([]NamedMention, 0, len(mentions))
	for _, m := range mentions {
		n := NamedMention{Mention: m, Name: m.CompetitorID, Role: domain.RoleCompetitor, Share: float64(m.Answers) / float64(max(counted, 1))}
		for _, c := range companies {
			if c.ID == m.CompetitorID {
				n.Name, n.Role = c.Name, c.Role
			}
		}
		out = append(out, n)
	}
	return out
}

// Results reads the latest runs and their answers, each named company matched to the list as it is now.
func (s *Service) Results(ctx context.Context) (Results, error) {
	runs, err := s.repo.Runs(ctx, allRuns)
	if err != nil {
		return Results{}, fmt.Errorf("list the runs: %w", err)
	}
	set, err := s.newestSet(ctx)
	if err != nil {
		return Results{}, err
	}
	m, companies, err := s.matcher(ctx)
	if err != nil {
		return Results{}, err
	}
	out := Results{PromptSetVersion: set.Version, Runs: make([]RunResult, 0, len(runs)), Mentions: []NamedMention{}, WrongClaims: []domain.WrongClaim{}}
	mentioned := false
	for _, r := range runs {
		answers, err := s.repo.Answers(ctx, r.ID, false)
		if err != nil {
			return Results{}, fmt.Errorf("the answers of run %s: %w", r.ID, err)
		}
		answers = rematched(answers, m)
		cells := domain.Cells(answers)
		out.Runs = append(out.Runs, RunResult{Run: r, Progress: domain.Progressed(answers, r.Models), Cells: cells, CommonFailure: domain.CommonestFailure(answers)})
		if !mentioned && r.Complete() && len(cells) > 0 {
			out.Mentions, mentioned = named(domain.WithTribelt(domain.Mentions(answers), answers, companies), answers, companies), true
		}
		out.WrongClaims = append(out.WrongClaims, domain.WrongClaims(answers)...)
	}
	out.WrongClaims = out.WrongClaims[:min(len(out.WrongClaims), MaxWrongClaims)]
	return out, nil
}

// AnswerFilter picks a run's answers; an empty field keeps every value.
type AnswerFilter struct {
	Provider string
	Scope    domain.Scope
	Family   domain.Family
	Mode     domain.Mode
}

func (f AnswerFilter) keeps(a domain.Answer) bool {
	return (f.Provider == "" || a.Provider == f.Provider) && (f.Scope == "" || a.Prompt.Scope == f.Scope) &&
		(f.Family == "" || a.Prompt.Family == f.Family) && (f.Mode == "" || a.Mode == f.Mode)
}

// Answers returns a run, with its progress and measures, and the answers the filter keeps, with
// their text and how they were read; domain.ErrNotFound for a run nobody knows.
func (s *Service) Answers(ctx context.Context, runID uuid.UUID, filter AnswerFilter) (RunResult, []domain.Answer, error) {
	run, err := s.repo.Run(ctx, runID)
	if err != nil {
		return RunResult{}, nil, err
	}
	answers, err := s.repo.Answers(ctx, runID, true)
	if err != nil {
		return RunResult{}, nil, fmt.Errorf("the run's answers: %w", err)
	}
	m, _, err := s.matcher(ctx)
	if err != nil {
		return RunResult{}, nil, err
	}
	answers = rematched(answers, m)
	kept := make([]domain.Answer, 0, len(answers))
	for _, a := range answers {
		if filter.keeps(a) {
			kept = append(kept, a)
		}
	}
	return RunResult{
		Run: run, Progress: domain.Progressed(answers, run.Models), Cells: domain.Cells(answers), CommonFailure: domain.CommonestFailure(answers),
	}, kept, nil
}

// RunAnswers is a run with the answers that count, without their text.
type RunAnswers struct {
	Run     domain.Run
	Answers []domain.Answer
}

// Counted returns at most limit of the newest complete runs that have answers that count, newest
// first, each with only those answers: what other contexts compare companies on.
func (s *Service) Counted(ctx context.Context, limit int) ([]RunAnswers, error) {
	runs, err := s.repo.Runs(ctx, allRuns)
	if err != nil {
		return nil, fmt.Errorf("list the runs: %w", err)
	}
	m, _, err := s.matcher(ctx)
	if err != nil {
		return nil, err
	}
	out := []RunAnswers{}
	for _, r := range runs {
		if len(out) == limit {
			break
		}
		if !r.Complete() {
			continue
		}
		answers, err := s.repo.Answers(ctx, r.ID, false)
		if err != nil {
			return nil, fmt.Errorf("the answers of run %s: %w", r.ID, err)
		}
		answers = rematched(answers, m)
		kept := slices.DeleteFunc(answers, func(a domain.Answer) bool { return !a.Counts() })
		if len(kept) > 0 {
			out = append(out, RunAnswers{Run: r, Answers: kept})
		}
	}
	return out, nil
}
