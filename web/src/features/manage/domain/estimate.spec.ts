import { describe, expect, it } from 'vitest'
import { budgetFraction, confirmLine, estimateLine, leftUsd, overBudgetLine, readerOf, scheduleCosts, scheduleWarning } from './estimate'

describe('leftUsd and budgetFraction', () => {
  it('is what is left of the budget, never below nothing', () => {
    expect(leftUsd({ budgetUsd: 10, spentUsd: 2.5 })).toBe(7.5)
    expect(leftUsd({ budgetUsd: 10, spentUsd: 12 })).toBe(0)
  })

  it('is the share used, from 0 to 1', () => {
    expect(budgetFraction({ budgetUsd: 10, spentUsd: 2.5 })).toBe(0.25)
    expect(budgetFraction({ budgetUsd: 10, spentUsd: 20 })).toBe(1)
    expect(budgetFraction({ budgetUsd: 0, spentUsd: 0 })).toBe(0)
  })
})

describe('the estimate of a start', () => {
  it('says what it costs and what the budget has left', () => {
    expect(estimateLine(0.25, 7.5, 'OpenRouter')).toBe('About $0.25. $7.50 of this month’s OpenRouter budget is left.')
    expect(estimateLine(1.12, 8.12, 'OpenRouter', '200 responses')).toBe('About $1.12. 200 responses. $8.12 of this month’s OpenRouter budget is left.')
    expect(estimateLine(0, 8, 'Anthropic')).toBe('Free. $8.00 of this month’s Anthropic budget is left.')
  })

  it('warns before the click when it costs more than is left', () => {
    expect(overBudgetLine(0.25, 7.5, 'OpenRouter')).toBeNull()
    expect(overBudgetLine(9, 7.5, 'OpenRouter')).toBe('This costs more than the $7.50 left of this month’s OpenRouter budget.')
    expect(overBudgetLine(9, 7.5, 'Anthropic')).toBe('This costs more than the $7.50 left of this month’s Anthropic budget.')
    expect(confirmLine(9, 7.5, 'OpenRouter')).toBe(
      'This costs about $9.00. Only $7.50 is left of this month’s OpenRouter budget. The work stops when the budget runs out, and keeps what it has done.',
    )
  })
})

describe('the schedule', () => {
  it('costs a run, a month, and a year', () => {
    expect(scheduleCosts({ runUsd: 1.12, runsPerMonth: 1, monthUsd: 1.12 }, 'monthly')).toEqual({ run: '$1.12', month: '$1.12', year: '$13.44', runsPerYear: 12 })
    expect(scheduleCosts({ runUsd: 1, runsPerMonth: 4.33, monthUsd: 4.33 }, 'weekly')).toEqual({ run: '$1.00', month: '$4.33', year: '$52.00', runsPerYear: 52 })
  })

  it('warns when a month of runs costs more than the budget, with the fix for the schedule', () => {
    expect(scheduleWarning({ overBudget: false, monthUsd: 1, budgetUsd: 10 }, 'monthly')).toBeNull()
    expect(scheduleWarning({ overBudget: true, monthUsd: 12, budgetUsd: 10 }, 'weekly')).toEqual({
      text: 'At this schedule the runs cost about $12.00 a month, which is more than the $10.00 monthly budget. Runs will stop when the budget is spent. Choose monthly, or raise the budget.',
      setMonthly: true,
    })
    expect(scheduleWarning({ overBudget: true, monthUsd: 12, budgetUsd: 10 }, 'monthly')).toEqual({
      text: 'At this schedule the runs cost about $12.00 a month, which is more than the $10.00 monthly budget. Runs will stop when the budget is spent. Raise the budget, or take a provider off the list.',
      setMonthly: false,
    })
  })
})

describe('readerOf', () => {
  it('reads on the Anthropic key with a Claude model, else on OpenRouter', () => {
    expect(readerOf('anthropic: claude-haiku-5-5')).toEqual({ budget: 'Anthropic', model: 'claude-haiku-5-5' })
    expect(readerOf('google/gemini-2.5-flash-lite')).toEqual({ budget: 'OpenRouter', model: 'google/gemini-2.5-flash-lite' })
  })
})
