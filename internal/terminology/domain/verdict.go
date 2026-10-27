package domain

import (
	"slices"
	"time"
)

// A language model reviews the waiting candidates (ADR-0005). It accepts or rejects each one, and
// says whether it is sure. A sure verdict becomes a decision by the model, which an administrator
// can undo like any other; an unsure one leaves the candidate waiting for a person, with the
// model's suggestion. The model is asked about a candidate once.

// ModelReviewer is who a decision by the language model names.
const ModelReviewer = "language model"

// ReviewBatch is how many candidates the model is asked about at a time.
const ReviewBatch = 40

// Verdict is the language model's review of one candidate.
type Verdict struct {
	CandidateID string
	Phrase      string
	Accept      bool
	// Sure is false when the phrase could go either way; a person then decides.
	Sure   bool
	Reason string
	// Term is the term an accepted candidate becomes, without its companies; empty for a rejected one.
	Term  Acceptance
	Model string
	At    time.Time
}

// Usage is what one request to the model consumed, and when.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
	At           time.Time
}

// Unreviewed are the waiting candidates without the model's verdict, in their order.
func Unreviewed(candidates []Candidate) []Candidate {
	var out []Candidate
	for _, c := range candidates {
		if c.Status == StatusPending && c.Verdict == nil {
			out = append(out, c)
		}
	}
	return out
}

// Complete keeps the model's verdicts on the candidates it was asked about, one each, in their
// order, stamped with the model and the time. A candidate it gave no verdict on gets an unsure
// one, so it waits for a person rather than being asked again.
func Complete(asked []Candidate, answered []Verdict, model string, at time.Time) []Verdict {
	out := make([]Verdict, 0, len(asked))
	for _, c := range asked {
		v := Verdict{CandidateID: c.ID}
		if i := slices.IndexFunc(answered, func(a Verdict) bool { return a.CandidateID == c.ID }); i >= 0 {
			v = answered[i]
		}
		v.Phrase, v.Model, v.At = c.Phrase, model, at
		out = append(out, v)
	}
	return out
}

// Decision is the decision a sure verdict makes on candidate c, by the model. An unsure verdict,
// or an accepted term the map cannot take, makes none: the candidate waits for a person.
func (v Verdict) Decision(c Candidate, terms []Term) (Decision, bool) {
	if !v.Sure {
		return Decision{}, false
	}
	d := Decision{CandidateID: c.ID, Phrase: c.Phrase, Status: StatusRejected, By: ModelReviewer, At: v.At}
	if !v.Accept {
		return d, true
	}
	a := v.Term
	a.UsedBy = make([]string, 0, min(len(c.Companies), maxUsedBy))
	for _, co := range c.Companies[:min(len(c.Companies), maxUsedBy)] {
		a.UsedBy = append(a.UsedBy, co.Name)
	}
	term, err := NewTerm(a, c.Source, terms)
	if err != nil {
		return Decision{}, false
	}
	d.Status, d.Term = StatusAccepted, &term
	return d, true
}

// WithVerdicts attaches the model's verdict to each candidate it reviewed, and to a waiting one
// the term it suggests when the map, holding terms, could take it.
func WithVerdicts(candidates []Candidate, verdicts []Verdict, terms []Term) []Candidate {
	byID := map[string]Verdict{}
	for _, v := range verdicts {
		byID[v.CandidateID] = v
	}
	for i, c := range candidates {
		v, ok := byID[c.ID]
		if !ok {
			continue
		}
		candidates[i].Verdict = &v
		if _, err := NewTerm(v.Term, "", terms); c.Status == StatusPending && v.Accept && err == nil {
			s := v.Term
			candidates[i].Suggestion = &s
		}
	}
	return candidates
}

// Examples are up to perTier terms of each tier, in the map's order, for the model to calibrate on.
func Examples(terms []Term, perTier int) []Term {
	count := map[Tier]int{}
	var out []Term
	for _, t := range terms {
		if count[t.Tier] < perTier {
			count[t.Tier]++
			out = append(out, t)
		}
	}
	return out
}
