import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { promptResults, promptRun, RUN_ID } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { RUN_POLL_MS, useCandidateDecision, useCompetitorCandidates, usePromptAnswers, usePromptResults } from './usePromptTest'

describe('usePromptAnswers', () => {
  it('asks nothing without a run, then loads its answers', async () => {
    const runId = ref<string | undefined>(undefined)
    const { state, sent } = mountComposable(
      () => usePromptAnswers(runId, ref({ provider: 'openai' })),
      routes({ [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => json({ run: promptRun(), items: [] }) }),
    )
    await flushPromises()
    expect(sent).toEqual([])
    runId.value = RUN_ID
    await nextTick()
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(state.isLoading.value).toBe(false)
    expect(state.answers.value?.items).toEqual([])
    expect(state.error.value).toBeNull()
  })
})

describe('useCompetitorCandidates', () => {
  it('has no candidates before they load', async () => {
    const { state } = mountComposable(
      () => useCompetitorCandidates(ref('pending')),
      routes({ 'GET /api/v1/admin/competitor-candidates': () => json({ items: [], counts: { pending: 0, added: 0, dismissed: 0 } }) }),
    )
    expect(state.candidates.value).toEqual([])
    expect(state.counts.value).toBeUndefined()
    await flushPromises()
    expect(state.counts.value?.pending).toBe(0)
    expect(state.forbidden.value).toBe(false)
    expect(state.error.value).toBeNull()
  })

  it('says when the account may not review them', async () => {
    const { state } = mountComposable(
      () => useCompetitorCandidates(ref('pending')),
      routes({ 'GET /api/v1/admin/competitor-candidates': () => problem(403, 'Forbidden') }),
    )
    await flushPromises()
    expect(state.forbidden.value).toBe(true)
    expect(state.error.value).toBe('Only administrators can do this.')
  })
})

describe('useCandidateDecision', () => {
  const candidate = { id: '3a7bd3e2360a9f01', name: 'Intralox', answers: 4, providers: ['openai'], models: [], addresses: [], lastSeenAt: '2026-11-01T06:00:00Z', status: 'added' }

  it('adds, dismisses and undoes, and says so', async () => {
    const { state, sent } = mountComposable(
      useCandidateDecision,
      routes({
        'POST /api/v1/admin/competitor-candidates/3a7bd3e2360a9f01/add': () => json(candidate),
        'POST /api/v1/admin/competitor-candidates/3a7bd3e2360a9f01/dismiss': () => json({ ...candidate, status: 'dismissed' }),
        'DELETE /api/v1/admin/competitor-candidates/3a7bd3e2360a9f01/decision': () => new Response(null, { status: 204 }),
      }),
    )
    state.add('3a7bd3e2360a9f01', 'Intralox', 'https://www.intralox.com/')
    expect(state.isDeciding.value).toBe(true)
    await flushPromises()
    expect(state.notice.value).toBe('Added Intralox to the competitors.')
    state.dismiss('3a7bd3e2360a9f01', 'Intralox')
    expect(state.isDeciding.value).toBe(true)
    await flushPromises()
    expect(state.notice.value).toBe('Dismissed Intralox.')
    state.undo('3a7bd3e2360a9f01', 'Intralox')
    expect(state.isDeciding.value).toBe(true)
    await flushPromises()
    expect(state.notice.value).toBe('Intralox is back among the candidates to review.')
    expect(state.error.value).toBeNull()
    expect(sent.filter((r) => !r.startsWith('GET'))).toHaveLength(3)
  })

  it('reports the failure of the last action', async () => {
    const { state } = mountComposable(
      useCandidateDecision,
      routes({
        'POST /api/v1/admin/competitor-candidates/0123456789abcdef/add': () => problem(409, 'Conflict', 'Taken.'),
        'POST /api/v1/admin/competitor-candidates/0123456789abcdef/dismiss': () => problem(404, 'Not Found', 'Gone.'),
        'DELETE /api/v1/admin/competitor-candidates/0123456789abcdef/decision': () => problem(404, 'Not Found', 'Never decided.'),
      }),
    )
    state.add('0123456789abcdef', 'X', 'https://x.example/')
    await flushPromises()
    expect(state.error.value).toContain('Taken.')
    state.dismiss('0123456789abcdef', 'X')
    await flushPromises()
    expect(state.error.value).toContain('Gone.')
    state.undo('0123456789abcdef', 'X')
    await flushPromises()
    expect(state.error.value).toContain('Never decided.')
  })
})

describe('usePromptResults', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('asks again while a run is running, and stops once none is', async () => {
    vi.useFakeTimers()
    let running = true
    const { state, sent } = mountComposable(
      usePromptResults,
      routes({ 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [promptRun({ status: running ? 'running' : 'done' })] })) }),
    )
    await vi.advanceTimersByTimeAsync(0)
    expect(state.results.value?.runs[0]?.status).toBe('running')
    running = false
    await vi.advanceTimersByTimeAsync(RUN_POLL_MS)
    expect(state.results.value?.runs[0]?.status).toBe('done')
    await vi.advanceTimersByTimeAsync(RUN_POLL_MS * 2)
    expect(sent).toEqual(['GET /api/v1/prompt-test', 'GET /api/v1/prompt-test'])
  })
})
