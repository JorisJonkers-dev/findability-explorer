import type { LlmBudget } from '../../../infrastructure/api/types.gen'

const NUMBER = new Intl.NumberFormat('en-GB')

export type BudgetMeter = {
  /** How much of the month's tokens are used, from 0 to 1, for the bar. */
  fraction: number
  /** "48,211 of 2,000,000 tokens used this month (2%)". */
  text: string
  /** What one company scan uses on average: "A company scan uses 29,137 tokens on average, over 2 scans." */
  perScan: string
  /** How many more scans this month's tokens cover, or null before there is an average. */
  left: string | null
}

/** The token budget as a meter, and what one scan uses. */
export function budgetMeter(budget: LlmBudget): BudgetMeter {
  const fraction = budget.monthlyTokens > 0 ? Math.min(budget.usedThisMonth / budget.monthlyTokens, 1) : budget.usedThisMonth > 0 ? 1 : 0
  const share = `${String(Math.round(fraction * 100))}%`
  return {
    fraction,
    text: `${NUMBER.format(budget.usedThisMonth)} of ${NUMBER.format(budget.monthlyTokens)} tokens used this month (${share})`,
    ...perScan(budget),
  }
}

/** The average tokens of a scan, and how many more scans the month's remaining tokens cover. */
function perScan(budget: LlmBudget): Pick<BudgetMeter, 'perScan' | 'left'> {
  if (budget.scansMatched === 0 || budget.averageTokensPerScan === 0) {
    return { perScan: 'No scan has used the language model yet, so there is no average.', left: null }
  }
  const over = budget.scansMatched === 1 ? '1 scan' : `${NUMBER.format(budget.scansMatched)} scans`
  const text = `A company scan uses ${NUMBER.format(budget.averageTokensPerScan)} tokens on average, over ${over}.`
  const remaining = Math.max(budget.monthlyTokens - budget.usedThisMonth, 0)
  const more = Math.floor(remaining / budget.averageTokensPerScan)
  if (more === 0 && remaining > 0) {
    return { perScan: text, left: 'What is left this month covers less than one more scan.' }
  }
  if (more === 0) {
    return { perScan: text, left: 'This month’s tokens are used up; scans run without product matching until next month.' }
  }
  return { perScan: text, left: `What is left this month covers about ${NUMBER.format(more)} more ${more === 1 ? 'scan' : 'scans'}.` }
}
