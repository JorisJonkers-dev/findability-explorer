import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { competitor, summary } from '../../../testing/fixtures'
import { json, problem } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useCompany } from './useCompany'

describe('useCompany', () => {
  it('fetches one company and its scans', async () => {
    const { state, sent } = mountComposable(
      () => useCompany(ref('twentebelt')),
      () => json({ competitor: competitor('Twentebelt'), scans: [summary()] }),
    )
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(sent).toEqual(['GET /api/v1/competitors/twentebelt'])
    expect(state.detail.value?.competitor.name).toBe('Twentebelt')
    expect(state.detail.value?.scans).toHaveLength(1)
    expect(state.notFound.value).toBe(false)
    expect(state.error.value).toBeNull()
  })

  it('tells an unknown company from a failed request', async () => {
    const missing = mountComposable(() => useCompany(ref('nobody')), () => problem(404, 'Not Found'))
    await flushPromises()
    expect(missing.state.notFound.value).toBe(true)
    const failing = mountComposable(() => useCompany(ref('twentebelt')), () => problem(500, 'Internal Server Error'))
    await flushPromises()
    expect(failing.state.notFound.value).toBe(false)
    expect(failing.state.error.value).toBe('Internal Server Error')
  })
})
