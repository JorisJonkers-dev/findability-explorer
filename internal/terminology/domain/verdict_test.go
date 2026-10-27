package domain_test

import (
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

func waiting(phrase string, companies ...string) domain.Candidate {
	c := domain.Candidate{ID: domain.CandidateID(phrase), Phrase: phrase, Status: domain.StatusPending, Source: "https://www.twentebelt.com"}
	for _, name := range companies {
		c.Companies = append(c.Companies, domain.CandidateCompany{ID: name, Name: name})
	}
	return c
}

func TestOnlyWaitingCandidatesTheModelHasNotSeenAreReviewed(t *testing.T) {
	seen, fresh := waiting("spiral freezer belt"), waiting("wire mesh belt")
	decided := waiting("belts")
	decided.Status = domain.StatusRejected
	seen.Verdict = &domain.Verdict{CandidateID: seen.ID}
	got := domain.Unreviewed([]domain.Candidate{seen, decided, fresh})
	if len(got) != 1 || got[0].ID != fresh.ID {
		t.Errorf("Unreviewed = %+v", got)
	}
}

func TestEveryCandidateOfABatchGetsAVerdict(t *testing.T) {
	at := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	a, b := waiting("spiral freezer belt"), waiting("wire mesh belt")
	answered := []domain.Verdict{
		{CandidateID: b.ID, Accept: true, Sure: true, Reason: "A kind of belt."},
		{CandidateID: "ffffffffffffffff", Accept: true, Sure: true}, // not asked about
	}
	got := domain.Complete([]domain.Candidate{a, b}, answered, "claude-haiku-5-5", at)
	if len(got) != 2 {
		t.Fatalf("Complete = %+v", got)
	}
	if got[0].CandidateID != a.ID || got[0].Sure || got[0].Phrase != a.Phrase || got[0].Model != "claude-haiku-5-5" || !got[0].At.Equal(at) {
		t.Errorf("a candidate the model skipped = %+v", got[0])
	}
	if got[1].CandidateID != b.ID || !got[1].Sure || got[1].Phrase != b.Phrase || got[1].Model != "claude-haiku-5-5" {
		t.Errorf("an answered candidate = %+v", got[1])
	}
}

func TestASureVerdictIsADecisionByTheModel(t *testing.T) {
	at := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	c := waiting("spiral freezer belts", "Twentebelt", "Ashworth")
	accept := domain.Verdict{
		CandidateID: c.ID, Phrase: c.Phrase, Accept: true, Sure: true, At: at,
		Term: domain.Acceptance{Text: "spiral freezer belt", Language: "en", Family: "tri_flex_turn_curve", Tier: "application"},
	}
	d, ok := accept.Decision(c, nil)
	if !ok || d.Status != domain.StatusAccepted || d.By != domain.ModelReviewer || !d.At.Equal(at) || d.Phrase != c.Phrase {
		t.Fatalf("an accepted candidate = %+v, %v", d, ok)
	}
	if d.Term.Text != "spiral freezer belt" || d.Term.Origin != domain.OriginReview || len(d.Term.UsedBy) != 2 || d.Term.Source != c.Source {
		t.Errorf("its term = %+v", d.Term)
	}

	reject := domain.Verdict{CandidateID: c.ID, Phrase: c.Phrase, Sure: true, At: at}
	if d, ok := reject.Decision(c, nil); !ok || d.Status != domain.StatusRejected || d.Term != nil || d.By != domain.ModelReviewer {
		t.Errorf("a rejected candidate = %+v, %v", d, ok)
	}

	unsure := accept
	unsure.Sure = false
	if _, ok := unsure.Decision(c, nil); ok {
		t.Error("an unsure verdict decided")
	}
	// A term the map cannot take waits for a person.
	inMap := []domain.Term{{Text: "spiral freezer belt", Language: "en"}}
	if _, ok := accept.Decision(c, inMap); ok {
		t.Error("a term the map has decided")
	}
	bad := accept
	bad.Term.Family = "plastic"
	if _, ok := bad.Decision(c, nil); ok {
		t.Error("a term of no family decided")
	}
}

func TestCandidatesCarryTheModelsVerdictAndATermTheMapCouldTake(t *testing.T) {
	term := func(text string) domain.Acceptance {
		return domain.Acceptance{Text: text, Language: "en", Family: "tri_flex_turn_curve", Tier: "application"}
	}
	a, b, c, d := waiting("spiral freezer belt"), waiting("wire mesh belt"), waiting("spiral belts"), waiting("chain belt")
	d.Status = domain.StatusRejected
	verdicts := []domain.Verdict{
		{CandidateID: b.ID, Accept: true, Reason: "Unsure.", Term: term("wire mesh belt")},
		{CandidateID: c.ID, Accept: true, Term: term("spiral belt")}, // the map has it
		{CandidateID: d.ID, Accept: true, Term: term("chain belt")},  // decided already
	}
	inMap := []domain.Term{{Text: "spiral belt", Language: "en"}}
	got := domain.WithVerdicts([]domain.Candidate{a, b, c, d}, verdicts, inMap)
	if got[0].Verdict != nil || got[1].Verdict == nil || got[1].Verdict.Reason != "Unsure." {
		t.Errorf("verdicts = %+v", got)
	}
	if got[1].Suggestion == nil || got[1].Suggestion.Text != "wire mesh belt" || got[2].Suggestion != nil || got[3].Suggestion != nil {
		t.Errorf("suggestions = %+v, %+v, %+v", got[1].Suggestion, got[2].Suggestion, got[3].Suggestion)
	}
}

func TestADecisionNamesAtMostTwentyCompanies(t *testing.T) {
	names := make([]string, 0, 25)
	for i := range 25 {
		names = append(names, string(rune('a'+i)))
	}
	c := waiting("wire mesh belt", names...)
	v := domain.Verdict{Accept: true, Sure: true, Term: domain.Acceptance{Text: "wire mesh belt", Language: "en", Family: "all", Tier: "broad"}}
	if d, ok := v.Decision(c, nil); !ok || len(d.Term.UsedBy) != 20 {
		t.Errorf("a phrase at 25 companies = %+v, %v", d.Term, ok)
	}
}

func TestCountsSayHowManyWaitForTheModel(t *testing.T) {
	seen, fresh := waiting("a"), waiting("b")
	seen.Verdict = &domain.Verdict{}
	_, counts := domain.Count([]domain.Candidate{seen, fresh}, domain.StatusPending)
	if counts.Pending != 2 || counts.Unreviewed != 1 {
		t.Errorf("counts = %+v", counts)
	}
}

func TestExamplesAreAFewTermsOfEachTier(t *testing.T) {
	terms := []domain.Term{
		{Text: "a", Tier: domain.TierSpec},
		{Text: "b", Tier: domain.TierSpec},
		{Text: "c", Tier: domain.TierSpec},
		{Text: "d", Tier: domain.TierBroad},
	}
	got := domain.Examples(terms, 2)
	if len(got) != 3 || got[0].Text != "a" || got[1].Text != "b" || got[2].Text != "d" {
		t.Errorf("Examples = %+v", got)
	}
}
