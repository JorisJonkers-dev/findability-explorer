package app

import (
	"context"
	"fmt"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// PromptSets are the published Prompt sets, newest first, each with its questions, and every scope
// a question can be asked in.
type PromptSets struct {
	Sets   []domain.PublishedSet
	Scopes []domain.ScopeInfo
}

// PromptSets returns the published Prompt sets and the scopes.
func (s *Service) PromptSets(ctx context.Context) (PromptSets, error) {
	sets, err := s.promptSets(ctx)
	if err != nil {
		return PromptSets{}, err
	}
	scopes, err := s.repo.PromptScopes(ctx)
	if err != nil {
		return PromptSets{}, fmt.Errorf("the scopes: %w", err)
	}
	return PromptSets{Sets: sets, Scopes: scopes}, nil
}

func (s *Service) promptSets(ctx context.Context) ([]domain.PublishedSet, error) {
	sets, err := s.repo.PromptSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("the Prompt sets: %w", err)
	}
	return sets, nil
}

// newestSet is the newest published Prompt set, which a new run asks, or domain.ErrNoPromptSet.
func (s *Service) newestSet(ctx context.Context) (domain.PromptSet, error) {
	sets, err := s.promptSets(ctx)
	if err != nil {
		return domain.PromptSet{}, err
	}
	set, ok := domain.Newest(sets)
	if !ok {
		return domain.PromptSet{}, domain.ErrNoPromptSet
	}
	return set, nil
}

// Publish makes the next version of the Prompt set from an administrator's draft and stores it with
// the new scopes it asks in. A draft it cannot ask is a domain.DraftError; a draft that asks what the
// newest version asks is domain.ErrNothingChanged.
func (s *Service) Publish(ctx context.Context, actor string, d domain.Draft) (domain.PublishedSet, error) {
	sets, err := s.promptSets(ctx)
	if err != nil {
		return domain.PublishedSet{}, err
	}
	scopes, err := s.repo.PromptScopes(ctx)
	if err != nil {
		return domain.PublishedSet{}, fmt.Errorf("the scopes: %w", err)
	}
	set, added, err := domain.Publish(sets, scopes, d, actor, s.now())
	if err != nil {
		return domain.PublishedSet{}, err
	}
	if err := s.repo.PublishPromptSet(ctx, set, added); err != nil {
		return domain.PublishedSet{}, fmt.Errorf("store Prompt set version %d: %w", set.Version, err)
	}
	return set, nil
}
