// Package domain is the prompt-test context's pure core: the versioned prompt set, the choice of
// models, Prompt runs and their answers, the reading of answers into the companies they name and
// the addresses they cite, AI share of voice and citation share, competitor candidates, the
// schedule and the budget (docs/adr/0004). It imports no transport, persistence or generated code
// (enforced by depguard).
package domain

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

// Kind is the kind of question an answer records: for a supplier, for a belt for an application,
// or about a problem with a belt. It is the question's buyer intent (KindOf), with the product
// intent recorded as supplier, as answers did before the Prompt set moved to the database.
type Kind string

const (
	KindSupplier    Kind = "supplier"
	KindApplication Kind = "application"
	KindProblem     Kind = "problem"
)

// Scope is the id of a scope: a language a question is asked in, with or without a region
// (ScopeInfo). The scopes imported with versions 1 and 2 keep the ids answers recorded.
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

// LanguageOf is the language a scope asks in, as the stored scopes say; "" for a scope they do not
// know.
func LanguageOf(scopes []ScopeInfo, s Scope) string {
	for _, info := range scopes {
		if info.ID == s {
			return info.Language
		}
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

// PromptSet is the versioned set of questions a Prompt run asks: a published version's questions
// (PublishedSet.PromptSet). A change to any question is a new version, so results of different
// versions are never compared as the same.
type PromptSet struct {
	Version int
	Prompts []Prompt
}
