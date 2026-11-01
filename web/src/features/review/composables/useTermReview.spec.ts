import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useStartReview, useTermCandidates, useTermReviewControl } from './useTermReview'

describe('useTermCandidates', () => {
  it('has nothing before the response, then the candidates of the status asked for', async () => {
    const { state, sent } = mountComposable(
      () => useTermCandidates(ref('rejected')),
      () => json({ items: [], counts: { pending: 2, accepted: 0, rejected: 0 }, review: { on: true, running: false, waiting: 2, monthTokens: 0, monthCostUsd: 0 } }),
    )
    expect(state.candidates.value).toEqual([])
    expect(state.counts.value).toBeUndefined()
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(sent).toEqual(['GET /api/v1/admin/term-candidates?status=rejected'])
    expect(state.counts.value).toEqual({ pending: 2, accepted: 0, rejected: 0 })
    expect(state.review.value).toEqual({ on: true, running: false, waiting: 2, monthTokens: 0, monthCostUsd: 0 })
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
    const ok = mountComposable(useStartReview, () => json({ on: true, running: true, waiting: 3, monthTokens: 0, monthCostUsd: 0 }, 202))
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

describe('useTermReviewControl', () => {
  const STATE = { on: false, running: false, waiting: 0, monthTokens: 0, monthCostUsd: 0 }

  it('switches the review off and stops one that runs', async () => {
    const bodies: unknown[] = []
    const { state, sent } = mountComposable(
      useTermReviewControl,
      routes({
        'PUT /api/v1/admin/term-candidates/review/switch': async (r) => (bodies.push(await r.json()), json(STATE)),
        'POST /api/v1/admin/term-candidates/review/stop': () => new Response(null, { status: 204 }),
      }),
    )
    state.setOn(false)
    expect(state.isBusy.value).toBe(true)
    await flushPromises()
    state.stop()
    await flushPromises()
    expect(bodies).toEqual([{ on: false }])
    expect(sent.filter((r) => !r.startsWith('GET'))).toEqual(['PUT /api/v1/admin/term-candidates/review/switch', 'POST /api/v1/admin/term-candidates/review/stop'])
    expect(state.error.value).toBeNull()
  })

  it('says why it was refused', async () => {
    const { state } = mountComposable(useTermReviewControl, () => problem(403, 'Forbidden'))
    state.stop()
    await flushPromises()
    expect(state.error.value).toBe('Only administrators can review terms.')
  })
})
