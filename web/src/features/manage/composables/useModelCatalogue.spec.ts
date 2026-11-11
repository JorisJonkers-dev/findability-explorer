import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { modelCatalogue } from '../fixtures'
import { useModelCatalogue } from './useModelCatalogue'

describe('useModelCatalogue', () => {
  it('reads OpenRouter’s catalogue once the key works', async () => {
    const works = ref(false)
    const { state, sent } = mountComposable(() => useModelCatalogue(works), routes({ 'GET /api/v1/admin/prompt-test/catalogue': () => json(modelCatalogue()) }))
    await flushPromises()
    expect(sent).toEqual([])
    expect(state.models.value).toEqual([])
    expect(state.readAt.value).toBe('')
    works.value = true
    await flushPromises()
    expect(sent).toEqual(['GET /api/v1/admin/prompt-test/catalogue'])
    expect(state.models.value.map((m) => m.id)).toEqual(['openai/gpt-oss-20b', 'google/gemma:free', 'qwen/qwen3-8b:free', 'openai/gpt-5-nano'])
    expect(state.readAt.value).toBe('2026-11-01T06:00:00Z')
    expect(state.error.value).toBeNull()
  })

  it('says why it cannot be read', async () => {
    const { state } = mountComposable(() => useModelCatalogue(ref(true)), routes({ 'GET /api/v1/admin/prompt-test/catalogue': () => problem(422, 'No OpenRouter key works.') }))
    await flushPromises()
    expect(state.error.value).toBe('No OpenRouter key works.')
  })
})
