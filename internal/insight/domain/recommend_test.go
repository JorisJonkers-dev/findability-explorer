package domain_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

var written = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

func ids() func() string {
	n := 0
	return func() string { n++; return fmt.Sprintf("r%d", n) }
}

func TestTheModelPlacesAndWordsEachGap(t *testing.T) {
	gaps := []domain.Gap{
		{Key: "term:nl:spiraalband", Kind: domain.GapTerm, Title: `Use "Spiraalband" on tribelt.nl`, Specifics: []string{"Two competitors use it."}, Benefit: 4, Effort: 2},
		{Key: "listing:directindustry.com", Kind: domain.GapListing, Title: "Get Tribelt listed on directindustry.com", Specifics: []string{"Cited twice."}, Benefit: 3, Effort: 3},
		{Key: "fact:x", Kind: domain.GapFact, Title: "Correct it", Specifics: []string{"A claim."}, Benefit: 5, Effort: 1},
	}
	pages := []domain.Page{{URL: "https://www.tribelt.nl/spiraalbanden/"}}
	w := []domain.Written{
		{
			Key: "term:nl:spiraalband", Title: " Use 'spiraalband' on the Dutch spiral belts page ", Where: "https://www.tribelt.nl/spiraalbanden",
			Why: "Ashworth uses it.", Steps: []string{"Add it to the heading.", " ", strings.Repeat("x", 400), "a", "b", "c", "d", "e"},
			DoneWhen: "The next scan finds it.", Benefit: 1, Effort: 2, ScoreReason: "Few buyers search for it.",
		},
		// A page the scan did not read, and a score moved by more than a point.
		{Key: "listing:directindustry.com", Title: "List Tribelt", Where: "https://www.tribelt.nl/nowhere", Benefit: 3, Effort: 3, ScoreReason: "Unmoved."},
		{Key: "listing:directindustry.com", Title: "A second answer is ignored"},
		{Key: "unknown", Title: "Not asked about"},
	}
	got := domain.Place(gaps, w, pages, "run1", written, ids())
	if len(got) != 3 {
		t.Fatalf("placed = %+v", got)
	}
	term := got[0]
	if term.ID != "r1" || term.RunID != "run1" || term.Title != "Use 'spiraalband' on the Dutch spiral belts page" || term.Where != "https://www.tribelt.nl/spiraalbanden" ||
		term.WhereNote != "" || term.Benefit != 3 || term.Effort != 2 || term.ScoreReason != "Few buyers search for it." || len(term.Steps) != 6 ||
		term.Steps[0] != "Add it to the heading." || len([]rune(term.Steps[1])) != 300 || !term.CreatedAt.Equal(written) {
		t.Errorf("the term = %+v", term)
	}
	listing := got[1]
	if listing.Where != "" || listing.WhereNote != domain.ForTheAgency || listing.Title != "List Tribelt" || listing.ScoreReason != "" || listing.Why != "Cited twice." {
		t.Errorf("the listing = %+v", listing)
	}
	// Nothing written: the rules' draft, for the web agency to place.
	if f := got[2]; f.Title != "Correct it" || f.Why != "A claim." || f.WhereNote != domain.ForTheAgency || f.Benefit != 5 || f.Effort != 1 {
		t.Errorf("the fact = %+v", f)
	}
	// A score moved beyond the bounds stays within them.
	edge := domain.Place([]domain.Gap{{Key: "k", Benefit: 5, Effort: 1}}, []domain.Written{{Key: "k", Benefit: 7, Effort: -3, ScoreReason: "x"}}, nil, "r", written, ids())
	if edge[0].Benefit != 5 || edge[0].Effort != 1 || edge[0].ScoreReason != "" {
		t.Errorf("at the edge = %+v", edge[0])
	}
}

func TestARunReplacesTheOpenRecommendationsAndKeepsTheDoneAndEditedOnes(t *testing.T) {
	open := domain.Recommendation{ID: "open", Gap: domain.Gap{Key: "a"}}
	done := domain.Recommendation{ID: "done", Gap: domain.Gap{Key: "b"}, DoneAt: written, DoneBy: "joris"}
	edited := domain.Recommendation{ID: "edited", Gap: domain.Gap{Key: "c"}, EditedAt: written}
	fresh := []domain.Recommendation{{ID: "new-a", Gap: domain.Gap{Key: "a"}}, {ID: "new-b", Gap: domain.Gap{Key: "b"}}, {ID: "new-d", Gap: domain.Gap{Key: "d"}}}
	r := domain.Replace([]domain.Recommendation{open, done, edited}, fresh)
	if len(r.Replaced) != 1 || r.Replaced[0].ID != "open" || len(r.Kept) != 2 || len(r.Added) != 2 || r.Added[0].ID != "new-a" || r.Added[1].ID != "new-d" {
		t.Errorf("replacement = %+v", r)
	}
	if n := domain.Replaces([]domain.Recommendation{open, done, edited}); n != 1 {
		t.Errorf("Replaces = %d", n)
	}
}

func TestAWrittenRecommendationIsListedInThePlan(t *testing.T) {
	standings := []domain.Standing{{Measure: domain.MeasureTermsUsed}}
	r := domain.Recommendation{
		ID: "r1", RunID: "run2", Gap: domain.Gap{Kind: domain.GapTerm}, Title: "Use it", Why: "Because.", Steps: []string{"Do it."}, DoneWhen: "Found.", Benefit: 4, Effort: 2,
	}
	item := r.Item("run2", standings)
	if item.ID != "r1" || item.Measure != domain.MeasureTermsUsed || item.Moves == "" || !item.Manual() || item.Done || !item.New || item.Written == nil ||
		item.Benefit != 4 || item.Steps[0] != "Do it." {
		t.Errorf("item = %+v", item)
	}
	r.DoneAt, r.DoneBy = written, "joris"
	if item := r.Item("run2", standings); !item.Done || item.New || item.Tick == nil || item.Tick.By != "joris" {
		t.Errorf("done = %+v", item)
	}
	r.DoneAt = time.Time{}
	if item := r.Item("run3", standings); item.New {
		t.Error("an older run's recommendation is new")
	}
	for _, k := range []domain.GapKind{domain.GapPage, domain.GapListing, domain.GapFact, domain.GapReadiness} {
		r.Gap.Kind = k
		if item := r.Item("", nil); item.Moves == "" || item.Measure == "" {
			t.Errorf("%s = %+v", k, item)
		}
	}
}

func TestWrittenRecommendationsJoinThePlanByValue(t *testing.T) {
	hand := domain.PlanItem{Improvement: domain.Improvement{ID: "hand", Benefit: 2, Effort: 2}}
	recs := []domain.Recommendation{
		{ID: "best", RunID: "run1", Gap: domain.Gap{Kind: domain.GapFact}, Benefit: 5, Effort: 1},
		{ID: "worst", RunID: "run1", Gap: domain.Gap{Kind: domain.GapPage}, Benefit: 1, Effort: 5},
		{ID: "tie", RunID: "run1", Gap: domain.Gap{Kind: domain.GapTerm}, Benefit: 4, Effort: 4},
	}
	got := domain.WithWritten([]domain.PlanItem{hand}, recs, "run1", nil)
	var order []string
	for _, it := range got {
		order = append(order, it.ID)
	}
	if fmt.Sprint(order) != "[best tie hand worst]" {
		t.Errorf("order = %v", order)
	}
	runs := []domain.RecRun{{ID: "c", Ending: "failed"}, {ID: "b", Ending: "done"}, {ID: "a", Ending: "done"}}
	if domain.NewestDone(runs) != "b" || domain.NewestDone(runs[:1]) != "" {
		t.Error("NewestDone")
	}
}

func TestARunIsEstimatedFromItsGapsAndPages(t *testing.T) {
	gaps := make([]domain.Gap, 12)
	pages := []domain.Page{{URL: "https://www.tribelt.nl/", Text: strings.Repeat("x", 4000)}}
	one := domain.EstimateRun(gaps[:1], pages, 10, "claude-haiku-5-5")
	two := domain.EstimateRun(gaps, pages, 10, "claude-haiku-5-5")
	if one <= 0 || two <= one || domain.EstimateRun(nil, pages, 10, "claude-haiku-5-5") != 0 {
		t.Errorf("estimates = %s, %s", one, two)
	}
}

func TestAnItemsKindIsItsGapsOrWhetherAScanChecksIt(t *testing.T) {
	written := domain.Recommendation{Gap: domain.Gap{Kind: domain.GapListing}}
	if k := (domain.PlanItem{Written: &written}).Kind(); k != "listing" {
		t.Errorf("written = %s", k)
	}
	if k := (domain.PlanItem{Auto: true}).Kind(); k != "readiness" {
		t.Errorf("scanned = %s", k)
	}
	if k := (domain.PlanItem{}).Kind(); k != "general" {
		t.Errorf("by hand = %s", k)
	}
}
