import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { claudeStep } from '../../../testing/claudeStepFixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useClaudeStep, useClaudeSteps } from './useClaudeSteps'

describe('useClaudeSteps', () => {
  it('lists the steps, and says why it could not', async () => {
    const s = claudeStep()
    const ok = mountComposable(useClaudeSteps, () => json({ items: [{ id: s.id, title: s.title, fixedInstructions: false, current: s.current, jobCostUsd: s.jobCostUsd }] }))
    expect(ok.state.isLoading.value).toBe(true)
    await flushPromises()
    expect(ok.state.steps.value.map((x) => x.id)).toEqual(['matching'])
    expect(ok.state.error.value).toBeNull()
    const failing = mountComposable(useClaudeSteps, () => problem(500, 'Internal Server Error'))
    await flushPromises()
    expect(failing.state.steps.value).toEqual([])
    expect(failing.state.error.value).toBe('Internal Server Error')
  })
})

describe('useClaudeStep', () => {
  it('reads one step and saves a new version, keeping the step it answers with', async () => {
    const saved = claudeStep({ current: { ...claudeStep().current, number: 3, instructions: 'New.' } })
    const bodies: unknown[] = []
    const { state, sent } = mountComposable(
      () => useClaudeStep(ref('matching')),
      routes({
        'GET /api/v1/admin/claude-steps/matching': () => json(claudeStep()),
        'POST /api/v1/admin/claude-steps/matching/versions': async (r) => (bodies.push(await r.json()), json(saved, 201)),
        'GET /api/v1/admin/claude-steps': () => json({ items: [] }),
      }),
    )
    await flushPromises()
    expect(state.step.value?.current.number).toBe(2)
    let got = 0
    state.save({ model: 'claude-haiku-5-5', instructions: 'New.' }, (s) => (got = s.current.number))
    expect(state.isSaving.value).toBe(true)
    await flushPromises()
    expect(got).toBe(3)
    expect(state.step.value?.current.instructions).toBe('New.')
    expect(bodies).toEqual([{ model: 'claude-haiku-5-5', instructions: 'New.' }])
    expect(sent.filter((r) => r.startsWith('GET /api/v1/admin/claude-steps/matching'))).toHaveLength(1)
  })

  it('says why it could not read or save', async () => {
    const { state } = mountComposable(
      () => useClaudeStep(ref('matching')),
      routes({
        'GET /api/v1/admin/claude-steps/matching': () => problem(500, 'Internal Server Error'),
        'POST /api/v1/admin/claude-steps/matching/versions': () => problem(422, 'Unprocessable Entity', 'Nothing changed, so there is no new version.'),
      }),
    )
    await flushPromises()
    expect(state.error.value).toBe('Internal Server Error')
    state.save({ model: 'claude-haiku-5-5', instructions: 'x' }, () => undefined)
    await flushPromises()
    expect(state.saveError.value).toBe('Nothing changed, so there is no new version.')
  })
})
