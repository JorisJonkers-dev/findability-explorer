package domain

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Cell counts the answers of one provider, scope, belt family and mode that were read, and how
// many of them name and cite Tribelt. AI share of voice is Named / Answers, citation share
// Cited / Answers; any grouping sums the cells first.
type Cell struct {
	Provider string
	Scope    Scope
	Family   Family
	Mode     Mode
	Answers  int
	Named    int
	Cited    int
}

// counted reports whether an answer counts in the measures: it was answered and read.
func counted(a Answer) bool { return a.Failure == "" && a.Extraction != nil && !a.Extraction.Failed }

// Cells counts a run's answers per provider, scope, family and mode, in that order.
func Cells(answers []Answer) []Cell {
	var out []Cell
	for _, a := range answers {
		if !counted(a) {
			continue
		}
		i := slices.IndexFunc(out, func(c Cell) bool {
			return c.Provider == a.Provider && c.Scope == a.Prompt.Scope && c.Family == a.Prompt.Family && c.Mode == a.Mode
		})
		if i < 0 {
			out = append(out, Cell{Provider: a.Provider, Scope: a.Prompt.Scope, Family: a.Prompt.Family, Mode: a.Mode})
			i = len(out) - 1
		}
		out[i].Answers++
		if a.Extraction.NamesTribelt {
			out[i].Named++
		}
		if a.Extraction.CitesTribelt {
			out[i].Cited++
		}
	}
	slices.SortStableFunc(out, func(a, b Cell) int {
		return cmp.Or(strings.Compare(a.Provider, b.Provider), cmp.Compare(slices.Index(Scopes(), a.Scope), slices.Index(Scopes(), b.Scope)),
			cmp.Compare(slices.Index(Families(), a.Family), slices.Index(Families(), b.Family)),
			cmp.Compare(slices.Index(Modes(), a.Mode), slices.Index(Modes(), b.Mode)))
	})
	return out
}

// Mention is how often a run's answers name one company on the list: in how many answers, how
// often first, and its average place.
type Mention struct {
	CompetitorID string
	Answers      int
	First        int
	positions    int
}

// AveragePosition is the company's average place in the answers that name it, from 1.
func (m Mention) AveragePosition() float64 {
	if m.Answers == 0 {
		return 0
	}
	return float64(m.positions) / float64(m.Answers)
}

// Mentions counts, for every company on the list a run's answers name, the answers that name it
// (once per answer), most answers first, then most often first, then by id.
func Mentions(answers []Answer) []Mention {
	byID := map[string]*Mention{}
	for _, a := range answers {
		if counted(a) {
			mention(byID, a.Extraction.Companies)
		}
	}
	out := make([]Mention, 0, len(byID))
	for _, m := range byID {
		out = append(out, *m)
	}
	slices.SortFunc(out, func(a, b Mention) int {
		return cmp.Or(cmp.Compare(b.Answers, a.Answers), cmp.Compare(b.First, a.First), strings.Compare(a.CompetitorID, b.CompetitorID))
	})
	return out
}

// mention counts one answer's companies on the list, each once, at its first place.
func mention(byID map[string]*Mention, companies []Company) {
	seen := map[string]bool{}
	for _, c := range companies {
		if c.CompetitorID == "" || seen[c.CompetitorID] {
			continue
		}
		seen[c.CompetitorID] = true
		m := byID[c.CompetitorID]
		if m == nil {
			m = &Mention{CompetitorID: c.CompetitorID}
			byID[c.CompetitorID] = m
		}
		m.Answers++
		m.positions += c.Position
		if c.Position == 1 {
			m.First++
		}
	}
}

// WrongClaim is a claim about Tribelt an answer made that the fact sheet contradicts.
type WrongClaim struct {
	AnswerID    uuid.UUID
	RunID       uuid.UUID
	At          time.Time
	Provider    string
	Model       string
	PromptID    string
	Mode        Mode
	Correctness Correctness
	Claim       string
}

// WrongClaims lists every claim the extractor quoted as wrong or partly wrong, newest answer first.
func WrongClaims(answers []Answer) []WrongClaim {
	var out []WrongClaim
	for _, a := range answers {
		if !counted(a) {
			continue
		}
		for _, claim := range a.Extraction.WrongClaims {
			out = append(out, WrongClaim{
				AnswerID: a.ID, RunID: a.RunID, At: a.At, Provider: a.Provider, Model: a.Model, PromptID: a.Prompt.ID,
				Mode: a.Mode, Correctness: a.Extraction.Correctness, Claim: claim,
			})
		}
	}
	slices.SortStableFunc(out, func(a, b WrongClaim) int { return b.At.Compare(a.At) })
	return out
}

// Progressed sums a run's answers into its progress, per provider in the order of models.
func Progressed(answers []Answer, models []ModelChoice) Progress {
	var p Progress
	for _, m := range models {
		p.ByProvider = append(p.ByProvider, ProviderProgress{Provider: m.Provider, Models: []string{}})
	}
	for _, a := range answers {
		cost := a.Cost
		if a.Extraction != nil {
			cost += a.Extraction.Cost
			p.InputTokens += a.Extraction.InputTokens
			p.OutputTokens += a.Extraction.OutputTokens
		}
		p.InputTokens += a.InputTokens
		p.OutputTokens += a.OutputTokens
		p.Cost += cost
		i := slices.IndexFunc(p.ByProvider, func(pp ProviderProgress) bool { return pp.Provider == a.Provider })
		if i < 0 {
			p.ByProvider = append(p.ByProvider, ProviderProgress{Provider: a.Provider, Models: []string{}})
			i = len(p.ByProvider) - 1
		}
		pp := &p.ByProvider[i]
		pp.Cost += cost
		if a.Failure != "" {
			p.Failed++
			pp.Failed++
			continue
		}
		p.Answered++
		pp.Answered++
		if !slices.Contains(pp.Models, a.Model) {
			pp.Models = append(pp.Models, a.Model)
		}
	}
	return p
}
