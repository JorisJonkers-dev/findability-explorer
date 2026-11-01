package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

// The competitor definition (the spec's decision 19): who Tribelt sells to and who competes with
// it, the text every check of a company is made against. It is stored as versions an administrator
// edits in Manage; the newest is the one in use, and a check records the version it was made
// against. A check of an older version counts as no check, so the next check of the candidates or
// of the list makes it again; saving a version starts no check. Restoring a version saves its text
// as a new one. Versions 1 and 2 were in code: version 1 was the check by name and website alone
// (ADR-0006), version 2 the target market of ADR-0009, imported as the first stored version.

// MaxDefinition is the longest a definition may be, in characters.
const MaxDefinition = 8000

// Definition is one version of the competitor definition.
type Definition struct {
	Version int
	Text    string
	SavedAt time.Time
	SavedBy string
	// RestoredFrom is the version whose text this one restored; 0 for a new text.
	RestoredFrom int
}

// CheckDefinition is text trimmed, or ErrInvalid when it is empty or longer than MaxDefinition.
func CheckDefinition(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > MaxDefinition {
		return "", fmt.Errorf("%w: a definition has 1 to %d characters", ErrInvalid, MaxDefinition)
	}
	return text, nil
}

// NewestDefinition is the newest of versions, the one in use; false when there is none.
func NewestDefinition(versions []Definition) (Definition, bool) {
	var newest Definition
	for _, d := range versions {
		if d.Version > newest.Version {
			newest = d
		}
	}
	return newest, newest.Version > 0
}

// DefinitionVersion is the version numbered version; false when there is none.
func DefinitionVersion(versions []Definition, version int) (Definition, bool) {
	for _, d := range versions {
		if d.Version == version {
			return d, true
		}
	}
	return Definition{}, false
}

// Outdated counts what was checked against a version older than version: the competitors on the
// list the model checks, and the waiting candidates whose newest check with a reason is older.
// Both are checked again by the next check.
func Outdated(directory []DirectoryCompany, checks []ListCheck, candidates []Candidate, verdicts []CompanyVerdict, version int) (int, int) {
	return outdatedCompetitors(directory, checks, version), outdatedCandidates(candidates, verdicts, version)
}

func outdatedCompetitors(directory []DirectoryCompany, checks []ListCheck, version int) int {
	checked := map[string]int{}
	for _, c := range checks {
		checked[c.CompanyID] = c.Version
	}
	n := 0
	for _, c := range directory {
		if v, ok := checked[c.ID]; ok && c.Checkable() && v < version {
			n++
		}
	}
	return n
}

func outdatedCandidates(candidates []Candidate, verdicts []CompanyVerdict, version int) int {
	newest := map[string]int{}
	for _, v := range verdicts {
		if v.Reason != "" && v.Version > newest[v.CandidateID] {
			newest[v.CandidateID] = v.Version
		}
	}
	n := 0
	for _, c := range candidates {
		best := 0
		for _, id := range c.MemberIDs() {
			best = max(best, newest[id])
		}
		if c.Decision == nil && best > 0 && best < version {
			n++
		}
	}
	return n
}

// Typical tokens of checking one company, for an estimate before any check.
const (
	TypicalCheckIn  = 2500
	TypicalCheckOut = 120
)

// EstimateChecks is what checking n companies should cost on the Anthropic key, from what the
// checks so far used (used, over checked companies), or a typical check before any.
func EstimateChecks(n int, used ReviewUsage, checked int) llm.Micros {
	if n <= 0 {
		return 0
	}
	in, out := int64(TypicalCheckIn), int64(TypicalCheckOut)
	if checked > 0 {
		in, out = used.InputTokens/int64(checked), used.OutputTokens/int64(checked)
	}
	return llm.TokenUse{Model: llm.DefaultModel, Input: in * int64(n), Output: out * int64(n)}.Cost()
}
