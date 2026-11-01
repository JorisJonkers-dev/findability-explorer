package domain_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

func gapCompanies() []domain.Company {
	return []domain.Company{
		{ID: domain.TribeltID, Name: "Tribelt", Role: domain.RoleTribelt, Website: "https://www.tribelt.nl/"},
		{ID: "ashworth", Name: "Ashworth", Role: domain.RoleCompetitor, Website: "https://www.ashworth.com/"},
		{ID: "wirebelt", Name: "Wire Belt", Role: domain.RoleCompetitor, Website: "https://wirebelt.com/"},
		{ID: "spiraal", Name: "De Spiraal", Role: domain.RoleGroup},
	}
}

func byKey(gaps []domain.Gap) map[string]domain.Gap {
	out := map[string]domain.Gap{}
	for _, g := range gaps {
		out[g.Key] = g
	}
	return out
}

func TestSearchTermsCompetitorsUseAndTribeltDoesNotAreGaps(t *testing.T) {
	in := domain.GapInputs{
		Companies: gapCompanies(),
		Terms: domain.TermMap{Terms: []domain.MapTerm{
			// Two competitors, a buyer close to buying, and a page of its family: benefit 4, effort 2.
			{Text: "Spiraalband", Language: "nl", Family: "open_spiral", Tier: "product", FoundBy: []string{"ashworth", "wirebelt", "ashworth", "spiraal"}},
			// Credited by the research to one competitor, broad, and no page: benefit 2, effort 3.
			{Text: "metal belt", Language: "en", Family: "other", Tier: "broad", UsedBy: 1},
			// Tribelt uses it, or no competitor does: no gap.
			{Text: "eyelink belt", Language: "en", Family: "eyelink", Tier: "product", FoundBy: []string{"ashworth"}, UsedByTribelt: true},
			{Text: "groepsband", Language: "nl", Family: "eyelink", Tier: "product", FoundBy: []string{"spiraal"}},
		}},
		Pages: []domain.Page{{URL: "https://www.tribelt.nl/spiraalbanden", Language: "nl", Family: "open_spiral"}},
	}
	gaps := byKey(domain.Gaps(in))
	g := gaps["term:nl:spiraalband"]
	if g.Kind != domain.GapTerm || g.Benefit != 4 || g.Effort != 2 || g.Title != `Use "Spiraalband" on tribelt.nl` || g.Family != "open_spiral" ||
		g.Specifics[0] != `2 competitors use the Dutch search term "Spiraalband"; tribelt.nl does not.` {
		t.Errorf("Spiraalband = %+v", g)
	}
	if g := gaps["term:en:metal belt"]; g.Benefit != 2 || g.Effort != 3 {
		t.Errorf("metal belt = %+v", g)
	}
	if _, ok := gaps["term:en:eyelink belt"]; ok {
		t.Error("a term Tribelt uses is a gap")
	}
	if _, ok := gaps["term:nl:groepsband"]; ok {
		t.Error("a term only a group company uses is a gap")
	}
}

func TestAFamilyWithoutATribeltPageInALanguageCompetitorsUseIsAGap(t *testing.T) {
	run := &domain.Run{StartedAt: time.Now(), Answers: []domain.Answer{
		{Family: "eyelink", Named: []string{"ashworth", "ashworth"}}, {Family: "eyelink", Named: []string{"ashworth", domain.TribeltID}},
		{Family: "open_spiral", Named: []string{domain.TribeltID}},
	}}
	in := domain.GapInputs{
		Companies: gapCompanies(), Run: run,
		Terms: domain.TermMap{Terms: []domain.MapTerm{
			{Text: "Drahtösenband", Language: "de", Family: "eyelink", Tier: "product", FoundBy: []string{"ashworth", "wirebelt"}},
			{Text: "Spiralband", Language: "de", Family: "open_spiral", Tier: "product", FoundBy: []string{"ashworth"}},
			// Tribelt uses a term of the family in the language, or has a page: no page gap.
			{Text: "eyelink belt", Language: "en", Family: "eyelink", Tier: "product", FoundBy: []string{"ashworth"}},
			{Text: "eyelink", Language: "en", Family: "eyelink", Tier: "broad", UsedByTribelt: true},
			{Text: "spiraalband", Language: "nl", Family: "open_spiral", Tier: "product", FoundBy: []string{"ashworth"}},
			// An unknown family or language: no page gap.
			{Text: "x", Language: "fr", Family: "eyelink", FoundBy: []string{"ashworth"}},
		}},
		Pages: []domain.Page{{Language: "nl", Family: "open_spiral"}},
	}
	gaps := byKey(domain.Gaps(in))
	g := gaps["page:de:eyelink"]
	// Two competitors, and Ashworth named more often than Tribelt in eyelink responses.
	if g.Kind != domain.GapPage || g.Benefit != 4 || g.Effort != 4 || g.Title != "A German page about eyelink belts" || len(g.Specifics) != 2 ||
		g.Specifics[1] != "AI responses about eyelink belts name Ashworth more often than Tribelt." {
		t.Errorf("eyelink in German = %+v", g)
	}
	if g := gaps["page:de:open_spiral"]; g.Benefit != 2 || len(g.Specifics) != 1 {
		t.Errorf("open spiral in German = %+v", g)
	}
	// Without a run, the responses add nothing.
	in.Run = nil
	if g := byKey(domain.Gaps(in))["page:de:eyelink"]; g.Benefit != 3 || len(g.Specifics) != 1 {
		t.Errorf("without a run = %+v", g)
	}
	for _, k := range []string{"page:en:eyelink", "page:nl:open_spiral", "page:fr:eyelink"} {
		if _, ok := gaps[k]; ok {
			t.Errorf("%s is a gap", k)
		}
	}
}

func TestSitesWebResponsesCiteForCompetitorsButNotTribeltAreGaps(t *testing.T) {
	web := func(named []string, sources ...string) domain.Answer {
		return domain.Answer{Mode: domain.ModeWeb, Named: named, Sources: sources}
	}
	run := &domain.Run{Answers: []domain.Answer{
		web([]string{"ashworth"}, "https://www.directindustry.com/belts", "https://directindustry.com/x", "https://www.ashworth.com/", "not an address"),
		web([]string{"wirebelt"}, "https://www.directindustry.com/y", "https://www.europages.de/"),
		// Naming Tribelt, naming no competitor, or from knowledge: not counted.
		web([]string{"ashworth", domain.TribeltID}, "https://www.thomasnet.com/"),
		web([]string{"spiraal"}, "https://www.thomasnet.com/"),
		{Mode: domain.ModeKnowledge, Named: []string{"ashworth"}, Sources: []string{"https://www.thomasnet.com/"}},
	}}
	gaps := byKey(domain.Gaps(domain.GapInputs{Companies: gapCompanies(), Run: run}))
	if g := gaps["listing:directindustry.com"]; g.Kind != domain.GapListing || g.Benefit != 3 || g.Effort != 3 || g.Title != "Get Tribelt listed on directindustry.com" ||
		g.Specifics[0] != "2 web responses cite directindustry.com and name competitors, but not Tribelt." {
		t.Errorf("directindustry = %+v", g)
	}
	if g := gaps["listing:europages.de"]; g.Benefit != 2 {
		t.Errorf("europages = %+v", g)
	}
	for _, k := range []string{"listing:ashworth.com", "listing:thomasnet.com"} {
		if _, ok := gaps[k]; ok {
			t.Errorf("%s is a gap", k)
		}
	}
	if gaps := domain.Gaps(domain.GapInputs{Companies: gapCompanies()}); len(gaps) != 0 {
		t.Errorf("without a run = %+v", gaps)
	}
}

func TestClaimsTheFactCheckMarkedWrongAreGaps(t *testing.T) {
	run := &domain.Run{Answers: []domain.Answer{
		{WrongClaims: []string{"Tribelt is in Hengelo.", " "}}, {WrongClaims: []string{"tribelt is in Hengelo. "}}, {WrongClaims: []string{"Tribelt makes rubber belts."}},
	}}
	gaps := byKey(domain.Gaps(domain.GapInputs{Run: run}))
	if g := gaps["fact:tribelt is in hengelo."]; g.Kind != domain.GapFact || g.Benefit != 4 || g.Effort != 1 ||
		g.Specifics[0] != `2 responses claim: "Tribelt is in Hengelo."` {
		t.Errorf("Hengelo = %+v", g)
	}
	if g := gaps["fact:tribelt makes rubber belts."]; g.Benefit != 3 {
		t.Errorf("rubber = %+v", g)
	}
	if len(gaps) != 2 {
		t.Errorf("gaps = %+v", gaps)
	}
}

func TestReadinessChecksNoWrittenRecommendationCoversAreGaps(t *testing.T) {
	failing := domain.Readiness{Blocked: []string{"GPTBot", "ClaudeBot"}}
	gaps := byKey(domain.Gaps(domain.GapInputs{Tribelt: &failing}))
	if g := gaps["readiness:crawlers"]; g.Benefit != 5 || g.Effort != 1 || g.Specifics[0] != "robots.txt keeps out ClaudeBot, GPTBot." {
		t.Errorf("crawlers = %+v", g)
	}
	if _, ok := gaps["readiness:organization"]; !ok {
		t.Error("no Organization gap")
	}
	if _, ok := gaps["readiness:languages"]; !ok {
		t.Error("no languages gap")
	}
	passing := domain.Readiness{Organization: true, LanguageVersions: true}
	if gaps := domain.Gaps(domain.GapInputs{Tribelt: &passing}); len(gaps) != 0 {
		t.Errorf("passing = %+v", gaps)
	}
}

func TestTheGapsAreRankedBestValueFirstAndCapped(t *testing.T) {
	var claims []string
	for i := range domain.MaxGaps + 5 {
		claims = append(claims, fmt.Sprintf("Wrong claim %02d.", i))
	}
	run := &domain.Run{Answers: []domain.Answer{{WrongClaims: claims}, {WrongClaims: []string{"Wrong claim 07."}}}}
	gaps := domain.Gaps(domain.GapInputs{Run: run, Tribelt: &domain.Readiness{Organization: true, LanguageVersions: true, Blocked: []string{"GPTBot"}}})
	// Crawlers (5 of 1) first, then the claim made twice (4 of 1), then the rest by key.
	if len(gaps) != domain.MaxGaps || gaps[0].Key != "readiness:crawlers" || gaps[1].Key != "fact:wrong claim 07." || gaps[2].Key != "fact:wrong claim 00." {
		t.Errorf("first gaps = %s, %s, %s of %d", gaps[0].Key, gaps[1].Key, gaps[2].Key, len(gaps))
	}
	// A term with 8 competitors scores the most.
	many := domain.MapTerm{Text: "band", Language: "nl", Family: "other", Tier: "spec", UsedBy: 9}
	if g := domain.Gaps(domain.GapInputs{Terms: domain.TermMap{Terms: []domain.MapTerm{many}}}); g[0].Benefit != 5 {
		t.Errorf("a term of nine = %+v", g[0])
	}
	four := domain.MapTerm{Text: "band", Language: "nl", Family: "other", Tier: "broad", UsedBy: 4}
	if g := domain.Gaps(domain.GapInputs{Terms: domain.TermMap{Terms: []domain.MapTerm{four}}}); g[0].Benefit != 4 {
		t.Errorf("a term of four = %+v", g[0])
	}
}

func TestThePageTextIsStaleWhenTribeltWasScannedBeforeTheNewestRun(t *testing.T) {
	run := &domain.Run{StartedAt: time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC)}
	if !domain.Stale(run.StartedAt.Add(-time.Hour), run) || !domain.Stale(time.Time{}, run) {
		t.Error("an older scan, or none, is not stale")
	}
	if domain.Stale(run.StartedAt.Add(time.Hour), run) || domain.Stale(time.Time{}, nil) {
		t.Error("a newer scan, or no run, is stale")
	}
}
