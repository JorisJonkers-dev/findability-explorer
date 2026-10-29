package persistence_test

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// scan stores a queued scan of company, from a re-scan when rescan is set, and starts it when
// running is set.
func scan(t *testing.T, repo *persistence.Repository, company string, rescan, running bool) uuid.UUID {
	t.Helper()
	a := domain.Analysis{
		ID: uuid.New(), URL: "https://" + company + ".test/", CompetitorID: company, MaxPages: 5, Status: domain.StatusQueued,
		CreatedBy: "admin", CreatedAt: created, Rescan: rescan,
	}
	if _, err := repo.Create(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	if running {
		if err := repo.MarkRunning(t.Context(), a.ID); err != nil {
			t.Fatal(err)
		}
	}
	return a.ID
}

func TestStoppingRecordsWaitingScansAsStoppedAndAsksRunningOnesToStop(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	at := created.Add(time.Hour)
	waiting, scanning := scan(t, repo, "a", true, false), scan(t, repo, "b", true, true)
	single, alone := scan(t, repo, "c", false, false), scan(t, repo, "d", false, true)

	// A running scan keeps the pages it read when asked to stop, and runs until it sees the request.
	if err := repo.AddPage(ctx, scanning, domain.PageSummary{URL: "https://b.test/", Language: "en"}); err != nil {
		t.Fatal(err)
	}
	if asked, err := repo.StopRequested(ctx, scanning); err != nil || asked {
		t.Fatalf("StopRequested before = %v, %v", asked, err)
	}
	rescan := domain.StopScope{RescanOnly: true}
	if ids, err := repo.StopQueued(ctx, rescan, domain.StoppedByHandReason, at); err != nil || !slices.Equal(ids, []uuid.UUID{waiting}) {
		t.Fatalf("StopQueued of the rescan = %v, %v", ids, err)
	}
	if ids, err := repo.RequestStop(ctx, rescan, at); err != nil || !slices.Equal(ids, []uuid.UUID{scanning}) {
		t.Fatalf("RequestStop of the rescan = %v, %v", ids, err)
	}
	if asked, err := repo.StopRequested(ctx, scanning); err != nil || !asked {
		t.Fatalf("StopRequested after = %v, %v", asked, err)
	}
	if got, err := repo.Get(ctx, waiting); err != nil || !got.StoppedByHand() || !got.FinishedAt.Equal(at) {
		t.Fatalf("stopped waiting scan = %+v, %v", got, err)
	}
	if got, err := repo.Get(ctx, scanning); err != nil || got.Status != domain.StatusRunning || len(got.Pages) != 1 {
		t.Fatalf("asked running scan = %+v, %v", got, err)
	}
	if ids, err := repo.Queued(ctx); err != nil || !slices.Equal(ids, []uuid.UUID{single}) {
		t.Fatalf("still queued = %v, %v", ids, err)
	}

	// One scan by its id; a scan that neither waits nor runs is not reached.
	if ids, err := repo.RequestStop(ctx, domain.StopScope{ID: alone}, at); err != nil || !slices.Equal(ids, []uuid.UUID{alone}) {
		t.Fatalf("RequestStop of one = %v, %v", ids, err)
	}
	if ids, err := repo.StopQueued(ctx, domain.StopScope{ID: waiting}, domain.StoppedByHandReason, at); err != nil || len(ids) != 0 {
		t.Fatalf("StopQueued of a stopped scan = %v, %v", ids, err)
	}
	// Every waiting scan at once.
	if ids, err := repo.StopQueued(ctx, domain.StopScope{}, domain.StoppedByHandReason, at); err != nil || !slices.Equal(ids, []uuid.UUID{single}) {
		t.Fatalf("StopQueued of all = %v, %v", ids, err)
	}
	if asked, err := repo.StopRequested(ctx, uuid.New()); err != nil || asked {
		t.Fatalf("StopRequested of an unknown scan = %v, %v", asked, err)
	}
}
