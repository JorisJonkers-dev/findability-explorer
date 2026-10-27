package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"unicode"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/site"
)

// Correctness is how what an answer says about Tribelt holds against the fact sheet; empty when
// the answer does not mention Tribelt.
type Correctness string

const (
	CorrectnessNone    Correctness = ""
	CorrectnessCorrect Correctness = "correct"
	CorrectnessPartly  Correctness = "partly"
	CorrectnessWrong   Correctness = "wrong"
)

// Named is a company an answer names, as the extractor read it, with the address the answer
// gives for it, if any.
type Named struct {
	Name string
	URL  string
}

// Extracted is what the extractor read from one answer, in its strict format.
type Extracted struct {
	// Companies are the companies the answer names, in the order it names them.
	Companies    []Named
	Addresses    []string
	NamesTribelt bool
	CitesTribelt bool
	Correctness  Correctness
	WrongClaims  []string
}

// Company is a company an answer named, matched to the company list when it is on it.
type Company struct {
	Name string
	URL  string
	// CompetitorID is the matched company's id; empty for a name not on the list.
	CompetitorID string
	// Position is its place in the answer, from 1.
	Position int
}

// Extraction is how one answer was read; Failed when the extractor could not read it, after a
// retry, so the answer is marked rather than dropped.
type Extraction struct {
	Failed       bool
	Companies    []Company
	Addresses    []string
	NamesTribelt bool
	CitesTribelt bool
	Correctness  Correctness
	WrongClaims  []string
	InputTokens  int64
	OutputTokens int64
	Cost         Micros
	// HaikuTokens are the tokens Claude Haiku on an Anthropic key used reading the answer; they
	// count towards the month's language-model tokens. 0 when OpenRouter read it.
	HaikuTokens int64
}

// Role is what a company is to Tribelt.
type Role string

const (
	RoleTribelt    Role = "tribelt"
	RoleMirror     Role = "mirror"
	RoleGroup      Role = "group"
	RoleCompetitor Role = "competitor"
)

// DirectoryCompany is what the prompt test knows of a company on the list: the names it goes by,
// its website and the website's host, and whether it is archived or a person kept it.
type DirectoryCompany struct {
	ID   string
	Name string
	// OtherNames are the names answers may give it instead, such as "Ashworth Bros.".
	OtherNames []string
	Host       string
	Role       Role
	// Website is the home page to read; "" without one.
	Website string
	// Archived is true for a company taken off the list, by a person or the language model.
	Archived bool
	// Kept is true once a person restored the company or kept it on the list: the language model
	// never archives it.
	Kept bool
}

// legal are the words of a legal form an answer may add or leave out.
func legal() []string {
	return []string{"bv", "b.v.", "gmbh", "ltd", "inc", "nv", "n.v.", "ag", "llc", "sa", "s.a.", "spa", "s.p.a.", "sl", "co", "kg", "company", "group"}
}

// NameKey is a name as it compares: folded, without a remark in brackets, a legal form or
// punctuation. "Märtens Transportbänder GmbH" and "martens transportbander" have one key.
func NameKey(name string) string {
	base, _, _ := strings.Cut(name, "(")
	words := strings.FieldsFunc(fold.Fold(base), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' })
	words = slices.DeleteFunc(words, func(w string) bool { return slices.Contains(legal(), w) || strings.Trim(w, ".") == "" })
	for i, w := range words {
		words[i] = strings.Trim(w, ".")
	}
	return strings.Join(words, " ")
}

// aliases are the keys a directory company goes by: its name, its other names, and each name in
// a remark in brackets ("Wire Belt Company (Wire Belt, WBC)").
func aliases(c DirectoryCompany) []string {
	out := []string{NameKey(c.Name)}
	for _, n := range c.OtherNames {
		if k := NameKey(n); k != "" {
			out = append(out, k)
		}
	}
	if _, remark, ok := strings.Cut(c.Name, "("); ok {
		for _, part := range strings.Split(strings.TrimSuffix(strings.TrimSpace(remark), ")"), ",") {
			if k := NameKey(part); k != "" {
				out = append(out, k)
			}
		}
	}
	return out
}

// sameCompany reports whether an address is on the website of the company with host: the same
// registered domain, so "products.bandogrp.com" is Bando's. An address on a marketplace,
// directory or social network is nobody's website.
func sameCompany(address, host string) bool {
	d := site.CompanyDomain(address)
	return d != "" && d == site.Domain(host)
}

// Match finds the company on the list a name (and the address given for it) means: by website
// domain, by name or alias, or by a short name that starts one of them ("ESFO" for "ESFO
// Transportbanden"). It returns "" for a name not on the list.
func Match(n Named, directory []DirectoryCompany) string {
	if n.URL != "" {
		for _, c := range directory {
			if sameCompany(n.URL, c.Host) {
				return c.ID
			}
		}
	}
	key := NameKey(n.Name)
	if key == "" {
		return ""
	}
	for _, c := range directory {
		if slices.Contains(aliases(c), key) {
			return c.ID
		}
	}
	first, _, _ := strings.Cut(key, " ")
	if len([]rune(first)) < 4 && !strings.Contains(key, " ") {
		return ""
	}
	for _, c := range directory {
		if slices.ContainsFunc(aliases(c), func(a string) bool { return strings.HasPrefix(a, key+" ") }) {
			return c.ID
		}
	}
	return ""
}

// Read turns what the extractor read into the answer's extraction: each named company matched to
// the list, the answer's own citations added to the addresses, and Tribelt named or cited also
// when the extractor missed it but a name or an address shows it.
func Read(x Extracted, citations []string, directory []DirectoryCompany) Extraction {
	out := Extraction{
		NamesTribelt: x.NamesTribelt, CitesTribelt: x.CitesTribelt, Correctness: x.Correctness,
		WrongClaims: x.WrongClaims,
	}
	role := map[string]Role{}
	tribeltHosts := []string{}
	for _, c := range directory {
		role[c.ID] = c.Role
		if c.Role == RoleTribelt {
			tribeltHosts = append(tribeltHosts, c.Host)
		}
	}
	for i, n := range x.Companies {
		id := Match(n, directory)
		out.Companies = append(out.Companies, Company{Name: n.Name, URL: n.URL, CompetitorID: id, Position: i + 1})
		out.NamesTribelt = out.NamesTribelt || (id != "" && role[id] == RoleTribelt)
	}
	for _, a := range append(slices.Clone(x.Addresses), citations...) {
		if a = strings.TrimSpace(a); a != "" && !slices.Contains(out.Addresses, a) {
			out.Addresses = append(out.Addresses, a)
		}
	}
	for _, a := range out.Addresses {
		out.CitesTribelt = out.CitesTribelt || slices.ContainsFunc(tribeltHosts, func(t string) bool { return sameCompany(a, t) })
	}
	if !out.NamesTribelt {
		out.Correctness, out.WrongClaims = CorrectnessNone, nil
	}
	return out
}

// CandidateID is the id of a competitor candidate: made from its name's key, so the same name in
// every run is one candidate, and a dismissed one stays dismissed.
func CandidateID(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:16]
}

// CitedCompanies are the companies on the list whose website an answer's addresses point to, in
// the order first cited, each once.
func CitedCompanies(addresses []string, directory []DirectoryCompany) []string {
	var out []string
	for _, a := range addresses {
		for _, c := range directory {
			if sameCompany(a, c.Host) && !slices.Contains(out, c.ID) {
				out = append(out, c.ID)
			}
		}
	}
	return out
}

// NamedCompanies are the companies on the list an answer names, in its order, each once.
func (x Extraction) NamedCompanies() []string {
	var out []string
	for _, c := range x.Companies {
		if c.CompetitorID != "" && !slices.Contains(out, c.CompetitorID) {
			out = append(out, c.CompetitorID)
		}
	}
	return out
}
