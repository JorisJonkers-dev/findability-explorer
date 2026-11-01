package persistence

import (
	"context"
	"fmt"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// PromptSets returns every published Prompt set with its questions in order, newest first.
func (r *Repository) PromptSets(ctx context.Context) ([]domain.PublishedSet, error) {
	q := r.store.Queries()
	rows, err := q.ListPromptSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("select prompt sets: %w", err)
	}
	questions, err := q.ListPromptSetQuestions(ctx)
	if err != nil {
		return nil, fmt.Errorf("select prompt set questions: %w", err)
	}
	byVersion := map[int64][]domain.Question{}
	for _, row := range questions {
		byVersion[row.Version] = append(byVersion[row.Version], domain.Question{
			ID: row.PromptID, Family: domain.Family(row.Family), Intent: domain.Intent(row.Intent),
			Scope: domain.ScopeInfo{ID: domain.Scope(row.ScopeID), Language: row.Language, Region: row.Region, Name: row.ScopeName},
			Text:  row.Text, English: row.English,
		})
	}
	out := make([]domain.PublishedSet, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.PublishedSet{
			Version: int(row.Version), PublishedAt: row.PublishedAt.UTC(), PublishedBy: row.PublishedBy, Note: row.Note,
			Questions: byVersion[row.Version],
		})
	}
	return out, nil
}

// PromptScopes returns every scope a question can be asked in, by name.
func (r *Repository) PromptScopes(ctx context.Context) ([]domain.ScopeInfo, error) {
	rows, err := r.store.Queries().ListPromptScopes(ctx)
	if err != nil {
		return nil, fmt.Errorf("select prompt scopes: %w", err)
	}
	out := make([]domain.ScopeInfo, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ScopeInfo{ID: domain.Scope(row.ID), Language: row.Language, Region: row.Region, Name: row.Name})
	}
	return out, nil
}
