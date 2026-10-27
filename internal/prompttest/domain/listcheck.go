package domain

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

// Claude Haiku checks the companies on the competitor list as it checks the candidates (ADR-0010):
// against TargetMarket, from the names, the website, the home page and what the answers say of
// each company. A sure "not a competitor" archives the company, by ModelDecider; an unsure verdict
// leaves it on the list for a person, flagged; a sure competitor stays. A company a person
// restored or kept is never archived by the model, and is not checked again.

// ListCheck is the language model's check of a company on the competitor list: whether it is a
// competitor of Tribelt, whether it is sure, why, and the CheckVersion that made it.
type ListCheck struct {
	CompanyID  string
	Competitor bool
	Sure       bool
	Reason     string
	Model      string
	Version    int
	At         time.Time
}

// Archives reports whether the check archives its company: a sure "not a competitor".
func (c ListCheck) Archives() bool { return c.Sure && !c.Competitor }

// Checkable reports whether the model checks c: a competitor on the list, not archived, that no
// person kept. Tribelt, the mirror and the group's companies are never checked.
func (c DirectoryCompany) Checkable() bool {
	return c.Role == RoleCompetitor && !c.Archived && !c.Kept
}

// ToCheck are the companies of directory the model checks next, in order: each checkable one
// without a check of the current CheckVersion. Raising CheckVersion checks them all again.
func ToCheck(directory []DirectoryCompany, checks []ListCheck) []DirectoryCompany {
	version := map[string]int{}
	for _, c := range checks {
		version[c.CompanyID] = c.Version
	}
	var out []DirectoryCompany
	for _, c := range directory {
		if v, ok := version[c.ID]; c.Checkable() && (!ok || v < CheckVersion) {
			out = append(out, c)
		}
	}
	return out
}

// ShownChecks are the checks of the competitors in directory, in its order, to show a person. A
// check of an older CheckVersion is left out for a company the model checks again; on an archived
// or kept company it stays, as what the model said then.
func ShownChecks(directory []DirectoryCompany, checks []ListCheck) []ListCheck {
	byID := map[string]ListCheck{}
	for _, c := range checks {
		byID[c.CompanyID] = c
	}
	var out []ListCheck
	for _, c := range directory {
		check, ok := byID[c.ID]
		if !ok || c.Role != RoleCompetitor || (c.Checkable() && check.Version < CheckVersion) {
			continue
		}
		out = append(out, check)
	}
	return out
}

// Naming is one name an answer gave, with the company on the list it was matched to when the
// answer was read, if any.
type Naming struct {
	AnswerID     uuid.UUID
	Name         string
	URL          string
	CompetitorID string
}

// Quotes is how many answers name a company, and the first MaxQuoted of them, to quote.
type Quotes struct {
	Answers int
	Quoted  []uuid.UUID
}

// NamedIn is, for each company on the list, the answers that name it: matched when they were read,
// or by m now. Each answer counts once per company, and the first MaxQuoted are quoted.
func (m Matcher) NamedIn(namings []Naming) map[string]Quotes {
	out := map[string]Quotes{}
	seen := map[string][]uuid.UUID{}
	for _, n := range namings {
		id := n.CompetitorID
		if id == "" {
			id = m.Match(n.Name, n.URL)
		}
		if id == "" || slices.Contains(seen[id], n.AnswerID) {
			continue
		}
		seen[id] = append(seen[id], n.AnswerID)
		q := out[id]
		q.Answers++
		if len(q.Quoted) < MaxQuoted {
			q.Quoted = append(q.Quoted, n.AnswerID)
		}
		out[id] = q
	}
	return out
}

// AsCandidate is a company on the list as the model checks it, in the shape of a candidate: its
// id, its names, its website, and the answers that name it. The check of a candidate thus checks
// it, from the same evidence.
func AsCandidate(c DirectoryCompany, q Quotes) Candidate {
	members := []Member{{ID: c.ID, Name: c.Name, Answers: q.Answers}}
	for _, n := range c.OtherNames {
		members = append(members, Member{ID: c.ID, Name: n, Joined: JoinedByModel})
	}
	return Candidate{ID: c.ID, Name: c.Name, Answers: q.Answers, Website: c.Website, Members: members, Quoted: q.Quoted}
}

// ListChecksOf are the model's verdicts on companies it checked as candidates (AsCandidate), as
// checks of them.
func ListChecksOf(verdicts []CompanyVerdict) []ListCheck {
	out := make([]ListCheck, 0, len(verdicts))
	for _, v := range verdicts {
		out = append(out, ListCheck{
			CompanyID: v.CandidateID, Competitor: v.Competitor, Sure: v.Sure, Reason: v.Reason, Model: v.Model, Version: v.Version, At: v.At,
		})
	}
	return out
}
