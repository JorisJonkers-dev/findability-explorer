package domain

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

// ImprovementsVersion is the version of the improvements catalogue. A change to which
// improvements there are, or their order, is a new version, as the prompt set's is.
const ImprovementsVersion = 1

// AutoCheck is how a scan of tribelt.nl shows an improvement done; empty for one only an
// administrator can tick off.
type AutoCheck string

const (
	CheckNone           AutoCheck = ""
	CheckSitemaps       AutoCheck = "sitemaps"
	CheckLLMsFiles      AutoCheck = "llmsFiles"
	CheckMarkdown       AutoCheck = "markdown"
	CheckStructuredData AutoCheck = "structuredData"
)

// Improvement is one item of the plan's ordered list: what to do, the measure it moves, and how
// a scan checks it, if one can.
type Improvement struct {
	ID      string
	Title   string
	Moves   string
	Measure Measure
	Check   AutoCheck
}

// Improvements is the catalogue, in the order to work through it.
func Improvements() []Improvement {
	return []Improvement{
		{"submit-search-engines", "Submit tribelt.nl to Google and Bing", "AI answers in Gemini and Copilot", MeasureShareOfVoice, CheckNone},
		{"sitemap-per-language", "A sitemap for each language", "the pages search engines know of, in Dutch, German and English", MeasureSitemapPages, CheckSitemaps},
		{"llms-files", "Add llms.txt and llms-full.txt", "AI readiness", MeasureReadiness, CheckLLMsFiles},
		{"markdown-pages", "Offer each belt page as Markdown", "AI readiness", MeasureReadiness, CheckMarkdown},
		{"structured-data", "Mark up products and FAQs", "AI readiness and how answers describe Tribelt", MeasureReadiness, CheckStructuredData},
		{"buyer-words", "Use the words buyers search for", "the search terms used on the site", MeasureTermsUsed, CheckNone},
		{"family-pages", "One page per belt family in each language", "answers to German and English questions", MeasureShareOfVoice, CheckNone},
		{"listings", "Get listed where answers find sources", "how often web answers cite tribelt.nl", MeasureCitationShare, CheckNone},
	}
}

// Passes reports whether Tribelt's readiness shows the check done.
func (c AutoCheck) Passes(r Readiness) bool {
	switch c {
	case CheckSitemaps:
		return r.Sitemap && slices.Contains(r.SitemapLanguages, "nl") && slices.Contains(r.SitemapLanguages, "de") && slices.Contains(r.SitemapLanguages, "en")
	case CheckLLMsFiles:
		return r.LLMs && r.LLMsFull
	case CheckMarkdown:
		return r.Markdown
	case CheckStructuredData:
		return slices.ContainsFunc(r.StructuredData, func(t string) bool { return slices.Contains([]string{"Product", "ProductGroup", "ProductModel"}, t) }) &&
			slices.Contains(r.StructuredData, "FAQPage")
	case CheckNone:
	}
	return false
}

// Tick is an administrator's mark that an improvement is done.
type Tick struct {
	ImprovementID string
	Version       int
	By            string
	At            time.Time
}

// PlanItem is an improvement with whether it is done, how that is known, and the competitor to
// look at: the leader of its measure.
type PlanItem struct {
	Improvement
	Done bool
	// Auto is true when the latest scan of tribelt.nl decides Done, false when a tick does.
	Auto   bool
	Tick   *Tick
	Leader *Leader
}

// PlanItems are the improvements in order, each done by the latest scan of tribelt.nl when a scan
// can check it, or by an administrator's tick otherwise; each names its measure's leader.
func PlanItems(ticks []Tick, tribelt *Readiness, standings []Standing) []PlanItem {
	out := make([]PlanItem, 0, len(Improvements()))
	for _, imp := range Improvements() {
		item := PlanItem{Improvement: imp, Leader: LeaderOf(standings, imp.Measure)}
		if imp.Check != CheckNone {
			item.Auto = true
			item.Done = tribelt != nil && imp.Check.Passes(*tribelt)
		} else if i := slices.IndexFunc(ticks, func(t Tick) bool { return t.ImprovementID == imp.ID }); i >= 0 {
			item.Done, item.Tick = true, &ticks[i]
		}
		out = append(out, item)
	}
	return out
}

// Open are the items not done yet, in order, at most n.
func Open(items []PlanItem, n int) []PlanItem {
	var out []PlanItem
	for _, it := range items {
		if !it.Done && len(out) < n {
			out = append(out, it)
		}
	}
	return out
}

// Watched is a number to watch: its value now, when there is data, and its target.
type Watched struct {
	Target Target
	Now    *float64
	Goal   float64
}

// Watch is each target's value now, from the newest run and Tribelt's latest scan, against the goal.
func Watch(targets map[Target]float64, f Facts) []Watched {
	out := make([]Watched, 0, len(Targets()))
	for _, t := range Targets() {
		w := Watched{Target: t, Goal: targets[t]}
		if v, ok := f.watched(t); ok {
			w.Now = &v
		}
		out = append(out, w)
	}
	return out
}

func (f Facts) watched(t Target) (float64, bool) {
	switch t {
	case TargetShareOfVoice:
		return f.Value(MeasureShareOfVoice, TribeltID)
	case TargetTwenteShareOfVoice:
		if f.Run == nil {
			return 0, false
		}
		return ShareOfVoice(f.Run.Keep(func(a Answer) bool { return a.Scope == "twente" }), TribeltID)
	case TargetReadiness:
		return f.Value(MeasureReadiness, TribeltID)
	case TargetTermsUsed:
		return f.Value(MeasureTermsUsed, TribeltID)
	case TargetCitations:
	}
	// The web answers of the run that cite tribelt.nl.
	if f.Run == nil {
		return 0, false
	}
	return float64(len(f.Run.Keep(func(a Answer) bool { return a.Mode == ModeWeb && slices.Contains(a.Cited, TribeltID) }))), true
}

// Watchout is a competitor to keep an eye on, and why.
type Watchout string

const (
	// WatchMostNamed is the competitor the AI answers name most.
	WatchMostNamed Watchout = "mostNamed"
	// WatchBestPrepared is the competitor whose site passes the most readiness checks.
	WatchBestPrepared Watchout = "bestPrepared"
	// WatchTwente is the competitor the answers about Twente name most.
	WatchTwente Watchout = "twente"
)

// EyeOn is a competitor to keep an eye on.
type EyeOn struct {
	Why    Watchout
	Leader Leader
}

// KeepAnEyeOn are the competitors to watch: the one named most in AI answers, the best prepared
// site, and the strongest in Twente; one without data is left out.
func KeepAnEyeOn(companies []Company, f Facts, standings []Standing) []EyeOn {
	var out []EyeOn
	if l := LeaderOf(standings, MeasureShareOfVoice); l != nil && l.Value > 0 {
		out = append(out, EyeOn{Why: WatchMostNamed, Leader: *l})
	}
	if l := LeaderOf(standings, MeasureReadiness); l != nil {
		out = append(out, EyeOn{Why: WatchBestPrepared, Leader: *l})
	}
	if f.Run != nil {
		twente := Facts{Run: &Run{Answers: f.Run.Keep(func(a Answer) bool { return a.Scope == "twente" })}}
		if l := standing(MeasureShareOfVoice, companies, twente).Leader; l != nil && l.Value > 0 {
			out = append(out, EyeOn{Why: WatchTwente, Leader: *l})
		}
	}
	return out
}

// Errors the plan's changes return.
var (
	// ErrUnknownImprovement is returned for an improvement the catalogue does not hold.
	ErrUnknownImprovement = errors.New("no improvement has this id")
	// ErrCheckedByScan is returned when ticking an improvement a scan of tribelt.nl checks.
	ErrCheckedByScan = fmt.Errorf("%w: a scan of tribelt.nl checks this improvement", ErrInvalid)
)

// ImprovementByID returns the catalogue's improvement with id.
func ImprovementByID(id string) (Improvement, bool) {
	i := slices.IndexFunc(Improvements(), func(imp Improvement) bool { return imp.ID == id })
	if i < 0 {
		return Improvement{}, false
	}
	return Improvements()[i], true
}
