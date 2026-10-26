import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { competitor, summary, termSearch } from '../../../testing/fixtures'
import { json, problem } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { POLL_INTERVAL_MS } from '../../scan/domain/status'
import { useTermMap } from '../../terminology/composables/useTermSearch'
import { useCompetitors } from './useCompetitors'

afterEach(() => {
  vi.useRealTimers()
})

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

  it('asks for the map again when a scan finishes, so its makers follow the scan', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    let call = 0
    const { sent } = mountComposable(
      () => {
        const companies = useCompetitors()
        useTermMap()
        return companies
      },
      (request) =>
        new URL(request.url).pathname === '/api/v1/terms'
          ? json(termSearch([]))
          : json({ items: [competitor('Twentebelt', { scans: 1, lastScan: summary({ status: call++ === 0 ? 'running' : 'done' }) })] }),
    )
    await flushPromises()
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS)
    await flushPromises()
    expect(sent.filter((s) => s.startsWith('GET /api/v1/terms'))).toHaveLength(2)
  })

  it('asks again while a company is being scanned, and stops once none is', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    const statuses = ['running', 'done'] as const
    let call = 0
    const { state, sent } = mountComposable(useCompetitors, () =>
      json({ items: [competitor('Twentebelt', { scans: 1, lastScan: summary({ status: statuses[Math.min(call++, 1)] }) })] }),
    )
    await flushPromises()
    expect(state.competitors.value[0]?.lastScan?.status).toBe('running')
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS)
    await flushPromises()
    expect(state.competitors.value[0]?.lastScan?.status).toBe('done')
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS * 3)
    expect(sent).toHaveLength(2)
  })
})
