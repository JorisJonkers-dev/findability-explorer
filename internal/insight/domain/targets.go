package domain

import (
	"errors"
	"fmt"
	"slices"
)

// Target is a number to watch in the plan, with the value Tribelt aims for this year.
type Target string

const (
	TargetShareOfVoice       Target = "aiShareOfVoice"
	TargetTwenteShareOfVoice Target = "twenteShareOfVoice"
	TargetCitations          Target = "citations"
	TargetReadiness          Target = "readinessChecks"
	TargetTermsUsed          Target = "termsUsed"
)

// Targets lists them in the order the plan shows them.
func Targets() []Target {
	return []Target{TargetShareOfVoice, TargetTwenteShareOfVoice, TargetReadiness, TargetTermsUsed, TargetCitations}
}

// DefaultTargets are the targets until an administrator sets them: named in 40% of answers and
// 75% of answers about Twente, five of five readiness checks, 90 terms from the map found on
// tribelt.nl, and tribelt.nl cited in 50 web answers of a run. Each is above where Tribelt stood
// in October 2026 (5%, 16%, 3, 45 and 28): a target already reached is no goal. 90 terms is twice
// the 45 of tribelt.nl's scan then, about three more terms per belt family in each of the three
// languages; 50 web answers is half the 100 web answers of a run.
func DefaultTargets() map[Target]float64 {
	return map[Target]float64{
		TargetShareOfVoice: 0.40, TargetTwenteShareOfVoice: 0.75, TargetReadiness: 5, TargetTermsUsed: 90,
		TargetCitations: 50,
	}
}

// ErrInvalid wraps every rule a change breaks.
var ErrInvalid = errors.New("invalid")

// maxCountTarget is the most terms or citations a target may ask for.
const maxCountTarget = 10_000

// CheckTarget checks a target's value: a share from 0 to 1, a readiness score from 0 to 5 in
// halves, or a count of terms or citations.
func CheckTarget(t Target, v float64) (float64, error) {
	if !slices.Contains(Targets(), t) {
		return 0, fmt.Errorf("%w: no target is called %q", ErrInvalid, t)
	}
	ok := false
	switch t {
	case TargetShareOfVoice, TargetTwenteShareOfVoice:
		ok = v > 0 && v <= 1
	case TargetReadiness:
		ok = v > 0 && v <= 5 && v*2 == float64(int(v*2))
	case TargetTermsUsed, TargetCitations:
		ok = v > 0 && v <= maxCountTarget && v == float64(int(v))
	}
	if !ok {
		return 0, fmt.Errorf("%w: %v is no target for %s", ErrInvalid, v, t)
	}
	return v, nil
}
