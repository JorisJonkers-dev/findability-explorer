package domain

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/lang"
)

// ImprovementsVersion is the version of the improvements catalogue. A change to which
// improvements there are, or their order, is a new version, as the prompt set's is.
const ImprovementsVersion = 2

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

// Improvement is one item of the plan's ordered list: what to do, why, how, when it is done, the
// measure it moves (Moves says what it improves, as a sentence), and how a scan checks it, if one
// can.
type Improvement struct {
	ID       string
	Title    string
	Moves    string
	Measure  Measure
	Check    AutoCheck
	Why      string
	Steps    []string
	DoneWhen string
	// Benefit is how much it helps, and Effort how much work it takes, each from ScoreMin to ScoreMax.
	Benefit int
	Effort  int
}

// ScoreMin and ScoreMax bound an improvement's benefit and effort.
const (
	ScoreMin = 1
	ScoreMax = 5
)

// Improvements is the catalogue, in the order to work through it.
func Improvements() []Improvement {
	return []Improvement{
		{
			ID: "submit-search-engines", Title: "Submit tribelt.nl to Google and Bing", Moves: "Helps Tribelt appear in Gemini and Copilot answers.",
			Benefit: 4, Effort: 1,
			Measure: MeasureShareOfVoice, Check: CheckNone,
			Why: "Gemini answers from Google's index and Copilot from Bing's. A page these search engines don't know cannot turn up in their answers.",
			Steps: []string{
				"Add tribelt.nl to Google Search Console and to Bing Webmaster Tools, and confirm you own the domain.",
				"Submit the sitemap in both.",
				"A week later, check that the belt pages are indexed, and fix any page they leave out.",
			},
			DoneWhen: "Both search engines show tribelt.nl as confirmed and have read its sitemap.",
		},
		{
			ID: "sitemap-per-language", Title: "A sitemap for each language", Moves: "Helps search engines find every page in Dutch, German and English.",
			Benefit: 2, Effort: 2,
			Measure: MeasureSitemapCoverage, Check: CheckSitemaps,
			Why: "Search engines and AI crawlers find pages through the sitemap. German and English pages that no sitemap lists are found late, or not at all.",
			Steps: []string{
				"List every Dutch, German and English page in a sitemap, each with its versions in the other languages.",
				"Name the sitemaps in robots.txt.",
				"Submit them to Google and Bing.",
			},
			DoneWhen: "The sitemaps of tribelt.nl list pages in Dutch, German and English.",
		},
		{
			ID: "llms-files", Title: "Add llms.txt and llms-full.txt", Moves: "Improves AI readiness.",
			Benefit: 3, Effort: 1,
			Measure: MeasureReadiness, Check: CheckLLMsFiles,
			Why: "llms.txt gives AI tools a short map of the site in plain text, and llms-full.txt gives them the content itself. Both help an assistant describe Tribelt's belts correctly.",
			Steps: []string{
				"Write llms.txt with a line about Tribelt, then a link and one sentence for each belt family and each main page.",
				"Put the text of those pages together in llms-full.txt.",
				"Publish both at the root of tribelt.nl, next to robots.txt.",
			},
			DoneWhen: "tribelt.nl/llms.txt and tribelt.nl/llms-full.txt both open.",
		},
		{
			ID: "markdown-pages", Title: "Offer each belt page as Markdown", Moves: "Improves AI readiness.",
			Benefit: 2, Effort: 3,
			Measure: MeasureReadiness, Check: CheckMarkdown,
			Why: "AI tools read plain text more reliably than a page full of menus and scripts. A Markdown version of a belt page gives them the facts without the clutter.",
			Steps: []string{
				"Publish a Markdown version of each belt page at the same address ending in .md, and of the home page at tribelt.nl/index.md.",
				"Link to the Markdown versions from llms.txt.",
			},
			DoneWhen: "tribelt.nl/index.md opens, or llms.txt links to Markdown pages.",
		},
		{
			ID: "structured-data", Title: "Mark up products and FAQs", Moves: "Improves AI readiness, and helps AI answers describe Tribelt correctly.",
			Benefit: 3, Effort: 3,
			Measure: MeasureReadiness, Check: CheckStructuredData,
			Why: "Markup tells search engines and AI tools what a page is about: the product, its specifications and the questions it answers. They use it to describe a company in their answers.",
			Steps: []string{
				"Add Product markup to each belt page, with the name, a description and the main specifications.",
				"Add FAQPage markup to a page with buyers' common questions and their answers.",
				"Test the pages with Google's Rich Results Test.",
			},
			DoneWhen: "The pages of tribelt.nl carry both Product and FAQ markup.",
		},
		{
			ID: "buyer-words", Title: "Use the words buyers search for", Moves: "Increases the number of search terms tribelt.nl uses.",
			Benefit: 4, Effort: 3,
			Measure: MeasureTermsUsed, Check: CheckNone,
			Why: "Buyers search in their own words, which are often not Tribelt's product names. A page that uses their words turns up in their searches and in AI answers to them.",
			Steps: []string{
				"Open Search terms and pick the most used terms for each belt family that tribelt.nl does not use yet.",
				"Work them into the belt pages' headings and text, in each language.",
				"Scan tribelt.nl again and check the number of terms used goes up.",
			},
			DoneWhen: "Each belt family's page uses the terms buyers search for most.",
		},
		{
			ID: "family-pages", Title: "One page per belt family in each language", Moves: "Helps Tribelt appear in answers to German and English questions.",
			Benefit: 5, Effort: 5,
			Measure: MeasureShareOfVoice, Check: CheckNone,
			Why: "Answers to German and English questions draw on pages in those languages. Without a page per belt family in each language, those answers name competitors instead.",
			Steps: []string{
				"Write a page for each belt family in Dutch, German and English, rather than one translated overview.",
				"Use the terms buyers in each language search for.",
				"Link each page to its versions in the other languages.",
			},
			DoneWhen: "Each of the " + InWords(len(Families())) + " belt families has its own page in all " + InWords(len(lang.Focus())) + " languages.",
		},
		{
			ID: "listings", Title: "Get listed where answers find sources", Moves: "Makes web answers cite tribelt.nl more often.",
			Benefit: 5, Effort: 4,
			Measure: MeasureCitationShare, Check: CheckNone,
			Why: "Web answers cite sources they trust, such as trade directories, industry associations and articles. A listing there gives answers a reason to cite tribelt.nl.",
			Steps: []string{
				"Open the Prompt test and note the sites the web answers cite for belt questions.",
				"Get Tribelt listed on those sites, with a link to the right belt page.",
				"Ask partners and customers who mention Tribelt to link to tribelt.nl.",
			},
			DoneWhen: "Tribelt is listed on the sites the web answers cite most.",
		},
	}
}

// ByValue returns the improvements best value first: the higher the benefit against the effort, the
// earlier. A tie goes to the larger benefit, then to the catalogue's order.
func ByValue(imps []Improvement) []Improvement {
	out := slices.Clone(imps)
	slices.SortStableFunc(out, func(a, b Improvement) int {
		if c := b.Benefit*a.Effort - a.Benefit*b.Effort; c != 0 {
			return c
		}
		return b.Benefit - a.Benefit
	})
	return out
}

// InWords is a small count in words, as the catalogue writes it: "five"; a larger one in digits.
func InWords(n int) string {
	words := []string{"no", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}
	if n >= 0 && n < len(words) {
		return words[n]
	}
	return strconv.Itoa(n)
}

// Manual reports whether only an administrator can tick the improvement off: no scan checks it.
func (imp Improvement) Manual() bool { return imp.Check == CheckNone }

// Part is one thing a scan check looks for on tribelt.nl, and whether the scan found it.
type Part struct {
	Name    string
	Present bool
}

// Parts are what the check looks for in Tribelt's readiness, each found or missing; nil for an
// improvement no scan checks.
func (c AutoCheck) Parts(r Readiness) []Part {
	switch c {
	case CheckSitemaps:
		has := func(lang string) bool { return r.Sitemap && slices.Contains(r.SitemapLanguages, lang) }
		return []Part{{"a Dutch sitemap", has("nl")}, {"a German sitemap", has("de")}, {"an English sitemap", has("en")}}
	case CheckLLMsFiles:
		return []Part{{"llms.txt", r.LLMs}, {"llms-full.txt", r.LLMsFull}}
	case CheckMarkdown:
		return []Part{{"Markdown pages", r.Markdown}}
	case CheckStructuredData:
		return []Part{{"Product markup", r.Product}, {"FAQ markup", slices.Contains(r.StructuredData, "FAQPage")}}
	case CheckNone:
	}
	return nil
}

// Passes reports whether Tribelt's readiness shows the check done: every part found.
func (c AutoCheck) Passes(r Readiness) bool {
	parts := c.Parts(r)
	return len(parts) > 0 && !slices.ContainsFunc(parts, func(p Part) bool { return !p.Present })
}

// Tick is an administrator's mark that an improvement is done.
type Tick struct {
	ImprovementID string
	Version       int
	By            string
	At            time.Time
}

// PlanItem is an improvement with whether it is done, how that is known, Tribelt's value on its
// measure, and the competitor to look at: the leader of that measure.
type PlanItem struct {
	Improvement
	Done bool
	// Auto is true when the latest scan of tribelt.nl decides Done, false when a tick does.
	Auto bool
	// Parts are what the latest scan of tribelt.nl found of a scanned improvement; nil otherwise.
	Parts   []Part
	Tick    *Tick
	Tribelt *float64
	Leader  *Leader
}

// PlanItems are the improvements in order, each done by the latest scan of tribelt.nl when a scan
// can check it, or by an administrator's tick otherwise; each names its measure's leader.
func PlanItems(ticks []Tick, tribelt *Readiness, standings []Standing) []PlanItem {
	out := make([]PlanItem, 0, len(Improvements()))
	for _, imp := range ByValue(Improvements()) {
		item := PlanItem{Improvement: imp, Leader: LeaderOf(standings, imp.Measure), Tribelt: tribeltOn(standings, imp.Measure)}
		if !imp.Manual() {
			item.Auto = true
			if tribelt != nil {
				item.Parts, item.Done = imp.Check.Parts(*tribelt), imp.Check.Passes(*tribelt)
			}
		} else if i := slices.IndexFunc(ticks, func(t Tick) bool { return t.ImprovementID == imp.ID }); i >= 0 {
			item.Done, item.Tick = true, &ticks[i]
		}
		out = append(out, item)
	}
	return out
}

// tribeltOn is Tribelt's value on a measure, nil without data.
func tribeltOn(standings []Standing, m Measure) *float64 {
	i := slices.IndexFunc(standings, func(s Standing) bool { return s.Measure == m })
	if i < 0 {
		return nil
	}
	return standings[i].Tribelt
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
	// Met is whether the value now has reached the target: it is no goal any more until raised.
	Met bool
}

// Watch is each target's value now, from the newest run and Tribelt's latest scan, against the goal.
func Watch(targets map[Target]float64, f Facts) []Watched {
	out := make([]Watched, 0, len(Targets()))
	for _, t := range Targets() {
		w := Watched{Target: t, Goal: targets[t]}
		if v, ok := f.watched(t); ok {
			w.Now, w.Met = &v, v >= w.Goal
		}
		out = append(out, w)
	}
	return out
}

// watched is a target's value now, by the same functions the standings and the overview use.
func (f Facts) watched(t Target) (float64, bool) {
	switch t {
	case TargetShareOfVoice:
		return f.Value(MeasureShareOfVoice, TribeltID)
	case TargetReadiness:
		return f.Value(MeasureReadiness, TribeltID)
	case TargetTermsUsed:
		return f.Value(MeasureTermsUsed, TribeltID)
	case TargetTwenteShareOfVoice, TargetCitations:
	}
	if f.Run == nil {
		return 0, false
	}
	if t == TargetTwenteShareOfVoice {
		return TwenteShare(*f.Run, TribeltID)
	}
	return float64(Citations(f.Run.Answers, TribeltID)), true
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
		twente := Facts{Run: &Run{Answers: f.Run.Keep(func(a Answer) bool { return a.Scope == ScopeTwente })}}
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
