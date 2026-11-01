// Package domain is the Activity read model (the spec's section 6, Activity): every piece of work
// the app does in the background, running or ended, with how it ended, what it did and what it
// cost on each key, and this month's spending by kind of work with the key that paid. The pieces
// come from the Prompt runs and scans, which their own tables record, and from the work journal,
// which records the rest (package work).
package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"

	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
)

// RecentLimit is how many ended pieces of work Activity lists.
const RecentLimit = 20

// Work is one piece of work: running while FinishedAt is zero, else ended as Ending says. Counts is
// what it did; OpenRouter and Anthropic what it cost on each key.
type Work struct {
	// ID names the piece of work: a Prompt run's or a scan's id, which its stop takes, a re-scan's
	// first scan's id, or the journal's id.
	ID         string
	Kind       work.Kind
	StartedAt  time.Time
	FinishedAt time.Time
	Ending     work.Ending
	Counts     work.Counts
	OpenRouter llm.Micros
	Anthropic  llm.Micros
}

// Running reports whether the work has not ended.
func (w Work) Running() bool { return w.FinishedAt.IsZero() }

// Activity is the work running now, newest first, and the last RecentLimit pieces ended, the last
// ended first.
type Activity struct {
	Running []Work
	Recent  []Work
}

// Arrange sorts all into what runs and what ended recently.
func Arrange(all []Work) Activity {
	a := Activity{Running: []Work{}, Recent: []Work{}}
	for _, w := range all {
		if w.Running() {
			a.Running = append(a.Running, w)
		} else {
			a.Recent = append(a.Recent, w)
		}
	}
	slices.SortStableFunc(a.Running, func(x, y Work) int { return y.StartedAt.Compare(x.StartedAt) })
	slices.SortStableFunc(a.Recent, func(x, y Work) int { return y.FinishedAt.Compare(x.FinishedAt) })
	a.Recent = a.Recent[:min(len(a.Recent), RecentLimit)]
	return a
}

// anthropic is what tokens cost on the Anthropic key: every Claude step runs on Claude Haiku until
// the steps choose models.
func anthropic(in, out int64) llm.Micros {
	return llm.TokenUse{Model: llm.DefaultModel, Input: in, Output: out}.Cost()
}

// PromptRunRow is a Prompt run as its tables record it: its status, whether an administrator
// stopped it, how many responses it asked for, got and failed, and what it cost: the answers and
// the readings OpenRouter did on the OpenRouter key, Claude Haiku's readings' tokens on the
// Anthropic key.
type PromptRunRow struct {
	ID                      uuid.UUID
	StartedAt, FinishedAt   time.Time
	Status                  string
	StoppedByHand           bool
	Tasks, Answered, Failed int
	AnswerCost              llm.Micros
	OpenRouterReadingCost   llm.Micros
	HaikuInput, HaikuOutput int64
}

// Work is the run as a piece of work.
func (r PromptRunRow) Work() Work {
	w := Work{
		ID: r.ID.String(), Kind: work.KindPromptRun, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
		Counts:     work.Counts{Done: r.Answered, Total: r.Tasks, Failed: r.Failed},
		OpenRouter: r.AnswerCost + r.OpenRouterReadingCost, Anthropic: anthropic(r.HaikuInput, r.HaikuOutput),
	}
	switch {
	case r.Status == "running":
	case r.Status == "done" && r.Failed > 0:
		w.Ending = work.EndingDoneWithFailures
	case r.Status == "done":
		w.Ending = work.EndingDone
	case r.Status == "stopped" && r.StoppedByHand:
		w.Ending = work.EndingStoppedByHand
	case r.Status == "stopped":
		w.Ending = work.EndingStoppedByBudget
	default:
		w.Ending = work.EndingFailed
	}
	return w
}

// ScanRow is a scan as its table records it: whether a re-scan queued it, who and when, its
// status, whether an administrator stopped it, the pages it read of the most it reads, and the
// tokens its product matching used on the Anthropic key.
type ScanRow struct {
	ID                        uuid.UUID
	Rescan                    bool
	CreatedBy                 string
	CreatedAt, FinishedAt     time.Time
	Status                    string
	StopRequested             bool
	Pages, MaxPages           int
	InputTokens, OutputTokens int64
}

// rescanGap is the most time between two scans one re-scan queues: it queues them one after the
// other at once, so a longer gap, or another administrator, begins another re-scan.
const rescanGap = time.Minute

// ScanWork is the scans as pieces of work: each scan started alone is one, and the scans one
// re-scan queued are one together.
func ScanWork(rows []ScanRow) []Work {
	rows = slices.Clone(rows)
	slices.SortStableFunc(rows, func(x, y ScanRow) int { return x.CreatedAt.Compare(y.CreatedAt) })
	var out []Work
	var batch []ScanRow
	flush := func() {
		if len(batch) > 0 {
			out = append(out, rescanWork(batch))
			batch = nil
		}
	}
	for _, r := range rows {
		if !r.Rescan {
			out = append(out, scanWork(r))
			continue
		}
		if len(batch) > 0 {
			last := batch[len(batch)-1]
			if last.CreatedBy != r.CreatedBy || r.CreatedAt.Sub(last.CreatedAt) > rescanGap {
				flush()
			}
		}
		batch = append(batch, r)
	}
	flush()
	return out
}

func scanWork(r ScanRow) Work {
	w := Work{
		ID: r.ID.String(), Kind: work.KindScan, StartedAt: r.CreatedAt, FinishedAt: r.FinishedAt,
		Counts: work.Counts{Done: r.Pages, Total: r.MaxPages}, Anthropic: anthropic(r.InputTokens, r.OutputTokens),
	}
	switch {
	case r.Status == "done":
		w.Ending = work.EndingDone
	case r.Status != "failed":
		w.FinishedAt = time.Time{}
	case r.StopRequested:
		w.Ending = work.EndingStoppedByHand
	default:
		w.Ending = work.EndingFailed
	}
	return w
}

// rescanWork is the scans of one re-scan as one piece of work: running while one of them runs or
// waits, stopped by hand when an administrator stopped one, else done, done with failures, or
// failed as its scans ended.
func rescanWork(batch []ScanRow) Work {
	w := Work{ID: batch[0].ID.String(), Kind: work.KindRescan, StartedAt: batch[0].CreatedAt, Counts: work.Counts{Total: len(batch)}}
	running, hand := false, false
	var in, out int64
	for _, r := range batch {
		in, out = in+r.InputTokens, out+r.OutputTokens
		switch {
		case r.Status == "done":
			w.Counts.Done++
		case r.Status != "failed":
			running = true
			continue
		case r.StopRequested:
			w.Counts.Done++
			hand = true
		default:
			w.Counts.Done++
			w.Counts.Failed++
		}
		if r.FinishedAt.After(w.FinishedAt) {
			w.FinishedAt = r.FinishedAt
		}
	}
	w.Anthropic = anthropic(in, out)
	switch {
	case running:
		w.FinishedAt = time.Time{}
	case hand:
		w.Ending = work.EndingStoppedByHand
	case w.Counts.Failed == w.Counts.Total:
		w.Ending = work.EndingFailed
	case w.Counts.Failed > 0:
		w.Ending = work.EndingDoneWithFailures
	default:
		w.Ending = work.EndingDone
	}
	return w
}

// Recorded is a piece of work the journal recorded; running while FinishedAt is zero.
type Recorded struct {
	ID                    uuid.UUID
	Kind                  work.Kind
	StartedAt, FinishedAt time.Time
	Outcome               work.Outcome
}

// Work is the recorded work as a piece of work, its tokens priced on the Anthropic key.
func (r Recorded) Work() Work {
	return Work{
		ID: r.ID.String(), Kind: r.Kind, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt, Ending: r.Outcome.Ending,
		Counts: r.Outcome.Counts, Anthropic: anthropic(r.Outcome.InputTokens, r.Outcome.OutputTokens),
	}
}

// Key is the key that paid.
type Key string

const (
	KeyOpenRouter Key = "openRouter"
	KeyAnthropic  Key = "anthropic"
)

// SpendKind is a kind of work as Activity's spending names it.
type SpendKind string

const (
	SpendPromptRuns         SpendKind = "promptRuns"
	SpendScans              SpendKind = "scans"
	SpendReadingResponses   SpendKind = "readingResponses"
	SpendTermReview         SpendKind = "termReview"
	SpendCandidateChecks    SpendKind = "candidateChecks"
	SpendRecommendationRuns SpendKind = "recommendationRuns"
)

// SpendLine is what one kind of work cost on one key this month.
type SpendLine struct {
	Kind SpendKind
	Key  Key
	Cost llm.Micros
}

// MonthUsage is what each kind of work used this month: the Prompt runs' answers and the readings
// OpenRouter did in dollars, the rest in Claude tokens. The review of the competitor candidates and
// the check of the list share CandidateChecks.
type MonthUsage struct {
	PromptAnswers      llm.Micros
	OpenRouterReadings llm.Micros
	HaikuReadings      llm.TokenUse
	Scans              llm.TokenUse
	TermReview         llm.TokenUse
	CandidateChecks    llm.TokenUse
	RecommendationRuns llm.TokenUse
}

// SpendingOf is this month's spending by kind of work, each with the key that paid: reading
// responses is on both keys, as OpenRouter reads them when no Anthropic key is set up.
func SpendingOf(u MonthUsage) []SpendLine {
	priced := func(t llm.TokenUse) llm.Micros { return anthropic(t.Input, t.Output) }
	return []SpendLine{
		{Kind: SpendPromptRuns, Key: KeyOpenRouter, Cost: u.PromptAnswers},
		{Kind: SpendScans, Key: KeyAnthropic, Cost: priced(u.Scans)},
		{Kind: SpendReadingResponses, Key: KeyOpenRouter, Cost: u.OpenRouterReadings},
		{Kind: SpendReadingResponses, Key: KeyAnthropic, Cost: priced(u.HaikuReadings)},
		{Kind: SpendTermReview, Key: KeyAnthropic, Cost: priced(u.TermReview)},
		{Kind: SpendCandidateChecks, Key: KeyAnthropic, Cost: priced(u.CandidateChecks)},
		{Kind: SpendRecommendationRuns, Key: KeyAnthropic, Cost: priced(u.RecommendationRuns)},
	}
}
