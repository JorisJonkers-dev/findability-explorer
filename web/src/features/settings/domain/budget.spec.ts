import { describe, expect, it } from 'vitest'
import { budgetMeter } from './budget'

describe('the budget meter', () => {
  it('shows the tokens used against the month, and the hourly limit', () => {
    expect(budgetMeter({ monthlyTokens: 2_000_000, usedThisMonth: 48_211, analysesPerHour: 10 })).toEqual({
      fraction: 48_211 / 2_000_000,
      text: '48,211 of 2,000,000 tokens used this month (2%)',
      hourly: '10 analyses per account per hour',
    })
  })

  it('fills at most the whole bar, and copes with a budget of nothing', () => {
    expect(budgetMeter({ monthlyTokens: 100, usedThisMonth: 250, analysesPerHour: 1 }).fraction).toBe(1)
    expect(budgetMeter({ monthlyTokens: 100, usedThisMonth: 250, analysesPerHour: 1 }).hourly).toBe('1 analysis per account per hour')
    expect(budgetMeter({ monthlyTokens: 0, usedThisMonth: 5, analysesPerHour: 0 }).fraction).toBe(1)
    expect(budgetMeter({ monthlyTokens: 0, usedThisMonth: 0, analysesPerHour: 0 }).fraction).toBe(0)
  })
})
