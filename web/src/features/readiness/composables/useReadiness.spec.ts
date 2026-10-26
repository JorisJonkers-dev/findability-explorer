import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { readinessRow } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useReadinessFiles, useReadinessTable } from './useReadiness'

const SCAN = '5b0c1a9e-2d3f-4e5a-8b6c-7d8e9f0a1b2c'

describe('useReadinessTable', () => {
  it('loads every row once, and finds one company in it', async () => {
    const { state, sent } = mountComposable(useReadinessTable, routes({ 'GET /api/v1/ai-readiness': () => json({ items: [readinessRow('Twentebelt')] }) }))
    expect(state.rows.value).toEqual([])
    expect(state.rowOf('twentebelt')).toBeUndefined()
    await flushPromises()
    expect(state.rows.value.map((r) => r.name)).toEqual(['Twentebelt'])
    expect(state.rowOf('twentebelt')?.name).toBe('Twentebelt')
    expect(state.error.value).toBeNull()
    expect(sent).toEqual(['GET /api/v1/ai-readiness'])
  })

  it('reports a failure', async () => {
    const { state } = mountComposable(useReadinessTable, routes({ 'GET /api/v1/ai-readiness': () => problem(500, 'Internal Server Error', 'Down.') }))
    await flushPromises()
    expect(state.error.value).toContain('Down.')
  })
})

describe('useReadinessFiles', () => {
  it('asks nothing until enabled with a scan, then loads the files', async () => {
    const enabled = ref(false)
    const scan = ref<string | undefined>(undefined)
    const { state, sent } = mountComposable(
      () => useReadinessFiles(scan, enabled),
      routes({
        [`GET /api/v1/admin/analyses/${SCAN}/readiness-files`]: () =>
          json({ robotsTxt: 'User-agent: *', robotsTruncated: false, llmsTxt: '', llmsTruncated: false, llmsFullTxt: '', llmsFullTruncated: false }),
      }),
    )
    await flushPromises()
    expect(sent).toEqual([])
    enabled.value = true
    await flushPromises()
    expect(sent).toEqual([])
    scan.value = SCAN
    await nextTick()
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(state.files.value?.robotsTxt).toBe('User-agent: *')
    expect(state.isLoading.value).toBe(false)
    expect(state.missing.value).toBe(false)
  })

  it('tells a scan without readiness from a failure', async () => {
    const missing = mountComposable(
      () => useReadinessFiles(ref(SCAN), ref(true)),
      routes({ [`GET /api/v1/admin/analyses/${SCAN}/readiness-files`]: () => problem(404, 'Not Found', 'The scan recorded no AI readiness.') }),
    )
    await flushPromises()
    expect(missing.state.missing.value).toBe(true)
    expect(missing.state.error.value).toBeNull()
    const failed = mountComposable(
      () => useReadinessFiles(ref(SCAN), ref(true)),
      routes({ [`GET /api/v1/admin/analyses/${SCAN}/readiness-files`]: () => problem(500, 'Internal Server Error', 'Down.') }),
    )
    await flushPromises()
    expect(failed.state.missing.value).toBe(false)
    expect(failed.state.error.value).toContain('Down.')
  })
})
