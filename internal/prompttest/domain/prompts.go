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
// about a problem with a belt.
type Kind string

const (
	KindSupplier    Kind = "supplier"
	KindApplication Kind = "application"
	KindProblem     Kind = "problem"
)

// Kinds lists them in the order the results show them.
func Kinds() []Kind { return []Kind{KindSupplier, KindApplication, KindProblem} }

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

// Scopes lists them in the order the results show them.
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

// region is the place a scope names in its own language's sentence, or "" for a language alone.
func region(s Scope) string {
	switch s {
	case ScopeTwente:
		return "in Twente"
	case ScopeNetherlands:
		return "in Nederland"
	case ScopeGermany:
		return "in Deutschland"
	case ScopeDutch, ScopeGerman, ScopeEnglish:
	}
	return ""
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

// PromptSetVersion is the version of CurrentPromptSet.
const PromptSetVersion = 1

// CurrentPromptSet is the prompt set Prompt runs ask now: 5 belt families × a supplier, an
// application and a problem question × 6 scopes = 90 prompts, in general terms, never Tribelt's
// product names, so they measure what a buyer who does not know the names is told.
func CurrentPromptSet() PromptSet {
	set := PromptSet{Version: PromptSetVersion}
	for _, f := range Families() {
		for _, k := range Kinds() {
			for _, s := range Scopes() {
				set.Prompts = append(set.Prompts, Prompt{
					ID: fmt.Sprintf("%s-%s-%s", f, k, s), Family: f, Kind: k, Scope: s, Language: LanguageOf(s),
					Text: question(f, k, s),
				})
			}
		}
	}
	return set
}

// wording is what a family is called, used for and goes wrong with, in one language.
type wording struct {
	belt, application, problem string
}

// wordings are each family's words per language, in general terms.
func wordings(f Family, language string) wording {
	all := map[Family]map[string]wording{
		FamilyEyelink: {
			"nl": {"draadogenbanden (metalen transportbanden van gelaste draadogen)", "een doorloopoven in een industriële bakkerij", "Onze ovenband van draadogen loopt steeds uit het spoor."},
			"de": {"Drahtösengliederbänder (Förderbänder aus geschweißten Drahtösen)", "einen Durchlaufofen in einer Großbäckerei", "Unser Ofenband aus Drahtösen läuft immer wieder aus der Spur."},
			"en": {"wire eyelink conveyor belts", "a tunnel oven in an industrial bakery", "Our eyelink oven belt keeps running off track."},
		},
		FamilyOpenSpiral: {
			"nl": {"open spiraalbanden van metaal (open geweven gaasbanden)", "het drogen en koelen van producten op een open band", "In onze metalen gaasband breken steeds draden."},
			"de": {"Weitspiralbänder aus Metall (offen geflochtene Drahtgewebebänder)", "das Trocknen und Kühlen von Produkten auf einem offenen Band", "In unserem Drahtgewebeband brechen immer wieder Drähte."},
			"en": {"open spiral (balanced weave) metal conveyor belts", "drying and cooling products on an open belt", "Wires keep breaking in our woven wire mesh belt."},
		},
		FamilyClosedSpiral: {
			"nl": {"gesloten spiraalbanden (dicht geweven metalen transportbanden)", "het transport van kleine onderdelen door een oven", "Kleine producten vallen door onze metalen transportband heen."},
			"de": {"Engspiralbänder (dicht geflochtene Metallförderbänder)", "den Transport kleiner Teile durch einen Ofen", "Kleine Produkte fallen durch unser Metallförderband."},
			"en": {"closed spiral (compound balanced) metal conveyor belts", "carrying small parts through an oven", "Small products fall through our metal conveyor belt."},
		},
		FamilyTriFlex: {
			"nl": {"bochtgaande roestvrijstalen spiraaltransportbanden voor spiraaltorens", "een spiraalvriezer of spiraalkoeler", "Onze band in de spiraalvriezer loopt vast in de bochten."},
			"de": {"kurvengängige Edelstahl-Spiralförderbänder für Spiraltürme", "einen Spiralfroster oder Spiralkühler", "Unser Band im Spiralfroster verklemmt sich in den Kurven."},
			"en": {"curve-going stainless steel belts for spiral towers", "a spiral freezer or spiral cooler", "Our spiral freezer belt jams in the curves."},
		},
		FamilyOther: {
			"nl": {"metalen transportbanden zoals staafbanden, vlakdraadbanden en kettingbanden", "zware, hete producten in een hardingsoven", "Onze metalen band slijt snel onder zware, hete lasten."},
			"de": {"Metallförderbänder wie Stabgliederbänder, Flachdrahtbänder und Kettenbänder", "schwere, heiße Teile in einem Härteofen", "Unser Metallförderband verschleißt schnell unter schweren, heißen Lasten."},
			"en": {"metal conveyor belts such as rod belts, flat wire belts and chain belts", "heavy, hot parts in a hardening furnace", "Our metal conveyor belt wears out quickly under heavy, hot loads."},
		},
	}
	return all[f][language]
}

// question writes the prompt for a family, kind and scope in the scope's language.
func question(f Family, k Kind, s Scope) string {
	language := LanguageOf(s)
	w := wordings(f, language)
	where := region(s)
	if where != "" {
		where = " " + where
	}
	switch language {
	case "de":
		return map[Kind]string{
			KindSupplier:    "Welche Firmen liefern " + w.belt + where + "? Nenne Hersteller und Lieferanten.",
			KindApplication: "Welches Förderband eignet sich für " + w.application + ", und wer liefert es" + where + "?",
			KindProblem:     w.problem + " Woran liegt das, welches Band wäre besser, und wer liefert es" + where + "?",
		}[k]
	case "en":
		return map[Kind]string{
			KindSupplier:    "Which companies supply " + w.belt + "? Name manufacturers and suppliers.",
			KindApplication: "Which conveyor belt suits " + w.application + ", and who supplies it?",
			KindProblem:     w.problem + " What causes this, which belt would be better, and who supplies it?",
		}[k]
	}
	return map[Kind]string{
		KindSupplier:    "Welke bedrijven leveren " + w.belt + where + "? Noem fabrikanten en leveranciers.",
		KindApplication: "Welke transportband is geschikt voor " + w.application + ", en wie levert die" + where + "?",
		KindProblem:     w.problem + " Waar ligt dat aan, welke band is beter, en wie levert die" + where + "?",
	}[k]
}
