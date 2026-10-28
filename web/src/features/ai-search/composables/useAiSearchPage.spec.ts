import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { AiSearchSummary, Plan } from '../../../infrastructure/api/types.gen'
import { promptResults } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useAiSearchPage } from './useAiSearchPage'

const summary: AiSearchSummary = {
  goal: 0.4,
  followed: [
    { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt' },
    { competitorId: 'ashworth', name: 'Ashworth', role: 'competitor' },
    { competitorId: 'martens', name: 'Märtens', role: 'competitor' },
  ],
  trend: [{ runStartedAt: '2026-11-02T06:00:00Z', shares: [0.24, 0.57, 0.1] }],
  crawlers: [],
  keepingOut: [],
  ranking: [{ competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', score: 3.5 }],
  sites: 41,
  questions: 20,
  providers: 5,
  readinessChecks: 5,
  checks: [{ check: 'crawlers', scored: true, passing: 30, tribelt: true }],
}
const plan: Plan = { catalogueVersion: 1, improvements: [], watched: [{ target: 'aiShareOfVoice', goal: 0.5 }], eyeOn: [], readinessChecks: 5 }

const ok = {
  'GET /api/v1/ai-search': () => json(summary),
  'GET /api/v1/prompt-test': () => json(promptResults()),
  'GET /api/v1/plan': () => json(plan),
}

describe('useAiSearchPage', () => {
  it('reads the summary, the newest run and the plan, and writes the page from them', async () => {
    const { state, sent } = mountComposable(useAiSearchPage, routes(ok))
    expect(state.isLoading.value).toBe(true)
    expect(state.headline.value).toBe('AI assistants, and what they say about Tribelt.')
    expect(state.figures.value).toEqual([])
    expect(state.rows.value).toEqual([])
    expect(state.readinessLine.value).toBe('')
    await flushPromises()
    expect(state.isLoading.value).toBe(false)
    expect(state.error.value).toBeNull()
    expect(state.rows.value.map((r) => r.name)).toEqual(['Ashworth', 'Tribelt B.V.', 'Märtens'])
    expect(state.headline.value).toBe('Tribelt ranks second among the companies AI answers name.')
    expect(state.lede.value).toContain('gave 12 answers')
    expect(state.cells.value).toHaveLength(4)
    expect(state.figures.value.map((f) => f.big)).toEqual(['25%', '17%', '3½ of 5'])
    expect(state.figures.value[0]?.note).toBe('Target 50%.')
    expect(state.readinessLine.value).toBe('Tribelt passes all five checks. See how the other sites compare.')
    expect([...sent].sort()).toEqual(['GET /api/v1/ai-search', 'GET /api/v1/plan', 'GET /api/v1/prompt-test'])
  })

  it('keeps loading until the results are in, though the summary is', async () => {
    let release: (() => void) | undefined
    const slow = new Promise<void>((resolve) => (release = resolve))
    const { state } = mountComposable(useAiSearchPage, (request) => (request.url.endsWith('/prompt-test') ? slow.then(() => json(promptResults())) : routes(ok)(request)) as never)
    await flushPromises()
    expect(state.summary.value).toBeDefined()
    expect(state.isLoading.value).toBe(true)
    release?.()
    await flushPromises()
    expect(state.isLoading.value).toBe(false)
  })

  it('reports a failure of the summary or of the results', async () => {
    const summaryDown = mountComposable(useAiSearchPage, routes({ ...ok, 'GET /api/v1/ai-search': () => problem(500, 'Internal Server Error', 'Down.') }))
    await flushPromises()
    expect(summaryDown.state.error.value).toContain('Down.')
    const resultsDown = mountComposable(useAiSearchPage, routes({ ...ok, 'GET /api/v1/prompt-test': () => problem(500, 'Internal Server Error', 'Gone.') }))
    await flushPromises()
    expect(resultsDown.state.error.value).toContain('Gone.')
  })
})
