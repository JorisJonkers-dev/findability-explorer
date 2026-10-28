package domain_test

import (
	"slices"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

func TestTheAutomaticChecksFollowTribeltsScan(t *testing.T) {
	for _, tc := range []struct {
		name  string
		check domain.AutoCheck
		r     domain.Readiness
		want  bool
	}{
		{"a sitemap in all three languages", domain.CheckSitemaps, domain.Readiness{Sitemap: true, SitemapLanguages: []string{"nl", "de", "en"}}, true},
		{"a sitemap missing German", domain.CheckSitemaps, domain.Readiness{Sitemap: true, SitemapLanguages: []string{"nl", "en"}}, false},
		{"no sitemap", domain.CheckSitemaps, domain.Readiness{SitemapLanguages: []string{"nl", "de", "en"}}, false},
		{"both llms files", domain.CheckLLMsFiles, domain.Readiness{LLMs: true, LLMsFull: true}, true},
		{"llms.txt alone", domain.CheckLLMsFiles, domain.Readiness{LLMs: true}, false},
		{"Markdown pages", domain.CheckMarkdown, domain.Readiness{Markdown: true}, true},
		{"no Markdown", domain.CheckMarkdown, domain.Readiness{}, false},
		// Product markup is what the AI readiness page counts as such (Readiness.Product).
		{"products and FAQs", domain.CheckStructuredData, domain.Readiness{Product: true, StructuredData: []string{"FAQPage", "IndividualProduct"}}, true},
		{"products alone", domain.CheckStructuredData, domain.Readiness{Product: true, StructuredData: []string{"Product"}}, false},
		{"a manual improvement", domain.CheckNone, domain.Readiness{Markdown: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.check.Passes(tc.r); got != tc.want {
				t.Errorf("Passes = %v", got)
			}
		})
	}
}

func TestPlanItemsAreDoneByScanOrByTick(t *testing.T) {
	at := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	ticks := []domain.Tick{{ImprovementID: "submit-search-engines", Version: 1, By: "joris", At: at}}
	tribelt := &domain.Readiness{LLMs: true, LLMsFull: true}
	standings := domain.Standings(companies(), facts())
	items := domain.PlanItems(ticks, tribelt, standings)
	if len(items) != len(domain.Improvements()) {
		t.Fatalf("items = %d", len(items))
	}
	byID := map[string]domain.PlanItem{}
	for _, it := range items {
		byID[it.ID] = it
	}
	if it := byID["submit-search-engines"]; !it.Done || it.Auto || it.Tick.By != "joris" || it.Leader.CompanyID != "ashworth" {
		t.Errorf("a ticked improvement = %+v", it)
	}
	if it := byID["llms-files"]; !it.Done || !it.Auto || it.Leader.CompanyID != "martens" {
		t.Errorf("a scanned improvement = %+v", it)
	}
	if it := byID["markdown-pages"]; it.Done || !it.Auto {
		t.Errorf("an improvement the scan does not show = %+v", it)
	}
	if it := byID["listings"]; it.Done || it.Tick != nil {
		t.Errorf("an unticked improvement = %+v", it)
	}
	open := domain.Open(items, 5)
	if len(open) != 5 || open[0].ID != "buyer-words" {
		t.Errorf("open = %+v", open)
	}
	if none := domain.PlanItems(nil, nil, nil); none[2].Done || none[2].Leader != nil {
		t.Errorf("before any scan = %+v", none[2])
	}
}

func TestNumbersToWatchAndWhoToKeepAnEyeOn(t *testing.T) {
	f := facts()
	f.Run.Answers[0].Scope = "twente"
	f.Run.Answers[1].Scope = "twente"
	watched := domain.Watch(domain.DefaultTargets(), f)
	now := map[domain.Target]float64{}
	for _, w := range watched {
		if w.Now != nil {
			now[w.Target] = *w.Now
		}
	}
	if len(watched) != len(domain.Targets()) || now[domain.TargetShareOfVoice] != 0.25 || now[domain.TargetTwenteShareOfVoice] != 0.5 ||
		now[domain.TargetReadiness] != 2 || now[domain.TargetTermsUsed] != 38 || now[domain.TargetCitations] != 1 || watched[0].Goal != 0.40 {
		t.Errorf("watched = %v", now)
	}
	if empty := domain.Watch(domain.DefaultTargets(), domain.Facts{}); empty[0].Now != nil || empty[1].Now != nil || empty[4].Now != nil || empty[0].Met {
		t.Errorf("without data = %+v", empty)
	}
	// A target the value has reached is met: 38 terms against a target of 35, and exactly 1 citation.
	low := domain.DefaultTargets()
	low[domain.TargetTermsUsed], low[domain.TargetCitations] = 35, 1
	for _, w := range domain.Watch(low, f) {
		if met := w.Target == domain.TargetTermsUsed || w.Target == domain.TargetCitations; w.Met != met {
			t.Errorf("%s met = %v", w.Target, w.Met)
		}
	}
	eyes := domain.KeepAnEyeOn(companies(), f, domain.Standings(companies(), f))
	if len(eyes) != 3 || eyes[0].Why != domain.WatchMostNamed || eyes[0].Leader.CompanyID != "ashworth" ||
		eyes[1].Leader.CompanyID != "martens" || eyes[2].Why != domain.WatchTwente || eyes[2].Leader.CompanyID != "ashworth" {
		t.Errorf("eyes = %+v", eyes)
	}
	if none := domain.KeepAnEyeOn(companies(), domain.Facts{}, nil); len(none) != 0 {
		t.Errorf("without data = %+v", none)
	}
}

func TestImprovementsAreFoundByID(t *testing.T) {
	if imp, ok := domain.ImprovementByID("llms-files"); !ok || imp.Check != domain.CheckLLMsFiles {
		t.Errorf("llms-files = %+v, %v", imp, ok)
	}
	if _, ok := domain.ImprovementByID("nothing"); ok {
		t.Error("an unknown id was found")
	}
}

func TestTheAutomaticChecksSayWhatIsInPlaceAndWhatIsMissing(t *testing.T) {
	for _, tc := range []struct {
		name  string
		check domain.AutoCheck
		r     domain.Readiness
		want  []domain.Part
	}{
		{
			"a sitemap missing German", domain.CheckSitemaps,
			domain.Readiness{Sitemap: true, SitemapLanguages: []string{"nl", "en"}},
			[]domain.Part{{Name: "a Dutch sitemap", Present: true}, {Name: "a German sitemap"}, {Name: "an English sitemap", Present: true}},
		},
		{
			"llms.txt alone", domain.CheckLLMsFiles,
			domain.Readiness{LLMs: true},
			[]domain.Part{{Name: "llms.txt", Present: true}, {Name: "llms-full.txt"}},
		},
		{
			"Markdown pages", domain.CheckMarkdown,
			domain.Readiness{Markdown: true},
			[]domain.Part{{Name: "Markdown pages", Present: true}},
		},
		{
			"products alone", domain.CheckStructuredData,
			domain.Readiness{Product: true, StructuredData: []string{"ProductModel"}},
			[]domain.Part{{Name: "Product markup", Present: true}, {Name: "FAQ markup"}},
		},
		{"a manual improvement", domain.CheckNone, domain.Readiness{Markdown: true}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.check.Parts(tc.r); !slices.Equal(got, tc.want) {
				t.Errorf("Parts = %+v", got)
			}
		})
	}
}

func TestEveryImprovementSaysWhyHowAndWhenItIsDone(t *testing.T) {
	for _, imp := range domain.Improvements() {
		if imp.Why == "" || len(imp.Steps) == 0 || imp.DoneWhen == "" {
			t.Errorf("%s lacks its details: %+v", imp.ID, imp)
		}
	}
}

func TestPlanItemsCarryTribeltsValueAndWhatTheScanFound(t *testing.T) {
	tribelt := &domain.Readiness{LLMs: true}
	standings := domain.Standings(companies(), facts())
	byID := map[string]domain.PlanItem{}
	for _, it := range domain.PlanItems(nil, tribelt, standings) {
		byID[it.ID] = it
	}
	if it := byID["llms-files"]; it.Tribelt == nil || len(it.Parts) != 2 || !it.Parts[0].Present || it.Parts[1].Present {
		t.Errorf("a scanned improvement = %+v", it)
	}
	if it := byID["listings"]; it.Parts != nil {
		t.Errorf("a manual improvement has parts: %+v", it.Parts)
	}
	if none := domain.PlanItems(nil, nil, nil); none[2].Parts != nil || none[2].Tribelt != nil {
		t.Errorf("before any scan = %+v", none[2])
	}
}

func TestTheCatalogueCountsTheFamiliesAndLanguagesItNames(t *testing.T) {
	imp, _ := domain.ImprovementByID("family-pages")
	if imp.DoneWhen != "Each of the five belt families has its own page in all three languages." {
		t.Errorf("done when = %q", imp.DoneWhen)
	}
	if domain.InWords(0) != "no" || domain.InWords(10) != "ten" || domain.InWords(11) != "11" || domain.InWords(-1) != "-1" {
		t.Error("InWords writes small counts in words and others in digits")
	}
}

func TestImprovementsAreRankedByBenefitAgainstEffort(t *testing.T) {
	for _, imp := range domain.Improvements() {
		if imp.Benefit < domain.ScoreMin || imp.Benefit > domain.ScoreMax || imp.Effort < domain.ScoreMin || imp.Effort > domain.ScoreMax {
			t.Errorf("%s scores benefit %d and effort %d", imp.ID, imp.Benefit, imp.Effort)
		}
	}
	var ids []string
	for _, imp := range domain.ByValue(domain.Improvements()) {
		ids = append(ids, imp.ID)
	}
	// The ratio decides; a tie goes to the larger benefit, then to the catalogue's order.
	want := []string{"submit-search-engines", "llms-files", "buyer-words", "listings", "family-pages", "structured-data", "sitemap-per-language", "markdown-pages"}
	if !slices.Equal(ids, want) {
		t.Errorf("by value = %v", ids)
	}
	tied := []domain.Improvement{{ID: "a", Benefit: 1, Effort: 1}, {ID: "b", Benefit: 1, Effort: 1}, {ID: "c", Benefit: 2, Effort: 2}}
	if got := domain.ByValue(tied); got[0].ID != "c" || got[1].ID != "a" || got[2].ID != "b" {
		t.Errorf("a tie = %v", got)
	}
}
