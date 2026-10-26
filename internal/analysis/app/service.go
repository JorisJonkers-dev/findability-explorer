// Package app holds the analysis context's use cases: starting an analysis, running it in the
// background, and reading it back. They are the only entry points the inbound adapters call.
package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

const (
	// DefaultListLimit is how many analyses List returns when the caller names no limit.
	DefaultListLimit = 20
	// MaxListLimit is the most analyses one List returns: more than the queue holds, so a re-scan
	// of every company can be followed in one list.
	MaxListLimit = 300
	// maxFailure is the longest failure reason stored; the contract and the database agree.
	maxFailure = 500
)

var (
	// ErrRateLimited is returned when a user has started too many analyses in the past hour.
	ErrRateLimited = errors.New("too many analyses started in the past hour; try again later")
	// ErrBusy is returned when the queue is full.
	ErrBusy = errors.New("the analysis queue is full; try again in a few minutes")
	// ErrScanActive is returned when a scan of the same company is queued or running.
	ErrScanActive = errors.New("a scan of this company is still queued or running")
	// ErrUnknownCompany is returned for a company id nobody knows.
	ErrUnknownCompany = errors.New("no company has this id")
	// ErrInvalidRescan is returned for a re-scan that names no companies, or both names and all.
	ErrInvalidRescan = errors.New("a re-scan names companies or asks for all of them, not both")
)

// MaxHistory is how many of a company's scans the profile and the comparison look through.
const MaxHistory = 100

// Limits bound what analyses may cost.
type Limits struct {
	// PerUserPerHour is how many analyses one user may start in an hour.
	PerUserPerHour int
	// MonthlyTokens is how many language-model tokens all analyses of a calendar month may use.
	MonthlyTokens int64
	// SiteTextChars is how much site text the language model reads per analysis.
	SiteTextChars int
	// QueueSize is how many analyses may wait to run.
	QueueSize int
	// Workers is how many analyses run at once, each on its own site; at least one.
	Workers int
}

// Glossary returns the terminology map's terms; the terminology context provides it.
type Glossary func() []domain.GlossaryTerm

// Service runs the analysis use cases.
type Service struct {
	repo      domain.Repository
	directory domain.Directory
	crawler   domain.Crawler
	matchers  domain.Matchers
	glossary  Glossary
	now       func() time.Time
	limits    Limits
	queue     chan uuid.UUID
}

// New returns a Service. matchers names the matcher each analysis uses when it runs.
func New(repo domain.Repository, directory domain.Directory, crawler domain.Crawler, matchers domain.Matchers,
	glossary Glossary, now func() time.Time, limits Limits,
) *Service {
	return &Service{
		repo: repo, directory: directory, crawler: crawler, matchers: matchers, glossary: glossary, now: now, limits: limits,
		queue: make(chan uuid.UUID, max(limits.QueueSize, 1)),
	}
}

// Start validates the request, checks the user's hourly limit, names the company the website
// belongs to (adding it under companyName when it is new), stores a queued analysis and puts it
// on the queue.
func (s *Service) Start(ctx context.Context, user, rawURL string, maxPages int, companyName string) (domain.Analysis, error) {
	target, err := domain.NewTarget(rawURL)
	if err != nil {
		return domain.Analysis{}, err
	}
	pages, err := domain.CheckMaxPages(maxPages)
	if err != nil {
		return domain.Analysis{}, err
	}
	now := s.now()
	started, err := s.repo.CountCreatedSince(ctx, user, now.Add(-time.Hour))
	if err != nil {
		return domain.Analysis{}, fmt.Errorf("count analyses: %w", err)
	}
	if started >= s.limits.PerUserPerHour {
		return domain.Analysis{}, ErrRateLimited
	}
	competitor, err := s.directory.Resolve(ctx, user, target.URL(), companyName)
	if err != nil {
		return domain.Analysis{}, fmt.Errorf("name the company: %w", err)
	}
	return s.queueScan(ctx, user, target, competitor, pages, false)
}

// queueScan stores a queued scan of competitor and puts it on the queue, unless a scan of the
// company is queued or running already.
func (s *Service) queueScan(ctx context.Context, user string, target domain.Target, competitor string, pages int, rescan bool) (domain.Analysis, error) {
	active, err := s.repo.CountActive(ctx, competitor)
	if err != nil {
		return domain.Analysis{}, fmt.Errorf("count active scans: %w", err)
	}
	if active > 0 {
		return domain.Analysis{}, ErrScanActive
	}
	now := s.now()
	a := domain.Analysis{
		ID: uuid.New(), URL: target.String(), CompetitorID: competitor, MaxPages: pages, Status: domain.StatusQueued,
		CreatedBy: user, CreatedAt: now, Matching: domain.MatchingPending, Rescan: rescan,
	}
	if a.Number, err = s.repo.Create(ctx, a); err != nil {
		return domain.Analysis{}, fmt.Errorf("create analysis: %w", err)
	}
	select {
	case s.queue <- a.ID:
		return a, nil
	default:
		if err := s.repo.Fail(ctx, a.ID, ErrBusy.Error(), now); err != nil {
			return domain.Analysis{}, fmt.Errorf("fail analysis: %w", err)
		}
		return domain.Analysis{}, ErrBusy
	}
}

// Skipped is a company a re-scan left out, and why.
type Skipped struct {
	CompetitorID string
	Name         string
	Reason       string
}

// Rescan queues one scan of each company named, or with all of every company that is not
// archived, has a website and has data: the desk research studied it, or it was scanned before. A company without a website, or with a scan
// queued or running, is skipped; so is every company once the queue is full. The hourly limit on
// single scans does not apply.
func (s *Service) Rescan(ctx context.Context, user string, ids []string, all bool, maxPages int) ([]domain.Analysis, []Skipped, error) {
	if all == (len(ids) > 0) {
		return nil, nil, ErrInvalidRescan
	}
	pages, err := domain.CheckMaxPages(maxPages)
	if err != nil {
		return nil, nil, err
	}
	companies, err := s.directory.Companies(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list companies: %w", err)
	}
	if all {
		if companies, err = s.scanned(ctx, companies); err != nil {
			return nil, nil, err
		}
	} else {
		companies = pickCompanies(companies, ids)
	}
	var queued []domain.Analysis
	var skipped []Skipped
	for _, c := range companies {
		reason, err := s.rescanOne(ctx, user, c, pages, &queued)
		if err != nil {
			return nil, nil, err
		}
		if reason != "" {
			skipped = append(skipped, Skipped{CompetitorID: c.ID, Name: c.Name, Reason: reason})
		}
	}
	return queued, skipped, nil
}

// rescanOne queues a scan of c and appends it to queued, or returns why it was skipped.
func (s *Service) rescanOne(ctx context.Context, user string, c domain.Company, pages int, queued *[]domain.Analysis) (string, error) {
	if c.Website == nil {
		if c.Name == "" {
			return "No company has this id.", nil
		}
		return "It has no website to scan.", nil
	}
	target, err := domain.NewTarget(c.Website.String())
	if err != nil {
		return "Its website cannot be scanned.", nil //nolint:nilerr // the reason is the outcome
	}
	a, err := s.queueScan(ctx, user, target, c.ID, pages, true)
	switch {
	case errors.Is(err, ErrScanActive):
		return "A scan of it is still queued or running.", nil
	case errors.Is(err, ErrBusy):
		return "The queue is full; try it again in a few minutes.", nil
	case err != nil:
		return "", err
	}
	*queued = append(*queued, a)
	return "", nil
}

// scanned keeps the companies that are not archived, have a website and have data: researched,
// or scanned before.
func (s *Service) scanned(ctx context.Context, companies []domain.Company) ([]domain.Company, error) {
	stats, err := s.repo.CompetitorStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("scan statistics: %w", err)
	}
	var out []domain.Company
	for _, c := range companies {
		if !c.Archived && c.Website != nil && (c.Researched || stats[c.ID].Count > 0) {
			out = append(out, c)
		}
	}
	return out, nil
}

// pickCompanies returns the companies named by ids, in that order; an unknown id becomes a
// company with only its id.
func pickCompanies(companies []domain.Company, ids []string) []domain.Company {
	out := make([]domain.Company, 0, len(ids))
	for _, id := range ids {
		c := domain.Company{ID: id}
		for _, known := range companies {
			if known.ID == id {
				c = known
			}
		}
		out = append(out, c)
	}
	return out
}

// Profile returns a company's search terms from its latest finished scan, or ErrUnknownCompany.
func (s *Service) Profile(ctx context.Context, competitorID string) (domain.Profile, error) {
	finished, err := s.finished(ctx, competitorID)
	if err != nil {
		return domain.Profile{}, err
	}
	if len(finished) == 0 {
		return domain.Profile{Languages: []string{}, Groups: []domain.GroupTerms{}}, nil
	}
	latest, err := s.repo.Get(ctx, finished[0].ID)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("load scan: %w", err)
	}
	first, err := s.repo.FirstScans(ctx, competitorID)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("first scans: %w", err)
	}
	return domain.BuildProfile(latest, len(finished), first), nil
}

// Compare compares two finished scans of a company; a nil id means the default for that side
// (see domain.Pick). It returns the older scan, the newer and the changes.
func (s *Service) Compare(ctx context.Context, competitorID string, from, to uuid.UUID) (domain.Analysis, domain.Analysis, []domain.ComparisonGroup, error) {
	finished, err := s.finished(ctx, competitorID)
	if err != nil {
		return domain.Analysis{}, domain.Analysis{}, nil, err
	}
	older, newer, err := domain.Pick(finished, from, to)
	if err != nil {
		return domain.Analysis{}, domain.Analysis{}, nil, err
	}
	a, err := s.repo.Get(ctx, older.ID)
	if err != nil {
		return domain.Analysis{}, domain.Analysis{}, nil, fmt.Errorf("load scan: %w", err)
	}
	b, err := s.repo.Get(ctx, newer.ID)
	if err != nil {
		return domain.Analysis{}, domain.Analysis{}, nil, fmt.Errorf("load scan: %w", err)
	}
	return older, newer, domain.Compare(a, b), nil
}

// finished returns a known company's finished scans without results, newest first.
func (s *Service) finished(ctx context.Context, competitorID string) ([]domain.Analysis, error) {
	companies, err := s.directory.Companies(ctx)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}
	if !slices.ContainsFunc(companies, func(c domain.Company) bool { return c.ID == competitorID }) {
		return nil, ErrUnknownCompany
	}
	scans, err := s.repo.ListByCompetitor(ctx, competitorID, MaxHistory)
	if err != nil {
		return nil, fmt.Errorf("scans of %s: %w", competitorID, err)
	}
	return domain.Finished(scans), nil
}

// Get returns one analysis with its results, or domain.ErrNotFound.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.Analysis, error) {
	return s.repo.Get(ctx, id)
}

// List returns up to limit analyses, newest first. A limit below 1 means DefaultListLimit.
func (s *Service) List(ctx context.Context, limit int) ([]domain.Analysis, error) {
	switch {
	case limit < 1:
		limit = DefaultListLimit
	case limit > MaxListLimit:
		limit = MaxListLimit
	}
	return s.repo.List(ctx, limit)
}

// Recover fails every analysis a restart interrupted and queues every analysis still waiting.
// Call it once at startup, before Work.
func (s *Service) Recover(ctx context.Context) error {
	if err := s.repo.FailRunning(ctx, "The service restarted while this analysis ran. Start it again.", s.now()); err != nil {
		return fmt.Errorf("fail interrupted analyses: %w", err)
	}
	ids, err := s.repo.Queued(ctx)
	if err != nil {
		return fmt.Errorf("list queued analyses: %w", err)
	}
	for _, id := range ids {
		select {
		case s.queue <- id:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// Work runs queued analyses, Limits.Workers at a time, until ctx ends, and returns when every
// worker stopped. A company never has two scans running, so each site is crawled by one worker
// at a time. onError hears about every analysis that could not be recorded, which only the logs
// can report.
func (s *Service) Work(ctx context.Context, onError func(uuid.UUID, error)) {
	var wg sync.WaitGroup
	for range max(s.limits.Workers, 1) {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case id := <-s.queue:
					if err := s.Run(ctx, id); err != nil {
						onError(id, err)
					}
				}
			}
		})
	}
	wg.Wait()
}

// Run crawls the analysis's site, extracts its keywords and known terms, matches its products,
// and records each step. A failure of the site itself is recorded on the analysis; Run returns
// an error only when the analysis could not be recorded.
func (s *Service) Run(ctx context.Context, id uuid.UUID) error {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("load analysis: %w", err)
	}
	if a.Status != domain.StatusQueued {
		return nil
	}
	if err := s.repo.MarkRunning(ctx, id); err != nil {
		return fmt.Errorf("mark running: %w", err)
	}
	target, err := domain.NewTarget(a.URL)
	if err != nil {
		return s.fail(ctx, id, err.Error())
	}
	glossary := s.glossary()
	lexicon := domain.NewLexicon(glossary)
	var pages []domain.Page
	var recordErr error
	crawlErr := s.crawler.Crawl(ctx, target, a.MaxPages, domain.NewLinkScorer(lexicon), func(p domain.Page) error {
		if err := s.repo.AddPage(ctx, id, domain.Summarise(p, lexicon)); err != nil {
			recordErr = fmt.Errorf("add page: %w", err)
			return recordErr
		}
		pages = append(pages, p)
		return nil
	})
	if recordErr != nil {
		return recordErr
	}
	if len(pages) == 0 {
		reason := "No page could be fetched."
		if crawlErr != nil {
			reason = "No page could be fetched: " + crawlErr.Error()
		}
		return s.fail(ctx, id, reason)
	}
	keywords := domain.Keywords(pages, lexicon, domain.BrandWords(target.URL().Hostname()))
	if err := s.repo.SaveFindings(ctx, id, keywords, domain.FindKnownTerms(pages, glossary)); err != nil {
		return fmt.Errorf("save findings: %w", err)
	}
	matching, model, eqs, usage := s.match(ctx, pages)
	if err := s.repo.SaveMatching(ctx, id, matching, model, eqs, usage); err != nil {
		return fmt.Errorf("save matching: %w", err)
	}
	if err := s.repo.Finish(ctx, id, s.now()); err != nil {
		return fmt.Errorf("finish analysis: %w", err)
	}
	return nil
}

// match asks the language model the settings name now for the site's equivalents of Tribelt's
// families, within the monthly budget, and returns the outcome with the provider and model it
// asked. It never fails the analysis: the crawl results stand on their own.
func (s *Service) match(ctx context.Context, pages []domain.Page) (domain.Matching, string, []domain.Equivalent, domain.Usage) {
	matcher, model, err := s.matchers.Current(ctx)
	switch {
	case err != nil:
		return domain.MatchingFailed, "", nil, domain.Usage{}
	case matcher == nil:
		return domain.MatchingDisabled, "", nil, domain.Usage{}
	}
	now := s.now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	spent, err := s.repo.TokensSince(ctx, monthStart)
	if err != nil || spent >= s.limits.MonthlyTokens {
		return domain.MatchingSkipped, "", nil, domain.Usage{}
	}
	text, truncated := domain.SiteText(pages, s.limits.SiteTextChars)
	if strings.TrimSpace(text) == "" {
		return domain.MatchingSkipped, "", nil, domain.Usage{}
	}
	found, usage, err := matcher.Match(ctx, text, truncated, domain.TribeltFamilies())
	if err != nil {
		return domain.MatchingFailed, model, nil, usage
	}
	eqs := make([]domain.Equivalent, 0, len(found))
	for _, e := range found {
		if checked, err := domain.CheckEquivalent(e); err == nil {
			eqs = append(eqs, checked)
		}
	}
	return domain.MatchingDone, model, eqs, usage
}

func (s *Service) fail(ctx context.Context, id uuid.UUID, reason string) error {
	if r := []rune(reason); len(r) > maxFailure {
		reason = string(r[:maxFailure-1]) + "…"
	}
	if err := s.repo.Fail(ctx, id, reason, s.now()); err != nil {
		return fmt.Errorf("fail analysis: %w", err)
	}
	return nil
}
