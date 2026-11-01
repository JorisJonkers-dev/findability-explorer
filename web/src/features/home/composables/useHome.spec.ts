import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { overviewRoutes } from '../../../testing/homeFixtures'
import { problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useHome } from './useHome'

describe('useHome', () => {
  it('loads the goals, the figures, the newest run, and the scans, and builds the four sections', async () => {
    const { state, sent } = mountComposable(useHome, routes(overviewRoutes()))
    expect(state.isLoading.value).toBe(true)
    expect(state.rows.value).toEqual([])
    expect(state.summary.value).toBe('')
    expect(state.tiles.value).toEqual([])
    expect(state.latest.value).toEqual([])
    await flushPromises()
    expect(state.isLoading.value).toBe(false)
    expect(state.error.value).toBeNull()
    expect(state.rows.value).toHaveLength(5)
    expect(state.summary.value).toContain('tribelt.nl passes 3 of the 8 AI readiness checks')
    expect(state.tiles.value).toHaveLength(5)
    expect(state.dataLine.value).toBe('From the Prompt run of 1 November 2026 and the scans up to 28 October 2026.')
    expect(state.latest.value[0]?.next).toBe('')
    // A reader may not read the Prompt test settings, so they are not asked for.
    expect(sent).not.toContain('GET /api/v1/admin/prompt-test')
  })

  it('tells an administrator when the next Prompt run is', async () => {
    const { state, sent } = mountComposable(useHome, routes(overviewRoutes({}, true)))
    await flushPromises()
    await flushPromises()
    expect(sent).toContain('GET /api/v1/admin/prompt-test')
    expect(state.latest.value[0]?.next).toBe('The next run is scheduled for 8 November 2026.')
  })

  it('reports a failure of any part it needs', async () => {
    for (const path of ['GET /api/v1/plan', 'GET /api/v1/overview', 'GET /api/v1/prompt-test', 'GET /api/v1/ai-readiness']) {
      const { state } = mountComposable(useHome, routes(overviewRoutes({ [path]: () => problem(500, 'Internal Server Error', `${path} down.`) })))
      await flushPromises()
      expect(state.error.value, path).toContain(`${path} down.`)
      if (path === 'GET /api/v1/prompt-test') expect(state.latest.value[0]?.latest).toBe('No Prompt run has finished yet.')
    }
  })
})
