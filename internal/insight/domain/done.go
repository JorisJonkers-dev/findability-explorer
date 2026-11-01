package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/site"
)

// Rules decide when a recommendation a run wrote is done (the spec's section 10.5), never a model:
// the next scan of tribelt.nl finds the term on the named page; it has a page for the family in
// the language; a web response that cites the listing names Tribelt or links to tribelt.nl; the
// readiness check passes. A correction of the fact sheet no scan can see, so it is ticked by hand.

// DoneFacts are what the newest scans and Prompt run say, which the rules read.
type DoneFacts struct {
	Pages   []Page
	Run     *Run
	Tribelt *Readiness
}

// ScanChecked reports whether a rule decides the recommendation is done: every kind but a fact
// sheet correction, which is ticked by hand.
func (r Recommendation) ScanChecked() bool { return r.Gap.Kind != GapFact }

// DoneByRule reports whether the rule of r's kind finds it done, with the parts it looked for;
// false and none for a recommendation ticked by hand.
func (r Recommendation) DoneByRule(f DoneFacts) (bool, []Part) {
	var parts []Part
	switch r.Gap.Kind {
	case GapTerm:
		parts = []Part{termPart(r, f.Pages)}
	case GapPage:
		name := fmt.Sprintf("a %s page about %s", LanguageNames[r.Gap.Language], FamilyNames[r.Gap.Family])
		parts = []Part{{Name: name, Present: hasPage(f.Pages, r.Gap.Family, r.Gap.Language)}}
	case GapListing:
		d := strings.TrimPrefix(r.Gap.Key, "listing:")
		parts = []Part{{Name: "a web response citing " + d + " that names Tribelt", Present: listingNames(f.Run, d)}}
	case GapReadiness:
		parts = readinessParts(r.Gap.Key, f.Tribelt)
	case GapFact:
		return false, nil
	}
	done := len(parts) > 0 && !slices.ContainsFunc(parts, func(p Part) bool { return !p.Present })
	return done, parts
}

// termPart is whether the term is on the page the recommendation names, or, without one, on any
// page of tribelt.nl's newest scan.
func termPart(r Recommendation, pages []Page) Part {
	term := r.Gap.Key
	if _, rest, ok := strings.Cut(term, ":"); ok {
		if _, text, ok := strings.Cut(rest, ":"); ok {
			term = text
		}
	}
	where := strings.TrimRight(r.Where, "/")
	name := fmt.Sprintf("%q on tribelt.nl", term)
	if where != "" {
		name = fmt.Sprintf("%q on %s", term, strings.TrimPrefix(strings.TrimPrefix(where, "https://"), "http://"))
	}
	found := slices.ContainsFunc(pages, func(p Page) bool {
		return (where == "" || strings.TrimRight(p.URL, "/") == where) && strings.Contains(fold.Fold(p.Text), fold.Fold(term))
	})
	return Part{Name: name, Present: found}
}

// listingNames is whether a web response of run that cites domain names Tribelt or cites tribelt.nl.
func listingNames(run *Run, domain string) bool {
	if run == nil {
		return false
	}
	return slices.ContainsFunc(run.Answers, func(a Answer) bool {
		cites := slices.ContainsFunc(a.Sources, func(s string) bool { return site.Domain(s) == domain })
		ownSite := slices.ContainsFunc(a.Sources, func(s string) bool { return site.Domain(s) == "tribelt.nl" })
		return a.Mode == ModeWeb && cites && (slices.Contains(a.Named, TribeltID) || ownSite)
	})
}

// readinessParts are what the readiness check of key looks for in tribelt.nl's readiness.
func readinessParts(key string, r *Readiness) []Part {
	if r == nil {
		r = &Readiness{}
	}
	switch key {
	case "readiness:crawlers":
		return []Part{{Name: "every AI crawler let in", Present: len(r.Blocked) == 0 && r.Bots != nil}}
	case "readiness:organization":
		return []Part{{Name: "Organization structured data", Present: r.Organization}}
	case "readiness:languages":
		return []Part{{Name: "pages in Dutch, German, and English", Present: r.LanguageVersions}}
	}
	return nil
}

// Edit is an administrator's wording and scores of a recommendation a run wrote.
type Edit struct {
	Title    string
	Where    string
	Why      string
	Steps    []string
	DoneWhen string
	Benefit  int
	Effort   int
}

// ErrNotEditable is returned for a recommendation no run wrote: one written by hand stays in code.
var ErrNotEditable = fmt.Errorf("%w: only a recommendation a Recommendation run wrote can be edited", ErrInvalid)

// Apply checks e and returns r with its wording and scores, edited at at. A page left blank is the
// web agency's to place.
func (r Recommendation) Apply(e Edit, at time.Time) (Recommendation, error) {
	title := strings.TrimSpace(e.Title)
	var steps []string
	for _, s := range e.Steps {
		if s = strings.TrimSpace(s); s != "" {
			steps = append(steps, s)
		}
	}
	switch {
	case title == "" || utf8.RuneCountInString(title) > maxTitle:
		return Recommendation{}, fmt.Errorf("%w: a title has 1 to %d characters", ErrInvalid, maxTitle)
	case utf8.RuneCountInString(e.Why) > maxWhy || utf8.RuneCountInString(e.DoneWhen) > maxDoneWhen || utf8.RuneCountInString(e.Where) > maxWhere:
		return Recommendation{}, fmt.Errorf("%w: the text is too long", ErrInvalid)
	case len(steps) > maxSteps || slices.ContainsFunc(steps, func(s string) bool { return utf8.RuneCountInString(s) > maxStep }):
		return Recommendation{}, fmt.Errorf("%w: at most %d steps of %d characters", ErrInvalid, maxSteps, maxStep)
	case e.Benefit < ScoreMin || e.Benefit > ScoreMax || e.Effort < ScoreMin || e.Effort > ScoreMax:
		return Recommendation{}, fmt.Errorf("%w: a score is from %d to %d", ErrInvalid, ScoreMin, ScoreMax)
	}
	r.Title, r.Why, r.Steps, r.DoneWhen, r.Benefit, r.Effort = title, strings.TrimSpace(e.Why), steps, strings.TrimSpace(e.DoneWhen), e.Benefit, e.Effort
	r.Where, r.WhereNote = strings.TrimSpace(e.Where), ""
	if r.Where == "" {
		r.WhereNote = ForTheAgency
	}
	r.EditedAt = at
	return r, nil
}
