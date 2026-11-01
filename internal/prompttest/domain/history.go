package domain

import (
	"time"

	"github.com/google/uuid"
)

// Version is one published Prompt set as the history shows it: when it was published, how many
// questions it asks, and what it changed (VersionsOf).
type Version struct {
	Number      int
	PublishedAt time.Time
	Questions   int
	Note        string
}

// RunFigureInput is one run with how far it got and its measures.
type RunFigureInput struct {
	Run      Run
	Progress Progress
	Cells    []Cell
}

// RunFigures are one run's figures over time: the share of the responses read that name Tribelt,
// overall, per language and for Twente, and the share of web responses that cite tribelt.nl. A nil
// share had no responses to measure. A run that got fewer than half its responses is partial.
type RunFigures struct {
	RunID     uuid.UUID
	StartedAt time.Time
	Version   int
	Tasks     int
	Answered  int
	// Read is how many responses were read and count in the shares.
	Read    int
	Partial bool
	// ReadingModel is the model that read most of its responses, as the readings name it; empty
	// when they recorded none. A change of it breaks the charts.
	ReadingModel string

	All, Dutch, German, English, Twente, Citation *float64
}

// History gives the figures of each run with at least one response read, in the order given.
func History(runs []RunFigureInput) []RunFigures {
	out := []RunFigures{}
	for _, r := range runs {
		if f, ok := figuresOf(r); ok {
			out = append(out, f)
		}
	}
	return out
}

// figuresOf is a run's figures; false when it read no response.
func figuresOf(r RunFigureInput) (RunFigures, bool) {
	var all, twente, web share
	languages := map[string]*share{"nl": {}, "de": {}, "en": {}}
	for _, c := range r.Cells {
		all.add(c.Answers, c.Named)
		languages[LanguageOf(c.Scope)].add(c.Answers, c.Named)
		if c.Scope == ScopeTwente {
			twente.add(c.Answers, c.Named)
		}
		if c.Mode == ModeWeb {
			web.add(c.Answers, c.Cited)
		}
	}
	if all.of == 0 {
		return RunFigures{}, false
	}
	return RunFigures{
		RunID: r.Run.ID, StartedAt: r.Run.StartedAt, Version: r.Run.PromptSetVersion, Tasks: r.Run.Tasks,
		Answered: r.Progress.Answered, Read: all.of, Partial: r.Progress.Answered*2 < r.Run.Tasks,
		All: all.value(), Dutch: languages["nl"].value(), German: languages["de"].value(), English: languages["en"].value(),
		Twente: twente.value(), Citation: web.value(),
	}, true
}

// share counts how many of some responses do something.
type share struct{ of, yes int }

func (s *share) add(of, yes int) { s.of, s.yes = s.of+of, s.yes+yes }

// value is the share, nil when there was nothing to measure.
func (s *share) value() *float64 {
	if s.of == 0 {
		return nil
	}
	v := float64(s.yes) / float64(s.of)
	return &v
}

// CommonestFailure is the reason most of the failed answers give, the first alphabetically of a
// tie; "" when none failed.
func CommonestFailure(answers []Answer) string {
	n := map[string]int{}
	best := ""
	for _, a := range answers {
		if a.Failure == "" {
			continue
		}
		n[a.Failure]++
		if c := n[a.Failure]; c > n[best] || (c == n[best] && a.Failure < best) {
			best = a.Failure
		}
	}
	return best
}
