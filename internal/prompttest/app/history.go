package app

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// HistoryView is what the charts over time draw: the published Prompt sets, the figures of each
// run with a response read, oldest first, and for each partial run the reason most of its failed
// responses give.
type HistoryView struct {
	Versions  []domain.Version
	Runs      []domain.RunFigures
	FailureOf map[uuid.UUID]string
}

// History gives the published Prompt sets and the Prompt test's figures per run, from the runs
// Results reads.
func (s *Service) History(ctx context.Context) (HistoryView, error) {
	sets, err := s.promptSets(ctx)
	if err != nil {
		return HistoryView{}, err
	}
	scopes, err := s.repo.PromptScopes(ctx)
	if err != nil {
		return HistoryView{}, fmt.Errorf("the scopes: %w", err)
	}
	res, err := s.Results(ctx)
	if err != nil {
		return HistoryView{}, err
	}
	in := make([]domain.RunFigureInput, 0, len(res.Runs))
	failure := map[uuid.UUID]string{}
	// Results are newest first; the charts read left to right. A run still running is no point on
	// them yet: it would show as partial for want of the responses it has still to ask.
	for _, r := range slices.Backward(res.Runs) {
		if r.Run.Status == domain.RunRunning {
			continue
		}
		in = append(in, domain.RunFigureInput{Run: r.Run, Progress: r.Progress, Cells: r.Cells})
		failure[r.Run.ID] = r.CommonFailure
	}
	readers, err := s.repo.ReadingModels(ctx)
	if err != nil {
		return HistoryView{}, fmt.Errorf("read the reading models: %w", err)
	}
	out := HistoryView{Versions: domain.VersionsOf(sets), Runs: domain.History(in, scopes), FailureOf: map[uuid.UUID]string{}}
	for i := range out.Runs {
		out.Runs[i].ReadingModel = readers[out.Runs[i].RunID]
	}
	for _, f := range out.Runs {
		if f.Partial && failure[f.RunID] != "" {
			out.FailureOf[f.RunID] = failure[f.RunID]
		}
	}
	return out, nil
}
