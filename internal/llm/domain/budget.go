package domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Micros is an amount in millionths of a dollar.
type Micros int64

// MicrosOf is dollars in micros, rounded.
func MicrosOf(dollars float64) Micros { return Micros(math.Round(dollars * 1e6)) }

// Dollars is m in dollars.
func (m Micros) Dollars() float64 { return float64(m) / 1e6 }

// String is m as an admin reads it: "$12.50", or "$0.004" for less than a cent's precision.
func (m Micros) String() string {
	if m%10_000 == 0 {
		return fmt.Sprintf("$%.2f", m.Dollars())
	}
	return "$" + strings.TrimRight(strconv.FormatFloat(m.Dollars(), 'f', 6, 64), "0")
}

// Price is a model's list price per million tokens.
type Price struct {
	InputPerMillion  Micros
	OutputPerMillion Micros
}

// Of is what input and output tokens cost at p, rounded to the nearest micro.
func (p Price) Of(input, output int64) Micros {
	return (Micros(input)*p.InputPerMillion + Micros(output)*p.OutputPerMillion + 500_000) / 1_000_000
}

// listPrices are the Claude models' list prices on Anthropic's API, for prompts up to 100,000
// tokens. Claude Haiku 5.5 costs $0.10 per million input tokens and $0.50 per million output
// tokens (ADR-0007).
var listPrices = map[string]Price{ //nolint:gochecknoglobals // a constant table
	DefaultModel: {InputPerMillion: 100_000, OutputPerMillion: 500_000},
}

// ListPrice is model's list price on Anthropic's API. Every Claude step uses Claude Haiku 5.5 until
// the steps can choose other models, so a model without a price of its own here is priced as Haiku.
func ListPrice(model string) Price {
	if p, ok := listPrices[model]; ok {
		return p
	}
	return listPrices[DefaultModel]
}

// TokenUse is what one model used: its tokens in and out.
type TokenUse struct {
	Model  string
	Input  int64
	Output int64
}

// Tokens is the use's tokens in and out together.
func (u TokenUse) Tokens() int64 { return u.Input + u.Output }

// Cost is what the use cost at its model's list price.
func (u TokenUse) Cost() Micros { return ListPrice(u.Model).Of(u.Input, u.Output) }

// Limits of the Anthropic key's monthly budget.
const (
	// DefaultBudget is the month's budget until an admin sets one: $15.
	DefaultBudget Micros = 15_000_000
	// MinBudget is one cent.
	MinBudget Micros = 10_000
	// MaxBudget is $1,000.
	MaxBudget Micros = 1_000_000_000
)

// ErrInvalidBudget is returned for a month's budget under one cent or over $1,000.
var ErrInvalidBudget = fmt.Errorf("%w: a month's budget is $0.01 to $1,000", ErrInvalid)

// CheckBudget checks a month's budget.
func CheckBudget(m Micros) (Micros, error) {
	if m < MinBudget || m > MaxBudget {
		return 0, ErrInvalidBudget
	}
	return m, nil
}
