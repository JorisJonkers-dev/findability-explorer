import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { promptResults, promptRun, RUN_ID } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { RUN_POLL_MS, useCandidateDecision, useCompetitorCandidates, usePromptAnswers, usePromptResults, usePromptSettings, useRetryPlan, useRetryRun, useStartRun } from './usePromptTest'

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
      routes({ 'GET /api/v1/admin/competitor-candidates': () => json({ items: [], counts: { pending: 0, added: 0, dismissed: 0, unchecked: 0 }, review: { running: false } }) }),
    )
    expect(state.candidates.value).toEqual([])
    expect(state.counts.value).toBeUndefined()
    await flushPromises()
    expect(state.counts.value?.pending).toBe(0)
    expect(state.review.value).toEqual({ running: false })
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

describe('usePromptAnswers while the run runs', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('asks for the answers again until polling stops', async () => {
    vi.useFakeTimers()
    const poll = ref(true)
    const { sent } = mountComposable(
      () => usePromptAnswers(ref(RUN_ID), ref({}), poll),
      routes({ [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => json({ run: promptRun(), items: [] }) }),
    )
    await vi.advanceTimersByTimeAsync(RUN_POLL_MS + 10)
    expect(sent).toHaveLength(2)
    poll.value = false
    await vi.advanceTimersByTimeAsync(RUN_POLL_MS * 3)
    expect(sent).toHaveLength(2)
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

  it('adds or dismisses several at once, splits a name off, and starts Haiku', async () => {
    const { state, sent } = mountComposable(
      useCandidateDecision,
      routes({
        'POST /api/v1/admin/competitor-candidates/decisions': () =>
          json({ items: [{ id: '3a7bd3e2360a9f01', name: 'Intralox', outcome: 'done' }, { id: '0123456789abcdef', outcome: 'skipped', reason: 'No website.' }] }),
        'POST /api/v1/admin/competitor-candidates/0123456789abcdef/split': () => new Response(null, { status: 204 }),
        'POST /api/v1/admin/competitor-candidates/review': () => json({ running: true }, 202),
      }),
    )
    state.decideMany('add', ['3a7bd3e2360a9f01', '0123456789abcdef'])
    expect(state.isDeciding.value).toBe(true)
    await flushPromises()
    expect(state.notice.value).toBe('Added 1 company. Skipped 1.')
    expect(state.results.value).toEqual([{ id: '0123456789abcdef', outcome: 'skipped', reason: 'No website.' }])
    state.split('0123456789abcdef', 'Metiz')
    await flushPromises()
    expect(state.notice.value).toBe('Metiz is a company of its own again.')
    expect(state.results.value).toEqual([])
    state.startReview()
    await flushPromises()
    expect(state.notice.value).toBe('Haiku is merging the duplicates and checking the companies.')
    expect(state.error.value).toBeNull()
    expect(sent.filter((r) => !r.startsWith('GET'))).toEqual([
      'POST /api/v1/admin/competitor-candidates/decisions',
      'POST /api/v1/admin/competitor-candidates/0123456789abcdef/split',
      'POST /api/v1/admin/competitor-candidates/review',
    ])
  })
})

describe('usePromptSettings', () => {
  it('says when the account may not read the settings', async () => {
    const { state } = mountComposable(usePromptSettings, routes({ 'GET /api/v1/admin/prompt-test': () => problem(403, 'Forbidden') }))
    await flushPromises()
    expect(state.forbidden.value).toBe(true)
    expect(state.settings.value).toBeUndefined()
    expect(state.error.value).toBe('Only administrators can do this.')
  })
})

describe('useStartRun', () => {
  it('says why a run could not start', async () => {
    const { state } = mountComposable(useStartRun, routes({ 'POST /api/v1/admin/prompt-runs': () => problem(409, 'Conflict', 'A Prompt run is running already.') }))
    expect(state.error.value).toBeNull()
    state.start()
    await flushPromises()
    expect(state.error.value).toBe('A Prompt run is running already.')
    expect(state.notice.value).toBeNull()
  })
})

describe('useRetryPlan', () => {
  it('asks for the plan only when enabled and a run is named', async () => {
    const enabled = ref(false)
    const plan = { askTasks: 2, askCostUsd: 0.01, unknownTasks: 0, readTasks: 1, readCostUsd: 0.001, readingCostUsd: 0.0003, answerCosts: [] }
    const { state, sent } = mountComposable(() => useRetryPlan(ref(RUN_ID), enabled), routes({ [`GET /api/v1/admin/prompt-runs/${RUN_ID}/retry-plan`]: () => json(plan) }))
    await flushPromises()
    expect(sent).toEqual([])
    expect(state.plan.value).toBeUndefined()
    enabled.value = true
    await flushPromises()
    expect(state.plan.value?.askTasks).toBe(2)
  })

  it('asks nothing without a run', async () => {
    const { sent } = mountComposable(() => useRetryPlan(ref(undefined), ref(true)), routes({}))
    await flushPromises()
    expect(sent).toEqual([])
  })
})

describe('useRetryRun', () => {
  const accepted = (tasks: number) => () => json({ run: promptRun({ status: 'running' }), tasks, estimatedCostUsd: 0.42 }, 202)

  it('asks a run again, or one of its responses, and says how many and for what', async () => {
    const bodies: unknown[] = []
    const { state } = mountComposable(
      useRetryRun,
      routes({
        [`POST /api/v1/admin/prompt-runs/${RUN_ID}/retry`]: async (request) => {
          bodies.push(await request.json())
          return accepted(bodies.length === 1 ? 3 : 1)()
        },
      }),
    )
    state.retry(RUN_ID)
    await flushPromises()
    expect(state.notice.value).toBe('Asking 3 answers again, for about $0.42.')
    state.retry(RUN_ID, { promptId: 'p', provider: 'openai', mode: 'web' })
    await flushPromises()
    expect(state.notice.value).toBe('Asking one answer again, for about $0.42.')
    expect(bodies).toEqual([{}, { promptId: 'p', provider: 'openai', mode: 'web' }])
    expect(state.isRetrying.value).toBe(false)
    expect(state.error.value).toBeNull()
  })

  it('reads a run again, and reports a failure of either', async () => {
    const { state } = mountComposable(
      useRetryRun,
      routes({
        [`POST /api/v1/admin/prompt-runs/${RUN_ID}/reread`]: accepted(2),
        [`POST /api/v1/admin/prompt-runs/${RUN_ID}/retry`]: () => problem(409, 'Conflict', 'A run is running.'),
      }),
    )
    state.reread(RUN_ID)
    await flushPromises()
    expect(state.notice.value).toBe('Reading 2 answers again, for about $0.42.')
    state.retry(RUN_ID)
    await flushPromises()
    expect(state.error.value).toContain('A run is running.')
  })

  it('reports a failed reading', async () => {
    const { state } = mountComposable(useRetryRun, routes({ [`POST /api/v1/admin/prompt-runs/${RUN_ID}/reread`]: () => problem(409, 'Conflict', 'Busy.') }))
    state.reread(RUN_ID, { promptId: 'p', provider: 'openai', mode: 'web' })
    await flushPromises()
    expect(state.error.value).toContain('Busy.')
  })
})
