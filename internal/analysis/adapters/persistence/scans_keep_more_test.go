package persistence_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// A scan records the pages that use each known term, and the newest finished scan of each company
// answers which of its pages use a term.
func TestTermPagesFromEachCompanysNewestScan(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	old := queued(t, repo, "joris", created)
	a := queued(t, repo, "joris", created.Add(time.Hour))
	for _, id := range []domain.Analysis{old, a} {
		for _, p := range []domain.PageSummary{
			{URL: "https://www.example.test/", Title: "Home", Language: "en"},
			{URL: "https://www.example.test/eyelink", Title: "Eyelink belts", Language: "en"},
		} {
			if err := repo.AddPage(ctx, id.ID, p); err != nil {
				t.Fatal(err)
			}
		}
	}
	eyelink := domain.GlossaryTerm{Text: "eyelink belt", Language: "en", Family: "eyelink", Tier: "product"}
	if err := repo.SaveFindings(ctx, old.ID, nil, []domain.KnownTerm{{GlossaryTerm: eyelink, Occurrences: 1}}); err != nil {
		t.Fatal(err)
	}
	known := []domain.KnownTerm{
		{GlossaryTerm: eyelink, Occurrences: 4, Pages: []int{1, 0}},
		{GlossaryTerm: domain.GlossaryTerm{Text: "spiral belt", Language: "en", Family: "open_spiral", Tier: "product"}, Occurrences: 1},
	}
	if err := repo.SaveFindings(ctx, a.ID, nil, known); err != nil {
		t.Fatal(err)
	}
	for _, id := range []domain.Analysis{old, a} {
		if err := repo.MarkRunning(ctx, id.ID); err != nil {
			t.Fatal(err)
		}
		if err := repo.Finish(ctx, id.ID, created.Add(2*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	got, err := repo.Get(ctx, a.ID)
	if err != nil || !slices.Equal(got.KnownTerms[0].Pages, []int{1, 0}) || got.KnownTerms[1].Pages != nil {
		t.Fatalf("known terms = %+v, %v", got.KnownTerms, err)
	}
	rows, err := repo.TermPages(ctx, "Eyelink Belt", "en")
	want := []domain.TermPageRow{
		{CompetitorID: "example", Position: 1, URL: "https://www.example.test/eyelink", Title: "Eyelink belts"},
		{CompetitorID: "example", Position: 0, URL: "https://www.example.test/", Title: "Home"},
	}
	if err != nil || !slices.Equal(rows, want) {
		t.Fatalf("TermPages = %+v, %v; want the newest scan's pages, in its order, whatever the case", rows, err)
	}
	// A term found before scans recorded its pages gives the company with no page.
	if rows, err := repo.TermPages(ctx, "spiral belt", "en"); err != nil || len(rows) != 1 || rows[0].Position != -1 {
		t.Fatalf("TermPages of a term without pages = %+v, %v", rows, err)
	}
	if rows, err := repo.TermPages(ctx, "eyelink belt", "de"); err != nil || len(rows) != 0 {
		t.Fatalf("TermPages in another language = %+v, %v", rows, err)
	}
}

// Only the pages given a text keep one, at most 100 KB as the database checks.
func TestAScanOfTribeltKeepsItsPagesText(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	a := queued(t, repo, "joris", created)
	for _, p := range []domain.PageSummary{
		{URL: "https://www.tribelt.nl/", Language: "nl", Text: "Tribelt maakt transportbanden."},
		{URL: "https://www.tribelt.nl/leeg", Language: "nl"},
	} {
		if err := repo.AddPage(ctx, a.ID, p); err != nil {
			t.Fatal(err)
		}
	}
	texts, err := repo.PageTexts(ctx, a.ID)
	if err != nil || len(texts) != 1 || texts[0].URL != "https://www.tribelt.nl/" || texts[0].Text != "Tribelt maakt transportbanden." {
		t.Fatalf("PageTexts = %+v, %v", texts, err)
	}
	if err := repo.AddPage(ctx, a.ID, domain.PageSummary{URL: "https://www.tribelt.nl/lang", Text: strings.Repeat("x", 100_001)}); err == nil {
		t.Error("the database took a page text over 100 KB")
	}
}
