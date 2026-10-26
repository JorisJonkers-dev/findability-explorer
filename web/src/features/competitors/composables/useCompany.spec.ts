import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { competitor, summary } from '../../../testing/fixtures'
import { json, problem } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { POLL_INTERVAL_MS } from '../../scan/domain/status'
import { useCompany } from './useCompany'

afterEach(() => {
  vi.useRealTimers()
})

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

  it('asks again while its newest scan reads pages, and stops once it is done', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    const pages = [3, 15]
    let call = 0
    const { state, sent } = mountComposable(
      () => useCompany(ref('twentebelt')),
      () => {
        const n = pages[Math.min(call++, 1)] ?? 15
        return json({ competitor: competitor('Twentebelt'), scans: [summary({ status: n < 15 ? 'running' : 'done', pagesCrawled: n })] })
      },
    )
    await flushPromises()
    expect(state.detail.value?.scans[0]?.pagesCrawled).toBe(3)
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS)
    await flushPromises()
    expect(state.detail.value?.scans[0]?.status).toBe('done')
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS * 3)
    expect(sent).toHaveLength(2)
  })
})
