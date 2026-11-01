package domain

import (
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// Claude Haiku checks each company the answers name against Tribelt's target market, from evidence
// beyond the name, and decides what it is sure about (ADR-0009): a sure competitor with a website
// joins the competitor list, a sure non-competitor is dismissed, and the rest wait for a person.

// The definition every check is made against is the competitor definition, stored as versions
// (Definition); a check records the version it was made against.

// ModelDecider is who a decision by the language model names, as for its merges.
const ModelDecider = ModelMerger

// Decides is the decision a verdict makes on candidate c, by the language model: a sure competitor
// with a website is added, a sure non-competitor is dismissed. An unsure verdict, or a competitor
// without a website to add it with, makes none: the candidate waits for a person.
func (v CompanyVerdict) Decides(c Candidate) (CandidateStatus, bool) {
	switch {
	case !v.Sure:
		return CandidatePending, false
	case !v.Competitor:
		return CandidateDismissed, true
	case c.Website == "":
		return CandidatePending, false
	default:
		return CandidateAdded, true
	}
}

// HomePage is what a company's home page says of it, clipped short; Problem says why it could not
// be read, and is "" when it was.
type HomePage struct {
	URL         string
	Title       string
	Description string
	Headings    []string
	Text        string
	Problem     string
}

// Dossier is what the model reads of a company it checks: the company with its names and websites,
// what the answers say around its names, and its home page; Home is nil without a website.
type Dossier struct {
	Candidate Candidate
	Snippets  []string
	Home      *HomePage
}

// ReviewUsage is what one request to the model consumed, and when.
type ReviewUsage struct {
	InputTokens  int64
	OutputTokens int64
	At           time.Time
}

const (
	// MaxQuoted is how many answers a company is quoted from.
	MaxQuoted = 3
	// snippetReach is how many bytes of an answer a snippet shows on each side of the name.
	snippetReach = 160
)

// Snippets quotes each answer where it names the company by one of names: the words around the
// first name found, the whitespace collapsed, with an ellipsis where the answer goes on. An answer
// that gives none of the names is left out, and so is a snippet quoted already.
func Snippets(answers, names []string) []string {
	var patterns []*regexp.Regexp
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			patterns = append(patterns, regexp.MustCompile(`(?i)`+regexp.QuoteMeta(n)))
		}
	}
	var out []string
	for _, a := range answers {
		for _, p := range patterns {
			loc := p.FindStringIndex(a)
			if loc == nil {
				continue
			}
			if s := snippet(a, loc[0], loc[1]); !slices.Contains(out, s) {
				out = append(out, s)
			}
			break
		}
	}
	return out
}

// snippet is text from about snippetReach bytes before start to as far after end, on whole words.
func snippet(text string, start, end int) string {
	from, to := max(0, start-snippetReach), min(len(text), end+snippetReach)
	for from > 0 && !utf8.RuneStart(text[from]) {
		from--
	}
	for to < len(text) && !utf8.RuneStart(text[to]) {
		to++
	}
	part := text[from:to]
	if from > 0 {
		if i := strings.IndexAny(part, " \n\t"); i >= 0 && i < start-from {
			part = part[i:]
		}
		part = "…" + strings.TrimLeft(part, " \n\t")
	}
	if to < len(text) {
		if i := strings.LastIndexAny(part, " \n\t"); i >= len(part)-(to-end) {
			part = part[:i]
		}
		part = strings.TrimRight(part, " \n\t") + "…"
	}
	return strings.Join(strings.Fields(part), " ")
}
