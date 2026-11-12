import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { recommendationRuns } from '../../../testing/recommendationRuns'
import { usePlan } from './usePlan'
import { RUN_POLL_MS, useRecommendationRuns } from './useRecommendationRuns'

afterEach(() => {
  vi.useRealTimers()
})

describe('useRecommendationRuns', () => {
  it('reads the runs, starts one, and asks again while it runs', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    let running = false
    const { state, sent } = mountComposable(
      () => {
        usePlan()
        return useRecommendationRuns()
      },
      routes({
        'GET /api/v1/admin/recommendation-runs': () => json(recommendationRuns({ running })),
        'POST /api/v1/admin/recommendation-runs': () => {
          running = true
          return json(recommendationRuns({ running: true }), 202)
        },
        'GET /api/v1/plan': () => json({}),
      }),
    )
    await flushPromises()
    expect(state.runs.value?.replaces).toBe(12)
    expect(state.error.value).toBeNull()
    state.start()
    expect(state.isStarting.value).toBe(true)
    await flushPromises()
    expect(state.runs.value?.running).toBe(true)
    expect(state.notice.value).toBe('The Recommendation run started. It replaces the open recommendations only when it ends.')
    const before = sent.filter((s) => s === 'GET /api/v1/admin/recommendation-runs').length
    await vi.advanceTimersByTimeAsync(RUN_POLL_MS)
    expect(sent.filter((s) => s === 'GET /api/v1/admin/recommendation-runs').length).toBeGreaterThan(before)
    // The run ends: the plan is asked for again.
    running = false
    await vi.advanceTimersByTimeAsync(RUN_POLL_MS)
    expect(state.runs.value?.running).toBe(false)
    expect(sent.filter((s) => s === 'GET /api/v1/plan').length).toBeGreaterThan(1)
  })

  it('stops the run, and says why a start or stop was refused', async () => {
    const { state, sent } = mountComposable(
      useRecommendationRuns,
      routes({
        'GET /api/v1/admin/recommendation-runs': () => json(recommendationRuns({ running: true })),
        'POST /api/v1/admin/recommendation-runs': () => problem(409, 'Conflict', 'A Recommendation run is running already.'),
        'POST /api/v1/admin/recommendation-runs/stop': () => new Response(null, { status: 204 }),
      }),
    )
    await flushPromises()
    state.stop()
    expect(state.isStopping.value).toBe(true)
    await flushPromises()
    expect(sent).toContain('POST /api/v1/admin/recommendation-runs/stop')
    expect(state.notice.value).toBe('The run stops after the batch it writes. The recommendations stay as they are.')
    state.start()
    await flushPromises()
    expect(state.error.value).toBe('A Recommendation run is running already.')
  })

  it('says why the runs could not be read', async () => {
    const { state } = mountComposable(useRecommendationRuns, routes({ 'GET /api/v1/admin/recommendation-runs': () => problem(403, 'Forbidden') }))
    await flushPromises()
    expect(state.error.value).toBe('Only administrators can start a Recommendation run.')
  })
})
