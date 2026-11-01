import type { PromptTestHistory } from '../infrastructure/api/types.gen'

export const HISTORY_RUN_A = '0b6a1c52-3e0f-4c1e-9a8d-6f1e2d3c4b5a'
export const HISTORY_RUN_B = '7c2d9e41-5a6b-4f70-8e1d-2b3c4d5e6f70'

/**
 * A Prompt test history with two versions: a partial run of version 1, its key refused, then a
 * whole run of version 2. Made-up figures, never the live data.
 */
export function promptHistory(overrides: Partial<PromptTestHistory> = {}): PromptTestHistory {
  return {
    versions: [
      { version: 1, publishedAt: '2026-10-26T15:04:51Z', questions: 90, note: 'A supplier, an application, and a problem question for each belt family, in six scopes.' },
      { version: 2, publishedAt: '2026-10-27T06:35:33Z', questions: 20, note: 'Kept the application question a buyer most often asks, in four scopes.' },
    ],
    runs: [
      { runId: HISTORY_RUN_A, startedAt: '2026-10-27T06:03:12Z', promptSetVersion: 1, tasks: 1260, answered: 62, read: 47, partial: true, failure: 'OpenRouter answered 401: User not found.', all: 0.4, dutch: 0.48 },
      { runId: HISTORY_RUN_B, startedAt: '2026-10-27T08:36:32Z', promptSetVersion: 2, tasks: 200, answered: 200, read: 183, partial: false, all: 0.25, dutch: 0.41, english: 0.1 },
    ],
    ...overrides,
  }
}
