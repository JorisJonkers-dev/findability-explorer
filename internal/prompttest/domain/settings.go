package domain

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// Cadence is how often scheduled Prompt runs start.
type Cadence string

const (
	CadenceMonthly Cadence = "monthly"
	CadenceWeekly  Cadence = "weekly"
)

// scheduledHour is the hour, UTC, a scheduled run starts.
const scheduledHour = 6

// NextDue is when the next scheduled run starts after t: on the first of the next month, or the
// next Monday, at 06:00 UTC.
func NextDue(t time.Time, c Cadence) time.Time {
	t = t.UTC()
	if c == CadenceWeekly {
		days := (8 - int(t.Weekday())) % 7
		next := time.Date(t.Year(), t.Month(), t.Day()+days, scheduledHour, 0, 0, 0, time.UTC)
		if !next.After(t) {
			next = next.AddDate(0, 0, 7)
		}
		return next
	}
	return time.Date(t.Year(), t.Month()+1, 1, scheduledHour, 0, 0, 0, time.UTC)
}

// MonthStart is the start of t's calendar month, UTC, from which the budget counts.
func MonthStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// FactSheet is the administrator-kept statement of what is true about Tribelt, which answers
// about Tribelt are judged against. Each change is a new version.
type FactSheet struct {
	Text      string
	Version   int
	UpdatedBy string
	UpdatedAt time.Time
}

// DefaultFactSheet is the fact sheet until an administrator writes one.
func DefaultFactSheet() string {
	return "Tribelt B.V. makes metal conveyor belts in Haaksbergen, Twente, the Netherlands. It is part of the " +
		"Lesjöfors Group, like its sister company De Spiraal. It makes eyelink belts, open spiral (balanced " +
		"weave) belts, closed spiral (compound balanced) belts, Tri-Flex curve-going and spiral-tower belts, and " +
		"other metal belts, for food processing, baking, freezing and industry. Its website, www.tribelt.nl, is in " +
		"Dutch, English and German."
}

// Settings are what administrators decide about Prompt runs.
type Settings struct {
	// CredentialID is the stored OpenRouter key runs use; empty until one is chosen.
	CredentialID  string
	Providers     []string
	MonthlyBudget Micros
	Cadence       Cadence
	NextDue       time.Time
	FactSheet     FactSheet
}

// DefaultBudget is a month's budget for Prompt runs until an administrator sets one: $10.
const DefaultBudget Micros = 10_000_000

// DefaultSettings are the settings until an administrator changes them: no key, the large
// providers, $10 a month, monthly from the first of next month, the default fact sheet.
func DefaultSettings(now time.Time) Settings {
	return Settings{
		Providers: DefaultProviders(), MonthlyBudget: DefaultBudget, Cadence: CadenceMonthly,
		NextDue: NextDue(now, CadenceMonthly), FactSheet: FactSheet{Text: DefaultFactSheet(), Version: 1},
	}
}

// Rules for the settings.
const (
	MaxProviders  = 20
	MaxBudget     = Micros(1_000_000_000) // $1,000
	MaxFactSheet  = 4000
	minFactSheet  = 20
	maxProviderID = 40
)

// ErrInvalid wraps every settings rule broken.
var ErrInvalid = errors.New("invalid")

var providerID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// CheckProviders trims, lower-cases and de-duplicates provider ids: 1 to MaxProviders, each an
// OpenRouter provider id such as "openai" or "meta-llama".
func CheckProviders(raw []string) ([]string, error) {
	var out []string
	for _, p := range raw {
		p = strings.ToLower(strings.TrimSpace(p))
		if len(p) > maxProviderID || !providerID.MatchString(p) {
			return nil, fmt.Errorf("%w: %q is no provider id, such as openai or meta-llama", ErrInvalid, p)
		}
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	if len(out) == 0 || len(out) > MaxProviders {
		return nil, fmt.Errorf("%w: name 1 to %d providers", ErrInvalid, MaxProviders)
	}
	return out, nil
}

// CheckBudget checks a month's budget: more than $0 and at most $1,000.
func CheckBudget(m Micros) (Micros, error) {
	if m <= 0 || m > MaxBudget {
		return 0, fmt.Errorf("%w: a month's budget is more than $0 and at most $1,000", ErrInvalid)
	}
	return m, nil
}

// CheckCadence checks a cadence.
func CheckCadence(c string) (Cadence, error) {
	switch Cadence(c) {
	case CadenceMonthly, CadenceWeekly:
		return Cadence(c), nil
	}
	return "", fmt.Errorf("%w: the schedule is monthly or weekly", ErrInvalid)
}

// CheckFactSheet trims a fact sheet and checks its length.
func CheckFactSheet(text string) (string, error) {
	text = strings.TrimSpace(text)
	if n := utf8.RuneCountInString(text); n < minFactSheet || n > MaxFactSheet {
		return "", fmt.Errorf("%w: a fact sheet has %d to %d characters", ErrInvalid, minFactSheet, MaxFactSheet)
	}
	return text, nil
}
