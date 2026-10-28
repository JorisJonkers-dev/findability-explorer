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
	TargetCitationShare      Target = "citationShare"
	TargetReadiness          Target = "readinessChecks"
	TargetTermsUsed          Target = "termsUsed"
)

// Targets lists them in the order the plan shows them.
func Targets() []Target {
	return []Target{TargetShareOfVoice, TargetTwenteShareOfVoice, TargetReadiness, TargetTermsUsed, TargetCitationShare}
}

// DefaultTargets are the targets until an administrator sets them: named in 40% of answers and
// 75% of answers about Twente, every one of the readiness checks, 90 terms from the map found on
// tribelt.nl, and tribelt.nl cited in 50% of the web answers of a run. Each is above where Tribelt
// stood in October 2026 (5%, 16%, 3 of 5 checks, 45 terms and 36%): a target already reached is no
// goal. 90 terms is twice the 45 of tribelt.nl's scan then, about three more terms per belt family
// in each of the three languages. The citations are a share, not a count, since runs read
// different numbers of web answers (ADR-0014).
func DefaultTargets() map[Target]float64 {
	return map[Target]float64{
		TargetShareOfVoice: 0.40, TargetTwenteShareOfVoice: 0.75, TargetReadiness: float64(ReadinessChecks()), TargetTermsUsed: 90,
		TargetCitationShare: 0.50,
	}
}

// ErrInvalid wraps every rule a change breaks.
var ErrInvalid = errors.New("invalid")

// maxCountTarget is the most terms a target may ask for.
const maxCountTarget = 10_000

// MaxOf is the most a target's value can be: 1 for a share (the citations too), ReadinessChecks for the readiness
// checks; false for a count, which has no most.
func MaxOf(t Target) (float64, bool) {
	switch t {
	case TargetShareOfVoice, TargetTwenteShareOfVoice, TargetCitationShare:
		return 1, true
	case TargetReadiness:
		return float64(ReadinessChecks()), true
	case TargetTermsUsed:
	}
	return 0, false
}

// CheckTarget checks a target's value: a share from 0 to 1, a readiness score up to
// ReadinessChecks in halves, or a count of terms.
func CheckTarget(t Target, v float64) (float64, error) {
	if !slices.Contains(Targets(), t) {
		return 0, fmt.Errorf("%w: no target is called %q", ErrInvalid, t)
	}
	ok := false
	switch t {
	case TargetShareOfVoice, TargetTwenteShareOfVoice, TargetCitationShare:
		ok = v > 0 && v <= 1
	case TargetReadiness:
		ok = v > 0 && v <= float64(ReadinessChecks()) && v*2 == float64(int(v*2))
	case TargetTermsUsed:
		ok = v > 0 && v <= maxCountTarget && v == float64(int(v))
	}
	if !ok {
		return 0, fmt.Errorf("%w: %v is no target for %s", ErrInvalid, v, t)
	}
	return v, nil
}
