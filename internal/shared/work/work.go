// Package work names the pieces of work the app does in the background, how each can end, and the
// journal that records the ones no table of their own keeps: the AI readiness check, the term
// review, the review of the competitor candidates, and the check of the competitor list. Activity
// lists them with the Prompt runs and scans, which their own tables record.
package work

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Kind is a kind of work.
type Kind string

const (
	KindPromptRun         Kind = "promptRun"
	KindScan              Kind = "scan"
	KindRescan            Kind = "rescan"
	KindReadinessCheck    Kind = "readinessCheck"
	KindTermReview        Kind = "termReview"
	KindCandidateReview   Kind = "candidateReview"
	KindCompetitorCheck   Kind = "competitorCheck"
	KindRecommendationRun Kind = "recommendationRun"
)

// Kinds is every kind of work.
func Kinds() []Kind {
	return []Kind{
		KindPromptRun, KindScan, KindRescan, KindReadinessCheck, KindTermReview, KindCandidateReview, KindCompetitorCheck,
		KindRecommendationRun,
	}
}

// Ending is how a piece of work ended.
type Ending string

const (
	EndingDone Ending = "done"
	// EndingDoneWithFailures is work that finished with some of its parts failed, such as a Prompt
	// run with failed responses.
	EndingDoneWithFailures Ending = "doneWithFailures"
	EndingStoppedByHand    Ending = "stoppedByHand"
	EndingStoppedByBudget  Ending = "stoppedByBudget"
	EndingFailed           Ending = "failed"
)

// Endings is every ending.
func Endings() []Ending {
	return []Ending{EndingDone, EndingDoneWithFailures, EndingStoppedByHand, EndingStoppedByBudget, EndingFailed}
}

// Counts is what a piece of work did: Done of Total, of which Failed failed.
type Counts struct {
	Done, Total, Failed int
}

// Outcome is how a piece of work ended, what it did, and the language-model tokens it used on the
// Anthropic key.
type Outcome struct {
	Ending                    Ending
	Counts                    Counts
	InputTokens, OutputTokens int64
}

// Journal records a piece of work: begun, beating while it runs, and later how it ended.
type Journal interface {
	Begin(ctx context.Context, kind Kind, by string, at time.Time) (uuid.UUID, error)
	// Beat records that the work still runs, at.
	Beat(ctx context.Context, id uuid.UUID, at time.Time) error
	End(ctx context.Context, id uuid.UUID, o Outcome, at time.Time) error
}

const (
	// Heartbeat is how often running work tells its journal it still runs.
	Heartbeat = 30 * time.Second
	// Stale is how long running work may go unheard of before it counts as left behind by a server
	// that stopped: four heartbeats, so a slow beat or two does not end live work.
	Stale = 4 * Heartbeat
)

// beatEvery is Heartbeat; tests shorten it.
var beatEvery = Heartbeat //nolint:gochecknoglobals // a test seam for the heartbeat's interval

// Beating calls beat every interval, under ctx, until the stop it returns is called; stop waits for
// a beat under way, so none lands after it.
func Beating(ctx context.Context, every time.Duration, beat func(context.Context)) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Go(func() {
		tick := time.NewTicker(every)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				beat(ctx)
			}
		}
	})
	return func() {
		cancel()
		wg.Wait()
	}
}

// Entry is a piece of work begun in a journal, or nothing to record.
type Entry struct {
	journal Journal
	id      uuid.UUID
	stop    func()
}

// Start begins a piece of work in j, by whom, at; without a journal it records nothing. A journal
// that fails is reported and the work goes on unrecorded.
func Start(ctx context.Context, j Journal, kind Kind, by string, at time.Time, report func(error)) Entry {
	if j == nil {
		return Entry{}
	}
	id, err := j.Begin(ctx, kind, by, at)
	if err != nil {
		report(fmt.Errorf("record the %s begun: %w", kind, err))
		return Entry{}
	}
	stop := Beating(context.WithoutCancel(ctx), beatEvery, func(ctx context.Context) {
		if err := j.Beat(ctx, id, time.Now().UTC()); err != nil {
			report(fmt.Errorf("record that the %s still runs: %w", kind, err))
		}
	})
	return Entry{journal: j, id: id, stop: stop}
}

// Finish stops the work's heartbeat and records how it ended; a journal that fails is reported.
func (e Entry) Finish(ctx context.Context, o Outcome, at time.Time, report func(error)) {
	if e.journal == nil {
		return
	}
	e.stop()
	if err := e.journal.End(ctx, e.id, o, at); err != nil {
		report(fmt.Errorf("record how the work ended: %w", err))
	}
}
