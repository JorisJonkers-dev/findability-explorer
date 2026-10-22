import { flushPromises } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { summary } from '../../../testing/fixtures'
import { json, problem } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { POLL_INTERVAL_MS } from '../domain/status'
import { RECENT_LIMIT, useRecentScans } from './useRecentScans'

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useRecentScans', () => {
  it('lists the newest scans, asking again while one is in progress', async () => {
    const statuses = ['running', 'done'] as const
    let call = 0
    const { state, sent } = mountComposable(useRecentScans, () =>
      json({ items: [summary({ status: statuses[Math.min(call++, 1)] })] }),
    )
    expect(state.scans.value).toEqual([])
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(state.scans.value[0]?.status).toBe('running')
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS)
    await flushPromises()
    expect(state.scans.value[0]?.status).toBe('done')
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS * 3)
    expect(sent).toEqual([`GET /api/v1/analyses?limit=${String(RECENT_LIMIT)}`, 'GET /api/v1/analyses?limit=8'])
    expect(state.error.value).toBeNull()
  })

  it('describes a failed request', async () => {
    const { state } = mountComposable(useRecentScans, () => problem(500, 'Internal Server Error'))
    await flushPromises()
    expect(state.error.value).toBe('Internal Server Error')
  })
})
