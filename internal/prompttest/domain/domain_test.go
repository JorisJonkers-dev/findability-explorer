package domain_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func TestThePromptSetAsksTwentyApplicationQuestionsInGeneralTerms(t *testing.T) {
	set := domain.CurrentPromptSet()
	if domain.PromptSetVersion != 2 || set.Version != domain.PromptSetVersion || len(set.Prompts) != 20 {
		t.Fatalf("set v%d has %d prompts", set.Version, len(set.Prompts))
	}
	ids := map[string]bool{}
	languages := map[string]int{}
	for _, p := range set.Prompts {
		if ids[p.ID] {
			t.Errorf("duplicate id %s", p.ID)
		}
		ids[p.ID] = true
		languages[p.Language]++
		if p.Kind != domain.KindApplication || p.Language != domain.LanguageOf(p.Scope) || !slices.Contains(domain.SetScopes(), p.Scope) {
			t.Errorf("prompt %s = %+v", p.ID, p)
		}
		if lower := strings.ToLower(p.Text); strings.Contains(lower, "tribelt") || strings.Contains(lower, "tri-flex") || strings.Contains(lower, "(") {
			t.Errorf("prompt %s names Tribelt or its products, or explains a term: %s", p.ID, p.Text)
		}
	}
	if languages["nl"] != 10 || languages["de"] != 5 || languages["en"] != 5 {
		t.Errorf("languages = %v", languages)
	}
	byID := func(id string) string {
		i := slices.IndexFunc(set.Prompts, func(p domain.Prompt) bool { return p.ID == id })
		if i < 0 {
			t.Fatalf("no prompt %s", id)
		}
		return set.Prompts[i].Text
	}
	want := map[string]string{
		"eyelink-application-english":            "We are building a tunnel oven for an industrial bakery. What kind of conveyor belt should we use, and which manufacturers can supply it?",
		"eyelink-application-twente":             "We bouwen een doorloopoven voor een industriële bakkerij. Wat voor transportband moeten we daarvoor gebruiken, en welke fabrikanten in de regio Twente kunnen die leveren?",
		"tri_flex_turn_curve-application-german": "Wir installieren einen Spiralfroster für Tiefkühlprodukte. Welches Förderband sollten wir dafür einsetzen, und welche Hersteller können es liefern?",
		"other-application-dutch":                "We moeten zware, hete onderdelen door een hardingsoven transporteren. Wat voor transportband moeten we daarvoor gebruiken, en welke fabrikanten kunnen die leveren?",
	}
	for id, text := range want {
		if got := byID(id); got != text {
			t.Errorf("%s = %q", id, got)
		}
	}
}

func model(id string, prompt, completion float64, out ...string) domain.CatalogueModel {
	if len(out) == 0 {
		out = []string{"text"}
	}
	return domain.CatalogueModel{ID: id, PromptPrice: prompt, CompletionPrice: completion, InputModalities: []string{"text", "image"}, OutputModalities: out}
}

func TestChooseModelsTakesTheCheapestTextModelOfEachProvider(t *testing.T) {
	catalogue := []domain.CatalogueModel{
		model("openai/gpt-oss-20b", 0.00000003, 0.00000014),
		model("openai/gpt-oss-20b:batch", 0.00000001, 0.00000001),
		model("openai/gpt-image", 0, 0, "image"),
		model("openai/embedding", 0, 0, "embeddings"),
		model("google/gemma:free", 0, 0),
		model("google/lyria", 0, 0, "audio", "text"),
		model("google/gemini-flash-lite", 0.0000001, 0.0000004),
		model("meta-llama/llama-guard-4", 0, 0),
		model("meta-llama/llama-3.1-8b", 0.00000002, 0.00000005),
		model("openrouter/auto", -1, -1),
		model("mistralai/nemo:free", 0, 0),
		model("mistralai/aaa:free", 0, 0),
	}
	got := domain.ChooseModels(catalogue, []string{"openai", "google", "meta-llama", "mistralai", "x-ai"})
	want := []domain.ModelChoice{
		{Provider: "openai", Model: "openai/gpt-oss-20b"},
		{Provider: "google", Model: "google/gemma:free", Free: true, Fallback: "google/gemini-flash-lite"},
		{Provider: "meta-llama", Model: "meta-llama/llama-3.1-8b"},
		{Provider: "mistralai", Model: "mistralai/aaa:free", Free: true},
	}
	if !slices.Equal(got, want) {
		t.Errorf("choices = %+v\nwant %+v", got, want)
	}
	if catalogue[0].Provider() != "openai" || !catalogue[4].Free() {
		t.Error("provider or free")
	}
}

func TestTasksAskEveryPromptOfEveryModelInBothModes(t *testing.T) {
	models := []domain.ModelChoice{{Provider: "openai"}, {Provider: "google"}}
	tasks := domain.Tasks(domain.CurrentPromptSet(), models)
	if len(tasks) != 80 {
		t.Fatalf("%d tasks", len(tasks))
	}
	keys := map[domain.TaskKey]bool{}
	for _, task := range tasks {
		keys[task.Key()] = true
	}
	if len(keys) != 80 || tasks[1].Key() != (domain.TaskKey{PromptID: "eyelink-application-twente", Provider: "google", Mode: domain.ModeKnowledge}) {
		t.Errorf("tasks[1] = %+v", tasks[1].Key())
	}
	a := domain.Answer{Prompt: tasks[3].Prompt, Provider: tasks[3].Provider, Mode: tasks[3].Mode}
	if a.Key() != tasks[3].Key() || tasks[3].Mode != domain.ModeWeb {
		t.Errorf("answer key = %+v", a.Key())
	}
	if domain.MicrosOf(1.2345675) != 1234568 || domain.Micros(2_500_000).Dollars() != 2.5 {
		t.Error("micros")
	}
}

func directory() []domain.DirectoryCompany {
	return []domain.DirectoryCompany{
		{ID: "tribelt", Name: "Tribelt B.V.", Host: "tribelt.nl", Role: domain.RoleTribelt},
		{ID: "esfo", Name: "ESFO Transportbanden B.V. (ESFO Metal Belts, Narviflex Group)", Host: "esfo-metalbelts.com", Role: domain.RoleCompetitor},
		{ID: "martens", Name: "Märtens Transportbänder GmbH", OtherNames: []string{"MARTENS Belts", "GmbH"}, Host: "martens-belts.com", Role: domain.RoleCompetitor},
		{ID: "wire-belt", Name: "Wire Belt Company Ltd", Host: "wirebelt.co.uk", Role: domain.RoleCompetitor},
		{ID: "abc", Name: "ABC Belts", Role: domain.RoleCompetitor},
		{ID: "de-spiraal", Name: "De Spiraal B.V.", Role: domain.RoleGroup},
	}
}

func TestMatchFindsACompanyByNameAliasOrWebsite(t *testing.T) {
	cases := map[string]domain.Named{
		"esfo":       {Name: "ESFO"},
		"martens":    {Name: "Martens Transportbander"},
		"wire-belt":  {Name: "Wire Belt"},
		"tribelt":    {Name: "Some Dutch maker", URL: "https://shop.tribelt.nl/en"},
		"de-spiraal": {Name: "De Spiraal"},
	}
	for want, n := range cases {
		if got := domain.Match(n, directory()); got != want {
			t.Errorf("Match(%+v) = %q, want %q", n, got, want)
		}
	}
	// The same registered domain: a subdomain, or an address without a scheme.
	if got := domain.Match(domain.Named{Name: "WBC", URL: "shop.wirebelt.co.uk/belts"}, directory()); got != "wire-belt" {
		t.Errorf("a subdomain = %q", got)
	}
	if got := domain.Match(domain.Named{Name: "Martens Belts Ltd"}, directory()); got != "martens" {
		t.Errorf("another name = %q", got)
	}
	if got := domain.Match(domain.Named{Name: "Narviflex"}, directory()); got != "esfo" {
		t.Errorf("an alias's first word = %q", got)
	}
	// A page on a directory is nobody's website.
	for _, n := range []domain.Named{{Name: "Ashworth"}, {Name: "AB"}, {Name: "B.V."}, {Name: "Ashworth", URL: "http://[::1"}, {Name: "Rothstein", URL: "https://www.europages.nl/tribelt.nl"}} {
		if got := domain.Match(n, directory()); got != "" {
			t.Errorf("Match(%+v) = %q, want none", n, got)
		}
	}
	if domain.NameKey("Märtens Transportbänder GmbH (Germany)") != "martens transportbander" {
		t.Errorf("key = %q", domain.NameKey("Märtens Transportbänder GmbH (Germany)"))
	}
}

func TestReadMatchesNamesAndFindsTribeltByNameOrAddress(t *testing.T) {
	x := domain.Extracted{
		Companies:   []domain.Named{{Name: "Twentebelt"}, {Name: "ESFO"}},
		Addresses:   []string{"https://twentebelt.com", " "},
		Correctness: domain.CorrectnessWrong, WrongClaims: []string{"Tribelt is in Belgium."},
	}
	got := domain.Read(x, []string{"https://www.tribelt.nl/producten", "https://twentebelt.com"}, directory())
	if len(got.Companies) != 2 || got.Companies[0].CompetitorID != "" || got.Companies[1] != (domain.Company{Name: "ESFO", CompetitorID: "esfo", Position: 2}) {
		t.Errorf("companies = %+v", got.Companies)
	}
	if !got.CitesTribelt || got.NamesTribelt || got.Correctness != domain.CorrectnessNone || got.WrongClaims != nil {
		t.Errorf("an answer citing but not naming Tribelt has no correctness: %+v", got)
	}
	if !slices.Equal(got.Addresses, []string{"https://twentebelt.com", "https://www.tribelt.nl/producten"}) {
		t.Errorf("addresses = %v", got.Addresses)
	}
	named := domain.Read(domain.Extracted{Companies: []domain.Named{{Name: "Tribelt"}}, Correctness: domain.CorrectnessPartly, WrongClaims: []string{"x"}}, nil, directory())
	if !named.NamesTribelt || named.CitesTribelt || named.Correctness != domain.CorrectnessPartly {
		t.Errorf("an answer naming Tribelt = %+v", named)
	}
}

func answer(provider string, scope domain.Scope, mode domain.Mode, x *domain.Extraction) domain.Answer {
	return domain.Answer{
		ID: uuid.New(), Prompt: domain.Prompt{ID: "eyelink-supplier-" + string(scope), Family: domain.FamilyEyelink, Scope: scope},
		Provider: provider, Model: provider + "/m", Mode: mode, Extraction: x,
		Reply: domain.Reply{InputTokens: 10, OutputTokens: 5, Cost: 100},
	}
}

func TestCellsMentionsAndWrongClaims(t *testing.T) {
	tribelt := &domain.Extraction{NamesTribelt: true, CitesTribelt: true, Companies: []domain.Company{{CompetitorID: "tribelt", Position: 1}, {CompetitorID: "esfo", Position: 2}, {CompetitorID: "esfo", Position: 3}}, Correctness: domain.CorrectnessWrong, WrongClaims: []string{"Tribelt is in Belgium."}, Cost: 10, InputTokens: 3, OutputTokens: 2}
	others := &domain.Extraction{Companies: []domain.Company{{CompetitorID: "esfo", Position: 1}, {Name: "Unknown", Position: 2}}}
	failedRead := answer("google", domain.ScopeTwente, domain.ModeKnowledge, &domain.Extraction{Failed: true})
	noAnswer := answer("google", domain.ScopeTwente, domain.ModeKnowledge, nil)
	noAnswer.Failure = "rate limited"
	late := answer("openai", domain.ScopeGermany, domain.ModeWeb, tribelt)
	late.At = time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	answers := []domain.Answer{
		answer("openai", domain.ScopeTwente, domain.ModeWeb, tribelt),
		answer("openai", domain.ScopeTwente, domain.ModeKnowledge, others),
		answer("google", domain.ScopeTwente, domain.ModeKnowledge, others),
		late, failedRead, noAnswer,
	}
	cells := domain.Cells(answers)
	want := []domain.Cell{
		{Provider: "google", Scope: domain.ScopeTwente, Family: domain.FamilyEyelink, Mode: domain.ModeKnowledge, Answers: 1},
		{Provider: "openai", Scope: domain.ScopeTwente, Family: domain.FamilyEyelink, Mode: domain.ModeKnowledge, Answers: 1},
		{Provider: "openai", Scope: domain.ScopeTwente, Family: domain.FamilyEyelink, Mode: domain.ModeWeb, Answers: 1, Named: 1, Cited: 1},
		{Provider: "openai", Scope: domain.ScopeGermany, Family: domain.FamilyEyelink, Mode: domain.ModeWeb, Answers: 1, Named: 1, Cited: 1},
	}
	if !slices.Equal(cells, want) {
		t.Errorf("cells = %+v", cells)
	}
	mentions := domain.Mentions(answers)
	if len(mentions) != 2 || mentions[0].CompetitorID != "esfo" || mentions[0].Answers != 4 || mentions[0].First != 2 || mentions[0].AveragePosition() != 1.5 {
		t.Errorf("mentions = %+v", mentions)
	}
	if mentions[1].CompetitorID != "tribelt" || (domain.Mention{}).AveragePosition() != 0 {
		t.Errorf("mentions[1] = %+v", mentions[1])
	}
	// The reading names Tribelt in an answer whose names the list did not match to it: Tribelt's
	// mentions count it, as the cells do.
	unmatched := answer("google", domain.ScopeGermany, domain.ModeKnowledge, &domain.Extraction{NamesTribelt: true, Companies: []domain.Company{{CompetitorID: "esfo", Position: 1}}})
	withTribelt := domain.WithTribelt(domain.Mentions(append(slices.Clone(answers), unmatched)), append(slices.Clone(answers), unmatched), directory())
	if withTribelt[0].CompetitorID != "esfo" || withTribelt[1].CompetitorID != "tribelt" || withTribelt[1].Answers != 3 || withTribelt[1].AveragePosition() != 1 {
		t.Errorf("with Tribelt = %+v", withTribelt)
	}
	onlyRead := domain.WithTribelt(nil, []domain.Answer{unmatched}, directory())
	if len(onlyRead) != 1 || onlyRead[0].Answers != 1 || onlyRead[0].AveragePosition() != 0 {
		t.Errorf("named by the reading alone = %+v", onlyRead)
	}
	if got := domain.WithTribelt(mentions[:1], answers[1:3], directory()); len(got) != 1 {
		t.Errorf("no answer names Tribelt, yet %+v", got)
	}
	if got := domain.WithTribelt(nil, []domain.Answer{unmatched}, directory()[1:]); len(got) != 0 {
		t.Errorf("no Tribelt on the list, yet %+v", got)
	}
	claims := domain.WrongClaims(answers)
	if len(claims) != 2 || !claims[0].At.Equal(late.At) || claims[0].Claim != "Tribelt is in Belgium." || claims[0].Correctness != domain.CorrectnessWrong {
		t.Errorf("claims = %+v", claims)
	}
	p := domain.Progressed(append(answers, answer("x-ai", domain.ScopeDutch, domain.ModeWeb, nil)), []domain.ModelChoice{{Provider: "openai"}, {Provider: "google"}, {Provider: "deepseek"}})
	if p.Answered != 6 || p.Failed != 1 || p.Cost != 7*100+20 || p.InputTokens != 7*10+6 || len(p.ByProvider) != 4 {
		t.Errorf("progress = %+v", p)
	}
	if g := p.ByProvider[1]; g.Provider != "google" || g.Answered != 2 || g.Failed != 1 || !slices.Equal(g.Models, []string{"google/m"}) {
		t.Errorf("google = %+v", g)
	}
	if p.ByProvider[2].Answered != 0 || p.ByProvider[3].Provider != "x-ai" {
		t.Errorf("by provider = %+v", p.ByProvider)
	}
}

func TestCandidatesGatherUnmatchedNamesAcrossRuns(t *testing.T) {
	day := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	sightings := []domain.Sighting{
		{Name: "Ashworth Bros", URL: "https://ashworth.com", Provider: "openai", Model: "openai/gpt-oss-20b", At: day},
		{Name: "Ashworth Bros.", Provider: "google", Model: "google/gemma:free", At: day.Add(time.Hour)},
		{Name: "Ashworth Bros", Provider: "openai", At: day},
		{Name: "ESFO", Provider: "openai", At: day},          // on the list
		{Name: "Distributor X", Provider: "openai", At: day}, // dismissed
		{Name: "Newly Added", Provider: "openai", At: day},   // added, and on the list now
		{Name: "GmbH", Provider: "openai", At: day},          // no name at all
	}
	dir := append(directory(), domain.DirectoryCompany{ID: "newly-added", Name: "Newly Added", Role: domain.RoleCompetitor})
	decisions := []domain.CandidateDecision{
		{ID: domain.CandidateID(domain.NameKey("Distributor X")), Status: domain.CandidateDismissed, By: "joris"},
		{ID: domain.CandidateID(domain.NameKey("Newly Added")), Status: domain.CandidateAdded, CompetitorID: "newly-added"},
	}
	got := domain.Candidates(sightings, decisions, dir, nil)
	if len(got) != 3 {
		t.Fatalf("candidates = %+v", got)
	}
	ash := got[0]
	if ash.Name != "Ashworth Bros" || ash.Answers != 3 || !slices.Equal(ash.Providers, []string{"google", "openai"}) ||
		!slices.Equal(ash.Models, []string{"google/gemma:free", "openai/gpt-oss-20b"}) ||
		!slices.Equal(ash.Addresses, []string{"https://ashworth.com"}) || !ash.LastSeen.Equal(day.Add(time.Hour)) || ash.Status() != domain.CandidatePending {
		t.Errorf("ashworth = %+v", ash)
	}
	if got[1].Name != "Distributor X" || got[1].Status() != domain.CandidateDismissed || got[2].Status() != domain.CandidateAdded {
		t.Errorf("decided = %+v, %+v", got[1], got[2])
	}
	if len(domain.CandidateID("x")) != 16 {
		t.Error("id length")
	}
}

func TestSettingsRulesAndTheSchedule(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC) // a Monday
	s := domain.DefaultSettings(now)
	if s.MonthlyBudget != 10_000_000 || s.Cadence != domain.CadenceMonthly || !s.NextDue.Equal(time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC)) ||
		len(s.Providers) != 5 || s.FactSheet.Version != 1 || !strings.Contains(s.FactSheet.Text, "Haaksbergen") || s.CredentialID != "" {
		t.Errorf("defaults = %+v", s)
	}
	if got := domain.NextDue(now, domain.CadenceWeekly); !got.Equal(time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC)) {
		t.Errorf("weekly after Monday noon = %v", got)
	}
	if got := domain.NextDue(time.Date(2026, 10, 26, 5, 0, 0, 0, time.UTC), domain.CadenceWeekly); !got.Equal(time.Date(2026, 10, 26, 6, 0, 0, 0, time.UTC)) {
		t.Errorf("weekly before Monday six = %v", got)
	}
	if got := domain.NextDue(time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC), domain.CadenceMonthly); !got.Equal(time.Date(2027, 1, 1, 6, 0, 0, 0, time.UTC)) {
		t.Errorf("monthly at year end = %v", got)
	}
	if !domain.MonthStart(now).Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Error("month start")
	}
	if got, err := domain.CheckProviders([]string{" OpenAI", "openai", "meta-llama"}); err != nil || !slices.Equal(got, []string{"openai", "meta-llama"}) {
		t.Errorf("providers = %v, %v", got, err)
	}
	for _, bad := range [][]string{nil, {"open ai"}, {strings.Repeat("a", 41)}, make([]string, 0)} {
		if _, err := domain.CheckProviders(bad); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("providers %v accepted", bad)
		}
	}
	many := make([]string, 21)
	for i := range many {
		many[i] = "p" + string(rune('a'+i))
	}
	if _, err := domain.CheckProviders(many); !errors.Is(err, domain.ErrInvalid) {
		t.Error("21 providers accepted")
	}
	if b, err := domain.CheckBudget(domain.MicrosOf(25)); err != nil || b != 25_000_000 {
		t.Errorf("budget = %v, %v", b, err)
	}
	for _, bad := range []domain.Micros{0, -1, domain.MaxBudget + 1} {
		if _, err := domain.CheckBudget(bad); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("budget %d accepted", bad)
		}
	}
	if c, err := domain.CheckCadence("weekly"); err != nil || c != domain.CadenceWeekly {
		t.Errorf("cadence = %v, %v", c, err)
	}
	if _, err := domain.CheckCadence("daily"); !errors.Is(err, domain.ErrInvalid) {
		t.Error("daily accepted")
	}
	if text, err := domain.CheckFactSheet("  Tribelt makes metal belts in Twente.  "); err != nil || text != "Tribelt makes metal belts in Twente." {
		t.Errorf("fact sheet = %q, %v", text, err)
	}
	for _, bad := range []string{"short", strings.Repeat("x", domain.MaxFactSheet+1)} {
		if _, err := domain.CheckFactSheet(bad); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("fact sheet of %d accepted", len(bad))
		}
	}
}

func TestCitedAndNamedCompaniesAreOnTheListEachOnce(t *testing.T) {
	cited := domain.CitedCompanies([]string{"https://shop.tribelt.nl/en", "https://www.martens-belts.com/", "tribelt.nl", "https://example.org"}, directory())
	if strings.Join(cited, ",") != "tribelt,martens" {
		t.Errorf("cited = %v", cited)
	}
	x := domain.Extraction{Companies: []domain.Company{{CompetitorID: "esfo"}, {Name: "Unknown"}, {CompetitorID: "tribelt"}, {CompetitorID: "esfo"}}}
	if named := x.NamedCompanies(); strings.Join(named, ",") != "esfo,tribelt" {
		t.Errorf("named = %v", named)
	}
}

func TestUnansweredAndWhatAskingAgainCosts(t *testing.T) {
	set := domain.CurrentPromptSet()
	models := []domain.ModelChoice{{Provider: "openai"}, {Provider: "google"}}
	p := set.Prompts[0]
	answers := []domain.Answer{
		{Prompt: p, Provider: "openai", Mode: domain.ModeKnowledge, Reply: domain.Reply{Cost: 100}, Extraction: &domain.Extraction{Cost: 20}},
		{Prompt: p, Provider: "openai", Mode: domain.ModeWeb, Reply: domain.Reply{Cost: 300}},
		{Prompt: p, Provider: "google", Mode: domain.ModeKnowledge, Failure: "Rate limited.", Reply: domain.Reply{Cost: 5}},
	}
	left := domain.Unanswered(set, models, answers)
	if len(left) != len(domain.Tasks(set, models))-2 {
		t.Fatalf("unanswered = %d", len(left))
	}
	google := []domain.Task{{Prompt: p, Provider: "google", Mode: domain.ModeKnowledge}}
	openai := []domain.Task{{Prompt: p, Provider: "openai", Mode: domain.ModeWeb}}
	// Google has no answer to go by: OpenAI's in the same mode, 100, and reading it, 7.
	if got := domain.Estimate(answers, google, 7); got != 107 {
		t.Errorf("google estimate = %d", got)
	}
	// OpenAI's own web answer, its search included: 300.
	if got := domain.Estimate(answers, append(openai, google...), 7); got != 300+7+107 {
		t.Errorf("both = %d", got)
	}
	if got := domain.Estimate(nil, google, 7); got != 7 {
		t.Errorf("no answers = %d", got)
	}
	costs := domain.AnswerCosts(answers, models, 7)
	if len(costs) != 4 || costs[0] != (domain.ModeCost{Provider: "openai", Mode: domain.ModeKnowledge, Cost: 107}) ||
		costs[1] != (domain.ModeCost{Provider: "openai", Mode: domain.ModeWeb, Cost: 307}) || costs[3].Cost != 307 {
		t.Errorf("costs = %+v", costs)
	}
}

func TestUnreadAreTheAnswersWhoseReadingFailed(t *testing.T) {
	answers := []domain.Answer{
		{Provider: "openai", Extraction: &domain.Extraction{Failed: true}},
		{Provider: "openai", Extraction: &domain.Extraction{}},
		{Provider: "google"},
		{Provider: "google", Failure: "Rate limited.", Extraction: &domain.Extraction{Failed: true}},
	}
	if got := domain.Unread(answers); len(got) != 1 || got[0].Provider != "openai" {
		t.Errorf("unread = %+v", got)
	}
}

func TestReadingIsPricedAtTheReaderThatReads(t *testing.T) {
	answers := []domain.Answer{
		{Extraction: &domain.Extraction{InputTokens: 1000, OutputTokens: 200, Cost: 90}},
		{Extraction: &domain.Extraction{InputTokens: 3000, OutputTokens: 400, Cost: 110}},
		{Failure: "Rate limited."},
	}
	haiku := domain.HaikuPrice()
	// 2,000 in at $0.10 and 300 out at $0.50 per million: 200 + 150 µ$.
	if got := domain.ReadingCost(answers, &haiku); got != 350 {
		t.Errorf("Haiku's reading = %d", got)
	}
	// Before any reading: typical tokens, 1,500 in and 250 out.
	if got := domain.ReadingCost(nil, &haiku); got != 150+125 {
		t.Errorf("Haiku's reading before any = %d", got)
	}
	// Without a price: what the readings cost on average.
	if got, none := domain.ReadingCost(answers, nil), domain.ReadingCost(nil, nil); got != 100 || none != 0 {
		t.Errorf("readings by their own cost = %d, without any = %d", got, none)
	}
	catalogue := []domain.CatalogueModel{{ID: domain.ExtractorModel, PromptPrice: 0.0000001, CompletionPrice: 0.0000004}}
	if p, ok := domain.CataloguePrice(catalogue, domain.ExtractorModel); !ok || p != (domain.TokenPrice{Input: 0.0000001, Output: 0.0000004}) {
		t.Errorf("the extractor's price = %+v, %v", p, ok)
	}
	if _, ok := domain.CataloguePrice(catalogue, "x/unknown"); ok {
		t.Error("an unknown model has a price")
	}
}

func TestAWebSearchCostsItsFee(t *testing.T) {
	if got := domain.WebSearchFee(domain.CatalogueModel{ID: "openai/gpt-5-mini", WebSearchPrice: 0.01}); got != 10_000 {
		t.Errorf("a model's own search = %d", got)
	}
	// OpenRouter searches for a model that does not: five results at $4 per 1,000.
	if got := domain.WebSearchFee(domain.CatalogueModel{ID: "deepseek/deepseek-chat"}); got != 20_000 {
		t.Errorf("OpenRouter's search = %d", got)
	}
}

func TestPricesFromARun(t *testing.T) {
	p := domain.CurrentPromptSet().Prompts[0]
	answers := []domain.Answer{
		{Prompt: p, Provider: "openai", Mode: domain.ModeKnowledge, Reply: domain.Reply{Cost: 100}, Extraction: &domain.Extraction{Cost: 10}},
		{Prompt: p, Provider: "openai", Mode: domain.ModeKnowledge, Reply: domain.Reply{Cost: 300}, Extraction: &domain.Extraction{Cost: 30}},
		{Prompt: p, Provider: "openai", Mode: domain.ModeWeb, Reply: domain.Reply{Cost: 1000}, Extraction: &domain.Extraction{Cost: 20}},
		{Prompt: p, Provider: "google", Mode: domain.ModeWeb, Failure: "Rate limited.", Reply: domain.Reply{Cost: 5}},
	}
	models := []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}, {Provider: "google", Model: "google/gemma:free", Fallback: "google/gemini"}}
	got := domain.PricesFromRun(answers, models, 20, domain.CadenceWeekly, 10_000_000, nil)
	if got.Basis != domain.BasisRun || got.Reading != 20 || len(got.Providers) != 2 {
		t.Fatalf("prices = %+v", got)
	}
	openai, google := got.Providers[0], got.Providers[1]
	if openai.Knowledge != 200 || openai.Web != 1000 || openai.PerRun != 24_000 || google.Knowledge != 0 || google.Fallback != "google/gemini" {
		t.Errorf("providers = %+v", got.Providers)
	}
	// Each provider's 40 answers are read too: 2 × 40 × 20.
	if got.Run != 24_000+1600 || got.RunsPerMonth != 52.0/12.0 || got.Month != domain.MicrosOf(25_600.0/1e6*52/12) || got.OverBudget() {
		t.Errorf("run %d, month %d, over %v", got.Run, got.Month, got.OverBudget())
	}
	if domain.PricesFromRun(nil, models, 20, domain.CadenceMonthly, 1, nil).Reading != 0 {
		t.Error("no answers, no reading cost")
	}
	// Haiku reads the next run: its price, not what the run's readings cost.
	haiku := domain.HaikuPrice()
	if got := domain.PricesFromRun(answers, models, 20, domain.CadenceWeekly, 10_000_000, &haiku); got.Reading != 0 {
		t.Errorf("reading by Haiku without tokens = %d", got.Reading)
	}
}

func TestPricesBeforeAnyRun(t *testing.T) {
	catalogue := []domain.CatalogueModel{
		{ID: "openai/gpt-oss-20b", PromptPrice: 0.0000001, CompletionPrice: 0.0000004},
		{ID: domain.ExtractorModel, PromptPrice: 0.0000001, CompletionPrice: 0.0000004},
	}
	models := []domain.ModelChoice{{Provider: "openai", Model: "openai/gpt-oss-20b"}, {Provider: "x", Model: "x/unknown"}}
	reader, _ := domain.CataloguePrice(catalogue, domain.ExtractorModel)
	got := domain.PricesFromCatalogue(catalogue, models, 20, domain.CadenceMonthly, 1, &reader)
	// 300 × 0.1 µ$ + 900 × 0.4 µ$ = 390 µ$ an answer, and OpenRouter's search with the web;
	// reading 1500 × 0.1 + 250 × 0.4 = 250 µ$.
	if got.Basis != domain.BasisCatalogue || got.Providers[0].Knowledge != 390 || got.Providers[0].Web != 390+20_000 ||
		got.Providers[1].Knowledge != 0 || got.Providers[1].Web != 0 || got.Reading != 250 {
		t.Errorf("estimate = %+v", got)
	}
	// Haiku reads: 1500 × 0.1 + 250 × 0.5 = 275 µ$. Without a reader's price, reading is not priced.
	haiku := domain.HaikuPrice()
	if r := domain.PricesFromCatalogue(catalogue, models, 20, domain.CadenceMonthly, 1, &haiku).Reading; r != 275 {
		t.Errorf("reading by Haiku = %d", r)
	}
	if r := domain.PricesFromCatalogue(catalogue, models, 20, domain.CadenceMonthly, 1, nil).Reading; r != 0 {
		t.Errorf("reading without a price = %d", r)
	}
	if got.RunsPerMonth != 1 || got.Month != got.Run || !got.OverBudget() {
		t.Errorf("month = %+v", got)
	}
	none := domain.NoPrices(domain.CadenceMonthly, 10_000_000)
	if none.Basis != domain.BasisNone || none.Run != 0 || none.OverBudget() {
		t.Errorf("no prices = %+v", none)
	}
}

func TestAReopenedRunRunsAgain(t *testing.T) {
	r := domain.Run{Status: domain.RunStopped, FinishedAt: time.Now(), Failure: "Budget."}.Reopened()
	if r.Status != domain.RunRunning || !r.FinishedAt.IsZero() || r.Failure != "" {
		t.Errorf("reopened = %+v", r)
	}
}

func TestPricesFollowTheProvidersSetNow(t *testing.T) {
	models := []domain.ModelChoice{{Provider: "openai", Model: "openai/a"}, {Provider: "google", Model: "google/b"}}
	kept, missing := domain.Asked(models, []string{"openai", "deepseek"})
	if len(kept) != 1 || kept[0].Provider != "openai" || len(missing) != 1 || missing[0] != "deepseek" {
		t.Fatalf("kept %+v, missing %v", kept, missing)
	}
	p := domain.PricesFromRun([]domain.Answer{{Provider: "openai", Mode: domain.ModeWeb, Reply: domain.Reply{Cost: 100}}}, kept, 20, domain.CadenceMonthly, 10_000_000, nil)
	more := p.With([]domain.ProviderPrice{{Provider: "deepseek", Knowledge: 50, Web: 50}}, 20, domain.CadenceMonthly)
	if len(more.Providers) != 2 || more.Run != p.Run+2000 || more.Basis != domain.BasisRun || len(p.Providers) != 1 {
		t.Errorf("with deepseek = %+v", more)
	}
}

func TestQuestionsOfARunAreItsSetOrWhatItsAnswersKept(t *testing.T) {
	current := domain.Run{PromptSetVersion: domain.PromptSetVersion}
	if set := domain.QuestionsOf(current, nil); set.Version != domain.PromptSetVersion || len(set.Prompts) != 20 {
		t.Fatalf("the current set = %+v", set)
	}
	supplier := domain.Prompt{ID: "eyelink-supplier-twente", Family: domain.FamilyEyelink, Kind: domain.KindSupplier, Scope: domain.ScopeTwente, Language: "nl", Text: "Wie levert oogjesbanden?"}
	problem := domain.Prompt{ID: "eyelink-problem-germany", Family: domain.FamilyEyelink, Kind: domain.KindProblem, Scope: domain.ScopeGermany, Language: "de", Text: "Unser Band reißt."}
	models := []domain.ModelChoice{{Provider: "openai"}, {Provider: "google"}}
	earlier := domain.Run{PromptSetVersion: 1, Models: models, Tasks: 3 * 2 * 2}
	answers := []domain.Answer{
		{Prompt: supplier, Provider: "openai", Mode: domain.ModeWeb},
		{Prompt: problem, Provider: "google", Mode: domain.ModeKnowledge, Failure: "Rate limited."},
		{Prompt: supplier, Provider: "google", Mode: domain.ModeWeb},
	}
	set := domain.QuestionsOf(earlier, answers)
	if set.Version != 1 || len(set.Prompts) != 2 || set.Prompts[0] != supplier || set.Prompts[1] != problem {
		t.Fatalf("an earlier set = %+v", set)
	}
	// Three questions were asked; one of them no answer kept.
	if n := domain.Unknown(earlier, set); n != 4 {
		t.Errorf("unknown = %d", n)
	}
	if n := domain.Unknown(domain.Run{Models: models}, set); n != 0 {
		t.Errorf("unknown of a run asked as a whole = %d", n)
	}
	if missing := domain.Unanswered(set, models, answers); len(missing) != 6 {
		t.Errorf("unanswered = %d", len(missing))
	}
}
