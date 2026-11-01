import type { PromptCatalogueModel, PromptModelCatalogue, PromptSets, PromptTestHistory, PromptTestSettings } from '../../infrastructure/api/types.gen'
import { STORED_ID } from '../settings/fixtures'

/** The Prompt test's settings: two providers, a $10 budget with $2.50 spent, monthly runs. */
export function promptSettings(overrides: Partial<PromptTestSettings> = {}): PromptTestSettings {
  return {
    credentialId: STORED_ID,
    providers: ['openai', 'google'],
    modelList: [],
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

function catalogueModel(id: string, extra: Partial<PromptCatalogueModel> = {}): PromptCatalogueModel {
  return { id, name: id, provider: id.split('/')[0] ?? id, free: false, recommended: false, basis: 'catalogue', knowledgeUsd: 0.001, webUsd: 0.021, ...extra }
}

/** OpenRouter's catalogue as Add a model lists it: two recommended, a free one, and a paid one. */
export function modelCatalogue(overrides: Partial<PromptModelCatalogue> = {}): PromptModelCatalogue {
  return {
    readAt: '2026-11-01T06:00:00Z',
    questions: 2,
    models: [
      catalogueModel('openai/gpt-oss-20b', { name: 'OpenAI: gpt-oss-20b', recommended: true, basis: 'run', knowledgeUsd: 0.0012, webUsd: 0.0061, listedAt: '2025-08-05T17:17:09Z' }),
      catalogueModel('google/gemma:free', { name: 'Google: Gemma 3 27B (free)', recommended: true, free: true, knowledgeUsd: 0, webUsd: 0.02, listedAt: '2025-03-12T05:12:39Z' }),
      catalogueModel('qwen/qwen3-8b:free', { name: 'Qwen: Qwen3 8B (free)', free: true, knowledgeUsd: 0, webUsd: 0.02 }),
      catalogueModel('openai/gpt-5-nano', { name: 'OpenAI: GPT-5 Nano', knowledgeUsd: 0.0004, webUsd: 0.0104, listedAt: '2025-08-07T17:23:33Z' }),
    ],
    ...overrides,
  }
}
