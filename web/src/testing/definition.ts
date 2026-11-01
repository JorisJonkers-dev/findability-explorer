import type { CompetitorDefinition } from '../infrastructure/api/types.gen'

/** The competitor definition at version 3, with version 2 imported from the code, and what checking again would take. */
export function definition(over: Partial<CompetitorDefinition> = {}): CompetitorDefinition {
  const current = { version: 3, text: 'Only makers of eyelink belts compete.', savedAt: '2026-11-01T09:00:00Z', savedBy: 'Joris' }
  return {
    current,
    versions: [current, { version: 2, text: 'A competitor makes metal belts.', savedAt: '2026-10-27T19:46:26Z', savedBy: 'import' }],
    outdatedCompetitors: 12,
    outdatedCandidates: 40,
    recheckCostUsd: 0.26,
    ...over,
  }
}
