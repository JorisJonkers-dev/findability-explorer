package domain

import (
	"errors"
	"fmt"
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
	return []Target{TargetShareOfVoice, TargetTwenteShareOfVoice, TargetCitationShare, TargetReadiness, TargetTermsUsed}
}

// DefaultTargets are the targets until an administrator sets them: named in 40% of answers and
// 75% of answers about Twente, cited in 20% of web answers, five of five readiness checks, and 35
// terms from the map used on the site.
func DefaultTargets() map[Target]float64 {
	return map[Target]float64{
		TargetShareOfVoice: 0.40, TargetTwenteShareOfVoice: 0.75, TargetCitationShare: 0.20,
		TargetReadiness: 5, TargetTermsUsed: 35,
	}
}

// ErrInvalid wraps every rule a change breaks.
var ErrInvalid = errors.New("invalid")

// maxTermsTarget is the most terms a target may ask for.
const maxTermsTarget = 10_000

// CheckTarget checks a target's value: a share from 0 to 1, a readiness score from 0 to 5 in
// halves, or a count of terms.
func CheckTarget(t Target, v float64) (float64, error) {
	ok := false
	switch t {
	case TargetShareOfVoice, TargetTwenteShareOfVoice, TargetCitationShare:
		ok = v > 0 && v <= 1
	case TargetReadiness:
		ok = v > 0 && v <= 5 && v*2 == float64(int(v*2))
	case TargetTermsUsed:
		ok = v > 0 && v <= maxTermsTarget && v == float64(int(v))
	default:
		return 0, fmt.Errorf("%w: no target is called %q", ErrInvalid, t)
	}
	if !ok {
		return 0, fmt.Errorf("%w: %v is no target for %s", ErrInvalid, v, t)
	}
	return v, nil
}
