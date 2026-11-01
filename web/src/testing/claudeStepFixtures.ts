import type { ClaudeStep, ClaudeStepVersion } from '../infrastructure/api/types.gen'

/** A version of a Claude step. */
export function stepVersion(overrides: Partial<ClaudeStepVersion> = {}): ClaudeStepVersion {
  return {
    number: 2,
    model: 'claude-sonnet-5-5',
    modelName: 'Claude Sonnet 5.5',
    instructions: 'Match the metal belts.',
    note: '',
    createdBy: 'Joris',
    createdAt: '2026-11-01T09:00:00Z',
    ...overrides,
  }
}

/** Matching products on version 2, Claude Sonnet 5.5, after the original version 1. */
export function claudeStep(overrides: Partial<ClaudeStep> = {}): ClaudeStep {
  return {
    id: 'matching',
    title: 'Matching products',
    fixedInstructions: false,
    original: 'Compare the site with Tribelt.',
    current: stepVersion(),
    versions: [
      stepVersion({ number: 1, model: 'claude-haiku-5-5', modelName: 'Claude Haiku 5.5', instructions: 'Compare the site with Tribelt.', note: 'The instructions the tool came with.', createdBy: 'the tool', createdAt: '2026-10-20T08:00:00Z' }),
      stepVersion(),
    ],
    models: [
      { id: 'claude-haiku-5-5', name: 'Claude Haiku 5.5', jobCostUsd: 0.0044 },
      { id: 'claude-sonnet-5-5', name: 'Claude Sonnet 5.5', jobCostUsd: 0.088 },
      { id: 'claude-opus-5-5', name: 'Claude Opus 5.5', jobCostUsd: 0.176 },
    ],
    jobCostUsd: 0.088,
    ...overrides,
  }
}

/** A step on the instructions it came with, on Claude Haiku 5.5. */
export function originalStep(id: ClaudeStep['id'], title: string, fixedInstructions = false): ClaudeStep {
  return claudeStep({
    id,
    title,
    fixedInstructions,
    current: stepVersion({ number: 1, model: 'claude-haiku-5-5', modelName: 'Claude Haiku 5.5', createdBy: 'the tool', instructions: 'Compare the site with Tribelt.' }),
    jobCostUsd: 0.0044,
  })
}
