import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Plan } from '../../../infrastructure/api/types.gen'
import { details } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { usePlan } from './usePlan'

const plan = (done: boolean): Plan => ({
  catalogueVersion: 1,
  improvements: [{ id: 'listings', title: 'Get listed', moves: 'citations', measure: 'citationShare', auto: false, done, ...details }],
  watched: [],
  eyeOn: [],
  readinessChecks: 5,
})

describe('usePlan', () => {
  it('loads the plan, and takes the plan a tick or new targets answer with', async () => {
    const sent: unknown[] = []
    const { state } = mountComposable(
      usePlan,
      routes({
        'GET /api/v1/plan': () => json(plan(false)),
        'PUT /api/v1/admin/plan/improvements/listings': async (r) => {
          sent.push(await r.json())
          return json(plan(true))
        },
        'PUT /api/v1/admin/plan/targets': async (r) => {
          sent.push(await r.json())
          return json(plan(false))
        },
      }),
    )
    await flushPromises()
    expect(state.plan.value?.improvements[0]?.done).toBe(false)
    state.tick('listings', true)
    expect(state.isSaving.value).toBe(true)
    await flushPromises()
    expect(state.plan.value?.improvements[0]?.done).toBe(true)
    state.setTargets([{ target: 'termsUsed', value: 40 }])
    await flushPromises()
    expect(state.plan.value?.improvements[0]?.done).toBe(false)
    expect(sent).toEqual([{ done: true }, { targets: [{ target: 'termsUsed', value: 40 }] }])
    expect(state.error.value).toBeNull()
    expect(state.saveError.value).toBeNull()
  })

  it('reports failures to load and to save', async () => {
    const { state } = mountComposable(
      usePlan,
      routes({
        'GET /api/v1/plan': () => problem(500, 'Internal Server Error', 'Down.'),
        'PUT /api/v1/admin/plan/improvements/listings': () => problem(422, 'Unprocessable Entity', 'A scan checks this.'),
        'PUT /api/v1/admin/plan/targets': () => problem(422, 'Unprocessable Entity', 'Out of range.'),
      }),
    )
    await flushPromises()
    expect(state.error.value).toContain('Down.')
    state.tick('listings', true)
    await flushPromises()
    expect(state.saveError.value).toBe('A scan checks this.')
    state.setTargets([{ target: 'citations', value: 5 }])
    await flushPromises()
    expect(state.saveError.value).toBe('Out of range.')
  })
})
