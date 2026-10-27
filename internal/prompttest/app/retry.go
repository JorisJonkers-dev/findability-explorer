package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// Retry is a run asked again: the run, running once more, how many answers it asks, and what
// they should cost.
type Retry struct {
	Run      domain.Run
	Tasks    int
	Estimate domain.Micros
}

// Errors asking again returns besides the ports' own.
var (
	// ErrEarlierSet is returned for a run of a prompt set this release no longer asks.
	ErrEarlierSet = fmt.Errorf("%w: this run asked an earlier set of questions, which is no longer asked", domain.ErrInvalid)
	// ErrNothingFailed is returned when the run, or the one question named, has no failed answer.
	ErrNothingFailed = fmt.Errorf("%w: no answer to ask again", domain.ErrInvalid)
)

// Retry asks a finished run's failed answers again, or the one task named, in the same run: with
// the run's prompt set and models, each new answer replacing the failed one. It refuses while a
// run is running (domain.ErrRunActive) or once the month's budget is spent
// (domain.ErrBudgetReached), and stops at the budget like any run. The answers are asked in the
// background.
func (s *Service) Retry(ctx context.Context, runID uuid.UUID, task *domain.TaskKey) (Retry, error) {
	run, err := s.repo.Run(ctx, runID)
	if err != nil {
		return Retry{}, err
	}
	if run.Status == domain.RunRunning {
		return Retry{}, domain.ErrRunActive
	}
	set := domain.CurrentPromptSet()
	if run.PromptSetVersion != set.Version {
		return Retry{}, ErrEarlierSet
	}
	answers, err := s.repo.Answers(ctx, runID, false)
	if err != nil {
		return Retry{}, fmt.Errorf("the run's answers: %w", err)
	}
	tasks := domain.Unanswered(set, run.Models, answers)
	if task != nil {
		tasks = keep(tasks, *task)
	}
	if len(tasks) == 0 {
		return Retry{}, ErrNothingFailed
	}
	if err := s.affordable(ctx); err != nil {
		return Retry{}, err
	}
	if err := s.repo.ReopenRun(ctx, runID, s.cfg.Owner, s.now()); err != nil {
		if errors.Is(err, domain.ErrRunActive) {
			return Retry{}, err
		}
		return Retry{}, fmt.Errorf("reopen the run: %w", err)
	}
	only := map[domain.TaskKey]bool{}
	for _, t := range tasks {
		only[t.Key()] = true
	}
	run = run.Reopened()
	s.spawn(func() {
		if err := s.execute(context.WithoutCancel(context.Background()), run, only); err != nil {
			s.onFail(run.ID, err)
		}
	})
	return Retry{Run: run, Tasks: len(tasks), Estimate: domain.Estimate(answers, tasks)}, nil
}

// affordable checks that a key is chosen and the month's budget is not spent.
func (s *Service) affordable(ctx context.Context) error {
	settings, err := s.settings(ctx)
	if err != nil {
		return err
	}
	if err := s.gateways.Check(ctx, settings.CredentialID); err != nil {
		return err
	}
	spent, err := s.repo.SpentSince(ctx, domain.MonthStart(s.now()))
	if err != nil {
		return fmt.Errorf("this month's spend: %w", err)
	}
	if spent >= settings.MonthlyBudget {
		return domain.ErrBudgetReached
	}
	return nil
}

func keep(tasks []domain.Task, key domain.TaskKey) []domain.Task {
	for _, t := range tasks {
		if t.Key() == key {
			return []domain.Task{t}
		}
	}
	return nil
}
