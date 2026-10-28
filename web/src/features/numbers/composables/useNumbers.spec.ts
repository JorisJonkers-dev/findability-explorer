import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { planFixture, standingsFixture } from '../../../testing/numbersFixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useNumbers } from './useNumbers'

describe('useNumbers', () => {
  it('loads the plan and the standings, and joins them', async () => {
    const { state, sent } = mountComposable(
      useNumbers,
      routes({ 'GET /api/v1/plan': () => json(planFixture()), 'GET /api/v1/standings': () => json(standingsFixture()) }),
    )
    expect(state.facts.value).toBeUndefined()
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(state.facts.value?.plan.watched).toHaveLength(5)
    expect(state.facts.value?.standings.trailing).toBe(3)
    expect(state.isLoading.value).toBe(false)
    expect(state.error.value).toBeNull()
    expect([...sent].sort()).toEqual(['GET /api/v1/plan', 'GET /api/v1/standings'])
  })

  it('reports a failure of either request', async () => {
    const { state } = mountComposable(useNumbers, routes({ 'GET /api/v1/plan': () => problem(500, 'Internal Server Error', 'Down.'), 'GET /api/v1/standings': () => json(standingsFixture()) }))
    await flushPromises()
    expect(state.error.value).toContain('Down.')
    expect(state.facts.value).toBeUndefined()
  })
})
