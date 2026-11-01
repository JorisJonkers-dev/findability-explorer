import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { HISTORY_RUN_A as A, HISTORY_RUN_B as B, promptHistory } from '../../../testing/historyFixtures'
import { json, problem, routes, settle } from '../../../testing/mountApp'
import { mountAt } from '../../../testing/mountAt'
import { LANGUAGE_MEASURES } from '../domain/history'
import { useHistory } from './useHistory'

async function mountHistory(path: string, handle = routes({ 'GET /api/v1/prompt-test/history': () => json(promptHistory()) })) {
  let state: ReturnType<typeof useHistory> | undefined
  const Probe = defineComponent({ setup: () => ((state = useHistory(LANGUAGE_MEASURES)), () => h('div')) })
  const mounted = await mountAt(Probe, path, handle)
  if (!state) throw new Error('the composable did not mount')
  return { ...mounted, state }
}

describe('useHistory', () => {
  it('loads the history once and shows every version by default, with the break between them', async () => {
    const { state, requests } = await mountHistory('/ai-responses')
    expect(requests()).toEqual(['GET /api/v1/prompt-test/history'])
    expect(state.isLoading.value).toBe(false)
    expect(state.history.value?.versions).toHaveLength(2)
    expect(state.error.value).toBeNull()
    expect(state.version.value).toBe('all')
    expect(state.options.value.map((o) => o.label)).toEqual(['All versions', 'Version 2, current', 'Version 1'])
    expect(state.runs.value.map((r) => r.runId)).toEqual([A, B])
    expect(state.breaks.value.map((b) => b.label)).toEqual(['Prompt set version 2'])
    expect(state.note.value).toContain('Version 1 asked 90 questions, and version 2 asks 20 questions.')
    expect(state.series.value.map((s) => [s.label, s.points.length])).toEqual([['All', 2], ['Dutch', 2], ['German', 0], ['English', 1]])
    expect(state.notes.value).toHaveLength(2)
    expect(state.rows.value.map((r) => r.x)).toEqual(['2026-10-27T06:03:12Z', '2026-10-27T08:36:32Z'])
    expect(state.hasPoints.value).toBe(true)
  })

  it('reads the version from the address, and keeps a new choice there', async () => {
    const { state, router } = await mountHistory('/ai-responses?version=1&x=y')
    expect(state.version.value).toBe(1)
    expect(state.runs.value.map((r) => r.runId)).toEqual([A])
    expect(state.breaks.value).toEqual([])
    expect(state.note.value).toContain('It has one run so far, so there is no line yet.')
    state.choose(2)
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/ai-responses?version=2&x=y')
    expect(state.runs.value.map((r) => r.runId)).toEqual([B])
    state.choose('all')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/ai-responses?x=y')
    expect(state.version.value).toBe('all')
  })

  it('has no points before anything loads, and reports a failure', async () => {
    const { state } = await mountHistory('/ai-responses', routes({ 'GET /api/v1/prompt-test/history': () => problem(500, 'Internal Server Error', 'Down.') }))
    expect(state.error.value).toContain('Down.')
    expect(state.runs.value).toEqual([])
    expect(state.options.value).toEqual([{ value: 'all', label: 'All versions' }])
    expect(state.hasPoints.value).toBe(false)
  })
})
