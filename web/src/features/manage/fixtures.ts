import type { PromptSets, PromptTestHistory, PromptTestSettings } from '../../infrastructure/api/types.gen'
import { STORED_ID } from '../settings/fixtures'

/** The Prompt test's settings: two providers, a $10 budget with $2.50 spent, monthly runs. */
export function promptSettings(overrides: Partial<PromptTestSettings> = {}): PromptTestSettings {
  return {
    credentialId: STORED_ID,
    providers: ['openai', 'google'],
    monthlyBudgetUsd: 10,
    spentThisMonthUsd: 2.5,
    cadence: 'monthly',
    nextDueAt: '2026-12-01T06:00:00Z',
    factSheet: { text: 'Tribelt makes metal conveyor belts in Haaksbergen.', version: 2, updatedBy: 'joris', updatedAt: '2026-10-26T09:00:00Z' },
    promptSet: {
      version: 2,
      prompts: [
        { id: 'eyelink-supplier-twente', family: 'eyelink', kind: 'supplier', scope: 'twente', language: 'nl', text: 'Welke bedrijven leveren draadogenbanden in Twente?' },
        { id: 'other-supplier-english', family: 'other', kind: 'supplier', scope: 'english', language: 'en', text: 'Which companies supply rod belts?' },
      ],
    },
    extractorModel: 'google/gemini-2.5-flash-lite',
    models: [
      { provider: 'openai', model: 'openai/gpt-oss-20b', free: false },
      { provider: 'google', model: 'google/gemma:free', free: true, fallback: 'google/gemini-2.5-flash-lite' },
    ],
    keyWorks: true,
    prices: {
      basis: 'run',
      providers: [
        { provider: 'openai', model: 'openai/gpt-oss-20b', knowledgeUsd: 0.0012, webUsd: 0.0061, runUsd: 0.15 },
        { provider: 'google', model: 'google/gemma:free', fallback: 'google/gemini-2.5-flash-lite', knowledgeUsd: 0, webUsd: 0.004, runUsd: 0.08 },
      ],
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

/** Two published versions of the Prompt set, newest first. */
export function promptSets(): PromptSets {
  return {
    versions: [
      {
        version: 2,
        publishedAt: '2026-10-27T15:00:00Z',
        note: 'Adds German and English questions.',
        questions: [
          { id: 'eyelink-twente', family: 'eyelink', intent: 'product', scope: 'twente', scopeName: 'Dutch, Twente', language: 'nl', region: 'Twente', text: 'Wie levert draadogenbanden in Twente?', english: 'Who supplies eyelink belts in Twente?' },
          { id: 'other-english', family: 'other', intent: 'product', scope: 'english', scopeName: 'English', language: 'en', text: 'Which companies supply rod belts?' },
        ],
      },
      {
        version: 1,
        publishedAt: '2026-10-20T09:30:00Z',
        note: 'The first questions.',
        questions: [{ id: 'eyelink-twente', family: 'eyelink', intent: 'product', scope: 'twente', scopeName: 'Dutch, Twente', language: 'nl', text: 'Wie levert draadogenbanden in Twente?' }],
      },
    ],
    scopes: [],
  }
}

/** The history's versions, oldest first, and no runs. */
export function promptHistory(): PromptTestHistory {
  return {
    versions: [
      { version: 1, publishedAt: '2026-10-20T09:30:00Z', questions: 1, note: 'The first questions.' },
      { version: 2, publishedAt: '2026-10-27T15:00:00Z', questions: 2, note: 'Adds German and English questions.' },
    ],
    runs: [],
  }
}
