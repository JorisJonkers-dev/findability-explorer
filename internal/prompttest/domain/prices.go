package domain

import "slices"

// PriceBasis is where a price overview comes from: the latest run's own costs, OpenRouter's
// price list before any run, or nothing to go by.
type PriceBasis string

const (
	BasisRun       PriceBasis = "run"
	BasisCatalogue PriceBasis = "catalogue"
	BasisNone      PriceBasis = "none"
)

// Typical tokens of one answer and of reading one, for an estimate before any run.
const (
	typicalAnswerIn  = 300
	typicalAnswerOut = 900
	typicalReadIn    = 1500
	typicalReadOut   = 250
)

// TokenPrice is what a model charges, in dollars per input token and per output token.
type TokenPrice struct {
	Input  float64
	Output float64
}

// Of is what in input and out output tokens cost.
func (p TokenPrice) Of(in, out int64) Micros {
	return MicrosOf(p.Input*float64(in) + p.Output*float64(out))
}

// HaikuPrice is Claude Haiku 5.5's list price on Anthropic's API, for prompts up to 100,000
// tokens, which its readings are priced at whatever the subsidy (ADR-0007).
func HaikuPrice() TokenPrice { return TokenPrice{Input: 0.10 / 1e6, Output: 0.50 / 1e6} }

// CataloguePrice is a model's price on OpenRouter's price list; false when the list does not have it.
func CataloguePrice(catalogue []CatalogueModel, id string) (TokenPrice, bool) {
	for _, m := range catalogue {
		if m.ID == id {
			return TokenPrice{Input: m.PromptPrice, Output: m.CompletionPrice}, true
		}
	}
	return TokenPrice{}, false
}

// exaSearchFee is what OpenRouter charges for its own web search, for a model that does not search
// itself: five results at $4 per 1,000.
const exaSearchFee Micros = 5 * 4_000

// WebSearchFee is what searching the web for one answer costs on top of its tokens: the model's own
// search at its price, or OpenRouter's.
func WebSearchFee(m CatalogueModel) Micros {
	if m.WebSearchPrice > 0 {
		return MicrosOf(m.WebSearchPrice)
	}
	return exaSearchFee
}

// ReadingCost is what reading one answer costs at the reader's price: the tokens a reading of
// answers used on average, or typical ones before any, times the price. Without a price to go by
// (nil), it is what reading answers cost on average.
func ReadingCost(answers []Answer, reader *TokenPrice) Micros {
	var in, out int64
	var cost, reads Micros
	for _, a := range answers {
		if a.Extraction != nil {
			in, out, cost, reads = in+a.Extraction.InputTokens, out+a.Extraction.OutputTokens, cost+a.Extraction.Cost, reads+1
		}
	}
	switch {
	case reader != nil && reads > 0:
		return reader.Of(in/int64(reads), out/int64(reads))
	case reader != nil:
		return reader.Of(typicalReadIn, typicalReadOut)
	case reads > 0:
		return cost / reads
	}
	return 0
}

// ProviderPrice is what one provider's answers cost: one answer from its own knowledge, one with
// web search, and all of them in a run.
type ProviderPrice struct {
	Provider  string
	Model     string
	Fallback  string
	Knowledge Micros
	Web       Micros
	PerRun    Micros
}

// Prices is what Prompt runs cost: per provider, reading the answers, one run, and a month of
// runs at the schedule against the budget.
type Prices struct {
	Basis        PriceBasis
	Providers    []ProviderPrice
	Reading      Micros
	Run          Micros
	RunsPerMonth float64
	Month        Micros
	Budget       Micros
}

// OverBudget reports whether a month of runs would cost more than the budget.
func (p Prices) OverBudget() bool { return p.Month > p.Budget }

// RunsPerMonth is how many scheduled runs a cadence starts in an average month.
func RunsPerMonth(c Cadence) float64 {
	if c == CadenceWeekly {
		return 52.0 / 12.0
	}
	return 1
}

// PricesFromRun prices runs by a run's own answers: each provider's average answer per mode, its
// web search included, and reading one at the price of the reader that will read the next run
// (ReadingCost), for questions questions.
func PricesFromRun(answers []Answer, models []ModelChoice, questions int, cadence Cadence, budget Micros, reader *TokenPrice) Prices {
	p := Prices{Basis: BasisRun, Reading: ReadingCost(answers, reader)}
	for _, m := range models {
		pp := ProviderPrice{Provider: m.Provider, Model: m.Model, Fallback: m.Fallback}
		pp.Knowledge = averageAnswer(answers, m.Provider, ModeKnowledge)
		pp.Web = averageAnswer(answers, m.Provider, ModeWeb)
		p.Providers = append(p.Providers, pp)
	}
	return p.total(questions, cadence, budget)
}

func averageAnswer(answers []Answer, provider string, mode Mode) Micros {
	var sum, n Micros
	for _, a := range answers {
		if a.Provider == provider && a.Mode == mode && a.Failure == "" {
			sum += a.Cost
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / n
}

// PricesFromCatalogue estimates runs before any has run: each chosen model's price times typical
// tokens, with the web search's fee on an answer with web search, and reading one at the price of
// the reader that will read them (nothing without a price to go by).
func PricesFromCatalogue(catalogue []CatalogueModel, models []ModelChoice, questions int, cadence Cadence, budget Micros, reader *TokenPrice) Prices {
	p := Prices{Basis: BasisCatalogue}
	if reader != nil {
		p.Reading = reader.Of(typicalReadIn, typicalReadOut)
	}
	for _, m := range models {
		pp := ProviderPrice{Provider: m.Provider, Model: m.Model, Fallback: m.Fallback}
		if i := slices.IndexFunc(catalogue, func(c CatalogueModel) bool { return c.ID == m.Model }); i >= 0 {
			c := catalogue[i]
			pp.Knowledge = TokenPrice{Input: c.PromptPrice, Output: c.CompletionPrice}.Of(typicalAnswerIn, typicalAnswerOut)
			pp.Web = pp.Knowledge + WebSearchFee(c)
		}
		p.Providers = append(p.Providers, pp)
	}
	return p.total(questions, cadence, budget)
}

// With is the overview with more providers' prices added, as the price list gives them for
// providers a run has not asked yet, and the totals counted again.
func (p Prices) With(extra []ProviderPrice, questions int, cadence Cadence) Prices {
	p.Providers = append(append([]ProviderPrice{}, p.Providers...), extra...)
	return p.total(questions, cadence, p.Budget)
}

// Asked are the models of a run that providers still in the settings answered with, and the
// providers in the settings the run did not ask.
func Asked(models []ModelChoice, providers []string) (kept []ModelChoice, missing []string) {
	for _, m := range models {
		if slices.Contains(providers, m.Provider) {
			kept = append(kept, m)
		}
	}
	for _, p := range providers {
		if !slices.ContainsFunc(models, func(m ModelChoice) bool { return m.Provider == p }) {
			missing = append(missing, p)
		}
	}
	return kept, missing
}

// NoPrices is the overview with nothing to go by.
func NoPrices(cadence Cadence, budget Micros) Prices {
	return Prices{Basis: BasisNone}.total(0, cadence, budget)
}

// total adds up a run and a month from the per-answer prices.
func (p Prices) total(questions int, cadence Cadence, budget Micros) Prices {
	q := Micros(questions)
	p.Run = 0
	for i := range p.Providers {
		pp := &p.Providers[i]
		pp.PerRun = q * (pp.Knowledge + pp.Web)
		p.Run += pp.PerRun + q*Micros(len(Modes()))*p.Reading
	}
	p.RunsPerMonth, p.Budget = RunsPerMonth(cadence), budget
	p.Month = MicrosOf(p.Run.Dollars() * p.RunsPerMonth)
	return p
}
