import { describe, expect, it } from 'vitest'
import type { LlmBudget } from '../../../infrastructure/api/types.gen'
import { budgetMeter } from './budget'

function budget(b: Partial<LlmBudget>): LlmBudget {
  return {
    monthlyBudgetUsd: 15,
    spentThisMonthUsd: 0,
    leftUsd: 15,
    tokensThisMonth: 0,
    averageTokensPerScan: 0,
    averageScanUsd: 0,
    scansMatched: 0,
    ...b,
  }
}

describe('the budget meter', () => {
  it('shows the dollars spent against the Anthropic budget, and what one scan costs', () => {
    expect(
      budgetMeter(
        budget({ spentThisMonthUsd: 0.48, leftUsd: 14.52, tokensThisMonth: 48_211, averageTokensPerScan: 29_137, averageScanUsd: 0.0044, scansMatched: 2 }),
      ),
    ).toEqual({
      fraction: 0.48 / 15,
      text: '$0.48 of $15.00 spent this month (3%), $14.52 left',
      perScan: 'A company scan costs $0.0044 on average (29,137 tokens), over 2 scans.',
      left: 'What is left this month covers about 3,300 more scans.',
    })
  })

  it('says what one scan costs, and how many more fit, once a scan has used the language model', () => {
    const none = budgetMeter(budget({}))
    expect(none.perScan).toBe('No scan has used the language model yet, so there is no average.')
    expect(none.left).toBeNull()
    const one = budgetMeter(budget({ monthlyBudgetUsd: 1, spentThisMonthUsd: 1.2, leftUsd: 0, averageTokensPerScan: 25_000, averageScanUsd: 0.5, scansMatched: 1 }))
    expect(one.perScan).toBe('A company scan costs $0.50 on average (25,000 tokens), over 1 scan.')
    expect(one.left).toBe('This month’s Anthropic budget is spent; scans run without product matching until next month.')
    expect(budgetMeter(budget({ leftUsd: 0.2, averageTokensPerScan: 25_000, averageScanUsd: 0.5, scansMatched: 1 })).left).toBe(
      'What is left this month covers less than one more scan.',
    )
    expect(budgetMeter(budget({ leftUsd: 0.5, averageTokensPerScan: 25_000, averageScanUsd: 0.5, scansMatched: 1 })).left).toBe(
      'What is left this month covers about 1 more scan.',
    )
  })

  it('fills at most the whole bar, and copes with a budget of nothing', () => {
    expect(budgetMeter(budget({ monthlyBudgetUsd: 1, spentThisMonthUsd: 2.5, leftUsd: 0 })).fraction).toBe(1)
    expect(budgetMeter(budget({})).fraction).toBe(0)
    expect(budgetMeter(budget({ monthlyBudgetUsd: 0, leftUsd: 0 })).fraction).toBe(0)
  })
})
