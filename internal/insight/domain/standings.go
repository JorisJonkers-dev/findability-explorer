// Package domain is the insight context's pure core: how Tribelt stands against its competitors
// on each measure (the standings), the belt families compared (products), and the plan's ordered
// improvements. It owns no data of its own beyond the plan's ticks and targets: everything else
// it is handed, by value, from the contexts that own it. It imports no transport, persistence or
// generated code (enforced by depguard).
package domain

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
)

// Role is what a company is to Tribelt.
type Role string

const (
	RoleTribelt    Role = "tribelt"
	RoleMirror     Role = "mirror"
	RoleGroup      Role = "group"
	RoleCompetitor Role = "competitor"
)

// BeltName is a company's own name for a belt, and the Tribelt family it answers to.
type BeltName struct {
	Name   string
	Family string
	// FromScan is true for a product a scan matched, not one the desk research named.
	FromScan bool
}

// Company is a company that is not archived, as the companies context lists it.
type Company struct {
	ID        string
	Name      string
	Role      Role
	Website   string
	BeltNames []BeltName
}

// Mode is how a Prompt test question was asked: from the model's own knowledge, or with web search.
type Mode string

const (
	ModeKnowledge Mode = "knowledge"
	ModeWeb       Mode = "web"
)

// Answer is one answer of a Prompt run that was answered and read: the question's scope and belt
// family, and the companies on the list it names, in order, and whose sites it cites.
type Answer struct {
	PromptID string
	Mode     Mode
	Scope    string
	Family   string
	Named    []string
	Cited    []string
}

// Run is one Prompt run with the answers that count.
type Run struct {
	ID        string
	StartedAt time.Time
	Answers   []Answer
}

// Keep is the answers f keeps.
func (r Run) Keep(f func(Answer) bool) []Answer {
	var out []Answer
	for _, a := range r.Answers {
		if f(a) {
			out = append(out, a)
		}
	}
	return out
}

// ShareOfVoice is the share of answers that name the company, from 0 to 1; ok is false without answers.
func ShareOfVoice(answers []Answer, id string) (float64, bool) {
	return share(answers, func(a Answer) bool { return slices.Contains(a.Named, id) })
}

// CitationShare is the share of web answers that cite the company's site; ok is false without web answers.
func CitationShare(answers []Answer, id string) (float64, bool) {
	var web []Answer
	for _, a := range answers {
		if a.Mode == ModeWeb {
			web = append(web, a)
		}
	}
	return share(web, func(a Answer) bool { return slices.Contains(a.Cited, id) })
}

func share(answers []Answer, hit func(Answer) bool) (float64, bool) {
	if len(answers) == 0 {
		return 0, false
	}
	n := 0
	for _, a := range answers {
		if hit(a) {
			n++
		}
	}
	return float64(n) / float64(len(answers)), true
}

// Access is how far a site's robots.txt lets the AI crawlers in.
type Access string

const (
	// AccessOpen means no crawler is kept out of the whole site.
	AccessOpen Access = "open"
	// AccessSome means some crawlers are kept out.
	AccessSome Access = "some"
	// AccessClosed means every crawler is kept out.
	AccessClosed Access = "closed"
	// AccessUnknown means robots.txt could not be read.
	AccessUnknown Access = "unknown"
)

// Readiness is what a company's newest scan recorded of its AI readiness.
type Readiness struct {
	Crawlers Access
	// Blocked are the crawlers kept out of the whole site.
	Blocked          []string
	Sitemap          bool
	SitemapPages     int
	SitemapLanguages []string
	LLMs             bool
	LLMsFull         bool
	Markdown         bool
	StructuredData   []string
	ScannedAt        time.Time
}

// ReadinessScore is how many of the five readiness checks a site passes: its crawlers let in (half
// when some are kept out), a sitemap, llms.txt, llms-full.txt and Markdown pages.
func (r Readiness) ReadinessScore() float64 {
	score := 0.0
	switch r.Crawlers {
	case AccessOpen:
		score++
	case AccessSome:
		score += 0.5
	case AccessClosed, AccessUnknown:
	}
	for _, ok := range []bool{r.Sitemap, r.LLMs, r.LLMsFull, r.Markdown} {
		if ok {
			score++
		}
	}
	return score
}

// Measure is one thing Tribelt is compared on. Higher is better on every one.
type Measure string

const (
	MeasureShareOfVoice  Measure = "aiShareOfVoice"
	MeasureCitationShare Measure = "citationShare"
	MeasureReadiness     Measure = "readinessChecks"
	MeasureSitemapPages  Measure = "sitemapPages"
	MeasureTermsUsed     Measure = "termsUsed"
)

// Measures lists them in the order the standings show them.
func Measures() []Measure {
	return []Measure{MeasureShareOfVoice, MeasureCitationShare, MeasureReadiness, MeasureSitemapPages, MeasureTermsUsed}
}

// Facts are what the other contexts know of the companies now: the newest Prompt run with answers
// that count, each company's newest readiness, and the map terms each company's newest scan found.
type Facts struct {
	Run       *Run
	Readiness map[string]Readiness
	// TermsUsed is set for every company a scan read; a company never scanned is absent.
	TermsUsed map[string]int
}

// Value is a company's value on a measure; ok is false when there is no data for it.
func (f Facts) Value(m Measure, id string) (float64, bool) {
	switch m {
	case MeasureShareOfVoice, MeasureCitationShare:
		if f.Run == nil {
			return 0, false
		}
		if m == MeasureShareOfVoice {
			return ShareOfVoice(f.Run.Answers, id)
		}
		return CitationShare(f.Run.Answers, id)
	case MeasureReadiness, MeasureSitemapPages:
		r, ok := f.Readiness[id]
		if !ok {
			return 0, false
		}
		if m == MeasureReadiness {
			return r.ReadinessScore(), true
		}
		return float64(r.SitemapPages), true
	case MeasureTermsUsed:
		n, ok := f.TermsUsed[id]
		return float64(n), ok
	}
	return 0, false
}

// Leader is the competitor with the best value on a measure.
type Leader struct {
	CompanyID string
	Name      string
	Value     float64
}

// Standing is Tribelt against its competitors on one measure: Tribelt's value, the median over
// the competitors with data (the typical competitor) and the leader. Tribelt, its group and the
// mirror are never in the median or the leader.
type Standing struct {
	Measure Measure
	Tribelt *float64
	Median  *float64
	Leader  *Leader
	// Competitors is how many competitors have data on the measure.
	Competitors int
}

// Trails reports whether the leader is ahead of Tribelt; never when either has no value.
func (s Standing) Trails() bool {
	return s.Leader != nil && s.Tribelt != nil && s.Leader.Value > *s.Tribelt
}

// Leads reports whether Tribelt is level with or ahead of the leader.
func (s Standing) Leads() bool {
	return s.Leader != nil && s.Tribelt != nil && *s.Tribelt >= s.Leader.Value
}

// Standings compares Tribelt with the competitors on every measure. A tie for the lead goes to the
// competitor named first alphabetically (accents and case aside).
func Standings(companies []Company, f Facts) []Standing {
	out := make([]Standing, 0, len(Measures()))
	for _, m := range Measures() {
		out = append(out, standing(m, companies, f))
	}
	return out
}

func standing(m Measure, companies []Company, f Facts) Standing {
	s := Standing{Measure: m}
	var values []float64
	for _, c := range companies {
		v, ok := f.Value(m, c.ID)
		if !ok {
			continue
		}
		switch c.Role {
		case RoleTribelt:
			s.Tribelt = &v
		case RoleCompetitor:
			values = append(values, v)
			if s.Leader == nil || v > s.Leader.Value || (v == s.Leader.Value && byName(c.Name, s.Leader.Name) < 0) {
				s.Leader = &Leader{CompanyID: c.ID, Name: c.Name, Value: v}
			}
		case RoleGroup, RoleMirror:
		}
	}
	s.Competitors = len(values)
	if med, ok := Median(values); ok {
		s.Median = &med
	}
	return s
}

func byName(a, b string) int {
	return cmp.Or(strings.Compare(fold.Fold(a), fold.Fold(b)), strings.Compare(a, b))
}

// Median is the middle value, or the mean of the two middle ones; ok is false for no values.
func Median(values []float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid], true
	}
	return (sorted[mid-1] + sorted[mid]) / 2, true
}

// Trailing counts the measures where Tribelt trails the leader.
func Trailing(standings []Standing) int {
	n := 0
	for _, s := range standings {
		if s.Trails() {
			n++
		}
	}
	return n
}

// LeaderOf is the standing's leader on m, or nil.
func LeaderOf(standings []Standing, m Measure) *Leader {
	for _, s := range standings {
		if s.Measure == m {
			return s.Leader
		}
	}
	return nil
}
