package app_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
)

// journal records the work begun and how it ended; beginErr makes Begin fail.
type journal struct {
	mu       sync.Mutex
	begun    []work.Kind
	ended    []work.Outcome
	beginErr error
}

func (j *journal) Begin(_ context.Context, kind work.Kind, _ string, _ time.Time) (uuid.UUID, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.begun = append(j.begun, kind)
	return uuid.New(), j.beginErr
}

func (j *journal) End(_ context.Context, _ uuid.UUID, o work.Outcome, _ time.Time) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.ended = append(j.ended, o)
	return nil
}

func (j *journal) Beat(context.Context, uuid.UUID, time.Time) error { return nil }

func (j *journal) outcomes() []work.Outcome {
	j.mu.Lock()
	defer j.mu.Unlock()
	return append([]work.Outcome(nil), j.ended...)
}

func TestAReadinessCheckIsRecordedWithWhatItChecked(t *testing.T) {
	r := newRepo()
	scansOf(t, r, map[string]domain.Analysis{
		"x":    {URL: "https://x.test/", Status: domain.StatusDone},
		"down": {URL: "https://down.test/", Status: domain.StatusDone},
	})
	j := &journal{}
	svc := app.New(r, &directory{}, siteCrawler{}, source{}, glossary, time.Now, limits())
	svc.WithJournal(j, func(err error) { t.Errorf("reported %v", err) })
	if _, err := svc.StartReadinessCheck(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitForCheck(t, svc)
	want := work.Outcome{Ending: work.EndingDoneWithFailures, Counts: work.Counts{Done: 2, Total: 2, Failed: 1}}
	if got := j.outcomes(); len(j.begun) != 1 || j.begun[0] != work.KindReadinessCheck || len(got) != 1 || got[0] != want {
		t.Fatalf("begun %v, ended %+v", j.begun, got)
	}
	// Every site read: done.
	clean := newRepo()
	scansOf(t, clean, map[string]domain.Analysis{"x": {URL: "https://x.test/", Status: domain.StatusDone}})
	j = &journal{}
	svc = app.New(clean, &directory{}, siteCrawler{}, source{}, glossary, time.Now, limits())
	svc.WithJournal(j, func(err error) { t.Errorf("reported %v", err) })
	if _, err := svc.StartReadinessCheck(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitForCheck(t, svc)
	if got := j.outcomes(); len(got) != 1 || got[0].Ending != work.EndingDone {
		t.Errorf("a clean check = %+v", got)
	}
}

func TestAStoppedReadinessCheckIsRecordedAsStoppedByHand(t *testing.T) {
	r := newRepo()
	scansOf(t, r, map[string]domain.Analysis{"x": {URL: "https://x.test/", Status: domain.StatusDone}})
	release, reading := make(chan struct{}), make(chan struct{}, 1)
	j := &journal{}
	svc := app.New(r, &directory{}, siteCrawler{release: release, reading: reading}, source{}, glossary, time.Now, limits())
	svc.WithJournal(j, func(err error) { t.Errorf("reported %v", err) })
	if _, err := svc.StartReadinessCheck(context.Background()); err != nil {
		t.Fatal(err)
	}
	<-reading
	if err := svc.StopReadinessCheck(t.Context()); err != nil {
		t.Fatal(err)
	}
	close(release)
	waitForCheck(t, svc)
	if got := j.outcomes(); len(got) != 1 || got[0].Ending != work.EndingStoppedByHand || got[0].Counts != (work.Counts{Total: 1}) {
		t.Errorf("stopped = %+v", got)
	}
}

func TestAJournalThatFailsLeavesTheReadinessCheckAlone(t *testing.T) {
	r := newRepo()
	scansOf(t, r, map[string]domain.Analysis{"x": {URL: "https://x.test/", Status: domain.StatusDone}})
	j := &journal{beginErr: errBoom}
	reported := make(chan error, 1)
	svc := app.New(r, &directory{}, siteCrawler{}, source{}, glossary, time.Now, limits())
	svc.WithJournal(j, func(err error) { reported <- err })
	if _, err := svc.StartReadinessCheck(context.Background()); err != nil {
		t.Fatal(err)
	}
	if done := waitForCheck(t, svc); done.Checked != 1 {
		t.Errorf("the check = %+v", done)
	}
	if err := <-reported; !errors.Is(err, errBoom) || len(j.outcomes()) != 0 {
		t.Errorf("reported %v; ended %+v", err, j.outcomes())
	}
}
