package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// stopping is a model that, while it checks its first batch, has stop called.
type stopping struct {
	*model
	stop func()
}

func (s stopping) Check(ctx context.Context, ds []domain.Dossier) ([]domain.CompanyVerdict, domain.ReviewUsage, error) {
	if s.stop != nil && len(s.checked) == 0 {
		s.stop()
	}
	return s.model.Check(ctx, ds)
}

func (s stopping) Merge(ctx context.Context, cs []domain.Candidate, d []domain.DirectoryCompany) ([]domain.Cluster, domain.ReviewUsage, error) {
	return s.model.Merge(ctx, cs, d)
}

func TestAStoppedCandidateReviewKeepsWhatItDecidedAndIsRecordedAsStoppedByHand(t *testing.T) {
	names := make([][2]string, 0, app.CheckBatch+3)
	for i := range app.CheckBatch + 3 {
		names = append(names, [2]string{fmt.Sprintf("Company %02d", i), ""})
	}
	r, _ := naming(names...)
	m := &model{verdicts: competitors}
	var svc *app.Service
	stopErr := errors.New("not asked")
	sm := stopping{model: m, stop: func() { stopErr = svc.StopReview(t.Context()) }}
	svc = reviewing(r, &directory{}, models{m: sm}, func(err error) { t.Errorf("reported %v", err) })
	if err := svc.StopReview(t.Context()); !errors.Is(err, app.ErrNothingToStop) {
		t.Fatalf("a stop before any review = %v", err)
	}
	if !svc.StartReview(t.Context()) {
		t.Fatal("the review did not start")
	}
	if stopErr != nil {
		t.Fatalf("the stop while it ran = %v", stopErr)
	}
	// The batch under way when the stop came is checked and kept; no batch is asked after it.
	if len(m.checked) != 1 || len(r.verdicts) != app.CheckBatch-1 {
		t.Errorf("checked %d times, %d verdicts kept", len(m.checked), len(r.verdicts))
	}
	if st := svc.ReviewState(); st.Running || st.Problem != app.ReviewProblemStopped {
		t.Errorf("state = %+v", st)
	}
	if err := svc.StopReview(t.Context()); !errors.Is(err, app.ErrNothingToStop) {
		t.Errorf("a stop after the review = %v", err)
	}
	// The next review takes up the rest, and is not stopped by the stop before it.
	if !svc.StartReview(t.Context()) {
		t.Fatal("the next review did not start")
	}
	if st := svc.ReviewState(); st.Problem != app.ReviewProblemNone || len(m.checked) != 2 {
		t.Errorf("the next review: state %+v, checked %d times", st, len(m.checked))
	}
}

func TestAReviewStoppedBeforeItsMergeAsksNothing(t *testing.T) {
	r, _ := naming([2]string{"Kienzle GmbH", ""})
	m := &model{verdicts: competitors}
	var later func()
	svc := app.New(r, gateways{gw: newGateway()}, &directory{}, func() time.Time { return start }, app.Config{Owner: "a"}, func(uuid.UUID, error) {},
		app.WithSpawn(func(f func()) { later = f }), app.WithReviewers(models{m: m}, &pages{}, func(err error) { t.Errorf("reported %v", err) }))
	svc.StartReview(t.Context())
	if err := svc.StopReview(t.Context()); err != nil {
		t.Fatalf("stop = %v", err)
	}
	later()
	if st := svc.ReviewState(); st.Problem != app.ReviewProblemStopped || m.merged != 0 || len(m.checked) != 0 {
		t.Errorf("state %+v, merged %d, checked %d", st, m.merged, len(m.checked))
	}
}

func TestAStoppedCompetitorCheckKeepsWhatItArchivedAndIsRecordedAsStoppedByHand(t *testing.T) {
	d := &directory{}
	for i := range app.CheckBatch + 2 {
		d.added = append(d.added, domain.DirectoryCompany{ID: fmt.Sprintf("rubber-%02d", i), Name: fmt.Sprintf("Rubber %02d", i), Role: domain.RoleCompetitor})
	}
	r := newRepo()
	m := &model{verdicts: func(cs []domain.Candidate) []domain.CompanyVerdict {
		var out []domain.CompanyVerdict
		for _, c := range cs {
			out = append(out, domain.CompanyVerdict{CandidateID: c.ID, Sure: true, Reason: "Makes rubber belting."})
		}
		return out
	}}
	var svc *app.Service
	stopErr := errors.New("not asked")
	sm := stopping{model: m, stop: func() { stopErr = svc.StopListCheck(t.Context()) }}
	svc = reviewing(r, d, models{m: sm}, func(err error) { t.Errorf("reported %v", err) })
	if err := svc.StopListCheck(t.Context()); !errors.Is(err, app.ErrNothingToStop) {
		t.Fatalf("a stop before any check = %v", err)
	}
	if !svc.StartListCheck(t.Context()) {
		t.Fatal("the check did not start")
	}
	if stopErr != nil {
		t.Fatalf("the stop while it ran = %v", stopErr)
	}
	// Twentebelt is checked too, but it is not one of the companies added here, so not archived.
	if len(m.checked) != 1 || len(r.checks) != app.CheckBatch || len(d.archived) != app.CheckBatch-1 {
		t.Errorf("checked %d times, %d checks kept, %d archived", len(m.checked), len(r.checks), len(d.archived))
	}
	if st := svc.ListCheckState(); st.Running || st.Problem != app.ReviewProblemStopped {
		t.Errorf("state = %+v", st)
	}
	// Stopping the check leaves the candidate review alone.
	if err := svc.StopReview(t.Context()); !errors.Is(err, app.ErrNothingToStop) {
		t.Errorf("the candidate review's stop = %v", err)
	}
	if st := svc.ReviewState(); st.Problem != app.ReviewProblemNone {
		t.Errorf("the candidate review's state = %+v", st)
	}
}
