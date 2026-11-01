import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Plan } from '../../../infrastructure/api/types.gen'
import { promptAnswer, promptResults, promptRun, promptSets, RUN_ID } from '../../../testing/fixtures'
import { json, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useAiResponsesPage } from './useAiResponsesPage'

const plan: Plan = { catalogueVersion: 1, improvements: [], watched: [{ target: 'aiShareOfVoice', goal: 0.4 }], eyeOn: [], readinessChecks: 5 }

describe('useAiResponsesPage', () => {
  it('reads the newest run, its responses and questions, the goals, and the companies, and writes the page from them', async () => {
    const { state, sent } = mountComposable(
      useAiResponsesPage,
      routes({
        'GET /api/v1/prompt-test': () => json(promptResults()),
        [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => json({ run: promptRun(), items: [promptAnswer({}, ['Tribelt:tribelt'])] }),
        'GET /api/v1/prompt-test/prompt-sets': () => json(promptSets()),
        'GET /api/v1/plan': () => json(plan),
        'GET /api/v1/competitors': () => json({ items: [] }),
      }),
    )
    expect(state.dataLine.value).toBeUndefined()
    expect(state.models.value).toEqual([])
    expect(state.read.value).toBe(0)
    expect(state.named.value.map((n) => n.name)).toEqual(['Tribelt'])
    expect(state.questions.value).toEqual([])
    await flushPromises()
    expect(state.dataLine.value).toContain('From the Prompt run of 1 November 2026')
    expect(state.figures.value[0]).toEqual({ value: '25%', label: 'AI responses that name Tribelt', goal: 'Goal 40%' })
    expect(state.read.value).toBe(12)
    expect(state.named.value.map((n) => n.name)).toEqual(['Twentebelt', 'Tribelt'])
    expect(state.models.value).toHaveLength(3)
    expect(state.questions.value[0]?.questions[0]?.english).toBe('We are building a tunnel oven for an industrial bakery. Which makers in the Twente region can supply the belt?')
    expect(state.families.value).toHaveLength(1)
    expect([...sent].sort()).toEqual([
      'GET /api/v1/competitors',
      'GET /api/v1/plan',
      `GET /api/v1/prompt-runs/${RUN_ID}/answers`,
      'GET /api/v1/prompt-test',
      'GET /api/v1/prompt-test/prompt-sets',
    ])
  })

  it('has nothing to show before a run', async () => {
    const { state } = mountComposable(useAiResponsesPage, routes({ 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [], mentions: [] })) }))
    await flushPromises()
    expect(state.run.value).toBeUndefined()
    expect(state.named.value.map((n) => n.name)).toEqual(['Tribelt'])
    expect(state.figures.value.map((f) => f.value)).toEqual(['–', '–', '–'])
  })
})
