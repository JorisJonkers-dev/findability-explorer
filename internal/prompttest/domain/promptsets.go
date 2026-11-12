package domain

import (
	"cmp"
	"errors"
	"slices"
	"time"
)

// Intent is the buyer intent a question represents: the kind of search it is, as the search term
// list names them.
type Intent string

const (
	IntentSpec        Intent = "spec"
	IntentProduct     Intent = "product"
	IntentApplication Intent = "application"
	IntentProblem     Intent = "problem"
	IntentBroad       Intent = "broad"
)

// KindOf is the kind an answer to a question of intent records: product as supplier, the others
// by their own name.
func KindOf(intent Intent) Kind {
	if intent == IntentProduct {
		return KindSupplier
	}
	return Kind(intent)
}

// ScopeInfo is a scope as stored: its id, the language it asks in, the region it asks about (""
// for none), and its name.
type ScopeInfo struct {
	ID       Scope
	Language string
	Region   string
	Name     string
}

// Question is one question of a published Prompt set: its belt family, buyer intent, scope, and
// text, with its English translation when it is in another language and one is known.
type Question struct {
	ID      string
	Family  Family
	Intent  Intent
	Scope   ScopeInfo
	Text    string
	English string
}

// PublishedSet is one published version of the Prompt set: when and by whom it was published, what
// it asks or changed, and its questions in order.
type PublishedSet struct {
	Version     int
	PublishedAt time.Time
	PublishedBy string
	Note        string
	Questions   []Question
}

// PromptSet is the set's questions as a run asks them, each in its scope's language.
func (p PublishedSet) PromptSet() PromptSet {
	set := PromptSet{Version: p.Version, Prompts: make([]Prompt, 0, len(p.Questions))}
	for _, q := range p.Questions {
		set.Prompts = append(set.Prompts, Prompt{
			ID: q.ID, Family: q.Family, Kind: KindOf(q.Intent), Scope: q.Scope.ID, Language: q.Scope.Language, Text: q.Text,
		})
	}
	return set
}

// ErrNoPromptSet is returned when no Prompt set is published, so a run has nothing to ask.
var ErrNoPromptSet = errors.New("no Prompt set is published")

// Newest is the newest of the published sets as a run asks it; false when none is published.
func Newest(sets []PublishedSet) (PromptSet, bool) {
	if len(sets) == 0 {
		return PromptSet{}, false
	}
	newest := sets[0]
	for _, s := range sets[1:] {
		if s.Version > newest.Version {
			newest = s
		}
	}
	return newest.PromptSet(), true
}

// VersionsOf lists the published sets, oldest first.
func VersionsOf(sets []PublishedSet) []Version {
	out := make([]Version, 0, len(sets))
	for _, s := range sets {
		out = append(out, Version{Number: s.Version, PublishedAt: s.PublishedAt, Questions: len(s.Questions), Note: s.Note})
	}
	slices.SortFunc(out, func(a, b Version) int { return cmp.Compare(a.Number, b.Number) })
	return out
}
