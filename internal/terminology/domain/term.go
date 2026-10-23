// Package domain is the terminology context's pure core: the terms companies use for metal
// conveyor belts, the companies that use them, and how the map is searched. It imports no
// transport, persistence or generated code (enforced by depguard).
package domain

import (
	"fmt"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
)

// Language is a language as the terminology map records it; LanguageEnUS marks American usage.
type Language string

const (
	LanguageDe   Language = "de"
	LanguageEn   Language = "en"
	LanguageEnUS Language = "en-US"
	LanguageNl   Language = "nl"
	LanguageFr   Language = "fr"
	LanguagePl   Language = "pl"
	LanguageIt   Language = "it"
	LanguageEs   Language = "es"
	LanguageSv   Language = "sv"
	LanguageDa   Language = "da"
)

// Languages lists every language, in the order facets report them.
func Languages() []Language {
	return []Language{
		LanguageDe, LanguageEn, LanguageEnUS, LanguageNl, LanguageFr,
		LanguagePl, LanguageIt, LanguageEs, LanguageSv, LanguageDa,
	}
}

// Tier is the kind of search a term represents.
type Tier string

const (
	TierSpec        Tier = "spec"
	TierProduct     Tier = "product"
	TierApplication Tier = "application"
	TierProblem     Tier = "problem"
	TierBroad       Tier = "broad"
)

// Tiers lists every tier, from the most specific search to the broadest.
func Tiers() []Tier {
	return []Tier{TierSpec, TierProduct, TierApplication, TierProblem, TierBroad}
}

// Family is the Tribelt belt family a term names. FamilyAll is a broad category that covers every
// family, FamilySpecAll a material or specification word that applies to every family.
type Family string

const (
	FamilyEyelink      Family = "eyelink"
	FamilyOpenSpiral   Family = "open_spiral"
	FamilyClosedSpiral Family = "closed_spiral"
	FamilyTriFlex      Family = "tri_flex_turn_curve"
	FamilyOther        Family = "other"
	FamilyAll          Family = "all"
	FamilySpecAll      Family = "spec_all"
)

// Families lists every family, Tribelt's own four first.
func Families() []Family {
	return []Family{
		FamilyEyelink, FamilyOpenSpiral, FamilyClosedSpiral, FamilyTriFlex,
		FamilyOther, FamilyAll, FamilySpecAll,
	}
}

// ErrUnknownValue is returned when a language, tier or family is not one the map knows.
var ErrUnknownValue = fmt.Errorf("unknown value")

// ParseLanguage returns the Language named raw.
func ParseLanguage(raw string) (Language, error) { return parse(raw, Languages()) }

// ParseTier returns the Tier named raw.
func ParseTier(raw string) (Tier, error) { return parse(raw, Tiers()) }

// ParseFamily returns the Family named raw.
func ParseFamily(raw string) (Family, error) { return parse(raw, Families()) }

func parse[T ~string](raw string, known []T) (T, error) {
	for _, k := range known {
		if string(k) == raw {
			return k, nil
		}
	}
	var zero T
	return zero, fmt.Errorf("%w: %q", ErrUnknownValue, raw)
}

// Term is one name for a belt, material or specification, as one or more companies use it.
type Term struct {
	Text     string
	Language Language
	Family   Family
	Tier     Tier
	UsedBy   []string
	Source   string
	Origin   Origin
}

// Origin is where a term comes from: the desk research, or a scan's phrase an administrator
// accepted from the review queue.
type Origin string

const (
	OriginResearch Origin = "research"
	OriginReview   Origin = "review"
)

// BeltName is a company's own name for a belt, and the Tribelt family it corresponds to.
type BeltName struct {
	Name   string
	Family Family
}

// Company is a company in the map: a competitor, or a company in Tribelt's own group.
type Company struct {
	Name          string
	Country       string
	Website       string
	Languages     []string
	GroupCompany  bool
	BeltNames     []BeltName
	HomepageTitle string
}

// Fold reduces s to the form searches compare; see fold.Fold.
func Fold(s string) string { return fold.Fold(s) }
