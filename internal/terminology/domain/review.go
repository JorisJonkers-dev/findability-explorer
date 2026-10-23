package domain

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// The review queue: phrases finished scans found that the map does not hold are candidates, and
// an administrator accepts one into the map, with the text, language, family and tier the map
// should show, or rejects it.

// Observation is one finished scan's finding of a phrase outside the map: which company, when,
// and how the scan's classifier judged it.
type Observation struct {
	ScanID      string
	CompanyID   string
	CompanyName string
	ScanURL     string
	StartedAt   time.Time
	Phrase      string
	Language    string
	Score       float64
	Pages       int
	Tier        string
	Family      string
	Closest     string
}

// Status is where a candidate is in the review.
type Status string

const (
	StatusPending  Status = "pending"
	StatusAccepted Status = "accepted"
	StatusRejected Status = "rejected"
)

// Decision is an administrator's decision on a candidate; Term is the term an accepted one became.
type Decision struct {
	CandidateID string
	Phrase      string
	Status      Status
	By          string
	At          time.Time
	Term        *Term
}

// CandidateCompany is a company whose scan found a candidate.
type CandidateCompany struct {
	ID   string
	Name string
}

// Candidate is a phrase scans found outside the map, with its evidence across companies.
type Candidate struct {
	ID        string
	Phrase    string
	Status    Status
	Languages []string
	Companies []CandidateCompany
	Scans     int
	BestScore float64
	MaxPages  int
	// SuggestedTier and SuggestedFamily are the classifier's most frequent judgement; the family
	// is empty when it named none.
	SuggestedTier   Tier
	SuggestedFamily Family
	Closest         string
	LastSeenAt      time.Time
	// Source is the start address of the newest scan that found it.
	Source   string
	Decision *Decision
}

// CandidateID is a candidate's id: the first 16 hex characters of the SHA-256 of its folded phrase.
func CandidateID(phrase string) string {
	sum := sha256.Sum256([]byte(Fold(phrase)))
	return hex.EncodeToString(sum[:])[:16]
}

// Candidates aggregates observations by phrase. A phrase with a decision takes its status; one
// without is pending, unless the map has it by now (the map grew since the scan). They come with
// the most companies first, then the highest score, then by phrase.
func Candidates(obs []Observation, decisions []Decision, terms []Term) []Candidate {
	inMap := map[string]bool{}
	for _, t := range terms {
		for _, alt := range spellings(t.Text) {
			inMap[alt] = true
		}
	}
	byID := map[string]*tally{}
	var order []string
	for _, o := range obs {
		id := CandidateID(o.Phrase)
		t, ok := byID[id]
		if !ok {
			t = &tally{candidate: Candidate{ID: id, Phrase: o.Phrase}, scans: map[string]bool{}, languages: map[string]int{}, judgements: map[judgement]int{}}
			byID[id] = t
			order = append(order, id)
		}
		t.add(o)
	}
	decided := map[string]Decision{}
	for _, d := range decisions {
		decided[d.CandidateID] = d
	}
	out := make([]Candidate, 0, len(order))
	for _, id := range order {
		c := byID[id].result()
		if d, ok := decided[id]; ok {
			c.Status, c.Decision = d.Status, &d
		} else if inMap[Fold(c.Phrase)] || inMap[singular(Fold(c.Phrase))] {
			continue
		}
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b Candidate) int {
		return cmp.Or(cmp.Compare(len(b.Companies), len(a.Companies)), cmp.Compare(b.BestScore, a.BestScore), strings.Compare(a.Phrase, b.Phrase))
	})
	return out
}

// spellings are the folded forms a term stands for: each " / " alternative without remarks in
// brackets, and its singular.
func spellings(text string) []string {
	var out []string
	for _, part := range strings.Split(text, " / ") {
		base, _, _ := strings.Cut(part, "(")
		if f := Fold(strings.Join(strings.Fields(base), " ")); f != "" {
			out = append(out, f, singular(f))
		}
	}
	return out
}

// singular drops a final plural s of the last word, as "belts" for "belt".
func singular(s string) string {
	if len(s) > 3 && strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") {
		return s[:len(s)-1]
	}
	return s
}

type judgement struct {
	tier    string
	family  string
	closest string
}

type tally struct {
	candidate  Candidate
	scans      map[string]bool
	languages  map[string]int
	judgements map[judgement]int
	companies  []CandidateCompany
}

func (t *tally) add(o Observation) {
	c := &t.candidate
	t.scans[o.ScanID] = true
	t.languages[o.Language]++
	t.judgements[judgement{tier: o.Tier, family: o.Family, closest: o.Closest}]++
	if !slices.ContainsFunc(t.companies, func(x CandidateCompany) bool { return x.ID == o.CompanyID }) {
		t.companies = append(t.companies, CandidateCompany{ID: o.CompanyID, Name: o.CompanyName})
	}
	c.BestScore = max(c.BestScore, o.Score)
	c.MaxPages = max(c.MaxPages, o.Pages)
	if !o.StartedAt.Before(c.LastSeenAt) {
		c.LastSeenAt, c.Source = o.StartedAt, o.ScanURL
	}
}

func (t *tally) result() Candidate {
	c := t.candidate
	c.Status = StatusPending
	c.Scans = len(t.scans)
	c.Languages = keysByCount(t.languages)
	c.Companies = slices.SortedFunc(slices.Values(t.companies), func(a, b CandidateCompany) int {
		return strings.Compare(Fold(a.Name), Fold(b.Name))
	})
	best, n := judgement{}, 0
	for j, count := range t.judgements {
		if count > n || (count == n && compareJudgement(j, best) < 0) {
			best, n = j, count
		}
	}
	c.SuggestedTier, c.SuggestedFamily, c.Closest = Tier(best.tier), Family(best.family), best.closest
	if _, err := ParseTier(best.tier); err != nil {
		c.SuggestedTier = TierBroad
	}
	if _, err := ParseFamily(best.family); err != nil {
		c.SuggestedFamily = ""
	}
	return c
}

func compareJudgement(a, b judgement) int {
	return cmp.Or(strings.Compare(a.tier, b.tier), strings.Compare(a.family, b.family), strings.Compare(a.closest, b.closest))
}

func keysByCount(counts map[string]int) []string {
	out := make([]string, 0, len(counts))
	for k := range counts {
		out = append(out, k)
	}
	slices.SortFunc(out, func(a, b string) int { return cmp.Or(cmp.Compare(counts[b], counts[a]), strings.Compare(a, b)) })
	return out
}

// Counts is how many candidates have each status.
type Counts struct {
	Pending  int
	Accepted int
	Rejected int
}

// Count counts candidates per status, and returns those with status.
func Count(all []Candidate, status Status) ([]Candidate, Counts) {
	var counts Counts
	out := []Candidate{}
	for _, c := range all {
		switch c.Status {
		case StatusPending:
			counts.Pending++
		case StatusAccepted:
			counts.Accepted++
		case StatusRejected:
			counts.Rejected++
		}
		if c.Status == status {
			out = append(out, c)
		}
	}
	return out, counts
}

// Errors of the review queue.
var (
	// ErrCandidateNotFound is returned for a candidate id no scan's phrase has.
	ErrCandidateNotFound = errors.New("no candidate has this id")
	// ErrDecided is returned when accepting or rejecting a candidate that was decided already.
	ErrDecided = errors.New("the candidate was decided already; undo that first")
	// ErrPending is returned when undoing the decision on a pending candidate.
	ErrPending = errors.New("the candidate has no decision to undo")
	// ErrTermExists is returned when the map has the term in that language already.
	ErrTermExists = errors.New("the terminology map has this term in this language already")
	// ErrInvalidTerm is returned for a term the map cannot hold.
	ErrInvalidTerm = errors.New("a term has between 1 and 200 characters, and each company name between 1 and 200, at most 20")
)

// Acceptance is the term an administrator makes of a candidate.
type Acceptance struct {
	Text     string
	Language string
	Family   string
	Tier     string
	UsedBy   []string
}

const (
	maxTermLength = 200
	maxUsedBy     = 20
)

// NewTerm checks an acceptance against the map's terms and returns the term it adds, sourced
// from source.
func NewTerm(a Acceptance, source string, terms []Term) (Term, error) {
	text := strings.Join(strings.Fields(a.Text), " ")
	if n := utf8.RuneCountInString(text); n == 0 || n > maxTermLength || len(a.UsedBy) > maxUsedBy {
		return Term{}, ErrInvalidTerm
	}
	usedBy := make([]string, 0, len(a.UsedBy))
	for _, u := range a.UsedBy {
		u = strings.TrimSpace(u)
		if n := utf8.RuneCountInString(u); n == 0 || n > maxTermLength {
			return Term{}, ErrInvalidTerm
		}
		usedBy = append(usedBy, u)
	}
	lang, err := ParseLanguage(a.Language)
	if err != nil {
		return Term{}, fmt.Errorf("%w: language", err)
	}
	family, err := ParseFamily(a.Family)
	if err != nil {
		return Term{}, fmt.Errorf("%w: family", err)
	}
	tier, err := ParseTier(a.Tier)
	if err != nil {
		return Term{}, fmt.Errorf("%w: tier", err)
	}
	if slices.ContainsFunc(terms, func(t Term) bool { return t.Language == lang && Fold(t.Text) == Fold(text) }) {
		return Term{}, ErrTermExists
	}
	if len([]rune(source)) > maxSourceLength {
		source = string([]rune(source)[:maxSourceLength])
	}
	return Term{Text: text, Language: lang, Family: family, Tier: tier, UsedBy: usedBy, Source: source, Origin: OriginReview}, nil
}

const maxSourceLength = 500
