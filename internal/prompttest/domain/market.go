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

// TargetMarket is who Tribelt sells to and who competes with it: the definition every check of a
// company is made against. It was researched from tribelt.nl's sector and product pages and the
// client interviews. Changing it changes what the check means, so raise CheckVersion with it.
const TargetMarket = `Tribelt B.V. in Haaksbergen, the Netherlands, is part of the Lesjöfors Group, as is its sister company De Spiraal. It designs and makes metal conveyor belts and their drive elements in its own factory.

Its belts are eyelink (wire mesh) belts, open spiral belts (also called balanced weave), closed spiral belts (also called compound balanced), Tri-Flex stainless steel curve-going belts for spiral freezer and cooler towers, and chain, ladder, plate link, honeycomb, interlaced wire and woven baking belts. Its drive elements are sprockets, tube drums and cage drums.

The belts are for process conveying, where a product is treated while it moves. That is heat in ovens, dryers, furnaces, heat treatment, sintering and glass firing, up to about 1,150 °C. It is cold in spiral freezers and coolers and in cryogenic freezing. It is wet and hygienic steps such as washing, blanching, frying, pasteurising and sterilisation tunnels. And it is heavy, sharp or abrasive loads in recycling, waste and water filtration.

Its customers are industrial food producers of bread and pastry, meat, fish and dairy, potatoes, fruit and vegetables, and meat substitutes. They are the builders of ovens, freezers and lines and the system integrators who build these belts into their machines. And they are industry, such as recycling, waste treatment, water filtration, automotive paint and washing lines, solar and electronics firing furnaces, and pharma and medical sterilisation. Its markets are in Europe, above all the Netherlands and Germany, and English-speaking buyers.

A competitor makes metal conveyor belts (wire, wire mesh, stainless steel link, metal chain or plate belts) for those process uses and sells them to those customers, or has them made and sells them under its own name. A maker anywhere in the world that sells into Europe counts.

These are not competitors. A maker of rubber, textile or steel-cord belting for mining, quarrying and bulk handling is not. A maker of lightweight fabric, PVC or polyurethane belts for packaging, logistics and light conveying is not. A maker of conveyor systems, ovens or freezers that buys its belts rather than making them is a customer or partner, not a competitor. A distributor or trader that does not make belts is not. A maker of wire mesh for fencing, screening, sieving or architecture rather than conveyor belts is not. A directory, marketplace or publisher is not.

A maker of plastic modular belts used in the same food lines, such as in spiral freezers, makes a substitute, not the same product. It is not a competitor unless it also makes metal process belts, and the reason says it makes plastic modular belts.`

// CheckVersion is the version of the check: of TargetMarket and of the evidence the model reads.
// A waiting company whose check has an older version is checked again. Version 1 was the check of
// ADR-0006, by name and website alone.
const CheckVersion = 2

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
