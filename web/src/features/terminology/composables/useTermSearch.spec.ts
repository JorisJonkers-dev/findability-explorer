import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { term, termSearch } from '../../../testing/fixtures'
import { json, problem } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useTermMap } from './useTermSearch'

describe('useTermMap', () => {
  it('has nothing before the first response', () => {
    const { state } = mountComposable(useTermMap, () => new Promise<Response>(() => undefined))
    expect(state.terms.value).toEqual([])
    expect(state.total.value).toBe(0)
    expect(state.isLoading.value).toBe(true)
    expect(state.error.value).toBeNull()
  })

  it('asks for the whole map at once', async () => {
    const { state, sent } = mountComposable(useTermMap, () => json(termSearch([term()], { total: 146 })))
    await flushPromises()
    expect(sent).toEqual(['GET /api/v1/terms?limit=200'])
    expect(state.terms.value).toEqual([term()])
    expect(state.total.value).toBe(146)
  })

  it('describes a failed request', async () => {
    const { state } = mountComposable(useTermMap, () => problem(500, 'Internal Server Error', 'The map is not loaded.'))
    await flushPromises()
    expect(state.error.value).toBe('The map is not loaded.')
  })
})
