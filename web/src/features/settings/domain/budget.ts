import type { LlmBudget } from '../../../infrastructure/api/types.gen'

const NUMBER = new Intl.NumberFormat('en-GB')

export type BudgetMeter = {
  /** How much of the month's tokens are used, from 0 to 1, for the bar. */
  fraction: number
  /** "48,211 of 2,000,000 tokens used this month (2%)". */
  text: string
  /** "10 analyses per account per hour". */
  hourly: string
}

/** The token budget as a meter, and the hourly limit in words. */
export function budgetMeter(budget: LlmBudget): BudgetMeter {
  const fraction = budget.monthlyTokens > 0 ? Math.min(budget.usedThisMonth / budget.monthlyTokens, 1) : budget.usedThisMonth > 0 ? 1 : 0
  const share = `${String(Math.round(fraction * 100))}%`
  return {
    fraction,
    text: `${NUMBER.format(budget.usedThisMonth)} of ${NUMBER.format(budget.monthlyTokens)} tokens used this month (${share})`,
    hourly: `${NUMBER.format(budget.analysesPerHour)} ${budget.analysesPerHour === 1 ? 'analysis' : 'analyses'} per account per hour`,
  }
}
