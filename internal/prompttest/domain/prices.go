package domain

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

// PricesFromRun prices runs by a run's own answers: each provider's average answer per mode,
// and the average cost of reading one, for questions questions.
func PricesFromRun(answers []Answer, models []ModelChoice, questions int, cadence Cadence, budget Micros) Prices {
	p := Prices{Basis: BasisRun}
	var read, reads Micros
	for _, a := range answers {
		if a.Extraction != nil {
			read += a.Extraction.Cost
			reads++
		}
	}
	if reads > 0 {
		p.Reading = read / reads
	}
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

// PricesFromCatalogue estimates runs before any has run: each chosen model's price, and the
// extractor's, times typical tokens.
func PricesFromCatalogue(catalogue []CatalogueModel, models []ModelChoice, questions int, cadence Cadence, budget Micros) Prices {
	price := func(id string, in, out int) Micros {
		for _, m := range catalogue {
			if m.ID == id {
				return MicrosOf(m.PromptPrice*float64(in) + m.CompletionPrice*float64(out))
			}
		}
		return 0
	}
	p := Prices{Basis: BasisCatalogue, Reading: price(ExtractorModel, typicalReadIn, typicalReadOut)}
	for _, m := range models {
		one := price(m.Model, typicalAnswerIn, typicalAnswerOut)
		p.Providers = append(p.Providers, ProviderPrice{Provider: m.Provider, Model: m.Model, Fallback: m.Fallback, Knowledge: one, Web: one})
	}
	return p.total(questions, cadence, budget)
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
