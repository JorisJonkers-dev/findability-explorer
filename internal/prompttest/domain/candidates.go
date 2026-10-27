package domain

import (
	"cmp"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/site"
)

// Sighting is one name an answer gave that the extractor could not match to the company list.
type Sighting struct {
	// AnswerID is the answer that gave the name.
	AnswerID uuid.UUID
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

// Merge records that a candidate is one company with another candidate (IntoID) or with a
// company on the list (CompetitorID), as the language model or an administrator found; Separate
// marks one a person split off, which is never merged again.
type Merge struct {
	CandidateID  string
	IntoID       string
	CompetitorID string
	Separate     bool
	By           string
	At           time.Time
}

// ModelMerger is who a merge by the language model names.
const ModelMerger = "language model"

// Joined is how a name joined the candidate it is part of.
type Joined string

const (
	// JoinedAsLead is the name the candidate goes by: the one the answers give most.
	JoinedAsLead Joined = ""
	// JoinedBySite is a name given with an address on the same website.
	JoinedBySite Joined = "site"
	// JoinedByModel is a name the language model found the same company.
	JoinedByModel Joined = "model"
)

// Member is one name a candidate folds in, with how many answers give it.
type Member struct {
	ID      string
	Name    string
	Answers int
	Joined  Joined
}

// Candidate is a company the answers name that is not on the competitor list: every name it goes
// by, how often and by which providers' models it was named, its websites and the addresses
// given for it, and the decision on it.
type Candidate struct {
	// ID is the id of its lead name; any of its names' ids finds it.
	ID string
	// Name is the name the answers give most.
	Name      string
	Answers   int
	Providers []string
	// Models are the exact models that named it.
	Models    []string
	Addresses []string
	// Sites are the domains of its own websites the addresses point to, the most given first.
	Sites []string
	// Website is the home page to add it with, on its first site; "" without one.
	Website  string
	Members  []Member
	LastSeen time.Time
	// Quoted are a few of the answers that name it, to quote what they say around its name.
	Quoted   []uuid.UUID
	Decision *CandidateDecision
	// Verdict is the language model's check of it, once it has one with the current CheckVersion,
	// or, for a decided candidate, with any version.
	Verdict *CompanyVerdict
}

// Status is the candidate's status: its decision's, or pending.
func (c Candidate) Status() CandidateStatus {
	if c.Decision == nil {
		return CandidatePending
	}
	return c.Decision.Status
}

// MemberIDs are the ids of every name the candidate goes by.
func (c Candidate) MemberIDs() []string {
	out := make([]string, 0, len(c.Members))
	for _, m := range c.Members {
		out = append(out, m.ID)
	}
	return out
}

// Candidates gathers the names the answers give that the company list does not hold into one
// candidate per company: names given with an address on the same website, and names merged by the
// language model or a person, are one company. A company with a name or website on the list, or
// merged into a company on it, is left out, unless a decision on it is on record (an added one
// stays listed, so the add can be undone). A decision on one name covers them all. Most answers
// first, then by name.
func Candidates(sightings []Sighting, decisions []CandidateDecision, directory []DirectoryCompany, merges []Merge) []Candidate {
	names := gatherNames(sightings, directory, merges)
	groups := group(names, merges)
	decided := map[string]CandidateDecision{}
	for _, d := range decisions {
		decided[d.ID] = d
	}
	out := make([]Candidate, 0, len(groups))
	for _, g := range groups {
		c, known := combine(g, decided)
		if known && c.Decision == nil {
			continue
		}
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b Candidate) int {
		return cmp.Or(cmp.Compare(b.Answers, a.Answers), strings.Compare(a.Name, b.Name))
	})
	return out
}

// name is everything the answers say under one name key.
type name struct {
	id        string
	spellings map[string]int
	answers   int
	providers []string
	models    []string
	addresses []string
	sites     map[string]int
	quoted    []uuid.UUID
	lastSeen  time.Time
	// known is true when the list holds the name or its website, or it was merged into a company on it.
	known bool
}

func gatherNames(sightings []Sighting, directory []DirectoryCompany, merges []Merge) map[string]*name {
	toCompany := map[string]bool{}
	for _, m := range merges {
		if m.CompetitorID != "" {
			toCompany[m.CandidateID] = true
		}
	}
	out := map[string]*name{}
	for _, s := range sightings {
		key := NameKey(s.Name)
		if key == "" {
			continue
		}
		id := CandidateID(key)
		n := out[id]
		if n == nil {
			n = &name{id: id, spellings: map[string]int{}, providers: []string{}, models: []string{}, addresses: []string{}, sites: map[string]int{}, known: toCompany[id]}
			out[id] = n
		}
		n.add(s, directory)
	}
	return out
}

func (n *name) add(s Sighting, directory []DirectoryCompany) {
	n.answers++
	n.spellings[s.Name]++
	n.known = n.known || Match(Named{Name: s.Name, URL: s.URL}, directory) != ""
	if !slices.Contains(n.providers, s.Provider) {
		n.providers = append(n.providers, s.Provider)
	}
	if s.Model != "" && !slices.Contains(n.models, s.Model) {
		n.models = append(n.models, s.Model)
	}
	if s.URL != "" && !slices.Contains(n.addresses, s.URL) {
		n.addresses = append(n.addresses, s.URL)
	}
	if d := site.CompanyDomain(s.URL); d != "" {
		n.sites[d]++
	}
	if len(n.quoted) < MaxQuoted && !slices.Contains(n.quoted, s.AnswerID) {
		n.quoted = append(n.quoted, s.AnswerID)
	}
	if s.At.After(n.lastSeen) {
		n.lastSeen = s.At
	}
}

// sets keeps which names are one company: each name's set is found by following it to its root.
type sets map[string]string

func (s sets) find(id string) string {
	if p := s[id]; p != "" && p != id {
		s[id] = s.find(p)
		return s[id]
	}
	return id
}

func (s sets) join(a, b string) {
	if ra, rb := s.find(a), s.find(b); ra != rb {
		s[ra] = rb
	}
}

// group puts names that are one company together: those on one website, and those merged.
func group(names map[string]*name, merges []Merge) [][]*name {
	one := sets{}
	bySite := map[string]string{}
	ids := slices.Sorted(maps.Keys(names))
	for _, id := range ids {
		for d := range names[id].sites {
			if other, ok := bySite[d]; ok {
				one.join(id, other)
			} else {
				bySite[d] = id
			}
		}
	}
	for _, m := range merges {
		if m.IntoID != "" && !m.Separate && names[m.CandidateID] != nil && names[m.IntoID] != nil {
			one.join(m.CandidateID, m.IntoID)
		}
	}
	groups := map[string][]*name{}
	for _, id := range ids {
		root := one.find(id)
		groups[root] = append(groups[root], names[id])
	}
	out := make([][]*name, 0, len(groups))
	for _, root := range slices.Sorted(maps.Keys(groups)) {
		out = append(out, groups[root])
	}
	return out
}

// combine makes one candidate of a company's names, led by the name the answers give most. It
// reports whether the list knows the company: when it knows one of its names.
func combine(g []*name, decided map[string]CandidateDecision) (Candidate, bool) {
	slices.SortFunc(g, func(a, b *name) int {
		return cmp.Or(cmp.Compare(b.answers, a.answers), strings.Compare(mostCommon(a.spellings), mostCommon(b.spellings)))
	})
	lead := g[0]
	c := Candidate{ID: lead.id, Name: mostCommon(lead.spellings), Providers: []string{}, Models: []string{}, Addresses: []string{}}
	sites := map[string]int{}
	known := false
	for i, n := range g {
		known = known || n.known
		c.Answers += n.answers
		c.Providers = union(c.Providers, n.providers)
		c.Models = union(c.Models, n.models)
		c.Addresses = union(c.Addresses, n.addresses)
		for d, count := range n.sites {
			sites[d] += count
		}
		c.LastSeen = later(c.LastSeen, n.lastSeen)
		c.Quoted = quote(c.Quoted, n.quoted)
		c.Members = append(c.Members, member(n, i, g))
		c.Decision = newest(c.Decision, decided, n.id)
	}
	slices.Sort(c.Providers)
	slices.Sort(c.Models)
	c.Sites = slices.SortedFunc(maps.Keys(sites), func(a, b string) int { return cmp.Or(cmp.Compare(sites[b], sites[a]), strings.Compare(a, b)) })
	c.Website = website(c.Addresses, c.Sites)
	return c, known
}

// member is the i-th name of a company g as it joined: the lead first, then by website or merged.
func member(n *name, i int, g []*name) Member {
	m := Member{ID: n.id, Name: mostCommon(n.spellings), Answers: n.answers}
	switch {
	case i == 0:
	case sharesSite(n, g):
		m.Joined = JoinedBySite
	default:
		m.Joined = JoinedByModel
	}
	return m
}

// quote is quoted with the answers of more it lacks, up to MaxQuoted.
func quote(quoted, more []uuid.UUID) []uuid.UUID {
	for _, id := range more {
		if len(quoted) < MaxQuoted && !slices.Contains(quoted, id) {
			quoted = append(quoted, id)
		}
	}
	return quoted
}

// newest is the newer of current and the decision on id, if there is one.
func newest(current *CandidateDecision, decided map[string]CandidateDecision, id string) *CandidateDecision {
	if d, ok := decided[id]; ok && (current == nil || d.At.After(current.At)) {
		return &d
	}
	return current
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// website is the home page of the first address on the first of the company's sites; "" without one.
func website(addresses, sites []string) string {
	if len(sites) == 0 {
		return ""
	}
	i := slices.IndexFunc(addresses, func(a string) bool { return site.CompanyDomain(a) == sites[0] })
	return site.Root(addresses[i])
}

// sharesSite reports whether n has a website another of the company's names has too.
func sharesSite(n *name, g []*name) bool {
	for _, other := range g {
		if other == n {
			continue
		}
		for d := range n.sites {
			if other.sites[d] > 0 {
				return true
			}
		}
	}
	return false
}

// union is list with the items of more it lacks, in order.
func union(list, more []string) []string {
	for _, x := range more {
		if !slices.Contains(list, x) {
			list = append(list, x)
		}
	}
	return list
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

// Matcher matches the names answers gave to the company list as it is now: by website and name,
// and by the names added from the candidates or merged into a company on the list. Answers are
// matched when they are read, so a company added or merged later counts in them too.
type Matcher struct {
	directory []DirectoryCompany
	aliases   map[string]string
}

// NewMatcher is a Matcher on the list, with the candidates added to it or merged into it.
func NewMatcher(directory []DirectoryCompany, decisions []CandidateDecision, merges []Merge) Matcher {
	on := map[string]bool{}
	for _, c := range directory {
		on[c.ID] = true
	}
	aliases := map[string]string{}
	for _, d := range decisions {
		if d.Status == CandidateAdded && on[d.CompetitorID] {
			aliases[d.ID] = d.CompetitorID
		}
	}
	for _, m := range merges {
		if on[m.CompetitorID] {
			aliases[m.CandidateID] = m.CompetitorID
		}
	}
	return Matcher{directory: directory, aliases: aliases}
}

// Rematch matches each company the answer named that the list did not hold when it was read.
func (m Matcher) Rematch(x Extraction) Extraction {
	companies := slices.Clone(x.Companies)
	for i, c := range companies {
		if c.CompetitorID != "" {
			continue
		}
		companies[i].CompetitorID = m.Match(c.Name, c.URL)
	}
	x.Companies = companies
	return x
}

// Match is the id of the company on the list that name, given with url, is, or "".
func (m Matcher) Match(name, url string) string {
	if id := Match(Named{Name: name, URL: url}, m.directory); id != "" {
		return id
	}
	return m.aliases[CandidateID(NameKey(name))]
}

// Cluster is what the language model found of some candidates: one company, and the company on
// the list it is, if any.
type Cluster struct {
	IDs          []string
	CompetitorID string
}

// MergesOf turns the model's clusters into merges, by ModelMerger at at: each candidate into the
// company on the list, or into the cluster's first. A candidate it was not asked about or a person
// split off, a company not on the list, and a cluster of one with no company are left out.
func MergesOf(clusters []Cluster, asked []Candidate, directory []DirectoryCompany, merges []Merge, at time.Time) []Merge {
	ok := map[string]bool{}
	for _, c := range asked {
		ok[c.ID] = true
	}
	for _, m := range merges {
		if m.Separate {
			ok[m.CandidateID] = false
		}
	}
	on := func(id string) bool {
		return slices.ContainsFunc(directory, func(c DirectoryCompany) bool { return c.ID == id })
	}
	var out []Merge
	for _, cl := range clusters {
		ids := slices.DeleteFunc(slices.Clone(cl.IDs), func(id string) bool { return !ok[id] })
		ids = slices.Compact(ids)
		switch {
		case cl.CompetitorID != "" && on(cl.CompetitorID):
			for _, id := range ids {
				out = append(out, Merge{CandidateID: id, CompetitorID: cl.CompetitorID, By: ModelMerger, At: at})
			}
		case len(ids) > 1:
			for _, id := range ids[1:] {
				out = append(out, Merge{CandidateID: id, IntoID: ids[0], By: ModelMerger, At: at})
			}
		}
	}
	return out
}

// CompanyVerdict is the language model's check of a candidate: whether it is a competitor of
// Tribelt, whether it is sure, and why (ADR-0009). A sure one decides the candidate; an unsure one
// leaves it to a person.
type CompanyVerdict struct {
	CandidateID string
	Competitor  bool
	Sure        bool
	Reason      string
	Model       string
	// Version is the CheckVersion that made it.
	Version int
	At      time.Time
}

// WithVerdicts attaches to each candidate the newest verdict on any of its names. A waiting
// candidate's verdict of an older CheckVersion is left off, so the candidate is checked again.
func WithVerdicts(candidates []Candidate, verdicts []CompanyVerdict) []Candidate {
	byID := map[string]CompanyVerdict{}
	for _, v := range verdicts {
		byID[v.CandidateID] = v
	}
	for i, c := range candidates {
		for _, id := range c.MemberIDs() {
			v, ok := byID[id]
			// A check without a reason is the model having said nothing about the company: it
			// counts as no check, so the company is asked about again.
			if !ok || v.Reason == "" || (c.Decision == nil && v.Version < CheckVersion) {
				continue
			}
			if candidates[i].Verdict == nil || v.At.After(candidates[i].Verdict.At) {
				candidates[i].Verdict = &v
			}
		}
	}
	return candidates
}

// Unchecked are the waiting candidates the model has not checked, in their order.
func Unchecked(candidates []Candidate) []Candidate {
	var out []Candidate
	for _, c := range candidates {
		if c.Status() == CandidatePending && c.Verdict == nil {
			out = append(out, c)
		}
	}
	return out
}

// CompleteVerdicts keeps the model's verdicts on the candidates it was asked about, at most one
// each, in their order, stamped with the model, the CheckVersion and the time. A candidate it gave
// no verdict on, or one without a reason, gets none, so it is asked about again; a verdict on a
// candidate it was not asked about is dropped.
func CompleteVerdicts(asked []Candidate, answered []CompanyVerdict, model string, at time.Time) []CompanyVerdict {
	out := make([]CompanyVerdict, 0, len(asked))
	for _, c := range asked {
		i := slices.IndexFunc(answered, func(a CompanyVerdict) bool { return a.CandidateID == c.ID })
		if i < 0 || strings.TrimSpace(answered[i].Reason) == "" {
			continue
		}
		v := answered[i]
		v.Model, v.Version, v.At = model, CheckVersion, at
		out = append(out, v)
	}
	return out
}
