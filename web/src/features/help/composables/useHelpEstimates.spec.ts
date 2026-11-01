import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { promptSettings } from '../fixtures'
import { useHelpEstimates } from './useHelpEstimates'

const settings = promptSettings({ runUsd: 1.25, monthUsd: 5.4 }, 'weekly')

describe('useHelpEstimates', () => {
  it("gives an administrator the Prompt test's live estimates", async () => {
    const { state, sent } = mountComposable(() => useHelpEstimates(ref(true)), routes({ 'GET /api/v1/admin/prompt-test': () => json(settings) }))
    await flushPromises()
    expect(sent).toEqual(['GET /api/v1/admin/prompt-test'])
    expect(state.estimates.value).toEqual({ runUsd: 1.25, cadence: 'weekly' })
  })

  it('asks nothing for a reader, and gives no estimates', async () => {
    const { state, sent } = mountComposable(() => useHelpEstimates(ref(false)), routes({ 'GET /api/v1/admin/prompt-test': () => json(settings) }))
    await flushPromises()
    expect(sent).toEqual([])
    expect(state.estimates.value).toBeUndefined()
  })

  it('gives no estimates when the settings cannot be read', async () => {
    const { state } = mountComposable(() => useHelpEstimates(ref(true)), routes({ 'GET /api/v1/admin/prompt-test': () => problem(500, 'Broken') }))
    await flushPromises()
    expect(state.estimates.value).toBeUndefined()
  })
})
