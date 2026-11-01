import { flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { RUN_ID, summary } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { POLL_INTERVAL_MS } from '../../scan/domain/status'
import { usePromptHistory, usePromptSets, useStopWork, useWorkScans, WORK_SCANS } from './useManage'

const SCAN = '00000000-0000-4000-9000-000000000001'
const ok = () => new Response(null, { status: 204 })

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

describe('useStopWork', () => {
  it('stops a run, a scan, and the check, and empties the queue, saying so each time', async () => {
    const { state, sent } = mountComposable(
      useStopWork,
      routes({
        [`POST /api/v1/admin/prompt-runs/${RUN_ID}/stop`]: ok,
        [`POST /api/v1/admin/analyses/${SCAN}/stop`]: ok,
        'POST /api/v1/admin/readiness-checks/stop': ok,
        'DELETE /api/v1/admin/analyses/queue': ok,
      }),
    )
    state.stop({ kind: 'promptRun', id: RUN_ID, label: 'Prompt run' })
    expect(state.isStopping.value).toBe(true)
    await flushPromises()
    expect(state.notice.value).toBe('The Prompt run stops within a minute. Its responses so far are kept.')
    state.stop({ kind: 'scan', id: SCAN, label: 'Scan of tribelt.nl' })
    await flushPromises()
    expect(state.notice.value).toBe('The scan of tribelt.nl stops after the page it reads.')
    state.stop({ kind: 'check', id: 'check', label: 'AI readiness check' })
    await flushPromises()
    expect(state.notice.value).toBe('The AI readiness check is stopped.')
    state.removeQueue()
    await flushPromises()
    expect(state.notice.value).toBe('The waiting scans are taken off the queue.')
    expect(state.error.value).toBeNull()
    expect(sent).toEqual([
      `POST /api/v1/admin/prompt-runs/${RUN_ID}/stop`,
      `POST /api/v1/admin/analyses/${SCAN}/stop`,
      'POST /api/v1/admin/readiness-checks/stop',
      'DELETE /api/v1/admin/analyses/queue',
    ])
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
