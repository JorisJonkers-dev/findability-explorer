import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useOverview } from './useOverview'

const figures = {
  terms: { total: 387, tribelt: 30, languages: [{ language: 'en', terms: 200, tribelt: 20 }] },
  competitors: { makers: 24, families: [{ family: 'eyelink', makers: 9 }] },
  aiSearch: { answers: 0, companies: [] },
}

describe('useOverview', () => {
  it("loads the poster's figures in one request", async () => {
    const { state, sent } = mountComposable(useOverview, routes({ 'GET /api/v1/overview': () => json(figures) }))
    expect(state.overview.value).toBeUndefined()
    await flushPromises()
    expect(state.overview.value?.terms.total).toBe(387)
    expect(state.overview.value?.competitors.families).toEqual([{ family: 'eyelink', makers: 9 }])
    expect(state.error.value).toBeNull()
    expect(sent).toEqual(['GET /api/v1/overview'])
  })

  it('reports a failure', async () => {
    const { state } = mountComposable(useOverview, routes({ 'GET /api/v1/overview': () => problem(500, 'Internal Server Error', 'Down.') }))
    await flushPromises()
    expect(state.error.value).toContain('Down.')
  })
})
