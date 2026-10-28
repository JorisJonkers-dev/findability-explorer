import { describe, expect, it } from 'vitest'
import type { AiSearchSummary, Plan } from '../../../infrastructure/api/types.gen'
import { promptResults, promptRun } from '../../../testing/fixtures'
import { json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const summary: AiSearchSummary = {
  goal: 0.4,
  followed: [
    { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt' },
    { competitorId: 'ashworth', name: 'Ashworth', role: 'competitor' },
    { competitorId: 'lesjofors', name: 'Lesjöfors', role: 'group' },
  ],
  trend: [
    { runStartedAt: '2026-10-26T06:00:00Z', shares: [0.14, 0.62, 0.1] },
    { runStartedAt: '2026-11-02T06:00:00Z', shares: [0.24, 0.57, 0.1] },
  ],
  crawlers: [],
  keepingOut: [],
  ranking: [
    { competitorId: 'martens', name: 'Märtens', role: 'competitor', score: 5 },
    { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', score: 3.5 },
  ],
  sites: 41,
  questions: 20,
  providers: 5,
  readinessChecks: 8,
  checks: [
    { check: 'crawlers', scored: true, passing: 30, tribelt: true },
    { check: 'llmsTxt', scored: true, passing: 12, tribelt: false },
  ],
}

const plan: Plan = {
  catalogueVersion: 1,
  improvements: [],
  watched: [
    { target: 'aiShareOfVoice', goal: 0.5 },
    { target: 'twenteShareOfVoice', goal: 0.6 },
    { target: 'citationShare', goal: 0.5 },
  ],
  eyeOn: [],
  readinessChecks: 8,
}

const api = (extra: Record<string, () => Response> = {}) =>
  routes({
    'GET /api/v1/ai-search': () => json(summary),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'GET /api/v1/plan': () => json(plan),
    ...extra,
  })

describe('the AI search page', () => {
  it('heads the page with where Tribelt ranks among the companies the answers name, and says what the newest run asked', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search')
    expect(document.title).toBe('AI search · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Tribelt ranks second among the companies AI answers name.')
    expect(wrapper.text()).toContain("In the newest Prompt run, AI assistants from two providers gave 12 answers to buyers' questions. OpenAI names Tribelt most often, in 3 of its 6 answers.")
  })

  it('shows three key figures that lead to the numbers and to AI readiness', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search')
    const figures = wrapper.findAll('section[aria-label="Key figures"] a')
    expect(figures.map((f) => [f.attributes('href'), f.text()])).toEqual([
      ['/numbers/aiShareOfVoice', '25%of AI answers name TribeltTarget 50%.'],
      ['/numbers/citationShare', '17%of web answers cite tribelt.nlTarget 50%.'],
      ['/ai-search/readiness', '3½ of 8AI readiness checks passedSee how AI readiness is checked.'],
    ])
  })

  it('shows the share of answers naming Tribelt, for all answers or those about Twente', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search')
    const donut = wrapper.get('section[aria-labelledby="donut-title"]')
    expect(donut.text()).toContain('25%')
    expect(donut.text()).toContain('3 of 12')
    expect(donut.text()).toContain('3 answers name Tribelt')
    expect(donut.text()).toContain('The target for the year is 50% of all answers.')
    const pills = donut.findAll('[role="group"] button')
    expect(pills.map((p) => [p.text(), p.attributes('aria-pressed')])).toEqual([
      ['All answers', 'true'],
      ['Answers about Twente', 'false'],
    ])
    await pills[1]?.trigger('click')
    expect(donut.text()).toContain('3 of 6')
    expect(donut.text()).toContain('The target for the year is 60% of the answers about Twente.')
    expect(donut.get('a').attributes('href')).toBe('/ai-search/answers')
  })

  it('lists who the answers name, the most named first, with a page for each competitor', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search')
    const named = wrapper.get('section[aria-labelledby="named-title"]')
    expect(named.text()).toContain('Share of the 12 answers that name each company')
    expect(named.findAll('li').map((l) => l.text())).toEqual(['1Ashworth57%', '2Tribelt B.V.Target 40%24%', '3LesjöforsA group company of Tribelt, not a competitor10%'])
    expect(named.findAll('li a').map((a) => a.attributes('href'))).toEqual(['/competitors/ashworth'])
    expect(named.text()).toContain('the shares add up to more than 100%')
  })

  it('leads on to AI readiness by site, saying how Tribelt fares', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search')
    const card = wrapper.findAll('a[href="/ai-search/readiness"]').at(-1)
    if (!card) throw new Error('no card')
    expect(card.text()).toContain('AI readiness by site')
    expect(card.text()).toContain('Tribelt fails one of the eight checks. See which ones.')
  })

  it('says so before the first Prompt run', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/ai-search': () => json({ ...summary, trend: [], ranking: [] }), 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [promptRun({ cells: [] })] })) }), '/ai-search')
    await settle()
    expect(wrapper.get('h1').text()).toBe('AI assistants have not been asked about Tribelt yet.')
    expect(wrapper.text()).toContain('No Prompt run has finished yet.')
  })

  it('says when the summary or the results cannot be loaded', async () => {
    const summaryDown = await mountApp(api({ 'GET /api/v1/ai-search': () => problem(500, 'Internal Server Error', 'Down.') }), '/ai-search')
    expect(summaryDown.wrapper.get('[role="alert"]').text()).toContain('Down.')
    summaryDown.wrapper.unmount()
    const resultsDown = await mountApp(api({ 'GET /api/v1/prompt-test': () => problem(500, 'Internal Server Error', 'Gone.') }), '/ai-search')
    expect(resultsDown.wrapper.get('[role="alert"]').text()).toContain('Gone.')
  })
})
