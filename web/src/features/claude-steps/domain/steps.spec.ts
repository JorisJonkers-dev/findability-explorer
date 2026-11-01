import { describe, expect, it } from 'vitest'
import type { ClaudeStep, ClaudeStepVersion } from '../../../infrastructure/api/types.gen'
import { costLine, dataLine, isDirty, modelLabel, STEP_INFO, stepInfo, versionLine } from './steps'

const version = (o: Partial<ClaudeStepVersion> = {}): ClaudeStepVersion => ({
  number: 2,
  model: 'claude-sonnet-5-5',
  modelName: 'Claude Sonnet 5.5',
  instructions: 'Match.',
  note: '',
  createdBy: 'Joris',
  createdAt: '2026-11-01T09:00:00Z',
  ...o,
})

const step = (o: Partial<ClaudeStep> = {}): ClaudeStep => ({
  id: 'matching',
  title: 'Matching products',
  fixedInstructions: false,
  original: 'Original.',
  current: version(),
  versions: [version({ number: 1, model: 'claude-haiku-5-5', modelName: 'Claude Haiku 5.5', createdBy: 'the tool' }), version()],
  models: [
    { id: 'claude-haiku-5-5', name: 'Claude Haiku 5.5', jobCostUsd: 0.0044 },
    { id: 'claude-sonnet-5-5', name: 'Claude Sonnet 5.5', jobCostUsd: 0.088 },
  ],
  jobCostUsd: 0.088,
  ...o,
})

describe('a Claude step', () => {
  it('has what it does, what it gets, what it replies, and where it belongs', () => {
    const info = stepInfo('matching')
    expect(info).toBe(STEP_INFO.matching)
    expect(info.section).toEqual({ label: 'Website scans', to: '/manage/scans' })
    expect(info.gets.map((g) => g.name)).toEqual(['Tribelt’s belt families', 'The website text'])
    expect(info.replies.map((r) => r.field)).toEqual(['product', 'family', 'reason', 'searchTerms'])
    expect(info.jobLabel).toBe('a scan')
  })

  it('says which version it uses, and who saved it when', () => {
    expect(dataLine(step())).toBe('Version 2, saved 1 November 2026 by Joris. Paid by the Anthropic key.')
    expect(dataLine(step({ current: version({ number: 1, createdBy: 'the tool' }) }))).toBe('Version 1, the instructions the tool came with. Paid by the Anthropic key.')
    expect(dataLine(step({ fixedInstructions: true }))).toBe('Claude Sonnet 5.5 since 1 November 2026. Paid by the Anthropic key.')
    expect(versionLine(version())).toBe('Version 2, 1 November 2026, by Joris')
  })

  it('names the default model, and prices a job with the chosen one against the one in use', () => {
    expect(modelLabel({ id: 'claude-haiku-5-5', name: 'Claude Haiku 5.5', jobCostUsd: 0 })).toBe('Claude Haiku 5.5 (default)')
    expect(modelLabel({ id: 'claude-sonnet-5-5', name: 'Claude Sonnet 5.5', jobCostUsd: 0 })).toBe('Claude Sonnet 5.5')
    const s = step()
    expect(costLine(s, 'claude-sonnet-5-5')).toBe('About $0.088 for a scan.')
    expect(costLine(s, 'claude-haiku-5-5')).toBe('About $0.0044 for a scan, $0.0836 less than now.')
    expect(costLine(step({ current: version({ model: 'claude-haiku-5-5' }), jobCostUsd: 0.0044 }), 'claude-sonnet-5-5')).toBe(
      'About $0.088 for a scan, $0.0836 more than now.',
    )
    expect(costLine(s, 'claude-unknown')).toBe('')
  })

  it('knows when the editor differs from the version in use', () => {
    const s = step()
    expect(isDirty(s, { text: 'Match.', model: 'claude-sonnet-5-5' })).toBe(false)
    expect(isDirty(s, { text: 'Match better.', model: 'claude-sonnet-5-5' })).toBe(true)
    expect(isDirty(s, { text: 'Match.', model: 'claude-haiku-5-5' })).toBe(true)
  })
})
