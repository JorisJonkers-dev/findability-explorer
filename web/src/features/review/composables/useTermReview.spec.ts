import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { json, problem } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useStartReview, useTermCandidates } from './useTermReview'

describe('useTermCandidates', () => {
  it('has nothing before the response, then the candidates of the status asked for', async () => {
    const { state, sent } = mountComposable(
      () => useTermCandidates(ref('rejected')),
      () => json({ items: [], counts: { pending: 2, accepted: 0, rejected: 0 }, review: { running: false, waiting: 2 } }),
    )
    expect(state.candidates.value).toEqual([])
    expect(state.counts.value).toBeUndefined()
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(sent).toEqual(['GET /api/v1/admin/term-candidates?status=rejected'])
    expect(state.counts.value).toEqual({ pending: 2, accepted: 0, rejected: 0 })
    expect(state.review.value).toEqual({ running: false, waiting: 2 })
    expect(state.forbidden.value).toBe(false)
    expect(state.error.value).toBeNull()
  })

  it('tells a refusal from a failure', async () => {
    const { state } = mountComposable(() => useTermCandidates(ref('pending')), () => problem(403, 'Forbidden'))
    await flushPromises()
    expect(state.forbidden.value).toBe(true)
    expect(state.error.value).toBe('Only administrators can review terms.')
  })
})

describe('useStartReview', () => {
  it('starts the language model\'s review, and says when it was refused', async () => {
    const ok = mountComposable(useStartReview, () => json({ running: true, waiting: 3 }, 202))
    ok.state.start()
    await flushPromises()
    expect(ok.sent).toEqual(['POST /api/v1/admin/term-candidates/review'])
    expect(ok.state.error.value).toBeNull()
    const refused = mountComposable(useStartReview, () => problem(403, 'Forbidden'))
    refused.state.start()
    await flushPromises()
    expect(refused.state.error.value).toBe('Only administrators can review terms.')
    expect(refused.state.isStarting.value).toBe(false)
  })
})
