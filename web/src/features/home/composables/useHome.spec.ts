import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { overviewFixture, planFixture, standingsFixture } from '../../../testing/numbersFixtures'
import { useHome } from './useHome'

const ok = {
  'GET /api/v1/plan': () => json(planFixture()),
  'GET /api/v1/standings': () => json(standingsFixture()),
  'GET /api/v1/overview': () => json(overviewFixture()),
}

describe('useHome', () => {
  it('loads the numbers and the figures of the sections', async () => {
    const { state } = mountComposable(useHome, routes(ok))
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(state.isLoading.value).toBe(false)
    expect(state.facts.value?.plan.watched).toHaveLength(5)
    expect(state.overview.value?.competitors.makers).toBe(24)
    expect(state.error.value).toBeNull()
  })

  it('reports a failure of the numbers', async () => {
    const { state } = mountComposable(useHome, routes({ ...ok, 'GET /api/v1/plan': () => problem(500, 'Internal Server Error', 'Plan down.') }))
    await flushPromises()
    expect(state.error.value).toContain('Plan down.')
  })

  it('reports a failure of the figures', async () => {
    const { state } = mountComposable(useHome, routes({ ...ok, 'GET /api/v1/overview': () => problem(500, 'Internal Server Error', 'Figures down.') }))
    await flushPromises()
    expect(state.error.value).toContain('Figures down.')
  })
})
