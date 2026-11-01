package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// DraftQuestion is one question of a draft: the id of the published question it keeps (changed or
// not), or "" for a new one; its belt family, buyer intent, scope, and text, with its English
// translation when one is known.
type DraftQuestion struct {
	ID      string
	Family  Family
	Intent  Intent
	Scope   Scope
	Text    string
	English string
}

// Draft is the Prompt set an administrator edited and publishes: its questions in order, the new
// scopes it asks in, and the version it was restored from (0 for none).
type Draft struct {
	Questions    []DraftQuestion
	Scopes       []ScopeInfo
	RestoredFrom int
}

// DraftError is why a draft cannot be published, in words an administrator can act on.
type DraftError struct{ Reason string }

func (e DraftError) Error() string { return e.Reason }

// ErrVersionTaken is returned when another publication stored the version number first.
var ErrVersionTaken = errors.New("another publication took this version number first; publish again")

// ErrNothingChanged is returned when the draft asks exactly what the newest version asks.
var ErrNothingChanged = errors.New("the draft asks what the published Prompt set asks")

// MaxQuestions is the most questions a Prompt set may ask.
const MaxQuestions = 1000

var (
	scopeIDPattern  = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,39}$`)
	languagePattern = regexp.MustCompile(`^[a-z]{2}$`)
	promptIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,79}$`)
)

// AskedIntents are the buyer intents a question of the Prompt test may have: the kinds its answers
// record and the results show.
func AskedIntents() []Intent { return []Intent{IntentProduct, IntentApplication, IntentProblem} }

// Publish makes the next version of the Prompt set from a draft: it checks the draft against the
// scopes known so far, gives each new question an id no version has used, says in its note what
// changed against the newest version, and returns the new scopes to store with it.
func Publish(sets []PublishedSet, known []ScopeInfo, d Draft, by string, at time.Time) (PublishedSet, []ScopeInfo, error) {
	scopes, err := scopesOf(known, d.Scopes)
	if err != nil {
		return PublishedSet{}, nil, err
	}
	if len(d.Questions) == 0 {
		return PublishedSet{}, nil, DraftError{"A Prompt set asks at least one question."}
	}
	if len(d.Questions) > MaxQuestions {
		return PublishedSet{}, nil, DraftError{fmt.Sprintf("A Prompt set asks at most %d questions.", MaxQuestions)}
	}
	used := map[string]bool{}
	for _, s := range sets {
		for _, q := range s.Questions {
			used[q.ID] = true
		}
	}
	questions, err := draftQuestions(d.Questions, scopes, used)
	if err != nil {
		return PublishedSet{}, nil, err
	}
	newest, version := newestOf(sets)
	if len(d.Scopes) == 0 && sameQuestions(newest.Questions, questions) {
		return PublishedSet{}, nil, ErrNothingChanged
	}
	return PublishedSet{
		Version: version + 1, PublishedAt: at, PublishedBy: by, Note: noteOf(newest.Questions, questions, d), Questions: questions,
	}, d.Scopes, nil
}

// scopesOf checks the new scopes and returns every scope by id.
func scopesOf(known, added []ScopeInfo) (map[Scope]ScopeInfo, error) {
	byID := map[Scope]ScopeInfo{}
	names := map[string]bool{}
	for _, s := range known {
		byID[s.ID] = s
		names[s.Name] = true
	}
	for _, s := range added {
		switch {
		case !scopeIDPattern.MatchString(string(s.ID)):
			return nil, DraftError{fmt.Sprintf("A scope's id is lowercase letters, digits and dashes; %q is not.", s.ID)}
		case !languagePattern.MatchString(s.Language):
			return nil, DraftError{fmt.Sprintf("A scope's language is a two-letter code; %q is not.", s.Language)}
		case len([]rune(s.Region)) > 100:
			return nil, DraftError{"A scope's region has at most 100 characters."}
		case strings.TrimSpace(s.Name) == "" || len([]rune(s.Name)) > 100:
			return nil, DraftError{"A scope's name has 1 to 100 characters."}
		}
		if _, ok := byID[s.ID]; ok {
			return nil, DraftError{fmt.Sprintf("The scope %s exists already.", s.ID)}
		}
		if names[s.Name] {
			return nil, DraftError{fmt.Sprintf("A scope is called %s already.", s.Name)}
		}
		byID[s.ID] = s
		names[s.Name] = true
	}
	return byID, nil
}

// draftQuestions checks the questions and gives each new one an id; used holds every id a version
// has used, so a new question never takes an old one's.
func draftQuestions(in []DraftQuestion, scopes map[Scope]ScopeInfo, used map[string]bool) ([]Question, error) {
	taken := map[string]bool{}
	for _, q := range in {
		if q.ID == "" {
			continue
		}
		if !promptIDPattern.MatchString(q.ID) {
			return nil, DraftError{fmt.Sprintf("A question's id is lowercase letters, digits and dashes; %q is not.", q.ID)}
		}
		if taken[q.ID] {
			return nil, DraftError{fmt.Sprintf("The question %s is in the draft twice.", q.ID)}
		}
		taken[q.ID] = true
	}
	out := make([]Question, 0, len(in))
	for _, q := range in {
		question, err := checkQuestion(q, scopes)
		if err != nil {
			return nil, err
		}
		if question.ID == "" {
			question.ID = freshID(fmt.Sprintf("%s-%s-%s", q.Family, KindOf(q.Intent), q.Scope), used, taken)
			taken[question.ID] = true
		}
		out = append(out, question)
	}
	return out, nil
}

func checkQuestion(q DraftQuestion, scopes map[Scope]ScopeInfo) (Question, error) {
	text, english := strings.TrimSpace(q.Text), strings.TrimSpace(q.English)
	scope, ok := scopes[q.Scope]
	switch {
	case !isOneOf(q.Family, Families()):
		return Question{}, DraftError{fmt.Sprintf("%q is no belt family.", q.Family)}
	case !isOneOf(q.Intent, AskedIntents()):
		return Question{}, DraftError{fmt.Sprintf("A question's buyer intent is product, application or problem; %q is not.", q.Intent)}
	case !ok:
		return Question{}, DraftError{fmt.Sprintf("The scope %s does not exist.", q.Scope)}
	case text == "" || len([]rune(text)) > 1000:
		return Question{}, DraftError{"A question has 1 to 1,000 characters."}
	case len([]rune(english)) > 1000:
		return Question{}, DraftError{"A question's English has at most 1,000 characters."}
	}
	return Question{ID: q.ID, Family: q.Family, Intent: q.Intent, Scope: scope, Text: text, English: english}, nil
}

func isOneOf[T comparable](v T, all []T) bool {
	for _, x := range all {
		if x == v {
			return true
		}
	}
	return false
}

// freshID is base, or base with the first number that makes it an id no version and no question of
// the draft has.
func freshID(base string, used, taken map[string]bool) string {
	if len(base) > 70 {
		base = strings.TrimRight(base[:70], "-")
	}
	id := base
	for n := 2; used[id] || taken[id]; n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}

// newestOf is the newest published set and its version; version 0 when none is published.
func newestOf(sets []PublishedSet) (PublishedSet, int) {
	var newest PublishedSet
	for _, s := range sets {
		if s.Version > newest.Version {
			newest = s
		}
	}
	return newest, newest.Version
}

func sameQuestions(a, b []Question) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID || !sameQuestion(a[i], b[i]) {
			return false
		}
	}
	return true
}

func sameQuestion(a, b Question) bool {
	return a.Family == b.Family && a.Intent == b.Intent && a.Scope.ID == b.Scope.ID && a.Text == b.Text && a.English == b.English
}

// noteOf says what the new version changes against the newest: "Restores version 1. Adds 2
// questions, changes 1, and removes none. Adds the scope French, Lyon."
func noteOf(old, now []Question, d Draft) string {
	before := map[string]Question{}
	for _, q := range old {
		before[q.ID] = q
	}
	added, changed := 0, 0
	kept := map[string]bool{}
	for _, q := range now {
		prev, ok := before[q.ID]
		switch {
		case !ok:
			added++
		case !sameQuestion(prev, q):
			changed++
		}
		kept[q.ID] = true
	}
	removed := 0
	for _, q := range old {
		if !kept[q.ID] {
			removed++
		}
	}
	var b strings.Builder
	if d.RestoredFrom > 0 {
		fmt.Fprintf(&b, "Restores version %d. ", d.RestoredFrom)
	}
	fmt.Fprintf(&b, "Adds %s, changes %s, and removes %s.", count(added, "question", "questions"), count(changed, "", ""), count(removed, "", ""))
	if len(d.Scopes) > 0 {
		names := make([]string, 0, len(d.Scopes))
		for _, s := range d.Scopes {
			names = append(names, s.Name)
		}
		label := "the scope"
		if len(names) > 1 {
			label = "the scopes"
		}
		fmt.Fprintf(&b, " Adds %s %s.", label, strings.Join(names, "; "))
	}
	note := b.String()
	if r := []rune(note); len(r) > 300 {
		note = string(r[:299]) + "…"
	}
	return note
}

// count is "none", "1 question", or "3 questions"; without words, "none", "1", or "3".
func count(n int, one, many string) string {
	switch {
	case n == 0:
		return "none"
	case one == "":
		return fmt.Sprint(n)
	case n == 1:
		return "1 " + one
	default:
		return fmt.Sprintf("%d %s", n, many)
	}
}
