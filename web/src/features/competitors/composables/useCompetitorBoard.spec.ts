import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { bots, competitor, overview, promptResults, readiness, readinessRow, readinessTable, summary, term, termSearch } from '../../../testing/fixtures'
import { type Handler, json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useCompetitorBoard } from './useCompetitorBoard'

const tribelt = competitor('Tribelt', { role: 'tribelt' })
const twentebelt = competitor('Twentebelt', { beltNames: [] })
const martens = competitor('Martens')
const group = competitor('Tribelt Group', { role: 'group' })
const old = competitor('Old Belts', { archived: true })

function api(extra: Record<string, Handler> = {}, items = [tribelt, twentebelt, martens, group, old]) {
  return routes({
    'GET /api/v1/competitors': () => json({ items }),
    'GET /api/v1/terms': () => json(termSearch([term({ usedBy: ['Twentebelt'] })])),
    'GET /api/v1/overview': () => json(overview({ competitors: { makers: 7, families: [{ family: 'eyelink', makers: 2 }] } })),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'GET /api/v1/ai-readiness': () =>
      json(readinessTable([readinessRow('Tribelt', { role: 'tribelt' }), readinessRow('Twentebelt', { readiness: readiness({ bots: bots({ GPTBot: 'blocked' }) }) })])),
    'GET /api/v1/standings': () => json({ measures: [], trailing: 0, open: [], readinessChecks: 9 }),
    ...extra,
  })
}

describe('useCompetitorBoard', () => {
  it('lists the competitors on the list with their share of AI answers and readiness score', async () => {
    const { state } = mountComposable(useCompetitorBoard, api())
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(state.rows.value).toEqual([
      { id: 'twentebelt', name: 'Twentebelt', share: 0.75, ready: 7.5 },
      { id: 'martens', name: 'Martens', share: 0, ready: undefined },
    ])
    expect(state.tribelt.value?.id).toBe('tribelt')
    expect(state.tribeltShare.value).toBe(0.25)
    expect(state.tribeltReady.value).toBe(8)
    expect(state.group.value.map((c) => c.id)).toEqual(['tribelt-group'])
    expect(state.error.value).toBeNull()
  })

  it('takes the count, the chart and the checks from the server', async () => {
    const { state } = mountComposable(useCompetitorBoard, api())
    await flushPromises()
    expect(state.count.value).toBe(7)
    expect(state.outOf.value).toBe(9)
    expect(state.title.value).toBe('7 competitors sell metal belts that Tribelt’s buyers could buy instead.')
    expect(state.figures.value.map((f) => f.big)).toEqual(['7', '1', '0'])
    expect(state.bars.value).toEqual([{ family: 'eyelink', label: 'Eyelink', makers: 2, tribelt: null, path: '/terms/families/eyelink' }])
  })

  it('counts what it lists and eight checks until the server says, and has nothing to compare without Tribelt', async () => {
    const { state } = mountComposable(
      useCompetitorBoard,
      api({ 'GET /api/v1/overview': () => problem(500, 'Internal Server Error'), 'GET /api/v1/standings': () => problem(500, 'Internal Server Error'), 'GET /api/v1/prompt-test': () => problem(500, 'Internal Server Error') }, [twentebelt, martens]),
    )
    await flushPromises()
    expect(state.count.value).toBe(2)
    expect(state.outOf.value).toBe(8)
    expect(state.bars.value).toEqual([])
    expect(state.tribelt.value).toBeUndefined()
    expect(state.tribeltShare.value).toBeUndefined()
    expect(state.tribeltReady.value).toBeUndefined()
    expect(state.figures.value).toHaveLength(1)
  })

  it('says when the companies cannot be loaded', async () => {
    const { state } = mountComposable(useCompetitorBoard, api({ 'GET /api/v1/competitors': () => problem(500, 'Internal Server Error') }))
    await flushPromises()
    expect(state.error.value).toBe('Internal Server Error')
  })
})

describe("the Competitors table's rows", () => {
  it('hold Tribelt and every competitor, archived too, with their figures', async () => {
    const scanned = competitor('Twentebelt', { beltNames: [], lastScan: summary({ createdAt: '2026-10-20T08:00:00Z', finishedAt: '2026-10-20T09:00:00Z' }) })
    const { state } = mountComposable(
      useCompetitorBoard,
      api({ 'GET /api/v1/terms': () => json(termSearch([term({ usedBy: ['Twentebelt'] }), term({ term: 'x', usedBy: [], tribeltUses: true })])) }, [tribelt, scanned, martens, group, old]),
    )
    await flushPromises()
    expect(state.tableRows.value.map((r) => [r.id, r.share, r.ready, r.terms, r.archived])).toEqual([
      ['tribelt', 0.25, 8, 1, false],
      ['twentebelt', 0.75, 7.5, 1, false],
      ['martens', 0, undefined, 0, false],
      ['old-belts', 0, undefined, 0, true],
    ])
    expect(state.newestScan.value).toBe('2026-10-20T09:00:00Z')
  })

  it('count no terms for a company the map credits none to, and have no newest scan before one', async () => {
    const { state } = mountComposable(
      useCompetitorBoard,
      api({ 'GET /api/v1/terms': () => json(termSearch([{ ...term(), makers: undefined }])) }, [twentebelt]),
    )
    await flushPromises()
    expect(state.tableRows.value.map((r) => [r.id, r.terms])).toEqual([['twentebelt', 0]])
    expect(state.newestScan.value).toBeUndefined()
  })
})
