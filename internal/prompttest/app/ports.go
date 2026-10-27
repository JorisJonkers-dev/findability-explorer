// Package app holds the prompt-test context's use cases: the settings, starting and running
// Prompt runs (on demand and on the schedule, one at a time across app instances, resumed after a
// restart, within the month's budget), reading their results, and reviewing competitor
// candidates. They are the only entry points the inbound adapters call.
package app

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// Gateway is OpenRouter, reached with the key Prompt runs use.
type Gateway interface {
	// Models lists OpenRouter's models with their prices.
	Models(ctx context.Context) ([]domain.CatalogueModel, error)
	// Ask asks model one prompt, as a buyer would, from its own knowledge or with web search. It
	// returns domain.ErrRateLimited when the model asks to slow down. A reply that failed still
	// counts what it cost.
	Ask(ctx context.Context, model, prompt string, web bool) (domain.Reply, error)
	// Extract has model read an answer in the strict format: the companies it names, the
	// addresses it cites, and whether what it says of Tribelt holds against the fact sheet.
	Extract(ctx context.Context, model string, in ExtractInput) (domain.Extracted, domain.Reply, error)
}

// ExtractInput is what the extractor reads.
type ExtractInput struct {
	Prompt    string
	Answer    string
	Citations []string
	FactSheet string
}

// Gateways hands out the Gateway for a stored credential: the language-model context's sealed
// OpenRouter keys (docs/adr/0002).
type Gateways interface {
	// Check returns nil when the credential is a readable OpenRouter key.
	Check(ctx context.Context, credentialID string) error
	// Gateway returns OpenRouter reached with the credential, or domain.ErrNoKey.
	Gateway(ctx context.Context, credentialID string) (Gateway, error)
}

// Directory is the port to the companies context: the companies names are matched to, and
// adding or removing a competitor candidate's company.
type Directory interface {
	Companies(ctx context.Context) ([]domain.DirectoryCompany, error)
	// Add adds a company by name and website and returns its id.
	Add(ctx context.Context, user, name, website string) (string, error)
	// Remove removes a company Add added.
	Remove(ctx context.Context, user, id string) error
}

// Repository stores the settings, the runs, their answers and the candidate decisions. The run
// methods are what keeps one run running at a time across app instances.
type Repository interface {
	// Settings returns the stored settings, or ok false when none are stored yet.
	Settings(ctx context.Context) (s domain.Settings, ok bool, err error)
	SaveSettings(ctx context.Context, s domain.Settings, by string, at time.Time) error

	// CreateRun stores a running run held by owner, or returns domain.ErrRunActive when one is
	// running already.
	CreateRun(ctx context.Context, r domain.Run, owner string, at time.Time) error
	// ClaimDue starts r when the schedule is due at now and no run is running: it moves the
	// schedule's next due time to next and stores r, held by owner, in one transaction, so only
	// one instance starts it. It reports whether it did.
	ClaimDue(ctx context.Context, now, next time.Time, r domain.Run, owner string) (bool, error)
	// AdoptStale takes over a running run whose holder has not been heard of since staleBefore,
	// as after a restart, and returns it; nil when there is none.
	AdoptStale(ctx context.Context, owner string, staleBefore, now time.Time) (*domain.Run, error)
	// Heartbeat records that owner still runs the run; false when another instance took it.
	Heartbeat(ctx context.Context, id uuid.UUID, owner string, at time.Time) (bool, error)
	FinishRun(ctx context.Context, id uuid.UUID, status domain.RunStatus, failure string, at time.Time) error
	// ReopenRun runs a finished run again, held by owner, or returns domain.ErrRunActive when a run
	// is running.
	ReopenRun(ctx context.Context, id uuid.UUID, owner string, at time.Time) error
	Run(ctx context.Context, id uuid.UUID) (domain.Run, error)
	// Runs returns at most limit runs, newest first.
	Runs(ctx context.Context, limit int) ([]domain.Run, error)

	// AddAnswer stores an answer with its extraction, if it has one. An answer to a task whose
	// answer failed replaces it, adding to what it cost; a task answered already is domain.ErrAnswered.
	AddAnswer(ctx context.Context, a domain.Answer) error
	// SetExtraction stores the extraction of an answer stored without one.
	SetExtraction(ctx context.Context, answerID uuid.UUID, x domain.Extraction, at time.Time) error
	// Answers returns a run's answers, with their text only when withText is true.
	Answers(ctx context.Context, runID uuid.UUID, withText bool) ([]domain.Answer, error)
	// SpentSince sums the cost of every answer and extraction at or after since.
	SpentSince(ctx context.Context, since time.Time) (domain.Micros, error)

	// Sightings returns every name an answer gave that was not matched to a company.
	Sightings(ctx context.Context) ([]domain.Sighting, error)
	CandidateDecisions(ctx context.Context) ([]domain.CandidateDecision, error)
	Decide(ctx context.Context, d domain.CandidateDecision) error
	// Undecide deletes a decision, or returns domain.ErrNotFound.
	Undecide(ctx context.Context, id string) error
	// Merges returns every merge of candidates.
	Merges(ctx context.Context) ([]domain.Merge, error)
	// SaveMerges stores merges, each replacing the one on its candidate.
	SaveMerges(ctx context.Context, merges []domain.Merge) error
	// CompanyVerdicts returns every check of a candidate by the language model.
	CompanyVerdicts(ctx context.Context) ([]domain.CompanyVerdict, error)
	// SaveCompanyVerdicts stores verdicts, each replacing the one on its candidate.
	SaveCompanyVerdicts(ctx context.Context, verdicts []domain.CompanyVerdict) error
}

// Reviewer is a language model that reviews the candidates: which are one company, or a company
// on the list, and which are competitors of Tribelt.
type Reviewer interface {
	Merge(ctx context.Context, candidates []domain.Candidate, directory []domain.DirectoryCompany) ([]domain.Cluster, error)
	Check(ctx context.Context, candidates []domain.Candidate) ([]domain.CompanyVerdict, error)
}

// Reviewers hands out the reviewer and the model it asks: Claude Haiku on an Anthropic key. A nil
// Reviewer without an error means no Anthropic key is set up.
type Reviewers interface {
	Current(ctx context.Context) (Reviewer, string, error)
}
