package domain_test

import (
	"errors"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

func TestClaudeHaikuIsPricedAtItsListPrice(t *testing.T) {
	// $0.10 per million input tokens and $0.50 per million output tokens.
	if got := domain.ListPrice(domain.DefaultModel).Of(1_000_000, 1_000_000); got != 600_000 {
		t.Fatalf("a million in and out = %d micros, want 600,000", got)
	}
	if got := (domain.TokenUse{Model: domain.DefaultModel, Input: 30_000, Output: 2_000}).Cost(); got != 4_000 {
		t.Fatalf("30,000 in and 2,000 out = %d micros, want 4,000", got)
	}
	if got := domain.ListPrice(domain.DefaultModel).Of(3, 0); got != 0 {
		t.Fatalf("three input tokens = %d micros, want 0 (rounded)", got)
	}
	if got := domain.ListPrice(domain.DefaultModel).Of(5, 0); got != 1 {
		t.Fatalf("five input tokens = %d micros, want 1 (rounded half up)", got)
	}
}

func TestAModelWithoutAListPriceHereIsPricedAsClaudeHaiku(t *testing.T) {
	if domain.ListPrice("claude-other") != domain.ListPrice(domain.DefaultModel) {
		t.Fatal("an unknown model is not priced as Claude Haiku")
	}
}

func TestAUseCountsItsTokensInAndOut(t *testing.T) {
	if n := (domain.TokenUse{Input: 1200, Output: 34}).Tokens(); n != 1234 {
		t.Fatalf("tokens = %d", n)
	}
}

func TestMicrosAreMillionthsOfADollar(t *testing.T) {
	if m := domain.MicrosOf(12.5); m != 12_500_000 || m.Dollars() != 12.5 || m.String() != "$12.50" {
		t.Fatalf("$12.50 = %d micros, %v dollars, %q", m, m.Dollars(), m.String())
	}
	if s := domain.Micros(4_000).String(); s != "$0.004" {
		t.Fatalf("4,000 micros reads %q", s)
	}
}

func TestAMonthsBudgetIsOneCentToAThousandDollars(t *testing.T) {
	for _, ok := range []domain.Micros{10_000, domain.DefaultBudget, 1_000_000_000} {
		if got, err := domain.CheckBudget(ok); err != nil || got != ok {
			t.Errorf("CheckBudget(%d) = %d, %v", ok, got, err)
		}
	}
	for _, bad := range []domain.Micros{-1, 0, 9_999, 1_000_000_001} {
		if _, err := domain.CheckBudget(bad); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("CheckBudget(%d) = %v, want ErrInvalid", bad, err)
		}
	}
	if domain.DefaultBudget != 15_000_000 {
		t.Fatalf("the default budget is %d micros, want $15", domain.DefaultBudget)
	}
}
