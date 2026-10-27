// Package domain is the prompt-test context's pure core: the versioned prompt set, the choice of
// models, Prompt runs and their answers, the reading of answers into the companies they name and
// the addresses they cite, AI share of voice and citation share, competitor candidates, the
// schedule and the budget (docs/adr/0004). It imports no transport, persistence or generated code
// (enforced by depguard).
package domain

import "fmt"

// Family is one of Tribelt's belt families, as the terminology map names them.
type Family string

const (
	FamilyEyelink      Family = "eyelink"
	FamilyOpenSpiral   Family = "open_spiral"
	FamilyClosedSpiral Family = "closed_spiral"
	FamilyTriFlex      Family = "tri_flex_turn_curve"
	FamilyOther        Family = "other"
)

// Families lists them in the order the results show them.
func Families() []Family {
	return []Family{FamilyEyelink, FamilyOpenSpiral, FamilyClosedSpiral, FamilyTriFlex, FamilyOther}
}

// Kind is the kind of question a prompt asks: for a supplier, for a belt for an application, or
// about a problem with a belt. Prompt set 2 asks only the application question; answers to
// prompt set 1 keep their kind.
type Kind string

const (
	KindSupplier    Kind = "supplier"
	KindApplication Kind = "application"
	KindProblem     Kind = "problem"
)

// Scope is where and in which language a prompt asks: a region in its language, or a language
// with no region.
type Scope string

const (
	ScopeTwente      Scope = "twente"
	ScopeNetherlands Scope = "netherlands"
	ScopeGermany     Scope = "germany"
	ScopeDutch       Scope = "dutch"
	ScopeGerman      Scope = "german"
	ScopeEnglish     Scope = "english"
)

// Scopes lists them in the order the results show them: every scope an answer may have, including
// the Netherlands and Germany that only prompt set 1 asked in.
func Scopes() []Scope {
	return []Scope{ScopeTwente, ScopeNetherlands, ScopeGermany, ScopeDutch, ScopeGerman, ScopeEnglish}
}

// LanguageOf is the language a scope asks in.
func LanguageOf(s Scope) string {
	switch s {
	case ScopeGermany, ScopeGerman:
		return "de"
	case ScopeEnglish:
		return "en"
	case ScopeTwente, ScopeNetherlands, ScopeDutch:
	}
	return "nl"
}

// Prompt is one buyer question of the prompt set.
type Prompt struct {
	ID       string
	Family   Family
	Kind     Kind
	Scope    Scope
	Language string
	Text     string
}

// PromptSet is the fixed, versioned set of questions every Prompt run asks. A change to any
// question is a new version, so results of different versions are never compared as the same.
type PromptSet struct {
	Version int
	Prompts []Prompt
}

// PromptSetVersion is the version of CurrentPromptSet. Version 1 asked 90 questions (a supplier,
// an application and a problem question in six scopes); version 2 keeps the one a buyer most
// often asks, so a run costs about a sixth.
const PromptSetVersion = 2

// SetScopes are the scopes the current prompt set asks in: Twente in Dutch, and Dutch, German
// and English with no region.
func SetScopes() []Scope {
	return []Scope{ScopeTwente, ScopeDutch, ScopeGerman, ScopeEnglish}
}

// CurrentPromptSet is the prompt set Prompt runs ask now: for each of the 5 belt families, the
// application question a buyer would type, in 4 scopes = 20 prompts. The buyer describes the job
// and asks which belt and who makes it, in general terms and never with Tribelt's product names,
// so a run measures what a buyer who does not know the names is told.
func CurrentPromptSet() PromptSet {
	set := PromptSet{Version: PromptSetVersion}
	for _, f := range Families() {
		for _, s := range SetScopes() {
			set.Prompts = append(set.Prompts, Prompt{
				ID: fmt.Sprintf("%s-%s-%s", f, KindApplication, s), Family: f, Kind: KindApplication, Scope: s,
				Language: LanguageOf(s), Text: question(f, s),
			})
		}
	}
	return set
}

// situation is what a buyer of a family is building or running, in a language.
func situation(f Family, language string) string {
	all := map[Family]map[string]string{
		FamilyEyelink: {
			"nl": "We bouwen een doorloopoven voor een industriële bakkerij.",
			"de": "Wir bauen einen Durchlaufofen für eine Großbäckerei.",
			"en": "We are building a tunnel oven for an industrial bakery.",
		},
		FamilyOpenSpiral: {
			"nl": "We willen producten drogen en daarna koelen op een open band waar de lucht goed doorheen kan.",
			"de": "Wir wollen Produkte auf einem offenen, gut luftdurchlässigen Band trocknen und anschließend kühlen.",
			"en": "We want to dry and then cool products on an open belt that lets the air through.",
		},
		FamilyClosedSpiral: {
			"nl": "We moeten kleine onderdelen door een industriële oven voeren zonder dat ze door de band vallen.",
			"de": "Wir müssen kleine Teile durch einen Industrieofen fördern, ohne dass sie durch das Band fallen.",
			"en": "We need to carry small parts through an industrial oven without them falling through the belt.",
		},
		FamilyTriFlex: {
			"nl": "We plaatsen een spiraalvriezer voor diepvriesproducten.",
			"de": "Wir installieren einen Spiralfroster für Tiefkühlprodukte.",
			"en": "We are installing a spiral freezer for frozen food.",
		},
		FamilyOther: {
			"nl": "We moeten zware, hete onderdelen door een hardingsoven transporteren.",
			"de": "Wir müssen schwere, heiße Teile durch einen Härteofen fördern.",
			"en": "We need to move heavy, hot parts through a hardening furnace.",
		},
	}
	return all[f][language]
}

// question writes a family's application question for a scope, in the scope's language, as a
// buyer would type it: the job, then which belt and who supplies it.
func question(f Family, s Scope) string {
	language := LanguageOf(s)
	ask := map[string]string{
		"nl": " Wat voor transportband moeten we daarvoor gebruiken, en welke fabrikanten kunnen die leveren?",
		"de": " Welches Förderband sollten wir dafür einsetzen, und welche Hersteller können es liefern?",
		"en": " What kind of conveyor belt should we use, and which manufacturers can supply it?",
	}[language]
	if s == ScopeTwente {
		ask = " Wat voor transportband moeten we daarvoor gebruiken, en welke fabrikanten in de regio Twente kunnen die leveren?"
	}
	return situation(f, language) + ask
}
