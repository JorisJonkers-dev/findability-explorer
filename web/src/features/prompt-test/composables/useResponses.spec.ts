import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { LocationQuery } from 'vue-router'
import { promptAnswer, promptResults, promptRun, promptSets, RUN_ID } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useEveryResponse, useOneResponse, usePromptSets } from './useResponses'

const OLDER = '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d'
const N = '00000000-0000-4000-8000-0000000000a1'
const U = '00000000-0000-4000-8000-0000000000a2'
const OLD = '00000000-0000-4000-8000-0000000000a3'
const named = promptAnswer({ id: N, mode: 'web' }, ['Tribelt:tribelt'])
const unread = promptAnswer({ id: U, provider: 'google', model: 'google/gemma:free', extraction: undefined })

const api = routes({
  'GET /api/v1/prompt-test': () => json(promptResults({ runs: [promptRun(), promptRun({ id: OLDER, startedAt: '2026-10-25T06:00:00Z', status: 'stopped' })] })),
  [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => json({ run: promptRun(), items: [named, unread] }),
  [`GET /api/v1/prompt-runs/${OLDER}/answers`]: () => json({ run: promptRun({ id: OLDER }), items: [promptAnswer({ id: OLD }, ['Twentebelt:twentebelt'])] }),
})

describe('usePromptSets', () => {
  it('loads the published Prompt sets, and reports a failure', async () => {
    const ok = mountComposable(usePromptSets, routes({ 'GET /api/v1/prompt-test/prompt-sets': () => json(promptSets()) }))
    await flushPromises()
    expect(ok.state.sets.value?.versions[0]?.version).toBe(1)
    expect(ok.state.error.value).toBeNull()
    const down = mountComposable(usePromptSets, routes({ 'GET /api/v1/prompt-test/prompt-sets': () => problem(500, 'Internal Server Error', 'Down.') }))
    await flushPromises()
    expect(down.state.error.value).toContain('Down.')
  })
})

describe('useEveryResponse', () => {
  it('shows the newest run, filtered as the address says', async () => {
    const query = ref<LocationQuery>({})
    const { state, sent } = mountComposable(() => useEveryResponse(query), api)
    expect(state.grid.value.rows).toEqual([])
    expect(state.models.value).toEqual([{ value: 'all', label: 'All models' }])
    expect(state.runs.value).toEqual([])
    await flushPromises()
    expect(state.run.value?.id).toBe(RUN_ID)
    expect(state.runs.value).toHaveLength(2)
    expect(state.list.value.map((a) => a.id)).toEqual([N, U])
    expect(state.grid.value.groups.map((g) => g.label)).toEqual(['OpenAI, gpt-oss-20b', 'Google, gemma:free'])
    expect(state.models.value.map((m) => m.label)).toEqual(['All models', 'OpenAI, gpt-oss-20b', 'Google, gemma:free'])
    expect(state.scopes.value.map((s) => s.value)).toEqual(['all', 'twente'])
    expect(state.key.value.map((k) => k.state)).toEqual(['first', 'named', 'not', 'failed', 'unread'])
    query.value = { names: 'yes' }
    await flushPromises()
    expect(state.list.value.map((a) => a.id)).toEqual([N])
    expect(state.key.value.map((k) => k.state)).toEqual(['first', 'named', 'not', 'failed'])
    query.value = { run: OLDER }
    await flushPromises()
    expect(state.list.value.map((a) => a.id)).toEqual([OLD])
    expect(sent).toContain(`GET /api/v1/prompt-runs/${OLDER}/answers`)
  })
})

describe('useOneResponse', () => {
  it('finds a response in the newest run, or in the run the address names', async () => {
    const id = ref(N)
    const runId = ref<string | undefined>(undefined)
    const { state } = mountComposable(() => useOneResponse(id, runId), api)
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(state.isLoading.value).toBe(false)
    expect(state.answer.value?.id).toBe(N)
    expect(state.run.value?.id).toBe(RUN_ID)
    id.value = OLD
    runId.value = OLDER
    await flushPromises()
    expect(state.answer.value?.id).toBe(OLD)
    id.value = 'nobody'
    expect(state.answer.value).toBeUndefined()
    expect(state.error.value).toBeNull()
  })

  it('reports a failure of the results or of the responses', async () => {
    const down = mountComposable(() => useOneResponse(ref(N), ref(undefined)), routes({ 'GET /api/v1/prompt-test': () => problem(500, 'Internal Server Error', 'Down.') }))
    await flushPromises()
    expect(down.state.error.value).toContain('Down.')
    const gone = mountComposable(() => useOneResponse(ref(N), ref(OLDER)), routes({ [`GET /api/v1/prompt-runs/${OLDER}/answers`]: () => problem(500, 'Internal Server Error', 'Gone.') }))
    await flushPromises()
    expect(gone.state.error.value).toContain('Gone.')
  })
})
