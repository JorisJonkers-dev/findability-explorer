import { flushPromises } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import type { AnalysisStatus } from '../../../infrastructure/api/types.gen'
import { ANALYSIS_ID, analysis, summary } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { POLL_INTERVAL_MS } from '../domain/status'
import { useRecentScans } from './useRecentScans'
import { useScan } from './useScan'

/** Lets the given time pass, with every response and render that falls in it. */
async function wait(ms: number): Promise<void> {
  await vi.advanceTimersByTimeAsync(ms)
  await flushPromises()
}

/** The server answers each request for the scan with the next status in line. */
function scanServer(statuses: AnalysisStatus[]) {
  return routes({
    [`GET /api/v1/analyses/${ANALYSIS_ID}`]: () => {
      const status = statuses.length > 1 ? statuses.shift() : statuses[0]
      return json(analysis({}, { status: status ?? 'done' }))
    },
    'GET /api/v1/analyses': () => json({ items: [summary()] }),
  })
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useScan', () => {
  it('asks again every two seconds while the scan runs, then lists the recent scans again', async () => {
    const { state, sent } = mountComposable(() => ({ scan: useScan(ref(ANALYSIS_ID)), recent: useRecentScans() }), scanServer(['queued', 'running', 'done']))
    await wait(0)
    expect(state.scan.scan.value?.summary.status).toBe('queued')
    await wait(POLL_INTERVAL_MS)
    expect(state.scan.scan.value?.summary.status).toBe('running')
    await wait(POLL_INTERVAL_MS)
    expect(state.scan.scan.value?.summary.status).toBe('done')
    await wait(POLL_INTERVAL_MS * 5)
    const polls = sent.filter((r) => r.startsWith(`GET /api/v1/analyses/${ANALYSIS_ID}`))
    const lists = sent.filter((r) => r.startsWith('GET /api/v1/analyses?'))
    expect(polls).toHaveLength(3)
    // Once when the sidebar mounts, once when the scan finished.
    expect(lists).toEqual(['GET /api/v1/analyses?limit=8', 'GET /api/v1/analyses?limit=8'])
  })

  it('never polls a scan that has failed', async () => {
    const { sent } = mountComposable(() => useScan(ref(ANALYSIS_ID)), scanServer(['failed']))
    await wait(POLL_INTERVAL_MS * 3)
    expect(sent).toHaveLength(1)
  })

  it('asks for nothing without an id', async () => {
    const { state, sent } = mountComposable(() => useScan(ref('')), scanServer(['done']))
    await wait(POLL_INTERVAL_MS)
    expect(sent).toEqual([])
    expect(state.scan.value).toBeUndefined()
  })

  it('describes a failed request', async () => {
    const { state } = mountComposable(
      () => useScan(ref(ANALYSIS_ID)),
      () => problem(404, 'Not Found', 'No such analysis.'),
    )
    expect(state.isLoading.value).toBe(true)
    await wait(0)
    expect(state.error.value).toBe('No such analysis.')
    expect(state.scan.value).toBeUndefined()
  })
})
