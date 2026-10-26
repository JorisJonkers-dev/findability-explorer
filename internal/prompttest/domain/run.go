package domain

import (
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
)

// Micros is an amount of US dollars in millionths, so costs add up exactly.
type Micros int64

// MicrosOf is dollars in millionths, rounded.
func MicrosOf(dollars float64) Micros { return Micros(math.Round(dollars * 1e6)) }

// Dollars is the amount in dollars.
func (m Micros) Dollars() float64 { return float64(m) / 1e6 }

// Mode is how a prompt is asked: from the model's own knowledge, or with live web search.
type Mode string

const (
	ModeKnowledge Mode = "knowledge"
	ModeWeb       Mode = "web"
)

// Modes lists both, in the order a run asks them.
func Modes() []Mode { return []Mode{ModeKnowledge, ModeWeb} }

// Task is one question a run asks: one prompt, of one provider's model, in one mode.
type Task struct {
	Prompt   Prompt
	Provider string
	Mode     Mode
}

// TaskKey identifies a task within a run; a run has one answer per key.
type TaskKey struct {
	PromptID string
	Provider string
	Mode     Mode
}

// Key is the task's key.
func (t Task) Key() TaskKey {
	return TaskKey{PromptID: t.Prompt.ID, Provider: t.Provider, Mode: t.Mode}
}

// Tasks are every prompt of set for every model chosen, in both modes, prompt by prompt.
func Tasks(set PromptSet, models []ModelChoice) []Task {
	out := make([]Task, 0, len(set.Prompts)*len(models)*len(Modes()))
	for _, p := range set.Prompts {
		for _, mode := range Modes() {
			for _, m := range models {
				out = append(out, Task{Prompt: p, Provider: m.Provider, Mode: mode})
			}
		}
	}
	return out
}

// RunStatus is where a Prompt run is in its life.
type RunStatus string

const (
	RunRunning RunStatus = "running"
	RunDone    RunStatus = "done"
	// RunStopped means the month's budget was reached; the answers so far are kept.
	RunStopped RunStatus = "stopped"
	RunFailed  RunStatus = "failed"
)

// Trigger is what started a run.
type Trigger string

const (
	TriggerManual   Trigger = "manual"
	TriggerSchedule Trigger = "schedule"
)

// Run is one Prompt run: every prompt of a prompt-set version, on every chosen model, in both
// modes, judged against one fact-sheet version.
type Run struct {
	ID               uuid.UUID
	PromptSetVersion int
	FactSheetVersion int
	// FactSheet is the fact sheet's text at that version, which the run's answers are judged against.
	FactSheet  string
	Trigger    Trigger
	StartedBy  string
	StartedAt  time.Time
	FinishedAt time.Time // zero while running
	Status     RunStatus
	Failure    string
	Models     []ModelChoice
	// Tasks is how many answers the run asks for.
	Tasks int
}

// Progress is how far a run got, overall and per provider, and what it cost.
type Progress struct {
	Answered     int
	Failed       int
	InputTokens  int64
	OutputTokens int64
	Cost         Micros
	ByProvider   []ProviderProgress
}

// ProviderProgress is one provider's part of a run.
type ProviderProgress struct {
	Provider string
	Answered int
	Failed   int
	Cost     Micros
	// Models are the models that answered, the free one and its fallback when one was needed.
	Models []string
}

// Reply is a model's answer to one prompt and what it consumed.
type Reply struct {
	Text         string
	Citations    []string
	InputTokens  int64
	OutputTokens int64
	Cost         Micros
}

// Answer is one model's reply to one prompt in a run, with how it was read. Failure is set when
// the model gave no answer.
type Answer struct {
	ID               uuid.UUID
	RunID            uuid.UUID
	Prompt           Prompt
	PromptSetVersion int
	Provider         string
	Model            string
	Mode             Mode
	Reply
	Failure    string
	At         time.Time
	Extraction *Extraction
}

// Key is the task the answer answers.
func (a Answer) Key() TaskKey {
	return TaskKey{PromptID: a.Prompt.ID, Provider: a.Provider, Mode: a.Mode}
}

// Errors the ports return.
var (
	// ErrRateLimited is returned when the provider asks to slow down (a 429).
	ErrRateLimited = errors.New("the model is rate limited")
	// ErrRunActive is returned when a run is running already.
	ErrRunActive = errors.New("a Prompt run is running already")
	// ErrNoKey is returned when no OpenRouter key is chosen for Prompt runs, or it cannot be read.
	ErrNoKey = errors.New("choose an OpenRouter key for Prompt runs in Settings first")
	// ErrNoModels is returned when none of the providers has a model on OpenRouter's list.
	ErrNoModels = errors.New("none of the providers has a text model on OpenRouter's price list")
	// ErrNotFound is returned for a run, answer or candidate nobody knows.
	ErrNotFound = errors.New("not found")
	// ErrCompanyKnown is returned when a candidate is added with a website another company has.
	ErrCompanyKnown = errors.New("another company has this website")
)
