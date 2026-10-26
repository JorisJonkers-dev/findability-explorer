import { describe, expect, it } from 'vitest'
import { budgetMeter } from './budget'

describe('the budget meter', () => {
  it('shows the tokens used against the month, and what one scan uses', () => {
    expect(budgetMeter({ monthlyTokens: 2_000_000, usedThisMonth: 48_211, averageTokensPerScan: 29_137, scansMatched: 2 })).toEqual({
      fraction: 48_211 / 2_000_000,
      text: '48,211 of 2,000,000 tokens used this month (2%)',
      perScan: 'A company scan uses 29,137 tokens on average, over 2 scans.',
      left: 'What is left this month covers about 66 more scans.',
    })
  })

  it('says what one scan uses, and how many more fit, once a scan has used the language model', () => {
    const none = budgetMeter({ monthlyTokens: 2_000_000, usedThisMonth: 0, averageTokensPerScan: 0, scansMatched: 0 })
    expect(none.perScan).toBe('No scan has used the language model yet, so there is no average.')
    expect(none.left).toBeNull()
    const one = budgetMeter({ monthlyTokens: 30_000, usedThisMonth: 40_000, averageTokensPerScan: 25_000, scansMatched: 1 })
    expect(one.perScan).toBe('A company scan uses 25,000 tokens on average, over 1 scan.')
    expect(one.left).toBe('This month’s tokens are used up; scans run without product matching until next month.')
    expect(budgetMeter({ monthlyTokens: 30_000, usedThisMonth: 10_000, averageTokensPerScan: 25_000, scansMatched: 1 }).left).toBe(
      'What is left this month covers less than one more scan.',
    )
    expect(budgetMeter({ monthlyTokens: 30_000, usedThisMonth: 0, averageTokensPerScan: 25_000, scansMatched: 1 }).left).toBe(
      'What is left this month covers about 1 more scan.',
    )
  })

  it('fills at most the whole bar, and copes with a budget of nothing', () => {
    expect(budgetMeter({ monthlyTokens: 100, usedThisMonth: 250, averageTokensPerScan: 0, scansMatched: 0 }).fraction).toBe(1)
    expect(budgetMeter({ monthlyTokens: 0, usedThisMonth: 5, averageTokensPerScan: 0, scansMatched: 0 }).fraction).toBe(1)
    expect(budgetMeter({ monthlyTokens: 0, usedThisMonth: 0, averageTokensPerScan: 0, scansMatched: 0 }).fraction).toBe(0)
  })
})
