import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { promptRun, RUN_ID } from '../../../testing/fixtures'
import { json, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useCompetitorCandidates, usePromptAnswers } from './usePromptTest'

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
  })
})
