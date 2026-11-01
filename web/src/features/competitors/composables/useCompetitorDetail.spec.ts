import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { competitor, overview, promptAnswer, promptResults, promptRun, readiness, readinessRow, readinessTable, summary, term, termSearch } from '../../../testing/fixtures'
import { type Handler, json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useCompetitorDetail } from './useCompetitorDetail'

const tribelt = competitor('Tribelt', { role: 'tribelt' })
const twentebelt = competitor('Twentebelt', { scans: 2 })
const martens = competitor('Martens', { scans: 0 })

function api(extra: Record<string, Handler> = {}, items = [tribelt, twentebelt, martens]) {
  return routes({
    'GET /api/v1/competitors': () => json({ items }),
    'GET /api/v1/competitors/twentebelt': () =>
      json({ competitor: twentebelt, scans: [summary({ id: '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d', status: 'running' }), summary()] }),
    'GET /api/v1/competitors/martens': () => json({ competitor: martens, scans: [] }),
    'GET /api/v1/competitors/nobody': () => problem(404, 'Not Found'),
    'GET /api/v1/terms': () =>
      json(termSearch([term({ term: 'TwenteFlex', usedBy: ['Twentebelt'] }), term({ term: 'Mirror', usedBy: ['Twentebelt', 'Tribelt'], tribeltUses: true }), term({ term: 'Other', usedBy: ['Martens'] })])),
    'GET /api/v1/overview': () => json(overview()),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'GET /api/v1/ai-readiness': () => json(readinessTable([readinessRow('Tribelt', { role: 'tribelt' }), readinessRow('Twentebelt', { readiness: readiness({ sitemap: { declared: true, files: 1, addresses: 5, languages: ['en'], coverage: { pages: 10, listed: 5, share: 0.5 } } }) })])),
    'GET /api/v1/standings': () => json({ measures: [], trailing: 0, open: [], readinessChecks: 9 }),
    ...extra,
  })
}

describe('useCompetitorDetail', () => {
  it('sets one competitor against Tribelt', async () => {
    const { state } = mountComposable(() => useCompetitorDetail(ref('twentebelt')), api())
    await flushPromises()
    expect(state.company.value?.shortName).toBe('Twentebelt')
    expect(state.tribelt.value?.id).toBe('tribelt')
    expect(state.outOf.value).toBe(9)
    expect(state.share.value).toBe(0.75)
    expect(state.ready.value).toBe(8)
    expect(state.title.value).toBe('Twentebelt is named in three times as many AI answers as Tribelt.')
    expect(state.where.value).toBe('Netherlands. Its website is in 2 languages.')
    expect(state.belts.value.find((b) => b.family === 'tri_flex_turn_curve')?.name).toBe('TwenteFlex spiral belts')
    expect(state.measures.value.map((m) => [m.id, m.themText, m.usText])).toEqual([
      ['answers', '75%', '25%'],
      ['ready', '8 of 9', '8 of 9'],
      ['sitemap', '50%', '85%'],
      ['terms', '2', '1'],
    ])
    expect(state.measures.value[3]?.text).toContain('of the 3 terms on the search term list')
    expect(state.latestScan.value?.status).toBe('done')
    expect(state.notFound.value).toBe(false)
  })

  it('knows nothing of a competitor no scan has read', async () => {
    const { state } = mountComposable(() => useCompetitorDetail(ref('martens')), api({ 'GET /api/v1/standings': () => problem(500, 'Internal Server Error'), 'GET /api/v1/overview': () => problem(500, 'Internal Server Error') }))
    await flushPromises()
    expect(state.outOf.value).toBe(8)
    expect(state.ready.value).toBeUndefined()
    expect(state.latestScan.value).toBeUndefined()
    const [, ready, sitemap, terms] = state.measures.value
    expect(ready?.themText).toBe('Not known')
    expect(sitemap?.themText).toBe('Not known')
    expect(terms?.themText).toBe('Not known')
    expect(terms?.text).toContain('of the 3 terms on the search term list')
  })

  it('has nothing to say before the company is known, or without Tribelt', async () => {
    const unknown = mountComposable(() => useCompetitorDetail(ref('nobody')), api())
    await flushPromises()
    expect(unknown.state.notFound.value).toBe(true)
    expect(unknown.state.company.value).toBeUndefined()
    expect(unknown.state.title.value).toBe('A competitor.')
    expect(unknown.state.where.value).toBe('')
    expect(unknown.state.belts.value).toEqual([])
    expect(unknown.state.measures.value).toEqual([])
    const alone = mountComposable(() => useCompetitorDetail(ref('twentebelt')), api({}, [twentebelt]))
    await flushPromises()
    expect(alone.state.tribelt.value).toBeUndefined()
    expect(alone.state.title.value).toBe('Twentebelt is named in 75% of AI answers.')
    const [, ready, sitemap, terms] = alone.state.measures.value
    expect(ready?.usText).toBe('Not known')
    expect(sitemap?.usText).toBe('Not known')
    expect(terms?.usText).toBe('Not known')
  })

  it("says what the competitor page needs: its purpose, its data line, where it is ahead, its terms, and each model's share", async () => {
    const run = promptResults().runs[0]
    const answers = [promptAnswer({ model: 'openai/gpt' }), promptAnswer({ model: 'openai/gpt' }, ['Tribelt:tribelt'])]
    const { state, sent } = mountComposable(
      () => useCompetitorDetail(ref('twentebelt')),
      api({ [`GET /api/v1/prompt-runs/${run?.id ?? ''}/answers`]: () => json({ run, items: answers }) }),
    )
    await flushPromises()
    expect(state.purpose.value).toBe('twentebelt.com, on the competitor list. It sells Tri-Flex belts.')
    expect(state.dataLine.value).toBe('From the Prompt run of 1 November 2026 and Twentebelt has not been scanned yet.')
    expect(state.ahead.value).toHaveLength(2)
    expect(state.ahead.value[0]).toContain('75%, against Tribelt’s 25%.')
    expect(state.terms.value.shown.map((t) => t.term)).toEqual(['TwenteFlex'])
    expect(state.modelBars.value).toEqual([{ label: 'openai/gpt', value: 0.5, display: '50%', tone: 'blue-grey', highlight: false }])
    expect(sent.some((r) => r.includes('/answers'))).toBe(true)
  })

  it('asks for no responses before a run, and says nothing of a company it does not know', async () => {
    const { state, sent } = mountComposable(() => useCompetitorDetail(ref('nobody')), api({ 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [] })) }))
    await flushPromises()
    expect(state.purpose.value).toBe('')
    expect(state.dataLine.value).toBe('')
    expect(state.modelBars.value).toEqual([])
    expect(sent.some((r) => r.includes('/answers'))).toBe(false)
  })

  it('dates a running run by its start, and a scan by its finish or its start', async () => {
    const running = promptRun({ finishedAt: undefined, status: 'running' })
    const finished = competitor('Twentebelt', { scans: 2, lastScan: summary({ createdAt: '2026-10-19T08:00:00Z', finishedAt: '2026-10-20T09:00:00Z' }) })
    const started = competitor('Martens', { lastScan: summary({ createdAt: '2026-10-21T08:00:00Z', finishedAt: undefined }) })
    const { state } = mountComposable(
      () => useCompetitorDetail(ref('twentebelt')),
      api({ 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [running] })) }, [tribelt, finished, started]),
    )
    await flushPromises()
    expect(state.dataLine.value).toBe('From the Prompt run of 1 November 2026 and the scan of Twentebelt on 20 October 2026.')
    const other = mountComposable(() => useCompetitorDetail(ref('martens')), api({}, [tribelt, finished, started]))
    await flushPromises()
    expect(other.state.dataLine.value).toBe('From the Prompt run of 1 November 2026 and the scan of Martens on 21 October 2026.')
  })
})
