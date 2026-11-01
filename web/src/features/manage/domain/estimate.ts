import type { PromptCadence, PromptPrices } from '../../../infrastructure/api/types.gen'
import { costLabel } from '../../prompt-test/domain/results'

/**
 * The budget a start is checked against. Prompt runs, their retries and their readings are paid from
 * the Prompt test's monthly budget, which the OpenRouter key spends; the dollar budget per key
 * arrives with its own ticket.
 */
export type Budget = { budgetUsd: number; spentUsd: number }

const BUDGET_NAME = 'OpenRouter'

/** What is left of the month's budget. */
export function leftUsd(budget: Budget): number {
  return Math.max(0, Math.round((budget.budgetUsd - budget.spentUsd) * 100) / 100)
}

/** How much of the budget is used, from 0 to 1, for its meter. */
export function budgetFraction(budget: Budget): number {
  return budget.budgetUsd > 0 ? Math.min(budget.spentUsd / budget.budgetUsd, 1) : 0
}

/** The estimate beside a start button: "About $0.25. $7.50 of this month’s OpenRouter budget is left." */
export function estimateLine(usd: number, left: number, note?: string): string {
  const cost = usd > 0 ? `About ${costLabel(usd)}.` : 'Free.'
  return `${cost}${note ? ` ${note}.` : ''} ${costLabel(left)} of this month’s ${BUDGET_NAME} budget is left.`
}

/** The warning shown before the click when the start costs more than the budget has left, else null. */
export function overBudgetLine(usd: number, left: number): string | null {
  return usd > left ? `This costs more than the ${costLabel(left)} left of this month’s ${BUDGET_NAME} budget.` : null
}

/** What the first click on an over-budget start says, before Start anyway. */
export function confirmLine(usd: number, left: number): string {
  return `This costs about ${costLabel(usd)}. Only ${costLabel(left)} is left of this month’s ${BUDGET_NAME} budget. The work stops when the budget runs out, and keeps what it has done.`
}

const RUNS_PER_YEAR: Record<PromptCadence, number> = { monthly: 12, weekly: 52 }

/** What the schedule costs: a run, a month, and a year. */
export function scheduleCosts(
  prices: Pick<PromptPrices, 'runUsd' | 'runsPerMonth' | 'monthUsd'>,
  cadence: PromptCadence,
): { run: string; month: string; year: string; runsPerYear: number } {
  const runsPerYear = RUNS_PER_YEAR[cadence]
  return { run: costLabel(prices.runUsd), month: costLabel(prices.monthUsd), year: costLabel(prices.runUsd * runsPerYear), runsPerYear }
}

/** The warning when a month of runs costs more than the budget, and whether Set to monthly would help. */
export function scheduleWarning(
  prices: Pick<PromptPrices, 'overBudget' | 'monthUsd' | 'budgetUsd'>,
  cadence: PromptCadence,
): { text: string; setMonthly: boolean } | null {
  if (!prices.overBudget) return null
  const weekly = cadence === 'weekly'
  return {
    text: `At this schedule the runs cost about ${costLabel(prices.monthUsd)} a month, which is more than the ${costLabel(prices.budgetUsd)} monthly budget. Runs will stop when the budget is spent. ${weekly ? 'Choose monthly, or raise the budget.' : 'Raise the budget, or take a provider off the list.'}`,
    setMonthly: weekly,
  }
}

