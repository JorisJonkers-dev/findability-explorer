// Package domaintest holds the Prompt set the tests ask: version 2 as the code built it before the
// Prompt set moved to the database, which the persistence tests compare the imported version 2
// with. Only tests import it.
package domaintest

import (
	"fmt"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// Scopes are the scopes as imported, in the order the results show them.
func Scopes() []domain.ScopeInfo {
	return []domain.ScopeInfo{
		{ID: domain.ScopeTwente, Language: "nl", Region: "Twente", Name: "Twente (Dutch)"},
		{ID: domain.ScopeNetherlands, Language: "nl", Region: "Netherlands", Name: "Netherlands (Dutch)"},
		{ID: domain.ScopeGermany, Language: "de", Region: "Germany", Name: "Germany (German)"},
		{ID: domain.ScopeDutch, Language: "nl", Name: "Dutch, no region"},
		{ID: domain.ScopeGerman, Language: "de", Name: "German, no region"},
		{ID: domain.ScopeEnglish, Language: "en", Name: "English, no region"},
	}
}

// Scope is the imported scope id.
func Scope(id domain.Scope) domain.ScopeInfo {
	for _, s := range Scopes() {
		if s.ID == id {
			return s
		}
	}
	panic("no scope " + string(id))
}

// Version2 is Prompt set version 2 as the code built it: for each of the 5 belt families, the
// application question a buyer would type, in Twente and in Dutch, German and English with no
// region = 20 questions, each with its English translation unless it is English already.
func Version2() domain.PublishedSet {
	set := domain.PublishedSet{
		Version: 2, PublishedAt: time.Date(2026, 10, 27, 6, 35, 33, 0, time.UTC), PublishedBy: "import",
		Note: "Kept the application question a buyer most often asks, in four scopes.",
	}
	for _, f := range domain.Families() {
		for _, id := range []domain.Scope{domain.ScopeTwente, domain.ScopeDutch, domain.ScopeGerman, domain.ScopeEnglish} {
			s := Scope(id)
			q := domain.Question{
				ID: fmt.Sprintf("%s-%s-%s", f, domain.KindApplication, id), Family: f, Intent: domain.IntentApplication, Scope: s,
				Text: question(f, s.Language, id == domain.ScopeTwente),
			}
			if s.Language != "en" {
				q.English = question(f, "en", id == domain.ScopeTwente)
			}
			set.Questions = append(set.Questions, q)
		}
	}
	return set
}

// situation is what a buyer of a family is building or running, in a language.
func situation(f domain.Family, language string) string {
	all := map[domain.Family]map[string]string{
		domain.FamilyEyelink: {
			"nl": "We bouwen een doorloopoven voor een industriële bakkerij.",
			"de": "Wir bauen einen Durchlaufofen für eine Großbäckerei.",
			"en": "We are building a tunnel oven for an industrial bakery.",
		},
		domain.FamilyOpenSpiral: {
			"nl": "We willen producten drogen en daarna koelen op een open band waar de lucht goed doorheen kan.",
			"de": "Wir wollen Produkte auf einem offenen, gut luftdurchlässigen Band trocknen und anschließend kühlen.",
			"en": "We want to dry and then cool products on an open belt that lets the air through.",
		},
		domain.FamilyClosedSpiral: {
			"nl": "We moeten kleine onderdelen door een industriële oven voeren zonder dat ze door de band vallen.",
			"de": "Wir müssen kleine Teile durch einen Industrieofen fördern, ohne dass sie durch das Band fallen.",
			"en": "We need to carry small parts through an industrial oven without them falling through the belt.",
		},
		domain.FamilyTriFlex: {
			"nl": "We plaatsen een spiraalvriezer voor diepvriesproducten.",
			"de": "Wir installieren einen Spiralfroster für Tiefkühlprodukte.",
			"en": "We are installing a spiral freezer for frozen food.",
		},
		domain.FamilyOther: {
			"nl": "We moeten zware, hete onderdelen door een hardingsoven transporteren.",
			"de": "Wir müssen schwere, heiße Teile durch einen Härteofen fördern.",
			"en": "We need to move heavy, hot parts through a hardening furnace.",
		},
	}
	return all[f][language]
}

// question writes a family's application question in a language, as a buyer would type it: the
// job, then which belt and who supplies it, in the Twente region when twente is true.
func question(f domain.Family, language string, twente bool) string {
	ask := map[string]string{
		"nl": " Wat voor transportband moeten we daarvoor gebruiken, en welke fabrikanten kunnen die leveren?",
		"de": " Welches Förderband sollten wir dafür einsetzen, und welche Hersteller können es liefern?",
		"en": " What kind of conveyor belt should we use, and which manufacturers can supply it?",
	}[language]
	if twente {
		ask = map[string]string{
			"nl": " Wat voor transportband moeten we daarvoor gebruiken, en welke fabrikanten in de regio Twente kunnen die leveren?",
			"en": " What kind of conveyor belt should we use, and which manufacturers in the Twente region can supply it?",
		}[language]
	}
	return situation(f, language) + ask
}
