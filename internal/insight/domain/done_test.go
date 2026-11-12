package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

func writtenRec(kind domain.GapKind, key string, over func(*domain.Recommendation)) domain.Recommendation {
	r := domain.Recommendation{ID: "r", RunID: "run", Gap: domain.Gap{Key: key, Kind: kind}, Title: "T", Benefit: 3, Effort: 2}
	if over != nil {
		over(&r)
	}
	return r
}

func TestATermIsDoneWhenTheScanFindsItOnTheNamedPage(t *testing.T) {
	pages := []domain.Page{
		{URL: "https://www.tribelt.nl/spiraalbanden/", Text: "Onze Spiraal-Band voor ovens."},
		{URL: "https://www.tribelt.nl/", Text: "Metalen transportbanden."},
	}
	named := writtenRec(domain.GapTerm, "term:nl:spiraal band", func(r *domain.Recommendation) { r.Where = "https://www.tribelt.nl/spiraalbanden" })
	done, parts := named.DoneByRule(domain.DoneFacts{Pages: pages})
	if !done || len(parts) != 1 || parts[0].Name != `"spiraal band" on www.tribelt.nl/spiraalbanden` {
		t.Errorf("on the named page = %v, %+v", done, parts)
	}
	elsewhere := writtenRec(domain.GapTerm, "term:nl:spiraal band", func(r *domain.Recommendation) { r.Where = "https://www.tribelt.nl/" })
	if done, _ := elsewhere.DoneByRule(domain.DoneFacts{Pages: pages}); done {
		t.Error("a term on another page is done")
	}
	anywhere := writtenRec(domain.GapTerm, "term:nl:transportbanden", nil)
	if done, parts := anywhere.DoneByRule(domain.DoneFacts{Pages: pages}); !done || parts[0].Name != `"transportbanden" on tribelt.nl` {
		t.Errorf("without a named page = %v, %+v", done, parts)
	}
	if done, _ := anywhere.DoneByRule(domain.DoneFacts{}); done {
		t.Error("done before any scan")
	}
}

func TestAPageIsDoneWhenTribeltHasOneForTheFamilyInTheLanguage(t *testing.T) {
	r := writtenRec(domain.GapPage, "page:de:eyelink", func(r *domain.Recommendation) { r.Gap.Language, r.Gap.Family = "de", "eyelink" })
	if done, parts := r.DoneByRule(domain.DoneFacts{Pages: []domain.Page{{Language: "de", Family: "eyelink"}}}); !done || parts[0].Name != "a German page about eyelink belts" {
		t.Errorf("with the page = %v, %+v", done, parts)
	}
	if done, _ := r.DoneByRule(domain.DoneFacts{Pages: []domain.Page{{Language: "nl", Family: "eyelink"}}}); done {
		t.Error("a Dutch page does it")
	}
}

func TestAListingIsDoneWhenAResponseCitingItNamesTribelt(t *testing.T) {
	r := writtenRec(domain.GapListing, "listing:directindustry.com", nil)
	web := func(named []string, sources ...string) domain.Answer {
		return domain.Answer{Mode: domain.ModeWeb, Named: named, Sources: sources}
	}
	cases := map[string]struct {
		answers []domain.Answer
		want    bool
	}{
		"names Tribelt":       {[]domain.Answer{web([]string{"tribelt"}, "https://www.directindustry.com/x")}, true},
		"links to tribelt.nl": {[]domain.Answer{web(nil, "https://directindustry.com/", "https://www.tribelt.nl/")}, true},
		"does not":            {[]domain.Answer{web([]string{"ashworth"}, "https://www.directindustry.com/x")}, false},
		"another site":        {[]domain.Answer{web([]string{"tribelt"}, "https://www.europages.de/")}, false},
		"from knowledge":      {[]domain.Answer{{Mode: domain.ModeKnowledge, Named: []string{"tribelt"}, Sources: []string{"https://directindustry.com/"}}}, false},
	}
	for name, c := range cases {
		if done, _ := r.DoneByRule(domain.DoneFacts{Run: &domain.Run{Answers: c.answers}}); done != c.want {
			t.Errorf("%s: %v", name, done)
		}
	}
	if done, parts := r.DoneByRule(domain.DoneFacts{}); done || parts[0].Name != "a web response citing directindustry.com that names Tribelt" {
		t.Errorf("without a run = %v, %+v", done, parts)
	}
}

func TestAReadinessCheckIsDoneWhenItPasses(t *testing.T) {
	open := &domain.Readiness{Bots: map[string]domain.BotAccess{}, Organization: true, LanguageVersions: true}
	shut := &domain.Readiness{Bots: map[string]domain.BotAccess{}, Blocked: []string{"GPTBot"}}
	for _, key := range []string{"readiness:crawlers", "readiness:organization", "readiness:languages"} {
		r := writtenRec(domain.GapReadiness, key, nil)
		if done, parts := r.DoneByRule(domain.DoneFacts{Tribelt: open}); !done || len(parts) != 1 {
			t.Errorf("%s passing = %v, %+v", key, done, parts)
		}
		if done, _ := r.DoneByRule(domain.DoneFacts{Tribelt: shut}); done {
			t.Errorf("%s failing is done", key)
		}
		if done, _ := r.DoneByRule(domain.DoneFacts{}); done {
			t.Errorf("%s before any scan is done", key)
		}
	}
	if done, parts := writtenRec(domain.GapReadiness, "readiness:unknown", nil).DoneByRule(domain.DoneFacts{Tribelt: open}); done || parts != nil {
		t.Errorf("an unknown check = %v, %+v", done, parts)
	}
}

func TestAFactSheetCorrectionIsTickedByHand(t *testing.T) {
	r := writtenRec(domain.GapFact, "fact:x", nil)
	if r.ScanChecked() {
		t.Error("a scan checks a correction")
	}
	if done, parts := r.DoneByRule(domain.DoneFacts{}); done || parts != nil {
		t.Errorf("by rule = %v, %+v", done, parts)
	}
	if item := r.Item("run", nil, domain.DoneFacts{}); item.Auto || !item.Manual() {
		t.Errorf("item = %+v", item)
	}
}

func TestTheRulesDecideTheItemsDoneAndARunKeepsWhatTheyFindDone(t *testing.T) {
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	f := domain.DoneFacts{Tribelt: &domain.Readiness{Organization: true}}
	done := writtenRec(domain.GapReadiness, "readiness:organization", nil)
	open := writtenRec(domain.GapReadiness, "readiness:languages", nil)
	item := done.Item("run", nil, f)
	if !item.Auto || !item.Done || item.New || len(item.Parts) != 1 || !item.Parts[0].Present {
		t.Errorf("done by rule = %+v", item)
	}
	if item := open.Item("run", nil, f); !item.Auto || item.Done || !item.New {
		t.Errorf("open = %+v", item)
	}
	kept := domain.KeepDone([]domain.Recommendation{done, open}, f, at)
	if !kept[0].DoneAt.Equal(at) || kept[0].DoneBy != "a scan" || !kept[1].DoneAt.IsZero() || !done.DoneAt.IsZero() {
		t.Errorf("KeepDone = %+v", kept)
	}
	if r := domain.Replace(kept, nil); len(r.Kept) != 1 || len(r.Replaced) != 1 {
		t.Errorf("replacement = %+v", r)
	}
}

func TestAnAdministratorEditsAWrittenRecommendation(t *testing.T) {
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	r := writtenRec(domain.GapTerm, "term:nl:x", func(r *domain.Recommendation) { r.WhereNote = domain.ForTheAgency })
	e := domain.Edit{
		Title: " Use x on the spiral page ", Where: " https://www.tribelt.nl/spiraal ", Why: " Because. ", Steps: []string{" One. ", " ", "Two."},
		DoneWhen: " Found. ", Benefit: 5, Effort: 1,
	}
	got, err := r.Apply(e, at)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Use x on the spiral page" || got.Where != "https://www.tribelt.nl/spiraal" || got.WhereNote != "" || got.Why != "Because." ||
		len(got.Steps) != 2 || got.DoneWhen != "Found." || got.Benefit != 5 || got.Effort != 1 || !got.EditedAt.Equal(at) || got.Open() {
		t.Errorf("edited = %+v", got)
	}
	e.Where = ""
	if got, _ := r.Apply(e, at); got.WhereNote != domain.ForTheAgency {
		t.Errorf("no page = %+v", got)
	}
	bad := map[string]func(*domain.Edit){
		"no title":       func(e *domain.Edit) { e.Title = " " },
		"a long title":   func(e *domain.Edit) { e.Title = strings.Repeat("x", 201) },
		"a long why":     func(e *domain.Edit) { e.Why = strings.Repeat("x", 501) },
		"too many steps": func(e *domain.Edit) { e.Steps = []string{"1", "2", "3", "4", "5", "6", "7"} },
		"a long step":    func(e *domain.Edit) { e.Steps = []string{strings.Repeat("x", 301)} },
		"a score of 6":   func(e *domain.Edit) { e.Benefit = 6 },
		"a score of 0":   func(e *domain.Edit) { e.Effort = 0 },
	}
	for name, breaks := range bad {
		e2 := e
		breaks(&e2)
		if _, err := r.Apply(e2, at); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("%s = %v", name, err)
		}
	}
}
