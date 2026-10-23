// Package domain is the companies context's pure core: every company Tribelt competes with, the
// ones the desk research studied and the ones users or scans added, and how a website is matched
// to one of them. It imports no transport, persistence or generated code (enforced by depguard).
package domain

import (
	"cmp"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/site"
)

// Source says where a company comes from.
type Source string

const (
	SourceResearch Source = "research"
	SourceAdded    Source = "added"
)

const (
	// MaxNameLength is the longest company name. db/migrations holds the same rule as a CHECK.
	MaxNameLength = 200
	maxIDLength   = 80
	maxURLLength  = 500
)

var (
	// ErrInvalidName is returned for a name that is empty after trimming, or too long.
	ErrInvalidName = fmt.Errorf("a company name has between 1 and %d characters", MaxNameLength)
	// ErrInvalidWebsite is returned for an address that is not http or https on a public host.
	ErrInvalidWebsite = errors.New("a company's website is an http or https address on a public host")
	// ErrNotFound is returned for a company id nobody knows.
	ErrNotFound = errors.New("no such company")
	// ErrNotArchivable is returned when archiving an added company, which is removed instead.
	ErrNotArchivable = errors.New("only a researched company is archived; an added one is removed")
)

// AlreadyKnownError is returned when a website's host belongs to a known company.
type AlreadyKnownError struct {
	ID   string
	Name string
}

func (e AlreadyKnownError) Error() string {
	return fmt.Sprintf("%s already has this website", e.Name)
}

// BeltName is a company's own name for a belt, and the Tribelt family it corresponds to.
type BeltName struct {
	Name   string
	Family string
}

// Company is a known company.
type Company struct {
	ID      string
	Source  Source
	Name    string
	Country string
	// Website is the address as the research noted it, or the address a user added.
	Website string
	// URL is the website to visit and scan; nil when the research found none.
	URL           *url.URL
	Host          string
	Languages     []string
	GroupCompany  bool
	BeltNames     []BeltName
	HomepageTitle string
	AddedBy       string
	AddedAt       time.Time // zero for a researched company
	// Archived is true for a researched company an administrator archived.
	Archived   bool
	ArchivedBy string
	ArchivedAt time.Time
}

// Override is an administrator's change to a researched company, which the binary embeds and
// never changes: another name or website, or archived. Empty fields keep the research's value.
type Override struct {
	ID         string
	Name       string
	Website    *url.URL
	Host       string
	ArchivedBy string
	ArchivedAt time.Time // zero unless archived
	UpdatedBy  string
	UpdatedAt  time.Time
}

// NewOverride is the override that turns the research's original into changed.
func NewOverride(original, changed Company, by string, at time.Time) Override {
	o := Override{ID: original.ID, UpdatedBy: by, UpdatedAt: at}
	if changed.Name != original.Name {
		o.Name = changed.Name
	}
	if changed.URL != nil && (original.URL == nil || changed.URL.String() != original.URL.String()) {
		o.Website, o.Host = changed.URL, changed.Host
	}
	if changed.Archived {
		o.ArchivedBy, o.ArchivedAt = changed.ArchivedBy, changed.ArchivedAt
	}
	return o
}

// ApplyOverrides returns the researched companies with the overrides applied.
func ApplyOverrides(research []Company, overrides []Override) []Company {
	out := make([]Company, 0, len(research))
	for _, c := range research {
		for _, o := range overrides {
			if o.ID != c.ID {
				continue
			}
			if o.Name != "" {
				c.Name = o.Name
			}
			if o.Website != nil {
				c.URL, c.Website, c.Host = o.Website, o.Website.String(), o.Host
			}
			c.Archived, c.ArchivedBy, c.ArchivedAt = !o.ArchivedAt.IsZero(), o.ArchivedBy, o.ArchivedAt
		}
		out = append(out, c)
	}
	return out
}

// Change is what an administrator changes about a company; a nil field stays as it is.
type Change struct {
	Name     *string
	URL      *string
	Archived *bool
}

// Apply returns c with ch applied by user at at, checked: the name's length, the website's
// address and that no other company in d has its host, and that only a researched company is
// archived.
func (d Directory) Apply(c Company, ch Change, user string, at time.Time) (Company, error) {
	if ch.Name != nil {
		name, err := CheckName(*ch.Name)
		if err != nil {
			return Company{}, err
		}
		c.Name = name
	}
	if ch.URL != nil {
		website, err := NewWebsite(*ch.URL)
		if err != nil {
			return Company{}, err
		}
		host := site.Host(website)
		if known, ok := d.ByHost(host); ok && known.ID != c.ID {
			return Company{}, AlreadyKnownError{ID: known.ID, Name: known.Name}
		}
		c.URL, c.Website, c.Host = website, website.String(), host
	}
	if ch.Archived != nil {
		if c.Source != SourceResearch {
			return Company{}, ErrNotArchivable
		}
		c.Archived, c.ArchivedBy, c.ArchivedAt = *ch.Archived, "", time.Time{}
		if c.Archived {
			c.ArchivedBy, c.ArchivedAt = user, at
		}
	}
	return c, nil
}

// Scan is what the companies context knows of one analysis of a company's website.
type Scan struct {
	ID           uuid.UUID
	URL          string
	Status       string
	Failure      string
	MaxPages     int
	PagesCrawled int
	CreatedAt    time.Time
	FinishedAt   time.Time
	// Number is the scan's number among the company's scans, from 1.
	Number        int
	MatchingModel string
}

// Stats is how often a company's website was scanned, and its newest scan.
type Stats struct {
	Count int
	Last  *Scan
}

// Listed is a company with its scan statistics.
type Listed struct {
	Company
	Stats Stats
}

// legalForms are the endings of a registered name that a company's id leaves out, as words.
func legalForms() [][]string {
	return [][]string{
		{"gmbh"},
		{"b", "v"},
		{"bv"},
		{"s", "a"},
		{"sa"},
		{"s", "p", "a"},
		{"spa"},
		{"s", "l"},
		{"ltd"},
		{"inc"},
		{"ag"},
		{"llc"},
		{"n", "v"},
		{"nv"},
	}
}

// ID makes a company's id from its name: folded, without a remark in brackets or a legal form,
// words joined by hyphens. "PELGER GmbH" is pelger, "Wire Belt Company Ltd" wire-belt-company.
func ID(name string) string {
	base, _, _ := strings.Cut(name, "(")
	words := strings.FieldsFunc(fold.Fold(base), func(r rune) bool { return r >= 128 || (!unicode.IsLetter(r) && !unicode.IsDigit(r)) })
	for trimmed := true; trimmed; {
		trimmed = false
		for _, form := range legalForms() {
			if len(words) > len(form) && slices.Equal(words[len(words)-len(form):], form) {
				words, trimmed = words[:len(words)-len(form)], true
			}
		}
	}
	id := strings.Join(words, "-")
	if len(id) > maxIDLength {
		id = strings.TrimRight(id[:maxIDLength], "-")
	}
	return id
}

// CheckName trims raw and checks its length.
func CheckName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if n := utf8.RuneCountInString(name); n == 0 || n > MaxNameLength {
		return "", fmt.Errorf("%w: got %d", ErrInvalidName, n)
	}
	return name, nil
}

// NewWebsite checks that raw is an absolute http or https address without credentials on a
// public host, and returns it without fragment.
func NewWebsite(raw string) (*url.URL, error) {
	if len(raw) > maxURLLength {
		return nil, fmt.Errorf("%w: longer than %d characters", ErrInvalidWebsite, maxURLLength)
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidWebsite, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || !site.PublicHost(strings.ToLower(u.Hostname())) {
		return nil, ErrInvalidWebsite
	}
	u.Fragment = ""
	if u.Path == "" {
		u.Path = "/"
	}
	return u, nil
}

// ResearchWebsite reads the research's note on a website ("https://www.cambridge-es.com
// (blocked by Cloudflare 403 on 2026-10-22)", "unverified (no official site found)") as an
// address, or nil when the note holds none.
func ResearchWebsite(note string) *url.URL {
	for _, field := range strings.Fields(note) {
		if strings.HasPrefix(field, "http://") || strings.HasPrefix(field, "https://") {
			if u, err := NewWebsite(field); err == nil {
				return u
			}
		}
	}
	return nil
}

// Directory is every known company, in order, to find one by id or by a website's host.
type Directory struct {
	companies []Company
}

// NewDirectory returns a Directory of research then added companies. On a shared host the first
// company wins, so Tribelt itself is found before a group company on its site.
func NewDirectory(research, added []Company) Directory {
	return Directory{companies: append(append([]Company{}, research...), added...)}
}

// ByID returns the company with id.
func (d Directory) ByID(id string) (Company, bool) {
	for _, c := range d.companies {
		if c.ID == id {
			return c, true
		}
	}
	return Company{}, false
}

// ByHost returns the first company whose website has host.
func (d Directory) ByHost(host string) (Company, bool) {
	for _, c := range d.companies {
		if c.Host != "" && c.Host == host {
			return c, true
		}
	}
	return Company{}, false
}

// NewCompany returns the company to add for name and website: its id made unique among the
// known ones with a number, or AlreadyKnownError when the website's host is taken.
func (d Directory) NewCompany(name string, website *url.URL, by string, at time.Time) (Company, error) {
	host := site.Host(website)
	if known, ok := d.ByHost(host); ok {
		return Company{}, AlreadyKnownError{ID: known.ID, Name: known.Name}
	}
	base := ID(name)
	if base == "" {
		base = ID(strings.ReplaceAll(host, ".", " "))
	}
	id := base
	for n := 2; ; n++ {
		if _, taken := d.ByID(id); !taken {
			break
		}
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return Company{
		ID: id, Source: SourceAdded, Name: name, Website: website.String(), URL: website, Host: host,
		AddedBy: by, AddedAt: at,
	}, nil
}

// Sorted returns the directory's companies with their stats: researched competitors by name,
// then added companies by name, then Tribelt's own group.
func (d Directory) Sorted(stats map[string]Stats) []Listed {
	out := make([]Listed, 0, len(d.companies))
	for _, c := range d.companies {
		out = append(out, Listed{Company: c, Stats: stats[c.ID]})
	}
	slices.SortStableFunc(out, func(a, b Listed) int {
		return cmp.Or(cmp.Compare(rank(a.Company), rank(b.Company)), strings.Compare(fold.Fold(a.Name), fold.Fold(b.Name)))
	})
	return out
}

func rank(c Company) int {
	switch {
	case c.GroupCompany:
		return 2
	case c.Source == SourceAdded:
		return 1
	default:
		return 0
	}
}
