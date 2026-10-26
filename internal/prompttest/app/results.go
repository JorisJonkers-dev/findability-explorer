package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// MaxRuns is how many runs the results go back.
const MaxRuns = 24

// MaxWrongClaims is how many wrong claims the results list.
const MaxWrongClaims = 200

// RunResult is one run with how far it got and its measures.
type RunResult struct {
	Run      domain.Run
	Progress domain.Progress
	Cells    []domain.Cell
}

// Results are the Prompt test's results: the latest runs, newest first, each with its measures
// (the trends); how often the newest run's answers name each company; and the claims about
// Tribelt the answers got wrong, newest first.
type Results struct {
	Runs        []RunResult
	Mentions    []domain.Mention
	Companies   []domain.DirectoryCompany
	WrongClaims []domain.WrongClaim
}

// Results reads the latest runs and their answers.
func (s *Service) Results(ctx context.Context) (Results, error) {
	runs, err := s.repo.Runs(ctx, MaxRuns)
	if err != nil {
		return Results{}, fmt.Errorf("list the runs: %w", err)
	}
	companies, err := s.directory.Companies(ctx)
	if err != nil {
		return Results{}, fmt.Errorf("list the companies: %w", err)
	}
	out := Results{Runs: make([]RunResult, 0, len(runs)), Mentions: []domain.Mention{}, Companies: companies, WrongClaims: []domain.WrongClaim{}}
	for i, r := range runs {
		answers, err := s.repo.Answers(ctx, r.ID, false)
		if err != nil {
			return Results{}, fmt.Errorf("the answers of run %s: %w", r.ID, err)
		}
		out.Runs = append(out.Runs, RunResult{Run: r, Progress: domain.Progressed(answers, r.Models), Cells: domain.Cells(answers)})
		if i == 0 {
			out.Mentions = domain.Mentions(answers)
		}
		out.WrongClaims = append(out.WrongClaims, domain.WrongClaims(answers)...)
	}
	out.WrongClaims = out.WrongClaims[:min(len(out.WrongClaims), MaxWrongClaims)]
	return out, nil
}

// Answers returns a run's answers with their text and how they were read, or domain.ErrNotFound.
func (s *Service) Answers(ctx context.Context, runID uuid.UUID) (domain.Run, []domain.Answer, error) {
	run, err := s.repo.Run(ctx, runID)
	if err != nil {
		return domain.Run{}, nil, err
	}
	answers, err := s.repo.Answers(ctx, runID, true)
	if err != nil {
		return domain.Run{}, nil, fmt.Errorf("the run's answers: %w", err)
	}
	return run, answers, nil
}
