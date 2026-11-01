import { flushPromises } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { summary } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { POLL_INTERVAL_MS } from '../domain/status'
import { QUEUE_LIMIT, useScanQueue, useStopReadinessCheck } from './useScanQueue'

const A1 = '00000000-0000-4000-8000-0000000000a1'
const B2 = '00000000-0000-4000-8000-0000000000b2'
const RUNNING = summary({ id: A1, status: 'running', createdAt: '2026-10-28T10:00:00Z' })
const WAITING = summary({ id: B2, status: 'queued', createdAt: '2026-10-28T10:01:00Z' })
const gone = () => new Response(null, { status: 204 })

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
})
afterEach(() => {
  vi.useRealTimers()
})

describe('useScanQueue', () => {
  it('splits the scans into running and waiting, and asks again while any is in progress', async () => {
    let items = [RUNNING, WAITING]
    const { state, sent } = mountComposable(useScanQueue, routes({ 'GET /api/v1/analyses': () => json({ items }) }))
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(state.running.value.map((s) => s.id)).toEqual([A1])
    expect(state.waiting.value.map((s) => s.id)).toEqual([B2])
    items = [summary()]
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS)
    await flushPromises()
    expect(state.running.value).toEqual([])
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS * 3)
    expect(sent).toEqual([`GET /api/v1/analyses?limit=${String(QUEUE_LIMIT)}`, `GET /api/v1/analyses?limit=${String(QUEUE_LIMIT)}`])
    expect(state.error.value).toBeNull()
  })

  it('stops a running scan, takes a waiting one off, clears the queue, and stops the rescan', async () => {
    const { state, sent } = mountComposable(
      useScanQueue,
      routes({
        'GET /api/v1/analyses': () => json({ items: [RUNNING, WAITING] }),
        [`POST /api/v1/admin/analyses/${A1}/stop`]: gone,
        [`POST /api/v1/admin/analyses/${B2}/stop`]: gone,
        'DELETE /api/v1/admin/analyses/queue': gone,
        'POST /api/v1/admin/rescans/stop': gone,
      }),
    )
    await flushPromises()
    state.stop(RUNNING, 'Twentebelt')
    await flushPromises()
    expect(state.notice.value).toBe('The scan of Twentebelt is stopped. It keeps the pages it read.')
    state.stop(WAITING, 'Ashworth')
    await flushPromises()
    expect(state.notice.value).toBe('The scan of Ashworth is removed from the queue.')
    state.removeWaiting()
    await flushPromises()
    expect(state.notice.value).toBe('Every waiting scan is removed from the queue.')
    state.stopRescan()
    await flushPromises()
    expect(state.notice.value).toBe('The rescan is stopped. The scans it finished are kept.')
    expect(sent.filter((r) => !r.startsWith('GET'))).toEqual([
      `POST /api/v1/admin/analyses/${A1}/stop`,
      `POST /api/v1/admin/analyses/${B2}/stop`,
      'DELETE /api/v1/admin/analyses/queue',
      'POST /api/v1/admin/rescans/stop',
    ])
    // Each stop reads the scans and the companies again.
    expect(sent.filter((r) => r.startsWith('GET /api/v1/analyses')).length).toBeGreaterThan(1)
    expect(state.isStopping.value).toBe(false)
  })

  it('says why a stop was refused, and why the list could not be read', async () => {
    const { state } = mountComposable(
      useScanQueue,
      routes({
        'GET /api/v1/analyses': () => problem(500, 'Internal Server Error'),
        'POST /api/v1/admin/rescans/stop': () => problem(404, 'Not Found', 'No scan of a rescan waits or runs.'),
      }),
    )
    await flushPromises()
    expect(state.error.value).toBe('Internal Server Error')
    state.stopRescan()
    await flushPromises()
    expect(state.stopError.value).toBe('No scan of a rescan waits or runs.')
    expect(state.notice.value).toBeNull()
  })
})

describe('useStopReadinessCheck', () => {
  it('stops the check and reads the table again', async () => {
    const { state, sent } = mountComposable(useStopReadinessCheck, routes({ 'POST /api/v1/admin/readiness-checks/stop': gone }))
    state.stop()
    await flushPromises()
    expect(sent).toEqual(['POST /api/v1/admin/readiness-checks/stop'])
    expect(state.error.value).toBeNull()
  })

  it('says why it could not stop', async () => {
    const { state } = mountComposable(useStopReadinessCheck, () => problem(404, 'Not Found', 'No check runs.'))
    state.stop()
    await flushPromises()
    expect(state.error.value).toBe('No check runs.')
  })
})
