package persistence_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
)

func repository(t *testing.T) *persistence.Repository {
	t.Helper()
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	return persistence.New(store)
}

var created = time.Date(2026, 10, 22, 9, 30, 0, 0, time.UTC)

func queued(t *testing.T, repo *persistence.Repository, user string, at time.Time) domain.Analysis {
	t.Helper()
	a := domain.Analysis{ID: uuid.New(), URL: "https://www.example.test/", CompetitorID: "example", MaxPages: 5, Status: domain.StatusQueued, CreatedBy: user, CreatedAt: at}
	n, err := repo.Create(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	a.Number = n
	return a
}

func TestAnAnalysisFromQueueToDone(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	a := queued(t, repo, "joris", created)

	if ids, err := repo.Queued(ctx); err != nil || len(ids) != 1 || ids[0] != a.ID {
		t.Fatalf("Queued = %v, %v", ids, err)
	}
	if err := repo.MarkRunning(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	for _, p := range []domain.PageSummary{
		{URL: "https://www.example.test/", Title: strings.Repeat("t", 600), Description: "d", Language: "en", WordCount: 12},
		{URL: "https://www.example.test/de", Title: "Förderbänder", Language: "de", WordCount: 3},
	} {
		if err := repo.AddPage(ctx, a.ID, p); err != nil {
			t.Fatal(err)
		}
	}
	kws := []domain.Keyword{
		{Phrase: "spiral belts", Language: "en", Score: 42.5, Pages: 2, Classification: domain.Classification{Tier: "product", Family: "tri_flex_turn_curve", Closest: "spiral belt", InMap: true}},
		{Phrase: "belt options", Language: "en", Score: 3, Pages: 1, Classification: domain.Classification{Tier: "broad"}},
	}
	known := []domain.KnownTerm{{GlossaryTerm: domain.GlossaryTerm{Text: "spiral belt", Language: "en", Family: "tri_flex_turn_curve", Tier: "product", UsedByTribelt: true}, Occurrences: 3}}
	if err := repo.SaveFindings(ctx, a.ID, kws, known); err != nil {
		t.Fatal(err)
	}
	eqs := []domain.Equivalent{
		{Product: "TwenteFlex", Family: "tri_flex_turn_curve", Reason: "spiral tower", SearchTerms: []domain.SearchTerm{{Term: "spiral belt", Language: "en"}, {Term: "Spiralband", Language: "de"}}},
		{Product: "Eyelink belts", Family: "eyelink", Reason: "eyelinks"},
	}
	if err := repo.SaveMatching(ctx, a.ID, domain.MatchingDone, "openrouter: anthropic/claude-haiku-5-5", eqs, domain.Usage{InputTokens: 900, OutputTokens: 100}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Finish(ctx, a.ID, created.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Get(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusDone || got.PagesCrawled != 2 || got.Matching != domain.MatchingDone || got.CompetitorID != "example" ||
		!got.FinishedAt.Equal(created.Add(time.Minute)) || !got.CreatedAt.Equal(created) || got.CreatedBy != "joris" ||
		got.MatchingModel != "openrouter: anthropic/claude-haiku-5-5" {
		t.Fatalf("analysis = %+v", got)
	}
	if len(got.Pages) != 2 || len([]rune(got.Pages[0].Title)) != 500 || got.Pages[1].Title != "Förderbänder" {
		t.Fatalf("pages = %+v", got.Pages)
	}
	if len(got.Keywords) != 2 || got.Keywords[0] != kws[0] || got.Keywords[1] != kws[1] || len(got.KnownTerms) != 1 || got.KnownTerms[0] != known[0] {
		t.Fatalf("findings = %+v %+v", got.Keywords, got.KnownTerms)
	}
	if len(got.Equivalents) != 2 || len(got.Equivalents[0].SearchTerms) != 2 || got.Equivalents[0].SearchTerms[1].Term != "Spiralband" ||
		len(got.Equivalents[1].SearchTerms) != 0 {
		t.Fatalf("equivalents = %+v", got.Equivalents)
	}
	if n, err := repo.TokensSince(ctx, created); err != nil || n != 1000 {
		t.Fatalf("TokensSince = %d, %v", n, err)
	}
	if n, err := repo.TokensSince(ctx, created.Add(time.Hour)); err != nil || n != 0 {
		t.Fatalf("TokensSince later = %d, %v", n, err)
	}
}

func TestFailingAndRecovering(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	failed := queued(t, repo, "u", created)
	running := queued(t, repo, "u", created.Add(time.Second))
	waiting := queued(t, repo, "v", created.Add(2*time.Second))
	if err := repo.Fail(ctx, failed.ID, strings.Repeat("x", 600), created); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkRunning(ctx, running.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.FailRunning(ctx, "restarted", created); err != nil {
		t.Fatal(err)
	}
	f, err := repo.Get(ctx, failed.ID)
	if err != nil || f.Status != domain.StatusFailed || len([]rune(f.Failure)) != 500 {
		t.Fatalf("failed = %+v, %v", f, err)
	}
	r, err := repo.Get(ctx, running.ID)
	if err != nil || r.Status != domain.StatusFailed || r.Failure != "restarted" {
		t.Fatalf("running = %+v, %v", r, err)
	}
	if ids, err := repo.Queued(ctx); err != nil || len(ids) != 1 || ids[0] != waiting.ID {
		t.Fatalf("Queued = %v, %v", ids, err)
	}
	list, err := repo.List(ctx, 2)
	if err != nil || len(list) != 2 || list[0].ID != waiting.ID || list[1].ID != running.ID {
		t.Fatalf("List = %+v, %v", list, err)
	}
	if n, err := repo.CountCreatedSince(ctx, "u", created); err != nil || n != 2 {
		t.Fatalf("CountCreatedSince(u) = %d, %v", n, err)
	}
}

func TestGetUnknownAndOutOfRange(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	if _, err := repo.Get(t.Context(), uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Get(unknown) = %v", err)
	}
	if _, err := repo.List(t.Context(), -1); err == nil {
		t.Fatal("List(-1) succeeded")
	}
}

func TestDatabaseRefusesWhatTheDomainWouldNot(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	a := queued(t, repo, "u", created)
	bad := []domain.KnownTerm{{GlossaryTerm: domain.GlossaryTerm{Text: "x", Language: "klingon", Family: "eyelink", Tier: "product"}, Occurrences: 1}}
	if err := repo.SaveFindings(t.Context(), a.ID, nil, bad); err == nil {
		t.Fatal("a known term in an unknown language was stored")
	}
	if err := repo.SaveMatching(t.Context(), a.ID, domain.MatchingDone, "", []domain.Equivalent{{Product: "x", Family: "plastic"}}, domain.Usage{}); err == nil {
		t.Fatal("an equivalent of an unknown family was stored")
	}
	if err := repo.SaveMatching(t.Context(), a.ID, domain.MatchingDone, "",
		[]domain.Equivalent{{Product: "x", Family: "other", SearchTerms: []domain.SearchTerm{{Term: "", Language: "en"}}}}, domain.Usage{}); err == nil {
		t.Fatal("an empty search term was stored")
	}
	if err := repo.SaveMatching(t.Context(), a.ID, "unknown", "", nil, domain.Usage{}); err == nil {
		t.Fatal("an unknown matching status was stored")
	}
	if err := repo.SaveFindings(t.Context(), a.ID, []domain.Keyword{{Phrase: "", Language: "en", Pages: 1}}, nil); err == nil {
		t.Fatal("an empty keyword was stored")
	}
	if _, err := repo.Create(t.Context(), domain.Analysis{ID: uuid.New(), URL: "x", MaxPages: 5, CreatedBy: "u", CreatedAt: created}); err == nil {
		t.Fatal("a too-short url was stored")
	}
	// The analysis is untouched by the refused writes.
	got, err := repo.Get(t.Context(), a.ID)
	if err != nil || got.Matching != domain.MatchingPending || len(got.KnownTerms) != 0 {
		t.Fatalf("after refused writes: %+v, %v", got, err)
	}
}

func TestAnalysesPerCompetitor(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	first := queued(t, repo, "u", created)
	second := queued(t, repo, "u", created.Add(time.Minute))
	other := domain.Analysis{ID: uuid.New(), URL: "https://other.test/", CompetitorID: "other", MaxPages: 1, Status: domain.StatusQueued, CreatedBy: "u", CreatedAt: created}
	legacy := domain.Analysis{ID: uuid.New(), URL: "https://legacy.test/", MaxPages: 1, Status: domain.StatusQueued, CreatedBy: "u", CreatedAt: created}
	for _, a := range []domain.Analysis{other, legacy} {
		if _, err := repo.Create(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	if first.Number != 1 || second.Number != 2 {
		t.Fatalf("numbers %d, %d", first.Number, second.Number)
	}
	list, err := repo.ListByCompetitor(ctx, "example", 10)
	if err != nil || len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID || list[0].CompetitorID != "example" {
		t.Fatalf("ListByCompetitor = %+v, %v", list, err)
	}
	stats, err := repo.CompetitorStats(ctx)
	if err != nil || len(stats) != 2 || stats["example"].Count != 2 || stats["example"].Last.ID != second.ID || stats["other"].Count != 1 {
		t.Fatalf("CompetitorStats = %+v, %v", stats, err)
	}
	if _, err := repo.ListByCompetitor(ctx, "example", -1); err == nil {
		t.Fatal("ListByCompetitor(-1) succeeded")
	}
	if got, err := repo.Get(ctx, second.ID); err != nil || got.Number != 2 {
		t.Fatalf("second scan = %+v, %v", got, err)
	}
	if got, err := repo.Get(ctx, legacy.ID); err != nil || got.CompetitorID != "" || got.Number != 0 {
		t.Fatalf("a scan without company = %+v, %v", got, err)
	}
}

func finished(t *testing.T, repo *persistence.Repository, competitor string, at time.Time, kws ...domain.Keyword) domain.Analysis {
	t.Helper()
	ctx := t.Context()
	a := domain.Analysis{
		ID: uuid.New(), URL: "https://" + competitor + ".test/", CompetitorID: competitor, MaxPages: 100,
		Status: domain.StatusQueued, CreatedBy: "u", CreatedAt: at, Rescan: true,
	}
	n, err := repo.Create(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	a.Number = n
	if err := repo.MarkRunning(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	for i := range 100 {
		if err := repo.AddPage(ctx, a.ID, domain.PageSummary{URL: "https://x.test/" + strings.Repeat("p", i), Family: "eyelink"}); err != nil {
			t.Fatalf("page %d: %v", i, err)
		}
	}
	if err := repo.SaveFindings(ctx, a.ID, kws, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.Finish(ctx, a.ID, at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestDeepScansHistoryAndFindings(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	eyelink := domain.Keyword{Phrase: "eyelink belts", Language: "en", Score: 40, Pages: 100, Classification: domain.Classification{Tier: "product", Family: "eyelink", InMap: true}}
	freezer := domain.Keyword{Phrase: "spiral freezer", Language: "en", Score: 12, Pages: 3, Classification: domain.Classification{Tier: "application", Family: "tri_flex_turn_curve", Closest: "spiral belt"}}
	first := finished(t, repo, "deep", created, eyelink)
	second := finished(t, repo, "deep", created.Add(time.Hour), eyelink, freezer)
	if first.Number != 1 || second.Number != 2 {
		t.Fatalf("numbers %d, %d", first.Number, second.Number)
	}
	got, err := repo.Get(ctx, second.ID)
	if err != nil || got.PagesCrawled != 100 || len(got.Pages) != 100 || got.Pages[0].Family != "eyelink" || got.MaxPages != 100 {
		t.Fatalf("a deep scan = %d pages, %v", len(got.Pages), err)
	}
	firstScans, err := repo.FirstScans(ctx, "deep")
	if err != nil || firstScans[domain.PhraseKey{Phrase: "eyelink belts", Language: "en"}] != 1 ||
		firstScans[domain.PhraseKey{Phrase: "spiral freezer", Language: "en"}] != 2 {
		t.Fatalf("FirstScans = %+v, %v", firstScans, err)
	}
	findings, err := repo.UnmappedFindings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var mine []domain.Finding
	for _, f := range findings {
		if f.CompetitorID == "deep" {
			mine = append(mine, f)
		}
	}
	if len(mine) != 1 || mine[0].Phrase != "spiral freezer" || mine[0].Closest != "spiral belt" || mine[0].AnalysisID != second.ID ||
		!mine[0].StartedAt.Equal(second.CreatedAt) || mine[0].URL != "https://deep.test/" {
		t.Fatalf("findings %+v", mine)
	}
	// Re-scans do not count against the hourly limit on single scans.
	if n, err := repo.CountCreatedSince(ctx, "u", created.Add(-time.Hour)); err != nil || n != 0 {
		t.Fatalf("CountCreatedSince = %d, %v", n, err)
	}
	if n, err := repo.CountActive(ctx, "deep"); err != nil || n != 0 {
		t.Fatalf("CountActive = %d, %v", n, err)
	}
	queued(t, repo, "u", created)
	if n, err := repo.CountActive(ctx, "example"); err != nil || n != 1 {
		t.Fatalf("CountActive(example) = %d, %v", n, err)
	}
	if err := repo.DeleteByCompetitor(ctx, "deep"); err != nil {
		t.Fatal(err)
	}
	if list, err := repo.ListByCompetitor(ctx, "deep", 10); err != nil || len(list) != 0 {
		t.Fatalf("after delete = %+v, %v", list, err)
	}
}
