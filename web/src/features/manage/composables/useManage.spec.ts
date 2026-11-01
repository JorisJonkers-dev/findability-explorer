import { flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { RUN_ID, summary } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { POLL_INTERVAL_MS } from '../../scan/domain/status'
import { ACTIVITY_POLL_MS, useActivity, usePromptHistory, usePromptSets, useStopWork, useWorkScans, WORK_SCANS } from './useManage'

const SCAN = '00000000-0000-4000-9000-000000000001'

describe('useWorkScans', () => {
  it('reads the latest scans', async () => {
    const { state, sent } = mountComposable(useWorkScans, routes({ 'GET /api/v1/analyses': () => json({ items: [summary()] }) }))
    expect(state.scans.value).toEqual([])
    await flushPromises()
    expect(sent).toEqual([`GET /api/v1/analyses?limit=${String(WORK_SCANS)}`])
    expect(state.scans.value).toHaveLength(1)
    expect(state.error.value).toBeNull()
  })

  it('asks again while a scan waits or runs', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    let call = 0
    const { state, sent } = mountComposable(useWorkScans, routes({ 'GET /api/v1/analyses': () => json({ items: [summary({ status: call++ === 0 ? 'queued' : 'done' })] }) }))
    await flushPromises()
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS)
    await flushPromises()
    expect(state.scans.value[0]?.status).toBe('done')
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS * 3)
    expect(sent).toHaveLength(2)
    vi.useRealTimers()
  })

  it('says when they cannot be read', async () => {
    const { state } = mountComposable(useWorkScans, routes({ 'GET /api/v1/analyses': () => problem(500, 'Broken') }))
    await flushPromises()
    expect(state.error.value).not.toBeNull()
  })
})

describe('usePromptSets and usePromptHistory', () => {
  it('reads the published sets and the history', async () => {
    const { state: sets } = mountComposable(usePromptSets, routes({ 'GET /api/v1/prompt-test/prompt-sets': () => json({ versions: [], scopes: [] }) }))
    const { state: history } = mountComposable(usePromptHistory, routes({ 'GET /api/v1/prompt-test/history': () => json({ versions: [], runs: [] }) }))
    await flushPromises()
    expect(sets.sets.value).toEqual({ versions: [], scopes: [] })
    expect(sets.isLoading.value).toBe(false)
    expect(sets.error.value).toBeNull()
    expect(history.history.value).toEqual({ versions: [], runs: [] })
    expect(history.error.value).toBeNull()
  })

  it('says when they cannot be read', async () => {
    const { state: sets } = mountComposable(usePromptSets, routes({}))
    const { state: history } = mountComposable(usePromptHistory, routes({}))
    await flushPromises()
    expect(sets.error.value).not.toBeNull()
    expect(history.error.value).not.toBeNull()
  })
})

describe('useActivity', () => {
  it('reads the work and asks again while some runs', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    const running = { id: 'w', kind: 'scan', startedAt: '2026-11-01T08:00:00Z', done: 1, total: 2, failed: 0, openRouterCostUsd: 0, anthropicCostUsd: 0 }
    let call = 0
    const { state, sent } = mountComposable(
      useActivity,
      routes({ 'GET /api/v1/admin/activity': () => json({ running: call++ === 0 ? [running] : [], recent: [], spending: { since: '2026-11-01T00:00:00Z', lines: [] } }) }),
    )
    await flushPromises()
    expect(state.activity.value?.running).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(ACTIVITY_POLL_MS)
    await flushPromises()
    expect(state.activity.value?.running).toEqual([])
    await vi.advanceTimersByTimeAsync(ACTIVITY_POLL_MS * 3)
    expect(sent).toHaveLength(2)
    expect(state.error.value).toBeNull()
    vi.useRealTimers()
  })

  it('says when it cannot be read', async () => {
    const { state } = mountComposable(useActivity, routes({}))
    await flushPromises()
    expect(state.error.value).toBe('Not Found')
  })
})

describe('useStopWork', () => {
  it('stops each kind of work by its own operation, and empties the queue, saying so each time', async () => {
    const ok204 = () => new Response(null, { status: 204 })
    const { state, sent } = mountComposable(
      useStopWork,
      routes({
        [`POST /api/v1/admin/prompt-runs/${RUN_ID}/stop`]: ok204,
        [`POST /api/v1/admin/analyses/${SCAN}/stop`]: ok204,
        'POST /api/v1/admin/rescans/stop': ok204,
        'POST /api/v1/admin/readiness-checks/stop': ok204,
        'POST /api/v1/admin/term-candidates/review/stop': ok204,
        'POST /api/v1/admin/competitor-candidates/review/stop': ok204,
        'POST /api/v1/admin/competitor-checks/stop': ok204,
        'DELETE /api/v1/admin/analyses/queue': ok204,
      }),
    )
    const notices: (string | null)[] = []
    const stops = [
      { kind: 'promptRun', id: RUN_ID },
      { kind: 'scan', id: SCAN },
      { kind: 'rescan', id: SCAN },
      { kind: 'readinessCheck', id: 'c' },
      { kind: 'termReview', id: 't' },
      { kind: 'candidateReview', id: 'c' },
      { kind: 'competitorCheck', id: 'c' },
    ] as const
    for (const s of stops) {
      state.stop({ ...s, label: 'Work' })
      expect(state.isStopping.value).toBe(true)
      await flushPromises()
      notices.push(state.notice.value)
    }
    state.removeQueue()
    await flushPromises()
    notices.push(state.notice.value)
    state.stop({ kind: 'recommendationRun', id: 'r', label: 'Recommendation run' })
    await flushPromises()
    expect(notices).toEqual([
      'The Prompt run stops within a minute. Its responses so far are kept.',
      'The scan stops after the page it reads.',
      'The rescan stops. The waiting scans are taken off the queue.',
      'The AI readiness check is stopped.',
      'The term review is stopped. What it decided is kept.',
      'The review of the competitor candidates is stopped. What it decided is kept.',
      'The check of the competitor list is stopped. What it decided is kept.',
      'The waiting scans are taken off the queue.',
    ])
    expect(state.error.value).toBeNull()
    expect(sent).toHaveLength(8)
  })

  it('says why a stop failed', async () => {
    const { state } = mountComposable(useStopWork, routes({}))
    expect(state.error.value).toBeNull()
    state.stop({ kind: 'promptRun', id: RUN_ID, label: 'Prompt run' })
    await flushPromises()
    expect(state.notice.value).toBeNull()
    expect(state.error.value).toBe('Not Found')
  })
})
