import type { AiSearchSummary, PromptTestSettings } from '../infrastructure/api/types.gen'
import { promptHistory } from './historyFixtures'
import { me, promptResults, readinessRow, readinessTable, summary } from './fixtures'
import { type Handler, json } from './mountApp'
import { overviewFixture, planFixture, standingsFixture } from './numbersFixtures'

/** The Prompt test settings an administrator reads: a monthly schedule, the next run on 8 November. */
export function promptSettingsFixture(overrides: Partial<PromptTestSettings> = {}): PromptTestSettings {
  return {
    providers: ['openai', 'google'],
    modelList: [],
    monthlyBudgetUsd: 10,
    spentThisMonthUsd: 2.5,
    cadence: 'monthly',
    nextDueAt: '2026-11-08T05:00:00Z',
    factSheet: { text: 'Tribelt makes metal conveyor belts in Haaksbergen.', version: 2, updatedBy: 'joris', updatedAt: '2026-10-26T09:00:00Z' },
    promptSet: { version: 1, prompts: [{ id: 'eyelink-supplier-twente', family: 'eyelink', kind: 'supplier', scope: 'twente', language: 'nl', text: 'Welke bedrijven leveren draadogenbanden in Twente?' }] },
    extractorModel: 'google/gemini-2.5-flash-lite',
    models: [{ provider: 'openai', model: 'openai/gpt-oss-20b', free: false }],
    keyWorks: true,
    prices: {
      basis: 'run',
      providers: [{ provider: 'openai', model: 'openai/gpt-oss-20b', knowledgeUsd: 0.0012, webUsd: 0.0061, runUsd: 0.15 }],
      readingUsd: 0.0002,
      runUsd: 0.25,
      runsPerMonth: 1,
      monthUsd: 0.25,
      budgetUsd: 10,
      overBudget: false,
    },
    ...overrides,
  }
}

/** The AI search summary, with how tribelt.nl does on four of the readiness checks. */
export function aiSearchFixture(): AiSearchSummary {
  return {
    goal: 0.4,
    followed: [{ competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt' }],
    trend: [],
    crawlers: [],
    keepingOut: [],
    ranking: [],
    sites: 41,
    questions: 20,
    providers: 2,
    readinessChecks: 8,
    checks: [
      { check: 'crawlers', scored: true, passing: 30, tribelt: true },
      { check: 'llmsTxt', scored: true, passing: 12, tribelt: false },
      { check: 'markdown', scored: true, passing: 2 },
      { check: 'product', scored: false, passing: 3, tribelt: false },
    ],
  }
}

/** Every request the Overview and the goal pages make, answered; extra replaces any of them. */
export function overviewRoutes(extra: Record<string, Handler> = {}, admin = false): Record<string, Handler> {
  return {
    'GET /api/v1/me': () => json(me({ isAdmin: admin })),
    'GET /api/v1/plan': () => json(planFixture()),
    'GET /api/v1/standings': () => json({ ...standingsFixture(), runStartedAt: '2026-11-01T06:00:00Z' }),
    'GET /api/v1/overview': () => json(overviewFixture()),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'GET /api/v1/ai-readiness': () =>
      json(
        readinessTable([
          readinessRow('Tribelt B.V.', { competitorId: 'tribelt', role: 'tribelt', scan: summary({ competitorId: 'tribelt', finishedAt: '2026-10-26T09:00:00Z' }) }),
          readinessRow('Ashworth', { competitorId: 'ashworth', role: 'competitor', scan: summary({ competitorId: 'ashworth', finishedAt: '2026-10-28T09:00:00Z' }) }),
        ]),
      ),
    'GET /api/v1/admin/prompt-test': () => json(promptSettingsFixture()),
    'GET /api/v1/ai-search': () => json(aiSearchFixture()),
    'GET /api/v1/prompt-test/history': () => json(promptHistory()),
    ...extra,
  }
}
