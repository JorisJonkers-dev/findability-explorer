// Package seed is the terminology context's catalogue: the terminology map from the group's desk
// research of 22 October 2026, embedded in the binary. data/terms.json and data/competitors.json
// are copies of project/report/research/ in the course-material repository; replace both to update.
package seed

import (
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

//go:embed data/terms.json data/competitors.json
var data embed.FS

type rawTerm struct {
	Term     string   `json:"term"`
	Language string   `json:"language"`
	Family   string   `json:"belt_family"`
	Tier     string   `json:"intent_tier"`
	UsedBy   []string `json:"used_by"`
	Source   string   `json:"source"`
}

type rawBeltName struct {
	Name       string `json:"competitor_name"`
	Equivalent string `json:"tribelt_equivalent"`
}

type rawCompany struct {
	Name          string        `json:"name"`
	Country       string        `json:"country"`
	Website       string        `json:"website"`
	Languages     []string      `json:"languages"`
	BeltNames     []rawBeltName `json:"belt_names"`
	HomepageTitle string        `json:"homepage_title"`
	GroupCompany  bool          `json:"group_company"`
}

// Catalogue holds the parsed map. It never changes after Load.
type Catalogue struct {
	terms     []domain.Term
	companies []domain.Company
}

// Terms returns every term in the map.
func (c *Catalogue) Terms() []domain.Term { return c.terms }

// Companies returns every company in the map.
func (c *Catalogue) Companies() []domain.Company { return c.companies }

// Load parses the embedded map. It fails on any value the domain does not know, so a bad data
// file stops the binary at startup instead of serving half a map.
func Load() (*Catalogue, error) {
	return parse(mustRead("data/terms.json"), mustRead("data/competitors.json"))
}

func mustRead(name string) []byte {
	b, err := data.ReadFile(name)
	if err != nil {
		panic(err) // cannot happen: the file is embedded at build time
	}
	return b
}

func parse(termsJSON, companiesJSON []byte) (*Catalogue, error) {
	var rawTerms []rawTerm
	if err := json.Unmarshal(termsJSON, &rawTerms); err != nil {
		return nil, fmt.Errorf("seed: terms: %w", err)
	}
	var rawCompanies []rawCompany
	if err := json.Unmarshal(companiesJSON, &rawCompanies); err != nil {
		return nil, fmt.Errorf("seed: companies: %w", err)
	}
	terms := make([]domain.Term, 0, len(rawTerms))
	for i, r := range rawTerms {
		t, err := toTerm(r)
		if err != nil {
			return nil, fmt.Errorf("seed: term %d (%q): %w", i, r.Term, err)
		}
		terms = append(terms, t)
	}
	companies := make([]domain.Company, 0, len(rawCompanies))
	for _, r := range rawCompanies {
		companies = append(companies, toCompany(r))
	}
	return &Catalogue{terms: terms, companies: companies}, nil
}

func toTerm(r rawTerm) (domain.Term, error) {
	lang, err := domain.ParseLanguage(r.Language)
	if err != nil {
		return domain.Term{}, err
	}
	family, err := domain.ParseFamily(r.Family)
	if err != nil {
		return domain.Term{}, err
	}
	tier, err := domain.ParseTier(r.Tier)
	if err != nil {
		return domain.Term{}, err
	}
	return domain.Term{Text: r.Term, Language: lang, Family: family, Tier: tier, UsedBy: r.UsedBy, Source: r.Source}, nil
}

func toCompany(r rawCompany) domain.Company {
	names := make([]domain.BeltName, 0, len(r.BeltNames))
	for _, b := range r.BeltNames {
		names = append(names, domain.BeltName{Name: b.Name, Family: familyOf(b.Equivalent)})
	}
	title := r.HomepageTitle
	if title == "" {
		title = "unverified"
	}
	return domain.Company{
		Name: r.Name, Country: r.Country, Website: r.Website, Languages: languageCodes(r.Languages),
		GroupCompany: r.GroupCompany, BeltNames: names, HomepageTitle: title,
	}
}

// languageCode is a language tag such as "de" or "en-GB".
var languageCode = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z]{2,4})?$`)

// languageCodes keeps the language tag at the start of each research note ("it (catalogue via
// Italian distributor)" is "it") and drops a note that names none ("others unverified").
func languageCodes(raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		code, _, _ := strings.Cut(strings.TrimSpace(r), " ")
		if languageCode.MatchString(code) {
			out = append(out, code)
		}
	}
	return out
}

// familyOf reads the research notes' free-text Tribelt equivalent, such as "Open spiral /
// balanced weave (open spiraalband / Weitspiralband)", as a belt family.
func familyOf(equivalent string) domain.Family {
	e := strings.ToLower(equivalent)
	switch {
	case strings.Contains(e, "eyelink") || strings.Contains(e, "eye-link"):
		return domain.FamilyEyelink
	case strings.Contains(e, "open spiral") || strings.Contains(e, "balanced weave"):
		return domain.FamilyOpenSpiral
	case strings.Contains(e, "closed spiral") || strings.Contains(e, "compound"):
		return domain.FamilyClosedSpiral
	case strings.Contains(e, "tri-flex") || strings.Contains(e, "turn") || strings.Contains(e, "spiral tower"):
		return domain.FamilyTriFlex
	default:
		return domain.FamilyOther
	}
}
