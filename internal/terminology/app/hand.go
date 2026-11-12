package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// HandTerms stores the terms administrators added by hand and the terms they removed.
type HandTerms interface {
	// Added returns every term added by hand, oldest first.
	Added(ctx context.Context) ([]domain.HandTerm, error)
	// Add stores a term added by hand, or returns domain.ErrTermExists when it was added meanwhile.
	Add(ctx context.Context, t domain.HandTerm) error
	// Removed returns every removal, oldest first.
	Removed(ctx context.Context) ([]domain.Removal, error)
	// Remove stores a removal.
	Remove(ctx context.Context, r domain.Removal) error
	// PutBack deletes the removal of the term with key; it reports whether there was one.
	PutBack(ctx context.Context, key domain.TermKey) (bool, error)
}

// ErrNoHandTerms is returned when no store for hand-added and removed terms was given.
var ErrNoHandTerms = errors.New("terms cannot be added or removed by hand here")

// WithHandTerms gives the service a store for the terms added and removed by hand. Call Load after.
func (s *Service) WithHandTerms(h HandTerms) *Service {
	s.hand = h
	return s
}

// loadHand reads the terms added by hand and the removals; none without a store.
func (s *Service) loadHand(ctx context.Context) ([]domain.HandTerm, []domain.Removal, error) {
	if s.hand == nil {
		return nil, nil, nil
	}
	added, err := s.hand.Added(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("load hand-added terms: %w", err)
	}
	removed, err := s.hand.Removed(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("load removed terms: %w", err)
	}
	return added, removed, nil
}

// HandTerms returns the terms added by hand, oldest first, removed ones included.
func (s *Service) HandTerms() []domain.HandTerm {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.HandTerm(nil), s.added...)
}

// RemovedTerms returns the removed terms, oldest removal first, each as the map has it.
func (s *Service) RemovedTerms() []domain.HandTerm {
	all := s.allTerms()
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.HandTerm, 0, len(s.removed))
	for _, r := range s.removed {
		t, ok := domain.FindTerm(all, r.Key)
		if !ok {
			t = domain.Term{Text: r.Key.Text, Language: r.Key.Language}
		}
		out = append(out, domain.HandTerm{Term: t, By: r.By, At: r.At})
	}
	return out
}

// AddTerm adds a term to the list by hand, recording user as who added it. A term the list has,
// or had and was removed, is refused with domain.ErrTermExists: a removed one is put back instead.
func (s *Service) AddTerm(ctx context.Context, user string, a domain.Acceptance) (domain.Term, error) {
	if s.hand == nil {
		return domain.Term{}, ErrNoHandTerms
	}
	t, err := domain.NewHandTerm(a, s.allTerms())
	if err != nil {
		return domain.Term{}, err
	}
	if err := s.hand.Add(ctx, domain.HandTerm{Term: t, By: user, At: s.now()}); err != nil {
		return domain.Term{}, fmt.Errorf("add term: %w", err)
	}
	return t, s.Load(ctx)
}

// RemoveTerm takes the term with key's text, in its language, off the list, recording user; it is
// no longer counted or looked for until put back. domain.ErrTermNotFound when the list has none.
func (s *Service) RemoveTerm(ctx context.Context, user string, key domain.TermKey) error {
	if s.hand == nil {
		return ErrNoHandTerms
	}
	t, ok := domain.FindTerm(s.Terms(), key)
	if !ok {
		return domain.ErrTermNotFound
	}
	if err := s.hand.Remove(ctx, domain.Removal{Key: t.Key(), By: user, At: s.now()}); err != nil {
		return fmt.Errorf("remove term: %w", err)
	}
	return s.Load(ctx)
}

// PutBack puts a removed term back on the list; domain.ErrNotRemoved when it was not removed.
func (s *Service) PutBack(ctx context.Context, key domain.TermKey) error {
	if s.hand == nil {
		return ErrNoHandTerms
	}
	found, err := s.hand.PutBack(ctx, key)
	if err != nil {
		return fmt.Errorf("put back term: %w", err)
	}
	if !found {
		return domain.ErrNotRemoved
	}
	return s.Load(ctx)
}
