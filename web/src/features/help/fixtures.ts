import type { PromptTestSettings } from '../../infrastructure/api/types.gen'

/** The Prompt test's settings as an administrator reads them, with the prices Help's figures come from. */
export function promptSettings(prices: { runUsd: number; monthUsd: number }, cadence: PromptTestSettings['cadence'] = 'monthly'): PromptTestSettings {
  return {
    providers: ['openai'],
    monthlyBudgetUsd: 10,
    spentThisMonthUsd: 0,
    cadence,
    nextDueAt: '2026-11-01T06:00:00Z',
    factSheet: { text: 'Tribelt makes metal conveyor belts.', version: 1, updatedBy: 'joris', updatedAt: '2026-10-26T09:00:00Z' },
    promptSet: { version: 1, prompts: [] },
    extractorModel: 'anthropic/claude-haiku',
    models: [],
    keyWorks: true,
    prices: { basis: 'run', providers: [], readingUsd: 0, runUsd: prices.runUsd, runsPerMonth: 1, monthUsd: prices.monthUsd, budgetUsd: 10, overBudget: false },
  }
}
