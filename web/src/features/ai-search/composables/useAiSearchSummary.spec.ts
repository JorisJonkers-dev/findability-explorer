import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useAiSearchSummary } from './useAiSearchSummary'

describe('useAiSearchSummary', () => {
  it('loads the summary once', async () => {
    const body = { goal: 0.4, followed: [], trend: [], crawlers: [], keepingOut: [], ranking: [], sites: 3, questions: 20, providers: 5 }
    const { state, sent } = mountComposable(useAiSearchSummary, routes({ 'GET /api/v1/ai-search': () => json(body) }))
    expect(state.summary.value).toBeUndefined()
    await flushPromises()
    expect(state.summary.value?.sites).toBe(3)
    expect(state.error.value).toBeNull()
    expect(sent).toEqual(['GET /api/v1/ai-search'])
  })

  it('reports a failure', async () => {
    const { state } = mountComposable(useAiSearchSummary, routes({ 'GET /api/v1/ai-search': () => problem(500, 'Internal Server Error', 'Down.') }))
    await flushPromises()
    expect(state.error.value).toContain('Down.')
  })
})
