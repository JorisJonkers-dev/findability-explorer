// Package domain is the analysis context's pure core: an analysis of one website, the rules for
// which sites may be analysed, and the extraction of keywords and known terms from crawled pages.
// It imports no transport, persistence or generated code (enforced by depguard).
package domain

import (
	"context"
	"fmt"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/site"
)

const (
	// DefaultMaxPages is how many pages a crawl fetches when the caller names no limit.
	DefaultMaxPages = 60
	// MaxPages is the most pages one crawl fetches. db/migrations holds the same rule as a CHECK.
	MaxPages     = 100
	maxURLLength = 500
)

// ErrInvalidTarget is returned for an address that is not http or https on a public host.
var ErrInvalidTarget = fmt.Errorf("only http and https addresses on a public host can be analysed")

// ErrInvalidMaxPages is returned for a page limit outside 1..MaxPages.
var ErrInvalidMaxPages = fmt.Errorf("an analysis fetches between 1 and %d pages", MaxPages)

// Target is a validated address to start a crawl from. The zero value is never handed out.
type Target struct {
	url *url.URL
}

// NewTarget checks that raw is an absolute http or https address without credentials, on a host
// that is not a private, loopback or link-local address literal or a local name. The crawler
// checks the address the host resolves to again when it connects.
func NewTarget(raw string) (Target, error) {
	if len(raw) > maxURLLength {
		return Target{}, fmt.Errorf("%w: longer than %d characters", ErrInvalidTarget, maxURLLength)
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return Target{}, fmt.Errorf("%w: %w", ErrInvalidTarget, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return Target{}, fmt.Errorf("%w: scheme %q", ErrInvalidTarget, u.Scheme)
	}
	if u.User != nil {
		return Target{}, fmt.Errorf("%w: credentials in the address", ErrInvalidTarget)
	}
	host := strings.ToLower(u.Hostname())
	if !PublicHost(host) {
		return Target{}, fmt.Errorf("%w: host %q", ErrInvalidTarget, host)
	}
	u.Fragment = ""
	if u.Path == "" {
		u.Path = "/"
	}
	return Target{url: u}, nil
}

// PublicHost reports whether host may be crawled; see site.PublicHost.
func PublicHost(host string) bool { return site.PublicHost(host) }

// PublicAddr reports whether addr may be connected to; see site.PublicAddr.
func PublicAddr(addr netip.Addr) bool { return site.PublicAddr(addr) }

// URL returns the address to start from.
func (t Target) URL() *url.URL { u := *t.url; return &u }

func (t Target) String() string { return t.url.String() }

// CheckMaxPages returns n when it is in 1..MaxPages, DefaultMaxPages for 0.
func CheckMaxPages(n int) (int, error) {
	switch {
	case n == 0:
		return DefaultMaxPages, nil
	case n < 1 || n > MaxPages:
		return 0, fmt.Errorf("%w: got %d", ErrInvalidMaxPages, n)
	default:
		return n, nil
	}
}

// Status is where an analysis is in its life: queued, running, then done or failed.
type Status string

const (
	StatusQueued  Status = "queued"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
)

// Matching is what happened to the product matching by the language model.
type Matching string

const (
	MatchingPending  Matching = "pending"
	MatchingDisabled Matching = "disabled"
	MatchingSkipped  Matching = "skipped"
	MatchingDone     Matching = "done"
	MatchingFailed   Matching = "failed"
)

// Analysis is one analysis of a website, with what it has found so far.
type Analysis struct {
	ID  uuid.UUID
	URL string
	// CompetitorID is the company the scanned website belongs to; empty for scans made before
	// companies were tracked.
	CompetitorID string
	// Number is the scan's number among its company's scans, from 1; 0 without a company.
	Number int
	// Rescan is true for a scan an administrator's re-scan of many companies queued.
	Rescan       bool
	MaxPages     int
	Status       Status
	Failure      string
	PagesCrawled int
	CreatedBy    string
	CreatedAt    time.Time
	FinishedAt   time.Time // zero until the analysis is done or failed
	Pages        []PageSummary
	Keywords     []Keyword
	KnownTerms   []KnownTerm
	Matching     Matching
	// MatchingModel names the provider and model that matched the products; empty when none was asked.
	MatchingModel string
	Equivalents   []Equivalent
}

// Usage is what one call to the language model consumed.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
}

// Total is every token of the call, which is what the monthly budget counts.
func (u Usage) Total() int64 { return u.InputTokens + u.OutputTokens }

// Repository is the port the persistence adapter implements.
type Repository interface {
	// Create stores a queued analysis and returns its number among its company's scans.
	Create(ctx context.Context, a Analysis) (int, error)
	// Get returns the analysis with every result, or ErrNotFound.
	Get(ctx context.Context, id uuid.UUID) (Analysis, error)
	// List returns at most limit analyses without results, newest first.
	List(ctx context.Context, limit int) ([]Analysis, error)
	// ListByCompetitor returns at most limit analyses of one company, without results, newest first.
	ListByCompetitor(ctx context.Context, competitorID string, limit int) ([]Analysis, error)
	// CompetitorStats counts each company's analyses and returns its newest one.
	CompetitorStats(ctx context.Context) (map[string]CompetitorStats, error)
	// CountActive counts a company's queued and running analyses.
	CountActive(ctx context.Context, competitorID string) (int, error)
	// FirstScans returns, per phrase, the number of the company's first finished scan that found it.
	FirstScans(ctx context.Context, competitorID string) (map[PhraseKey]int, error)
	// TokensSince sums the language-model tokens of analyses created at or after since.
	TokensSince(ctx context.Context, since time.Time) (int64, error)
	// Queued returns the ids of every queued analysis, oldest first.
	Queued(ctx context.Context) ([]uuid.UUID, error)
	// FailRunning marks every running analysis failed with reason; a restart interrupted them.
	FailRunning(ctx context.Context, reason string, at time.Time) error
	MarkRunning(ctx context.Context, id uuid.UUID) error
	// AddPage stores the next crawled page and raises the crawled count.
	AddPage(ctx context.Context, id uuid.UUID, page PageSummary) error
	SaveFindings(ctx context.Context, id uuid.UUID, keywords []Keyword, known []KnownTerm) error
	SaveMatching(ctx context.Context, id uuid.UUID, m Matching, model string, eqs []Equivalent, usage Usage) error
	Finish(ctx context.Context, id uuid.UUID, at time.Time) error
	Fail(ctx context.Context, id uuid.UUID, reason string, at time.Time) error
}

// CompetitorStats is how often one company's website was analysed, and its newest analysis.
type CompetitorStats struct {
	Count int
	Last  Analysis
}

// Directory is the port to the companies context: it names the company a website belongs to,
// adding one under name when nobody knows the website's host yet, and lists every company.
type Directory interface {
	Resolve(ctx context.Context, user string, website *url.URL, name string) (string, error)
	Companies(ctx context.Context) ([]Company, error)
}

// Company is what the analysis context knows of a company: whether and where it can be scanned.
type Company struct {
	ID   string
	Name string
	// Website is nil when the company has none.
	Website  *url.URL
	Archived bool
	// Researched is true for a company the desk research studied: it has data before any scan.
	Researched bool
}

// ErrNotFound is returned for an analysis id that does not exist.
var ErrNotFound = fmt.Errorf("no such analysis")

// Finding is a keyword one finished scan of a company found, with where and when; the review
// queue reads the ones the terminology map did not hold.
type Finding struct {
	AnalysisID   uuid.UUID
	CompetitorID string
	URL          string
	StartedAt    time.Time
	Keyword
}
