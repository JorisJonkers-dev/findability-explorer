package domain

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// Sighting is one name an answer gave that the extractor could not match to the company list.
type Sighting struct {
	Name     string
	URL      string
	Provider string
	Model    string
	At       time.Time
}

// CandidateStatus is where a competitor candidate is in the review.
type CandidateStatus string

const (
	CandidatePending   CandidateStatus = "pending"
	CandidateAdded     CandidateStatus = "added"
	CandidateDismissed CandidateStatus = "dismissed"
)

// CandidateDecision is an administrator's decision on a candidate. An added candidate names the
// company it became.
type CandidateDecision struct {
	ID           string
	Name         string
	Status       CandidateStatus
	CompetitorID string
	By           string
	At           time.Time
}

// Candidate is a company the answers name that is not on the competitor list: how often and by
// which providers' models it was named, the addresses given for it, and the decision on it.
type Candidate struct {
	ID string
	// Name is how the answers spell it most often.
	Name      string
	Answers   int
	Providers []string
	// Models are the exact models that named it.
	Models    []string
	Addresses []string
	LastSeen  time.Time
	Decision  *CandidateDecision
}

// Status is the candidate's status: its decision's, or pending.
func (c Candidate) Status() CandidateStatus {
	if c.Decision == nil {
		return CandidatePending
	}
	return c.Decision.Status
}

// Candidates gathers the sightings into one candidate per name key, leaving out a name the
// company list now holds unless a decision on it is still on record (an added candidate stays
// listed, so the add can be undone). Most answers first, then by name.
func Candidates(sightings []Sighting, decisions []CandidateDecision, directory []DirectoryCompany) []Candidate {
	g := gathering{byID: map[string]*Candidate{}, spellings: map[string]map[string]int{}, decided: map[string]CandidateDecision{}}
	for _, d := range decisions {
		g.decided[d.ID] = d
	}
	for _, s := range sightings {
		key := NameKey(s.Name)
		id := CandidateID(key)
		if _, ok := g.decided[id]; key == "" || (!ok && Match(Named{Name: s.Name, URL: s.URL}, directory) != "") {
			continue
		}
		g.add(id, s)
	}
	out := make([]Candidate, 0, len(g.byID))
	for id, c := range g.byID {
		c.Name = mostCommon(g.spellings[id])
		if d, ok := g.decided[id]; ok {
			c.Decision = &d
		}
		slices.Sort(c.Providers)
		slices.Sort(c.Models)
		out = append(out, *c)
	}
	slices.SortFunc(out, func(a, b Candidate) int {
		return cmp.Or(cmp.Compare(b.Answers, a.Answers), strings.Compare(a.Name, b.Name))
	})
	return out
}

// gathering collects the sightings of each candidate.
type gathering struct {
	byID      map[string]*Candidate
	spellings map[string]map[string]int
	decided   map[string]CandidateDecision
}

func (g gathering) add(id string, s Sighting) {
	c := g.byID[id]
	if c == nil {
		c = &Candidate{ID: id, Providers: []string{}, Models: []string{}, Addresses: []string{}}
		g.byID[id] = c
		g.spellings[id] = map[string]int{}
	}
	c.Answers++
	g.spellings[id][s.Name]++
	if !slices.Contains(c.Providers, s.Provider) {
		c.Providers = append(c.Providers, s.Provider)
	}
	if s.Model != "" && !slices.Contains(c.Models, s.Model) {
		c.Models = append(c.Models, s.Model)
	}
	if s.URL != "" && !slices.Contains(c.Addresses, s.URL) {
		c.Addresses = append(c.Addresses, s.URL)
	}
	if s.At.After(c.LastSeen) {
		c.LastSeen = s.At
	}
}

// mostCommon is the spelling seen most often, the first alphabetically on a tie.
func mostCommon(counts map[string]int) string {
	best, n := "", 0
	for s, c := range counts {
		if c > n || (c == n && s < best) {
			best, n = s, c
		}
	}
	return best
}
