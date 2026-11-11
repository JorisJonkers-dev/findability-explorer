import type { LlmBudget } from '../../../infrastructure/api/types.gen'
import { dollars } from './usage'

const NUMBER = new Intl.NumberFormat('en-GB')

export type BudgetMeter = {
  /** How much of the month's Anthropic budget is spent, from 0 to 1, for the bar. */
  fraction: number
  /** "$0.48 of $15.00 spent this month (3%), $14.52 left". */
  text: string
  /** What one company scan costs on average: "A company scan costs $0.0044 on average (29,137 tokens), over 2 scans." */
  perScan: string
  /** How many more scans what is left covers, or null before there is an average. */
  left: string | null
}

/** The Anthropic key's dollar budget as a meter, and what one scan costs. */
export function budgetMeter(budget: LlmBudget): BudgetMeter {
  const fraction = budget.monthlyBudgetUsd > 0 ? Math.min(budget.spentThisMonthUsd / budget.monthlyBudgetUsd, 1) : 0
  const share = `${String(Math.round(fraction * 100))}%`
  return {
    fraction,
    text: `${dollars(budget.spentThisMonthUsd)} of ${dollars(budget.monthlyBudgetUsd)} spent this month (${share}), ${dollars(budget.leftUsd)} left`,
    ...perScan(budget),
  }
}

/** What a scan costs on average, and how many more scans what is left of the month's budget covers. */
function perScan(budget: LlmBudget): Pick<BudgetMeter, 'perScan' | 'left'> {
  if (budget.scansMatched === 0 || budget.averageScanUsd === 0) {
    return { perScan: 'No scan has used the language model yet, so there is no average.', left: null }
  }
  const over = budget.scansMatched === 1 ? '1 scan' : `${NUMBER.format(budget.scansMatched)} scans`
  const text = `A company scan costs ${dollars(budget.averageScanUsd)} on average (${NUMBER.format(budget.averageTokensPerScan)} tokens), over ${over}.`
  // In whole micros, as the API counts them, so $14.52 at $0.0044 a scan is 3,300 scans, not 3,299.
  const more = Math.floor(micros(budget.leftUsd) / micros(budget.averageScanUsd))
  if (more === 0 && budget.leftUsd > 0) {
    return { perScan: text, left: 'What is left this month covers less than one more scan.' }
  }
  if (more === 0) {
    return { perScan: text, left: 'This month’s Anthropic budget is spent; scans run without product matching until next month.' }
  }
  return { perScan: text, left: `What is left this month covers about ${NUMBER.format(more)} more ${more === 1 ? 'scan' : 'scans'}.` }
}

function micros(usd: number): number {
  return Math.round(usd * 1e6)
}

/** The most and least a month's budget may be, in dollars, as the API takes it. */
export const BUDGET_RANGE = { min: 0.01, max: 1000 } as const

/** A typed budget in dollars, checked: the amount, or why it cannot be saved. */
export function checkBudget(typed: string): { kind: 'ok'; usd: number } | { kind: 'invalid'; message: string } {
  const usd = Number(typed.trim().replace(/^\$/, '').replace(',', '.'))
  if (typed.trim() === '' || !Number.isFinite(usd) || usd < BUDGET_RANGE.min || usd > BUDGET_RANGE.max) {
    return { kind: 'invalid', message: 'Type a budget from $0.01 to $1,000.' }
  }
  return { kind: 'ok', usd: Math.round(usd * 100) / 100 }
}
