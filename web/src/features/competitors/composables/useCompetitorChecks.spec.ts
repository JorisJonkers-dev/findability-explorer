import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { competitor } from '../../../testing/fixtures'
import { json, problem } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { REVIEW_POLL_MS } from '../../review/domain/candidates'
import { useCompetitorChecks, useListDecision } from './useCompetitorChecks'
import { useCompetitors } from './useCompetitors'

afterEach(() => {
  vi.useRealTimers()
})

const check = { competitorId: 'fenner', competitor: false, sure: true, reason: 'Makes rubber belting.', model: 'haiku', at: '2026-10-27T09:00:00Z' }

describe('useCompetitorChecks', () => {
  it('asks nothing for a viewer', async () => {
    const { state, sent } = mountComposable(() => useCompetitorChecks(ref(false)), () => json({}))
    await flushPromises()
    expect(sent).toEqual([])
    expect(state.checks.value.size).toBe(0)
    expect(state.unchecked.value).toBe(0)
    expect(state.review.value).toBeUndefined()
  })

  it('asks again while Haiku checks, and then for the companies it archived', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    let running = true
    const { state, sent } = mountComposable(
      () => {
        useCompetitors()
        return useCompetitorChecks(ref(true))
      },
      (r) =>
        new URL(r.url).pathname === '/api/v1/competitors'
          ? json({ items: [] })
          : json({ items: [check], unchecked: running ? 3 : 0, review: { running } }),
    )
    await flushPromises()
    sent.length = 0
    expect(state.checks.value.get('fenner')?.reason).toBe('Makes rubber belting.')
    expect(state.unchecked.value).toBe(3)
    running = false
    await vi.advanceTimersByTimeAsync(REVIEW_POLL_MS)
    await flushPromises()
    expect(state.unchecked.value).toBe(0)
    expect(sent).toContain('GET /api/v1/competitors')
    expect(state.error.value).toBeNull()
  })

  it('starts a check and asks for the checks again', async () => {
    const { state, sent } = mountComposable(
      () => useCompetitorChecks(ref(true)),
      (r) => (r.method === 'POST' ? json({ running: true }, 202) : json({ items: [], unchecked: 2, review: { running: false } })),
    )
    await flushPromises()
    state.start()
    await flushPromises()
    expect(sent.filter((s) => s === 'GET /api/v1/admin/competitor-checks')).toHaveLength(2)
    expect(sent).toContain('POST /api/v1/admin/competitor-checks')
  })

  it('says why it was refused', async () => {
    const { state } = mountComposable(() => useCompetitorChecks(ref(true)), () => problem(403, 'Forbidden'))
    await flushPromises()
    expect(state.error.value).toBe('Only administrators can change the companies.')
  })
})

describe('useListDecision', () => {
  it('keeps, archives and restores a company, and says so', async () => {
    const bodies: unknown[] = []
    const { state, sent } = mountComposable(useListDecision, async (r) => {
      bodies.push(await r.json())
      return json(competitor('Intralox'))
    })
    const intralox = { id: 'intralox', name: 'Intralox' }
    state.keep(intralox)
    await flushPromises()
    expect(state.notice.value).toBe('Intralox stays on the list, and Haiku will not archive it.')
    state.archive(intralox)
    await flushPromises()
    expect(state.notice.value).toBe('Archived Intralox. Its scans stay.')
    state.restore(intralox)
    await flushPromises()
    expect(state.notice.value).toBe('Intralox is back on the list, and Haiku will not archive it again.')
    expect(bodies).toEqual([{ kept: true }, { archived: true }, { archived: false }])
    expect(sent.filter((s) => s.startsWith('PATCH'))).toHaveLength(3)
  })
})
