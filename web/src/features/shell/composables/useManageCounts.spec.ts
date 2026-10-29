import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { manageCounts } from '../../../testing/manageFixtures'
import { mountComposable } from '../../../testing/mountComposable'
import { problem, routes } from '../../../testing/mountApp'
import { useManageCounts } from './useManageCounts'

describe('useManageCounts', () => {
  it('counts what waits and what runs, by the Manage section it belongs to', async () => {
    const { state, sent } = mountComposable(useManageCounts, routes(manageCounts({ terms: 8, candidates: 3, runs: 1, scans: 2 })))
    await flushPromises()
    expect(state.counts.value).toEqual({ '/manage/prompt-test': 1, '/manage/scans': 2, '/manage/competitors': 3, '/manage/terms': 8 })
    expect(sent).toEqual(
      expect.arrayContaining([
        'GET /api/v1/admin/term-candidates?status=pending',
        'GET /api/v1/admin/competitor-candidates?status=pending',
        'GET /api/v1/prompt-test',
        'GET /api/v1/analyses?limit=8',
      ]),
    )
  })

  it('counts nothing while the answers are missing', async () => {
    const { state } = mountComposable(
      useManageCounts,
      routes({ 'GET /api/v1/admin/term-candidates': () => problem(500, 'Down'), 'GET /api/v1/admin/competitor-candidates': () => problem(500, 'Down') }),
    )
    await flushPromises()
    expect(state.counts.value).toEqual({ '/manage/prompt-test': 0, '/manage/scans': 0, '/manage/competitors': 0, '/manage/terms': 0 })
  })
})
