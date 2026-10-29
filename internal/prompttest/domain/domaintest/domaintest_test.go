package domaintest_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain/domaintest"
)

func TestVersionTwoAsksTwentyApplicationQuestionsInGeneralTerms(t *testing.T) {
	v := domaintest.Version2()
	set := v.PromptSet()
	if set.Version != 2 || len(set.Prompts) != 20 {
		t.Fatalf("set v%d has %d prompts", set.Version, len(set.Prompts))
	}
	ids := map[string]bool{}
	languages := map[string]int{}
	asked := []domain.Scope{domain.ScopeTwente, domain.ScopeDutch, domain.ScopeGerman, domain.ScopeEnglish}
	for _, p := range set.Prompts {
		if ids[p.ID] {
			t.Errorf("duplicate id %s", p.ID)
		}
		ids[p.ID] = true
		languages[p.Language]++
		if p.Kind != domain.KindApplication || p.Language != domain.LanguageOf(p.Scope) || !slices.Contains(asked, p.Scope) {
			t.Errorf("prompt %s = %+v", p.ID, p)
		}
		if lower := strings.ToLower(p.Text); strings.Contains(lower, "tribelt") || strings.Contains(lower, "tri-flex") || strings.Contains(lower, "(") {
			t.Errorf("prompt %s names Tribelt or its products, or explains a term: %s", p.ID, p.Text)
		}
	}
	if languages["nl"] != 10 || languages["de"] != 5 || languages["en"] != 5 {
		t.Errorf("languages = %v", languages)
	}
	byID := func(id string) domain.Question {
		i := slices.IndexFunc(v.Questions, func(q domain.Question) bool { return q.ID == id })
		if i < 0 {
			t.Fatalf("no prompt %s", id)
		}
		return v.Questions[i]
	}
	want := map[string][2]string{
		"eyelink-application-english": {"We are building a tunnel oven for an industrial bakery. What kind of conveyor belt should we use, and which manufacturers can supply it?", ""},
		"eyelink-application-twente": {
			"We bouwen een doorloopoven voor een industriële bakkerij. Wat voor transportband moeten we daarvoor gebruiken, en welke fabrikanten in de regio Twente kunnen die leveren?",
			"We are building a tunnel oven for an industrial bakery. What kind of conveyor belt should we use, and which manufacturers in the Twente region can supply it?",
		},
		"tri_flex_turn_curve-application-german": {
			"Wir installieren einen Spiralfroster für Tiefkühlprodukte. Welches Förderband sollten wir dafür einsetzen, und welche Hersteller können es liefern?",
			"We are installing a spiral freezer for frozen food. What kind of conveyor belt should we use, and which manufacturers can supply it?",
		},
		"other-application-dutch": {
			"We moeten zware, hete onderdelen door een hardingsoven transporteren. Wat voor transportband moeten we daarvoor gebruiken, en welke fabrikanten kunnen die leveren?",
			"We need to move heavy, hot parts through a hardening furnace. What kind of conveyor belt should we use, and which manufacturers can supply it?",
		},
	}
	for id, text := range want {
		if q := byID(id); q.Text != text[0] || q.English != text[1] {
			t.Errorf("%s = %q in English %q", id, q.Text, q.English)
		}
	}
}

func TestEveryImportedScopeIsALanguageWithAnOptionalRegion(t *testing.T) {
	scopes := domaintest.Scopes()
	if len(scopes) != len(domain.Scopes()) {
		t.Fatalf("scopes = %+v", scopes)
	}
	for i, s := range scopes {
		if s.ID != domain.Scopes()[i] || s.Language != domain.LanguageOf(s.ID) || s.Name == "" {
			t.Errorf("scope %d = %+v", i, s)
		}
	}
	if domaintest.Scope(domain.ScopeGermany).Region != "Germany" || domaintest.Scope(domain.ScopeGerman).Region != "" {
		t.Error("Germany has a region, German has none")
	}
	defer func() {
		if recover() == nil {
			t.Error("an unknown scope is no scope")
		}
	}()
	domaintest.Scope("french")
}
