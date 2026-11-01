package work_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
)

var at = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

// journal records what was begun and ended; the err fields make a call fail.
type journal struct {
	begun          []work.Kind
	by             string
	ended          []work.Outcome
	beginErr       error
	endErr         error
	id             uuid.UUID
	endedID        uuid.UUID
	beganAt, endAt time.Time
}

func (j *journal) Begin(_ context.Context, kind work.Kind, by string, t time.Time) (uuid.UUID, error) {
	j.begun, j.by, j.beganAt = append(j.begun, kind), by, t
	j.id = uuid.New()
	return j.id, j.beginErr
}

func (j *journal) End(_ context.Context, id uuid.UUID, o work.Outcome, t time.Time) error {
	j.ended, j.endedID, j.endAt = append(j.ended, o), id, t
	return j.endErr
}

func TestAPieceOfWorkIsBegunAndEndedInTheJournal(t *testing.T) {
	j := &journal{}
	var reported []error
	report := func(err error) { reported = append(reported, err) }
	e := work.Start(t.Context(), j, work.KindTermReview, "language model", at, report)
	o := work.Outcome{Ending: work.EndingDone, Counts: work.Counts{Done: 40, Total: 45}, InputTokens: 100, OutputTokens: 10}
	e.Finish(t.Context(), o, at.Add(time.Minute), report)
	if len(j.begun) != 1 || j.begun[0] != work.KindTermReview || j.by != "language model" || !j.beganAt.Equal(at) {
		t.Errorf("begun %v by %q at %s", j.begun, j.by, j.beganAt)
	}
	if len(j.ended) != 1 || j.ended[0] != o || j.endedID != j.id || !j.endAt.Equal(at.Add(time.Minute)) || len(reported) != 0 {
		t.Errorf("ended %v (%s) at %s; reported %v", j.ended, j.endedID, j.endAt, reported)
	}
}

func TestWithoutAJournalNothingIsRecorded(t *testing.T) {
	e := work.Start(t.Context(), nil, work.KindScan, "joris", at, func(err error) { t.Errorf("reported %v", err) })
	e.Finish(t.Context(), work.Outcome{Ending: work.EndingDone}, at, func(err error) { t.Errorf("reported %v", err) })
}

func TestAJournalThatFailsIsReportedAndTheWorkGoesOn(t *testing.T) {
	boom := errors.New("boom")
	var reported []error
	report := func(err error) { reported = append(reported, err) }
	j := &journal{beginErr: boom}
	e := work.Start(t.Context(), j, work.KindReadinessCheck, "joris", at, report)
	e.Finish(t.Context(), work.Outcome{Ending: work.EndingDone}, at, report)
	if len(reported) != 1 || !errors.Is(reported[0], boom) || len(j.ended) != 0 {
		t.Errorf("a failed begin: reported %v, ended %v", reported, j.ended)
	}
	reported = nil
	j = &journal{endErr: boom}
	work.Start(t.Context(), j, work.KindReadinessCheck, "joris", at, report).Finish(t.Context(), work.Outcome{Ending: work.EndingFailed}, at, report)
	if len(reported) != 1 || !errors.Is(reported[0], boom) {
		t.Errorf("a failed end: reported %v", reported)
	}
}

func TestEveryKindAndEndingIsKnown(t *testing.T) {
	if len(work.Kinds()) != 8 || work.Kinds()[7] != work.KindRecommendationRun {
		t.Errorf("kinds = %v", work.Kinds())
	}
	if len(work.Endings()) != 5 {
		t.Errorf("endings = %v", work.Endings())
	}
}
