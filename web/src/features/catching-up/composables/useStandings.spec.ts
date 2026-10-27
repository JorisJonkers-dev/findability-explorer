import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useStandings } from './useStandings'

describe('useStandings', () => {
  it('loads the measures and the count Tribelt trails on', async () => {
    const { state, sent } = mountComposable(
      useStandings,
      routes({ 'GET /api/v1/standings': () => json({ measures: [{ measure: 'termsUsed', competitors: 0, trails: false, leads: false }], trailing: 3, open: [] }) }),
    )
    expect(state.measures.value).toEqual([])
    expect(state.trailing.value).toBe(0)
    await flushPromises()
    expect(state.measures.value.map((m) => m.measure)).toEqual(['termsUsed'])
    expect(state.trailing.value).toBe(3)
    expect(state.open.value).toEqual([])
    expect(state.error.value).toBeNull()
    expect(sent).toEqual(['GET /api/v1/standings'])
  })

  it('reports a failure', async () => {
    const { state } = mountComposable(useStandings, routes({ 'GET /api/v1/standings': () => problem(500, 'Internal Server Error', 'Down.') }))
    await flushPromises()
    expect(state.error.value).toContain('Down.')
  })
})
