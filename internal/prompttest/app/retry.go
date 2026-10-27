package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// Retry is a run asked or read again: the run, running once more, how many answers it asks or
// reads, and what that should cost.
type Retry struct {
	Run      domain.Run
	Tasks    int
	Estimate domain.Micros
}

var (
	// ErrNothingFailed is returned when the run, or the one question named, has no failed answer.
	ErrNothingFailed = fmt.Errorf("%w: no answer to ask again", domain.ErrInvalid)
	// ErrNothingUnread is returned when the run, or the one answer named, has no failed reading.
	ErrNothingUnread = fmt.Errorf("%w: no answer to read again", domain.ErrInvalid)
)

// Plan is what asking a run again would do: the failed and missing answers it would ask and what
// they should cost, how many it cannot ask because no answer kept their question, the answers whose
// reading failed and what reading them again should cost, and one answer of each provider and mode
// asked again, reading included, and one reading.
type Plan struct {
	Ask      int
	AskCost  domain.Micros
	Unknown  int
	Read     int
	ReadCost domain.Micros
	Costs    []domain.ModeCost
	Reading  domain.Micros
}

// RetryPlan is what asking a run again would ask and read, and cost, with reading priced at the
// reader that reads now.
func (s *Service) RetryPlan(ctx context.Context, runID uuid.UUID) (Plan, error) {
	run, answers, err := s.runAndAnswers(ctx, runID)
	if err != nil {
		return Plan{}, err
	}
	settings, err := s.settings(ctx)
	if err != nil {
		return Plan{}, err
	}
	reading, err := s.readingCost(ctx, answers, settings.CredentialID)
	if err != nil {
		return Plan{}, err
	}
	set := domain.QuestionsOf(run, answers)
	ask := domain.Unanswered(set, run.Models, answers)
	unread := len(domain.Unread(answers))
	return Plan{
		Ask: len(ask), AskCost: domain.Estimate(answers, ask, reading), Unknown: domain.Unknown(run, set),
		Read: unread, ReadCost: domain.Micros(unread) * reading, Costs: domain.AnswerCosts(answers, run.Models, reading), Reading: reading,
	}, nil
}

// Retry asks a finished run's failed answers again, or the one task named, in the same run: with
// the questions the run asked (domain.QuestionsOf, so a run of an earlier prompt set asks the
// questions its answers kept) and its models, each new answer replacing the failed one. It refuses while a
// run is running (domain.ErrRunActive) or once the month's budget is spent
// (domain.ErrBudgetReached), and stops at the budget like any run. The answers are asked in the
// background.
func (s *Service) Retry(ctx context.Context, runID uuid.UUID, task *domain.TaskKey) (Retry, error) {
	run, answers, err := s.finished(ctx, runID)
	if err != nil {
		return Retry{}, err
	}
	tasks := domain.Unanswered(domain.QuestionsOf(run, answers), run.Models, answers)
	if task != nil {
		tasks = keep(tasks, *task)
	}
	if len(tasks) == 0 {
		return Retry{}, ErrNothingFailed
	}
	only := map[domain.TaskKey]bool{}
	for _, t := range tasks {
		only[t.Key()] = true
	}
	reading, err := s.again(ctx, run, answers, redo{ask: only})
	if err != nil {
		return Retry{}, err
	}
	return Retry{Run: run.Reopened(), Tasks: len(tasks), Estimate: domain.Estimate(answers, tasks, reading)}, nil
}

// Reread reads a finished run's answers whose reading failed again, or the one answer named, with
// the reader that reads now: the stored answer is read again, the provider is not asked again. Each
// new reading replaces the failed one, what the failed one used still counted. It refuses as Retry
// does, and reads in the background.
func (s *Service) Reread(ctx context.Context, runID uuid.UUID, task *domain.TaskKey) (Retry, error) {
	run, answers, err := s.finished(ctx, runID)
	if err != nil {
		return Retry{}, err
	}
	read := map[uuid.UUID]bool{}
	for _, a := range domain.Unread(answers) {
		if task == nil || a.Key() == *task {
			read[a.ID] = true
		}
	}
	if len(read) == 0 {
		return Retry{}, ErrNothingUnread
	}
	reading, err := s.again(ctx, run, answers, redo{ask: map[domain.TaskKey]bool{}, read: read})
	if err != nil {
		return Retry{}, err
	}
	return Retry{Run: run.Reopened(), Tasks: len(read), Estimate: domain.Micros(len(read)) * reading}, nil
}

// finished is a run that is not running, with its answers.
func (s *Service) finished(ctx context.Context, runID uuid.UUID) (domain.Run, []domain.Answer, error) {
	run, answers, err := s.runAndAnswers(ctx, runID)
	if err != nil {
		return domain.Run{}, nil, err
	}
	if run.Status == domain.RunRunning {
		return domain.Run{}, nil, domain.ErrRunActive
	}
	return run, answers, nil
}

func (s *Service) runAndAnswers(ctx context.Context, runID uuid.UUID) (domain.Run, []domain.Answer, error) {
	run, err := s.repo.Run(ctx, runID)
	if err != nil {
		return domain.Run{}, nil, err
	}
	answers, err := s.repo.Answers(ctx, runID, false)
	if err != nil {
		return domain.Run{}, nil, fmt.Errorf("the run's answers: %w", err)
	}
	return run, answers, nil
}

// again runs a finished run once more, doing only what r names, once the key and the budget allow
// it, and returns what reading one answer costs now.
func (s *Service) again(ctx context.Context, run domain.Run, answers []domain.Answer, r redo) (domain.Micros, error) {
	settings, err := s.affordable(ctx)
	if err != nil {
		return 0, err
	}
	reading, err := s.readingCost(ctx, answers, settings.CredentialID)
	if err != nil {
		return 0, err
	}
	if err := s.repo.ReopenRun(ctx, run.ID, s.cfg.Owner, s.now()); err != nil {
		if errors.Is(err, domain.ErrRunActive) {
			return 0, err
		}
		return 0, fmt.Errorf("reopen the run: %w", err)
	}
	run = run.Reopened()
	s.spawn(func() {
		if err := s.execute(context.WithoutCancel(context.Background()), run, &r); err != nil {
			s.onFail(run.ID, err)
		}
	})
	return reading, nil
}

// affordable checks that a key is chosen and the month's budget is not spent, and returns the settings.
func (s *Service) affordable(ctx context.Context) (domain.Settings, error) {
	settings, err := s.settings(ctx)
	if err != nil {
		return domain.Settings{}, err
	}
	if err := s.gateways.Check(ctx, settings.CredentialID); err != nil {
		return domain.Settings{}, err
	}
	spent, err := s.repo.SpentSince(ctx, domain.MonthStart(s.now()))
	if err != nil {
		return domain.Settings{}, fmt.Errorf("this month's spend: %w", err)
	}
	if spent >= settings.MonthlyBudget {
		return domain.Settings{}, domain.ErrBudgetReached
	}
	return settings, nil
}

func keep(tasks []domain.Task, key domain.TaskKey) []domain.Task {
	for _, t := range tasks {
		if t.Key() == key {
			return []domain.Task{t}
		}
	}
	return nil
}
