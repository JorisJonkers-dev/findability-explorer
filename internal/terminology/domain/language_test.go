package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// The review pairs a phrase only with terms in its own language: each language's use of a phrase
// is a candidate of its own, the map holds a phrase only in the language of its term (American
// English is English), and a closest term in another language is dropped.
func TestCandidatesKeepLanguagesApart(t *testing.T) {
	terms := []domain.Term{
		{Text: "Transportband", Language: domain.LanguageDe},
		{Text: "conveyor belt", Language: "en-US"},
	}
	observed := []domain.Observation{
		obs("s1", "alpha", "transportband", "nl", 9, 1, "broad", "all", "Transportband", at),
		obs("s1", "alpha", "transportband", "DE", 8, 1, "broad", "all", "Transportband", at),
		obs("s1", "alpha", "conveyor belts", "en", 7, 1, "broad", "all", "conveyor belt", at),
		obs("s1", "alpha", "gurtband", "de", 6, 1, "broad", "all", "Gurtband", at), // a term the map no longer has
	}
	got := domain.Candidates(observed, nil, terms)
	if len(got) != 2 {
		t.Fatalf("candidates %+v", got)
	}
	if c := got[0]; c.Phrase != "transportband" || c.Language != "nl" || c.Closest != "" || c.ID != domain.CandidateID("Transportband", "nl") {
		t.Errorf("Dutch transportband = %+v", c)
	}
	if c := got[1]; c.Phrase != "gurtband" || c.Language != "de" || c.Closest != "Gurtband" {
		t.Errorf("gurtband = %+v", c)
	}
	if domain.CandidateID("transportband", "nl") == domain.CandidateID("transportband", "de") ||
		domain.CandidateID("belt", "en") != domain.CandidateID("belt", "en-US") {
		t.Error("a candidate's id does not keep to its language")
	}
}

// Whether the map holds a phrase is decided by the scans' own spelling rule, in the phrase's
// language, whatever the scan stored: a Dutch phrase a scan stored as in the map through the
// German term comes back as a candidate, and every word of a phrase may be plural.
func TestTheMapHoldsAPhraseByTheScansSpellingRule(t *testing.T) {
	terms := []domain.Term{
		{Text: "Gliederband", Language: domain.LanguageDe},
		{Text: "belt conveyor (s) / nid d'abeilles", Language: domain.LanguageEn},
	}
	observed := []domain.Observation{
		obs("s1", "alpha", "gliederband", "nl", 3, 1, "product", "eyelink", "Gliederband", at), // stored in the map
		obs("s1", "alpha", "gliederbander", "de", 3, 1, "product", "eyelink", "", at),
		obs("s1", "alpha", "belts conveyors", "en", 2, 1, "broad", "", "", at),
		obs("s1", "alpha", "nid d abeilles", "en", 1, 1, "product", "", "", at),
	}
	got := domain.Candidates(observed, nil, terms)
	phrases := make([]string, 0, len(got))
	for _, c := range got {
		phrases = append(phrases, c.Phrase+" "+c.Language)
	}
	if len(phrases) != 2 || phrases[0] != "gliederband nl" || phrases[1] != "gliederbander de" {
		t.Fatalf("candidates %q", phrases)
	}
}

// Decisions made on a phrase before candidates were kept apart by language still count: a
// rejection in every language, an acceptance only in its term's language. One made on the
// candidate's own id comes first.
func TestOldDecisionsHoldWhereTheyApply(t *testing.T) {
	observed := []domain.Observation{
		obs("s1", "alpha", "spiral belt", "nl", 5, 1, "product", "", "", at),
		obs("s1", "alpha", "spiral belt", "de", 4, 1, "product", "", "", at),
		obs("s1", "alpha", "rod belt", "nl", 3, 1, "product", "", "", at),
		obs("s1", "alpha", "rod belt", "en", 2, 1, "product", "", "", at),
		obs("s1", "alpha", "wire belt", "en", 1, 1, "product", "", "", at),
	}
	accepted := domain.Term{Text: "spiral belt", Language: domain.LanguageDe}
	wire := domain.Term{Text: "wire belt", Language: domain.LanguageEn}
	decisions := []domain.Decision{
		{CandidateID: domain.PhraseID("spiral belt"), Status: domain.StatusAccepted, Term: &accepted},
		{CandidateID: domain.PhraseID("rod belt"), Status: domain.StatusRejected},
		{CandidateID: domain.PhraseID("wire belt"), Status: domain.StatusAccepted, Term: &wire},
		{CandidateID: domain.CandidateID("wire belt", "en"), Status: domain.StatusRejected},
	}
	got := map[string]domain.Status{}
	for _, c := range domain.Candidates(observed, decisions, []domain.Term{accepted}) {
		got[c.Phrase+" "+c.Language] = c.Status
	}
	want := map[string]domain.Status{
		"spiral belt nl": domain.StatusPending, "spiral belt de": domain.StatusAccepted,
		"rod belt nl": domain.StatusRejected, "rod belt en": domain.StatusRejected,
		"wire belt en": domain.StatusRejected,
	}
	if len(got) != len(want) {
		t.Fatalf("statuses = %v", got)
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s = %s, want %s", k, got[k], w)
		}
	}
}

// A candidate becomes a term only in the language it was found in; American English is English.
func TestACandidateBecomesATermOnlyInItsLanguage(t *testing.T) {
	c := domain.Candidate{Phrase: "transportband", Language: "nl", Source: "https://belts.test/"}
	a := domain.Acceptance{Text: "transportband", Language: "de", Family: "all", Tier: "broad"}
	if _, err := c.NewTerm(a, nil); !errors.Is(err, domain.ErrOtherLanguage) || !errors.Is(err, domain.ErrUnknownValue) {
		t.Errorf("a Dutch candidate as a German term: %v", err)
	}
	a.Language = "nl"
	if term, err := c.NewTerm(a, nil); err != nil || term.Language != domain.LanguageNl || term.Source != c.Source {
		t.Errorf("NewTerm = %+v, %v", term, err)
	}
	en := domain.Candidate{Phrase: "conveyor belt", Language: "en"}
	if _, err := en.NewTerm(domain.Acceptance{Text: "conveyor belt", Language: "en-US", Family: "all", Tier: "broad"}, nil); err != nil {
		t.Errorf("an English candidate as an American English term: %v", err)
	}
}

// The model's verdicts keep to the candidate's language too: a sure acceptance in another
// language decides nothing, and a verdict given on the phrase alone counts only where it applies.
func TestVerdictsKeepToTheCandidatesLanguage(t *testing.T) {
	at := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	nl := domain.Candidate{ID: domain.CandidateID("spiraalband", "nl"), Phrase: "spiraalband", Language: "nl", Status: domain.StatusPending}
	de := domain.Candidate{ID: domain.CandidateID("spiraalband", "de"), Phrase: "spiraalband", Language: "de", Status: domain.StatusPending}
	german := domain.Acceptance{Text: "spiraalband", Language: "de", Family: "open_spiral", Tier: "product"}
	if _, ok := (domain.Verdict{Accept: true, Sure: true, At: at, Term: german}).Decision(nl, nil); ok {
		t.Error("a sure verdict made a Dutch candidate a German term")
	}

	old := []domain.Verdict{{CandidateID: domain.PhraseID("spiraalband"), Accept: true, Term: german}}
	got := domain.WithVerdicts([]domain.Candidate{nl, de}, old, nil)
	if got[0].Verdict != nil || got[1].Verdict == nil || got[1].Suggestion == nil {
		t.Errorf("an old acceptance in German = %+v", got)
	}
	nl.Verdict, de.Verdict, de.Suggestion = nil, nil, nil
	rejected := []domain.Verdict{{CandidateID: domain.PhraseID("spiraalband"), Reason: "A brand."}}
	got = domain.WithVerdicts([]domain.Candidate{nl, de}, rejected, nil)
	if got[0].Verdict == nil || got[1].Verdict == nil {
		t.Errorf("an old rejection = %+v", got)
	}
	nl.Verdict = nil
	other := []domain.Verdict{{CandidateID: nl.ID, Accept: true, Term: german}}
	got = domain.WithVerdicts([]domain.Candidate{nl}, other, nil)
	if got[0].Verdict == nil || got[0].Suggestion != nil {
		t.Errorf("a suggestion in another language = %+v", got[0])
	}
}

// A phrase a scan stored in no language takes the one its own words tell when the review reads
// it; one they do not tell stays in no language, a candidate all the same, never in the map.
func TestAPhraseInNoLanguageTakesTheOneItsWordsTell(t *testing.T) {
	terms := []domain.Term{{Text: "metalen transportband", Language: domain.LanguageNl}}
	observed := []domain.Observation{
		obs("s1", "alpha", "draadogenbanden", "und", 9, 1, "product", "eyelink", "", at),
		obs("s2", "beta", "draadogenbanden", "nl", 8, 1, "product", "eyelink", "", at),
		obs("s1", "alpha", "transportbänder", "", 7, 1, "broad", "all", "", at),
		obs("s1", "alpha", "1.4301", "und", 6, 1, "spec", "spec_all", "", at),
		obs("s1", "alpha", "metalen transportband", "und", 5, 1, "broad", "all", "", at), // in the map in Dutch
	}
	got := domain.Candidates(observed, nil, terms)
	byPhrase := map[string]domain.Candidate{}
	for _, c := range got {
		byPhrase[c.Phrase] = c
	}
	if len(got) != 3 {
		t.Fatalf("candidates %+v", got)
	}
	if c := byPhrase["draadogenbanden"]; c.Language != "nl" || len(c.Companies) != 2 || c.ID != domain.CandidateID("draadogenbanden", "nl") {
		t.Errorf("draadogenbanden = %+v, want one Dutch candidate of both companies", c)
	}
	if c := byPhrase["transportbänder"]; c.Language != "de" {
		t.Errorf("transportbänder = %+v, want German", c)
	}
	if c := byPhrase["1.4301"]; c.Language != "und" || c.Languages[0] != "und" || c.ID != domain.CandidateID("1.4301", "und") {
		t.Errorf("1.4301 = %+v, want a candidate in no language", c)
	}
}

// A candidate in no language joins the map in the language a person sets, any focus one; the
// language model's acceptance of it waits for that person, its rejection does not.
func TestAPersonSetsTheLanguageOfACandidateInNoLanguage(t *testing.T) {
	c := domain.Candidate{
		ID: domain.CandidateID("1.4301", "und"), Phrase: "1.4301", Language: "und", Source: "https://a.test/",
		Companies: []domain.CandidateCompany{{ID: "a", Name: "A"}},
	}
	a := domain.Acceptance{Text: "1.4301", Language: "de", Family: "spec_all", Tier: "spec"}
	term, err := c.NewTerm(a, nil)
	if err != nil || term.Language != domain.LanguageDe {
		t.Fatalf("NewTerm = %+v, %v", term, err)
	}
	if _, err := c.NewTerm(domain.Acceptance{Text: "1.4301", Language: "fr", Family: "spec_all", Tier: "spec"}, nil); !errors.Is(err, domain.ErrUnknownValue) {
		t.Errorf("a French term = %v, want ErrUnknownValue", err)
	}
	accept := domain.Verdict{CandidateID: c.ID, Accept: true, Sure: true, Term: a, At: at}
	if d, ok := accept.Decision(c, nil); ok {
		t.Errorf("the model accepted a candidate in no language: %+v", d)
	}
	reject := domain.Verdict{CandidateID: c.ID, Sure: true, At: at}
	if d, ok := reject.Decision(c, nil); !ok || d.Status != domain.StatusRejected {
		t.Errorf("rejection = %+v, %v", d, ok)
	}
	// The model's acceptance still suggests the term to the person.
	got := domain.WithVerdicts([]domain.Candidate{{ID: c.ID, Phrase: c.Phrase, Language: "und", Status: domain.StatusPending}}, []domain.Verdict{accept}, nil)
	if got[0].Suggestion == nil || got[0].Suggestion.Language != "de" {
		t.Errorf("suggestion = %+v", got[0].Suggestion)
	}
}

// A decision or verdict made while a phrase was in no language holds once its words tell one,
// where it applies: a rejection in any language, an acceptance in its term's.
func TestADecisionInNoLanguageHoldsOnceTheLanguageIsTold(t *testing.T) {
	undID := domain.CandidateID("draadogenbanden", "und")
	observed := []domain.Observation{
		obs("s1", "alpha", "draadogenbanden", "nl", 9, 1, "product", "eyelink", "", at),
		obs("s1", "alpha", "conveyor belts", "en", 9, 1, "broad", "all", "", at),
	}
	decisions := []domain.Decision{
		{CandidateID: undID, Phrase: "draadogenbanden", Status: domain.StatusAccepted, By: "j", At: at, Term: &domain.Term{Text: "draadogenband", Language: domain.LanguageNl}},
		{CandidateID: domain.CandidateID("conveyor belts", "und"), Phrase: "conveyor belts", Status: domain.StatusAccepted, By: "j", At: at, Term: &domain.Term{Text: "conveyor belts", Language: domain.LanguageDe}},
	}
	got := domain.Candidates(observed, decisions, nil)
	byPhrase := map[string]domain.Candidate{}
	for _, c := range got {
		byPhrase[c.Phrase] = c
	}
	if c := byPhrase["draadogenbanden"]; c.Status != domain.StatusAccepted || c.Decision.CandidateID != undID {
		t.Errorf("draadogenbanden = %+v, want accepted by the decision made in no language", c)
	}
	if c := byPhrase["conveyor belts"]; c.Status != domain.StatusPending {
		t.Errorf("conveyor belts = %+v, want waiting: its acceptance made a German term", c)
	}
	verdicts := []domain.Verdict{{CandidateID: undID, Phrase: "draadogenbanden", Sure: true, Reason: "no"}}
	with := domain.WithVerdicts([]domain.Candidate{{ID: domain.CandidateID("draadogenbanden", "nl"), Phrase: "draadogenbanden", Language: "nl"}}, verdicts, nil)
	if with[0].Verdict == nil || with[0].Verdict.Reason != "no" {
		t.Errorf("verdict = %+v, want the one given in no language", with[0].Verdict)
	}
}
