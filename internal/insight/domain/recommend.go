package domain

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

// ForTheAgency is where a recommendation goes when the model could not place it on a tribelt.nl page.
const ForTheAgency = "Where to put this is for the web agency to decide."

// Written is what the model wrote for one gap: what to do, on which page, why, how, when it is done,
// and its scores, each at most one point from the rules' with a reason when moved.
type Written struct {
	Key         string
	Title       string
	Where       string
	Why         string
	Steps       []string
	DoneWhen    string
	Benefit     int
	Effort      int
	ScoreReason string
}

// Recommendation is one a Recommendation run wrote: placed and worded by the model from a gap.
type Recommendation struct {
	ID    string
	RunID string
	Gap   Gap
	// Where is the tribelt.nl page it belongs on; empty when the model could not place it, and
	// WhereNote then says so.
	Where       string
	WhereNote   string
	Title       string
	Why         string
	Steps       []string
	DoneWhen    string
	Benefit     int
	Effort      int
	ScoreReason string
	CreatedAt   time.Time
	// DoneBy and DoneAt mark one ticked off; EditedAt one an administrator edited. Either keeps it
	// when the next run replaces the others.
	DoneBy   string
	DoneAt   time.Time
	EditedAt time.Time
}

// Open reports whether a run may replace the recommendation: not done, not edited.
func (r Recommendation) Open() bool { return r.DoneAt.IsZero() && r.EditedAt.IsZero() }

// Limits of what the model writes, as the contract and the store keep them.
const (
	maxTitle    = 200
	maxWhy      = 500
	maxStep     = 300
	maxSteps    = 6
	maxDoneWhen = 300
	maxReason   = 300
	maxWhere    = 500
)

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:n-1])) + "…"
}

// within is the model's score, moved at most one point from the rules' and kept within bounds;
// moved reports whether it differs.
func within(rule, model int) (int, bool) {
	if model == 0 {
		return rule, false
	}
	score := clamp(max(rule-1, min(rule+1, model)))
	return score, score != rule
}

// Place makes a recommendation of each gap from what the model wrote for it. A page it names that
// the newest scan of tribelt.nl did not read, or none, leaves the page blank with ForTheAgency; a
// gap it wrote nothing for keeps the rules' draft. A score moves at most one point, and keeps the
// reason only when it moved.
func Place(gaps []Gap, written []Written, pages []Page, runID string, at time.Time, newID func() string) []Recommendation {
	byKey := map[string]Written{}
	for _, w := range written {
		if _, ok := byKey[w.Key]; !ok {
			byKey[w.Key] = w
		}
	}
	known := map[string]bool{}
	for _, p := range pages {
		known[strings.TrimRight(p.URL, "/")] = true
	}
	out := make([]Recommendation, 0, len(gaps))
	for _, g := range gaps {
		out = append(out, place(g, byKey[g.Key], known, Recommendation{ID: newID(), RunID: runID, CreatedAt: at}))
	}
	return out
}

// place fills r from gap g and what the model wrote for it.
func place(g Gap, w Written, known map[string]bool, r Recommendation) Recommendation {
	r.Gap, r.Title, r.Why, r.DoneWhen = g, firstOf(clip(w.Title, maxTitle), g.Title), firstOf(clip(w.Why, maxWhy), clip(strings.Join(g.Specifics, " "), maxWhy)), clip(w.DoneWhen, maxDoneWhen)
	for _, s := range w.Steps {
		if s = clip(s, maxStep); s != "" && len(r.Steps) < maxSteps {
			r.Steps = append(r.Steps, s)
		}
	}
	if where := strings.TrimSpace(w.Where); known[strings.TrimRight(where, "/")] {
		r.Where = clip(where, maxWhere)
	} else {
		r.WhereNote = ForTheAgency
	}
	benefit, bMoved := within(g.Benefit, w.Benefit)
	effort, eMoved := within(g.Effort, w.Effort)
	r.Benefit, r.Effort = benefit, effort
	if bMoved || eMoved {
		r.ScoreReason = clip(w.ScoreReason, maxReason)
	}
	return r
}

// firstOf is s, or else when s is empty.
func firstOf(s, otherwise string) string {
	if s == "" {
		return otherwise
	}
	return s
}

// Replacement is what a run does to the recommendations there are: the open ones it replaces, the
// done and edited ones it keeps, and the new ones it adds; a gap a kept one covers already gets
// no new one.
type Replacement struct {
	Replaced []Recommendation
	Kept     []Recommendation
	Added    []Recommendation
}

// Replace decides what a run that wrote fresh does to existing.
func Replace(existing, fresh []Recommendation) Replacement {
	var out Replacement
	covered := map[string]bool{}
	for _, r := range existing {
		if r.Open() {
			out.Replaced = append(out.Replaced, r)
		} else {
			out.Kept = append(out.Kept, r)
			covered[r.Gap.Key] = true
		}
	}
	for _, r := range fresh {
		if !covered[r.Gap.Key] {
			out.Added = append(out.Added, r)
		}
	}
	return out
}

// Replaces counts the open recommendations a run would replace now.
func Replaces(existing []Recommendation) int {
	return len(slices.DeleteFunc(slices.Clone(existing), func(r Recommendation) bool { return !r.Open() }))
}

// RecRun is one Recommendation run: when it ran, how it ended, the model, what it added, replaced
// and kept, and the tokens it used.
type RecRun struct {
	ID                        string
	StartedAt                 time.Time
	FinishedAt                time.Time
	Ending                    string
	By                        string
	Model                     string
	Gaps                      int
	Added, Replaced, Kept     int
	InputTokens, OutputTokens int64
}

// Moves says what a recommendation of each kind improves, and Measure the measure it moves.
var kindMoves = map[GapKind]struct { //nolint:gochecknoglobals // a constant table
	moves   string
	measure Measure
}{
	GapTerm:      {"Helps buyers who search for this term find Tribelt.", MeasureTermsUsed},
	GapPage:      {"Gives Tribelt a page where its competitors have one.", MeasureTermsUsed},
	GapListing:   {"Helps AI responses that cite this site name Tribelt.", MeasureCitationShare},
	GapFact:      {"Makes what AI responses say about Tribelt correct.", MeasureShareOfVoice},
	GapReadiness: {"Improves AI readiness.", MeasureReadiness},
}

// Item is the recommendation as the plan lists it: ticked off by hand until the rules decide when
// a written recommendation is done, and new when the newest run wrote it.
func (r Recommendation) Item(newest string, standings []Standing) PlanItem {
	k := kindMoves[r.Gap.Kind]
	item := PlanItem{
		Improvement: Improvement{
			ID: r.ID, Title: r.Title, Moves: k.moves, Measure: k.measure, Check: CheckNone, Why: r.Why, Steps: r.Steps, DoneWhen: r.DoneWhen,
			Benefit: r.Benefit, Effort: r.Effort,
		},
		Leader: LeaderOf(standings, k.measure), Tribelt: tribeltOn(standings, k.measure),
		Written: &r, New: r.RunID == newest && r.Open(),
	}
	if !r.DoneAt.IsZero() {
		item.Done, item.Tick = true, &Tick{ImprovementID: r.ID, By: r.DoneBy, At: r.DoneAt}
	}
	return item
}

// Typical tokens of the instructions and of one written recommendation, for an estimate.
const (
	typicalInstructions = 1500
	typicalWritten      = 300
)

// EstimateRun is what writing gaps should cost with model, a batch at a time: each request reads
// the instructions, its gaps and every page, and writes a recommendation per gap. Text is counted
// at four characters a token.
func EstimateRun(gaps []Gap, pages []Page, batch int, model string) llm.Micros {
	if len(gaps) == 0 {
		return 0
	}
	pageChars := 0
	for _, p := range pages {
		pageChars += len(p.URL) + len(p.Title) + len(p.Text)
	}
	gapChars := 0
	for _, g := range gaps {
		gapChars += len(g.Title) + len(strings.Join(g.Specifics, " ")) + 60
	}
	requests := int64((len(gaps) + batch - 1) / batch)
	in := requests*(typicalInstructions+int64(pageChars/4)) + int64(gapChars/4)
	out := int64(len(gaps)) * typicalWritten
	return llm.ListPrice(model).Of(in, out)
}

// WithWritten adds the recommendations the runs wrote to the plan's items, best value first as
// ByValue orders them; newest is the newest finished run, whose open recommendations are new.
func WithWritten(items []PlanItem, recs []Recommendation, newest string, standings []Standing) []PlanItem {
	out := slices.Clone(items)
	for _, r := range recs {
		out = append(out, r.Item(newest, standings))
	}
	slices.SortStableFunc(out, func(a, b PlanItem) int {
		if c := b.Benefit*a.Effort - a.Benefit*b.Effort; c != 0 {
			return c
		}
		return b.Benefit - a.Benefit
	})
	return out
}

// NewestDone is the id of the newest run of runs (newest first) that ended done; "" for none.
func NewestDone(runs []RecRun) string {
	for _, r := range runs {
		if r.Ending == "done" {
			return r.ID
		}
	}
	return ""
}

// Kind is the item's kind: a written recommendation's gap's, or for one written by hand, AI
// readiness when a scan checks it and general otherwise.
func (it PlanItem) Kind() string {
	switch {
	case it.Written != nil:
		return string(it.Written.Gap.Kind)
	case it.Auto:
		return string(GapReadiness)
	default:
		return "general"
	}
}
