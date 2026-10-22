import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { competitor } from '../../../testing/fixtures'
import { json, problem } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useCompetitors } from './useCompetitors'

describe('useCompetitors', () => {
  it('has nothing before the response, then the companies', async () => {
    const { state } = mountComposable(useCompetitors, () => json({ items: [competitor('Twentebelt')] }))
    expect(state.competitors.value).toEqual([])
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(state.competitors.value.map((c) => c.name)).toEqual(['Twentebelt'])
    expect(state.error.value).toBeNull()
  })

  it('describes a failed request', async () => {
    const { state } = mountComposable(useCompetitors, () => problem(401, 'Unauthorized'))
    await flushPromises()
    expect(state.error.value).toBe('Unauthorized')
  })
})
