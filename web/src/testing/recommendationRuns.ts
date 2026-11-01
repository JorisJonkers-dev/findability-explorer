import type { RecommendationRuns } from '../infrastructure/api/types.gen'

/** The Recommendation run's page data: none running, 12 to replace, tribelt.nl scanned before the newest Prompt run, and two past runs. */
export function recommendationRuns(over: Partial<RecommendationRuns> = {}): RecommendationRuns {
  return {
    running: false,
    noKey: false,
    model: 'claude-haiku-5-5',
    replaces: 12,
    stale: true,
    promptRunAt: '2026-11-01T06:00:00Z',
    scanAt: '2026-10-31T09:00:00Z',
    terms: 420,
    gaps: 40,
    estimatedCostUsd: 0.07,
    runs: [
      {
        id: '3fa85f64-5717-4562-b3fc-2c963f66afa6', startedAt: '2026-11-01T08:00:00Z', finishedAt: '2026-11-01T08:02:00Z', ending: 'done',
        startedBy: 'Joris', model: 'claude-haiku-5-5', gaps: 40, added: 38, replaced: 10, kept: 2, costUsd: 0.06,
      },
      { id: '4fa85f64-5717-4562-b3fc-2c963f66afa6', startedAt: '2026-10-30T08:00:00Z', finishedAt: '2026-10-30T08:01:00Z', ending: 'stoppedByHand', startedBy: 'Joris', model: 'claude-haiku-5-5', gaps: 40, added: 0, replaced: 0, kept: 0, costUsd: 0.01 },
    ],
    ...over,
  }
}
