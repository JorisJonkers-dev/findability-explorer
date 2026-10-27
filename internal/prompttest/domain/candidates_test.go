package domain_test

import (
	"slices"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func idOf(name string) string { return domain.CandidateID(domain.NameKey(name)) }

func TestCandidatesFoldByWebsiteAndByTheModel(t *testing.T) {
	day := time.Date(2026, 10, 27, 8, 0, 0, 0, time.UTC)
	seen := func(name, url string, n int) []domain.Sighting {
		out := make([]domain.Sighting, 0, n)
		for i := range n {
			out = append(out, domain.Sighting{Name: name, URL: url, Provider: "openai", Model: "openai/gpt-oss-20b", At: day.Add(time.Duration(i) * time.Minute)})
		}
		return out
	}
	sightings := slices.Concat(
		// One company under four names, on one website.
		seen("Taiwan Metiz Alliance", "https://metiz.tw/product_conveyor_belt.htm", 4),
		seen("Metiz Alliance", "metiz.tw", 2),
		seen("Metiz", "http://www.metiz.tw/", 1),
		seen("Wire Mesh Conveyor Belts – Taiwan Metiz Alliance", "https://metiz.tw/product_conveyor_belt.htm", 1),
		// Merged by the language model: one name without a website.
		seen("Jonge Poerink Conveyors", "jpconveyors.com/", 3),
		seen("Jonge Poerink", "", 1),
		// A directory page is nobody's website: these stay apart.
		seen("Rothstein", "https://www.europages.nl/ondernemingen/rothstein", 1),
		seen("Kabelwerke", "https://www.europages.nl/ondernemingen/kabelwerke", 1),
		// On the list by website, under another name, with a second name on that website.
		seen("WBC Europe", "https://shop.wirebelt.co.uk/belts", 2),
		seen("Wire Belt Germany", "wirebelt.co.uk", 1),
		// Merged by the model into a company on the list.
		seen("Narvi", "", 1),
		// One company with two websites, the most given first.
		seen("Berner", "https://www.berner-gmbh.de/", 1),
		seen("Berner", "https://www.berner.de/", 2),
		// Dismissed, then a second spelling joined it.
		seen("Kienzle GmbH", "https://www.kienzle.com/", 2),
		seen("Kienzle Drahtförderbänder", "https://www.kienzle.com/de/", 1),
	)
	decisions := []domain.CandidateDecision{{ID: idOf("Kienzle GmbH"), Status: domain.CandidateDismissed, By: "joris", At: day}}
	merges := []domain.Merge{
		{CandidateID: idOf("Jonge Poerink"), IntoID: idOf("Jonge Poerink Conveyors"), By: domain.ModelMerger},
		{CandidateID: idOf("Narvi"), CompetitorID: "esfo", By: domain.ModelMerger},
		{CandidateID: idOf("Rothstein"), Separate: true, By: "joris"},
		{CandidateID: "ffffffffffffffff", IntoID: idOf("Rothstein"), By: domain.ModelMerger}, // a name no answer gives any more
	}
	got := domain.Candidates(sightings, decisions, directory(), merges)
	byName := map[string]domain.Candidate{}
	for _, c := range got {
		byName[c.Name] = c
	}
	if len(got) != 6 {
		t.Fatalf("candidates = %v", names(got))
	}

	metiz := byName["Taiwan Metiz Alliance"]
	if metiz.ID != idOf("Taiwan Metiz Alliance") || metiz.Answers != 8 || !slices.Equal(metiz.Sites, []string{"metiz.tw"}) ||
		metiz.Website != "https://metiz.tw/" || len(metiz.Members) != 4 || metiz.Members[0].Name != "Taiwan Metiz Alliance" ||
		metiz.Members[1].Joined != domain.JoinedBySite || metiz.Status() != domain.CandidatePending {
		t.Errorf("metiz = %+v", metiz)
	}
	if !slices.Contains(metiz.MemberIDs(), idOf("Metiz")) || len(metiz.Addresses) != 3 {
		t.Errorf("metiz members %v, addresses %v", metiz.MemberIDs(), metiz.Addresses)
	}
	jp := byName["Jonge Poerink Conveyors"]
	if jp.Answers != 4 || len(jp.Members) != 2 || jp.Members[1].Joined != domain.JoinedByModel || jp.Website != "https://jpconveyors.com/" {
		t.Errorf("jonge poerink = %+v", jp)
	}
	if r := byName["Rothstein"]; r.Answers != 1 || len(r.Sites) != 0 || r.Website != "" {
		t.Errorf("rothstein = %+v", r)
	}
	if b := byName["Berner"]; !slices.Equal(b.Sites, []string{"berner.de", "berner-gmbh.de"}) || b.Website != "https://www.berner.de/" {
		t.Errorf("berner = %+v", b)
	}
	if _, ok := byName["Kabelwerke"]; !ok {
		t.Error("two names on one directory were folded")
	}
	// The dismissal covers the spelling that joined later.
	if k := byName["Kienzle GmbH"]; k.Status() != domain.CandidateDismissed || len(k.Members) != 2 || k.Answers != 3 {
		t.Errorf("kienzle = %+v", k)
	}
	for _, gone := range []string{"WBC Europe", "Wire Belt Germany", "Narvi"} {
		if _, ok := byName[gone]; ok {
			t.Errorf("%s is on the list already, yet a candidate", gone)
		}
	}
	// The most answers first, then by name.
	if got[0].Name != "Taiwan Metiz Alliance" || got[1].Name != "Jonge Poerink Conveyors" {
		t.Errorf("order = %v", names(got))
	}
}

func names(cs []domain.Candidate) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, c.Name)
	}
	return out
}

func TestADecidedCompanyOnTheListStaysListedToUndo(t *testing.T) {
	sightings := []domain.Sighting{{Name: "Newly Added", URL: "https://newly.example/", Provider: "openai"}}
	dir := append(directory(), domain.DirectoryCompany{ID: "newly-added", Name: "Newly Added", Host: "newly.example", Role: domain.RoleCompetitor})
	decisions := []domain.CandidateDecision{{ID: idOf("Newly Added"), Status: domain.CandidateAdded, CompetitorID: "newly-added"}}
	got := domain.Candidates(sightings, decisions, dir, nil)
	if len(got) != 1 || got[0].Status() != domain.CandidateAdded || got[0].Decision.CompetitorID != "newly-added" {
		t.Errorf("candidates = %+v", got)
	}
}

func TestAnswersAreMatchedToTheListAsItIsNow(t *testing.T) {
	x := domain.Extraction{Companies: []domain.Company{
		{Name: "ESFO", CompetitorID: "esfo", Position: 1},
		{Name: "WBC", URL: "https://shop.wirebelt.co.uk/", Position: 2}, // on the list by website now
		{Name: "Jonge Poerink", Position: 3},                            // added from the candidates
		{Name: "Narvi", Position: 4},                                    // merged into a company on the list
		{Name: "Removed Since", Position: 5},                            // added, but no longer on the list
		{Name: "Unknown", Position: 6},
	}}
	decisions := []domain.CandidateDecision{
		{ID: idOf("Jonge Poerink"), Status: domain.CandidateAdded, CompetitorID: "jp"},
		{ID: idOf("Removed Since"), Status: domain.CandidateAdded, CompetitorID: "gone"},
		{ID: idOf("Unknown"), Status: domain.CandidateDismissed},
	}
	merges := []domain.Merge{{CandidateID: idOf("Narvi"), CompetitorID: "esfo"}, {CandidateID: idOf("Unknown"), IntoID: idOf("Narvi")}}
	dir := append(directory(), domain.DirectoryCompany{ID: "jp", Name: "Jonge Poerink Conveyors", Host: "jpconveyors.com", Role: domain.RoleCompetitor})
	got := domain.NewMatcher(dir, decisions, merges).Rematch(x)
	ids := make([]string, 0, len(got.Companies))
	for _, c := range got.Companies {
		ids = append(ids, c.CompetitorID)
	}
	if !slices.Equal(ids, []string{"esfo", "wire-belt", "jp", "esfo", "", ""}) || x.Companies[1].CompetitorID != "" {
		t.Errorf("rematched = %v; the stored extraction changed: %v", ids, x.Companies[1].CompetitorID != "")
	}
}

func TestTheModelsClustersBecomeMerges(t *testing.T) {
	at := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	asked := []domain.Candidate{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}, {ID: "e"}}
	got := domain.MergesOf([]domain.Cluster{
		{IDs: []string{"a", "b", "b", "zz"}},       // "zz" was not asked about
		{IDs: []string{"c"}, CompetitorID: "esfo"}, // one company on the list
		{IDs: []string{"d", "e"}},                  // "e" a person split off
		{IDs: []string{"c", "d"}, CompetitorID: "not-on-the-list"},
	}, asked, directory(), []domain.Merge{{CandidateID: "e", Separate: true}}, at)
	want := []domain.Merge{
		{CandidateID: "b", IntoID: "a", By: domain.ModelMerger, At: at},
		{CandidateID: "c", CompetitorID: "esfo", By: domain.ModelMerger, At: at},
		{CandidateID: "d", IntoID: "c", By: domain.ModelMerger, At: at},
	}
	if !slices.Equal(got, want) {
		t.Errorf("merges = %+v", got)
	}
}

func TestTheModelsCheckAttachesToTheWholeCompany(t *testing.T) {
	early, late := time.Date(2026, 10, 27, 8, 0, 0, 0, time.UTC), time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	cs := []domain.Candidate{
		{ID: "a", Members: []domain.Member{{ID: "a"}, {ID: "a2"}}},
		{ID: "b", Members: []domain.Member{{ID: "b"}}},
		{ID: "c", Members: []domain.Member{{ID: "c"}}, Decision: &domain.CandidateDecision{Status: domain.CandidateDismissed}},
	}
	got := domain.WithVerdicts(cs, []domain.CompanyVerdict{
		{CandidateID: "a", Competitor: true, Version: domain.CheckVersion, At: early},
		{CandidateID: "a2", Reason: "Plastic belts.", Version: domain.CheckVersion, At: late},
		// An older check: a waiting candidate is checked again, a decided one keeps it.
		{CandidateID: "b", Reason: "By name alone.", Version: 1, At: late},
		{CandidateID: "c", Reason: "By name alone.", Version: 1, At: late},
	})
	if got[0].Verdict == nil || got[0].Verdict.Reason != "Plastic belts." || got[1].Verdict != nil || got[2].Verdict == nil {
		t.Errorf("verdicts = %+v, %+v, %+v", got[0].Verdict, got[1].Verdict, got[2].Verdict)
	}
	if u := domain.Unchecked(got); len(u) != 1 || u[0].ID != "b" {
		t.Errorf("unchecked = %+v", u)
	}
}

func TestEveryCheckedCandidateGetsAVerdict(t *testing.T) {
	at := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	got := domain.CompleteVerdicts([]domain.Candidate{{ID: "a"}, {ID: "b"}},
		[]domain.CompanyVerdict{{CandidateID: "b", Competitor: true, Sure: true, Reason: "Makes mesh belts."}, {CandidateID: "zz", Sure: true}}, "m", at)
	if len(got) != 2 || got[0].CandidateID != "a" || got[0].Sure || got[0].Reason != "" || !got[1].Competitor || got[1].Model != "m" ||
		!got[0].At.Equal(at) || got[0].Version != domain.CheckVersion || got[1].Version != domain.CheckVersion {
		t.Errorf("verdicts = %+v", got)
	}
}
