// Package domain holds the Claude steps: each piece of work a Claude model does here, with the
// instructions it came with, its saved versions of model and instructions, and what one job of it
// costs with a model. The data a step sends and the reply format it reads stay in its call site's
// code; only the system message and the model are an administrator's to change.
package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

// ID names a Claude step.
type ID string

// The steps.
const (
	// Reading is reading a Prompt run's responses; its instructions are set in the code.
	Reading ID = "reading"
	// Matching is matching the products on a scan's pages to Tribelt's belt families.
	Matching ID = "matching"
	// TidyingNames is grouping the names in the responses that are one company.
	TidyingNames ID = "candidateMerge"
	// CheckingCompanies is deciding, against the competitor definition, whether a company competes.
	CheckingCompanies ID = "candidateCheck"
	// ReviewingTerms is deciding which search terms the scans found join the list.
	ReviewingTerms ID = "termReview"
)

// ByTheTool is who made version 1 of a step: the tool itself.
const ByTheTool = "the tool"

// MaxInstructions is the longest a step's instructions may be, in characters.
const MaxInstructions = 20_000

// ErrInvalid is returned for a version that cannot be saved.
var ErrInvalid = errors.New("invalid version")

// ErrNotFound is returned for a step that does not exist.
var ErrNotFound = errors.New("no such Claude step")

// Tokens is a mix of tokens read and written.
type Tokens struct {
	Input  int64
	Output int64
}

// Definition is a Claude step as the tool comes with it.
type Definition struct {
	ID    ID
	Title string
	// Fixed is true when the instructions are set in the code, so jobs stay comparable: only the
	// model is chosen then.
	Fixed bool
	// Original is the system message the tool came with.
	Original string
	// Typical is the mix of tokens one job reads and writes, which scales its cost to a model.
	Typical Tokens
}

// definitions are the steps, in the order the overview lists them. A step's typical tokens are
// those of the job its estimate names: one scan, reading 100 responses, tidying the waiting names,
// checking 100 companies, reviewing 100 search terms.
var definitions = []Definition{ //nolint:gochecknoglobals // a constant table
	{ID: Reading, Title: "Reading responses", Fixed: true, Original: ReadingInstructions, Typical: Tokens{Input: 150_000, Output: 25_000}},
	{ID: Matching, Title: "Matching products", Original: matchingInstructions, Typical: Tokens{Input: 15_000, Output: 1_500}},
	{ID: TidyingNames, Title: "Tidying candidate names", Original: tidyingInstructions, Typical: Tokens{Input: 6_000, Output: 1_500}},
	{ID: CheckingCompanies, Title: "Checking companies", Original: checkingInstructions, Typical: Tokens{Input: 250_000, Output: 12_000}},
	{ID: ReviewingTerms, Title: "Reviewing search terms", Original: reviewingInstructions, Typical: Tokens{Input: 25_000, Output: 12_000}},
}

// Definitions returns every step.
func Definitions() []Definition { return slices.Clone(definitions) }

// Lookup returns the step with this id; false when there is none.
func Lookup(id ID) (Definition, bool) {
	i := slices.IndexFunc(definitions, func(d Definition) bool { return d.ID == id })
	if i < 0 {
		return Definition{}, false
	}
	return definitions[i], true
}

// Version is one saved model and instructions of a step.
type Version struct {
	Step         ID
	Number       int
	Model        string
	Instructions string
	Note         string
	By           string
	At           time.Time
}

// Original is version 1 of d: the instructions it came with, on Claude Haiku 5.5.
func Original(d Definition) Version {
	return Version{Step: d.ID, Number: 1, Model: llm.DefaultModel, Instructions: d.Original, Note: "The instructions the tool came with.", By: ByTheTool}
}

// CurrentOf is the version d uses now: the newest of versions (oldest first), or its original
// without any. A step whose instructions are set in the code always uses the code's.
func CurrentOf(d Definition, versions []Version) Version {
	v := Original(d)
	if len(versions) > 0 {
		v = versions[len(versions)-1]
	}
	if d.Fixed {
		v.Instructions = d.Original
	}
	return v
}

// Change is what an administrator saves: a model and instructions, and the version they restore.
type Change struct {
	Model        string
	Instructions string
	// RestoredFrom is the version restored, or 0.
	RestoredFrom int
}

// Next checks ch against the current version and returns the version it makes.
func Next(d Definition, current Version, ch Change, by string, at time.Time) (Version, error) {
	if _, ok := llm.ClaudeModel(ch.Model); !ok {
		return Version{}, fmt.Errorf("%w: %q is no Claude model listed", ErrInvalid, ch.Model)
	}
	text := strings.TrimSpace(ch.Instructions)
	switch {
	case text == "":
		return Version{}, fmt.Errorf("%w: the instructions are empty", ErrInvalid)
	case len([]rune(text)) > MaxInstructions:
		return Version{}, fmt.Errorf("%w: the instructions are longer than %d characters", ErrInvalid, MaxInstructions)
	case d.Fixed && text != strings.TrimSpace(d.Original):
		return Version{}, fmt.Errorf("%w: these instructions are set in the code; only the model can be chosen", ErrInvalid)
	case text == strings.TrimSpace(current.Instructions) && ch.Model == current.Model:
		return Version{}, fmt.Errorf("%w: nothing changed, so there is no new version", ErrInvalid)
	}
	note := ""
	if ch.RestoredFrom > 0 {
		note = fmt.Sprintf("Restored from version %d.", ch.RestoredFrom)
	}
	return Version{Step: d.ID, Number: current.Number + 1, Model: ch.Model, Instructions: text, Note: note, By: by, At: at}, nil
}

// Scale is how much one job of d costs with model against Claude Haiku 5.5, for d's typical mix
// of tokens read and written.
func Scale(d Definition, model string) float64 {
	haiku := llm.ListPrice(llm.DefaultModel).Of(d.Typical.Input, d.Typical.Output)
	if haiku == 0 {
		return 1
	}
	return float64(llm.ListPrice(model).Of(d.Typical.Input, d.Typical.Output)) / float64(haiku)
}

// JobCost is what one job of d costs with model: base, the tool's own estimate with Claude Haiku
// 5.5, scaled by Scale; without a base, d's typical tokens priced at the model.
func JobCost(d Definition, base llm.Micros, model string) llm.Micros {
	if base <= 0 {
		return llm.ListPrice(model).Of(d.Typical.Input, d.Typical.Output)
	}
	return llm.Micros(float64(base) * Scale(d, model))
}
