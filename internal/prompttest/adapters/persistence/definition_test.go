package persistence_test

import (
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func TestTheCompetitorDefinitionIsStoredAsVersions(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	store, err := pg.Open(ctx, pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	repo := persistence.New(store)
	// The target market the code held is the first stored version, version 2; version 3 adds that
	// an oven, freezer or furnace builder with its own metal belts competes too.
	got, err := repo.Definitions(ctx)
	if err != nil || len(got) != 2 || got[1].Version != 2 || got[1].SavedBy != "import" || got[1].RestoredFrom != 0 ||
		!strings.Contains(got[1].Text, "A competitor makes metal conveyor belts") || !strings.Contains(got[1].Text, "1,150 °C") {
		t.Fatalf("Definitions = %+v, %v", got, err)
	}
	if got[0].Version != 3 || got[0].SavedBy != "import" || !strings.Contains(got[0].Text, "ovens, freezers or furnaces") ||
		got[0].Text == got[1].Text {
		t.Fatalf("newest definition = %+v", got[0])
	}
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	saved, err := repo.AddDefinition(ctx, "Only makers of eyelink belts compete.", "joris", 0, at)
	if err != nil || saved.Version != 4 || saved.Text != "Only makers of eyelink belts compete." || saved.SavedBy != "joris" || !saved.SavedAt.Equal(at) {
		t.Fatalf("AddDefinition = %+v, %v", saved, err)
	}
	restored, err := repo.AddDefinition(ctx, got[0].Text, "anna", 3, at.Add(time.Hour))
	if err != nil || restored.Version != 5 || restored.RestoredFrom != 3 {
		t.Fatalf("restored = %+v, %v", restored, err)
	}
	all, err := repo.Definitions(ctx)
	if err != nil || len(all) != 4 || all[0] != restored || all[1] != saved {
		t.Fatalf("all = %+v, %v", all, err)
	}
	if _, err := repo.AddDefinition(ctx, strings.Repeat("x", domain.MaxDefinition+1), "joris", 0, at); err == nil {
		t.Error("a definition too long was stored")
	}

	// What checking used: nothing yet; then the usage and the companies checked.
	if used, n, err := repo.CheckUsage(ctx); err != nil || used.InputTokens != 0 || n != 0 {
		t.Fatalf("CheckUsage = %+v, %d, %v", used, n, err)
	}
	if err := repo.SaveReviewUsage(ctx, []domain.ReviewUsage{{InputTokens: 900, OutputTokens: 90, At: at}}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveListChecks(ctx, []domain.ListCheck{{CompanyID: "fenner", Reason: "Rubber.", Model: "m", Version: 3, At: at}}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveCompanyVerdicts(ctx, []domain.CompanyVerdict{{CandidateID: "00000000000000a1", Reason: "Mesh.", Model: "m", Version: 3, At: at}}); err != nil {
		t.Fatal(err)
	}
	if used, n, err := repo.CheckUsage(ctx); err != nil || used.InputTokens != 900 || used.OutputTokens != 90 || n != 2 {
		t.Errorf("CheckUsage = %+v, %d, %v", used, n, err)
	}
}
