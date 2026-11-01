package domain

import (
	"errors"
	"slices"
	"time"
)

// OriginHand marks a term an administrator added to the map by hand.
const OriginHand Origin = "hand"

var (
	// ErrTermNotFound is returned when the map has no term with this text in this language.
	ErrTermNotFound = errors.New("the list has no term with this text in this language")
	// ErrNotRemoved is returned when putting back a term that was not removed.
	ErrNotRemoved = errors.New("this term was not removed")
)

// HandTerm is a term an administrator added by hand, with who added it and when.
type HandTerm struct {
	Term Term
	By   string
	At   time.Time
}

// Removal is a term an administrator took off the map, with who did and when. A removed term is
// no longer counted or looked for until it is put back.
type Removal struct {
	Key TermKey
	By  string
	At  time.Time
}

// NewHandTerm checks a term an administrator adds by hand against every term the map has, removed
// ones included, and returns it: in Dutch, German or English, used by nobody yet, marked as added
// by hand.
func NewHandTerm(a Acceptance, terms []Term) (Term, error) {
	t, err := NewTerm(Acceptance{Text: a.Text, Language: a.Language, Family: a.Family, Tier: a.Tier}, "", terms)
	if err != nil {
		return Term{}, err
	}
	t.Origin = OriginHand
	return t, nil
}

// WithoutRemoved returns the terms not removed.
func WithoutRemoved(terms []Term, removed []Removal) []Term {
	return slices.DeleteFunc(slices.Clone(terms), func(t Term) bool {
		return slices.ContainsFunc(removed, func(r Removal) bool { return r.Key == t.Key() })
	})
}

// FindTerm returns the term with key's text, without regard to case or accents, in its language.
func FindTerm(terms []Term, key TermKey) (Term, bool) {
	i := slices.IndexFunc(terms, func(t Term) bool { return t.Language == key.Language && Fold(t.Text) == Fold(key.Text) })
	if i < 0 {
		return Term{}, false
	}
	return terms[i], true
}
