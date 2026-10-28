import { describe, expect, it } from 'vitest'
import type { AiSearchSummary } from '../../../infrastructure/api/types.gen'
import { json, mountApp, problem, routes } from '../../../testing/mountApp'

const summary: AiSearchSummary = {
  goal: 0.4,
  followed: [
    { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt' },
    { competitorId: 'ashworth', name: 'Ashworth', role: 'competitor' },
  ],
  trend: [
    { runStartedAt: '2026-10-26T06:00:00Z', shares: [0.14, 0.62] },
    { runStartedAt: '2026-11-02T06:00:00Z', shares: [0.24, 0.57] },
  ],
  crawlers: [
    { bot: 'OAI-SearchBot', lab: 'OpenAI', gets: 'Links in ChatGPT search answers', feeds: 'today', access: 'allowed' },
    { bot: 'GPTBot', lab: 'OpenAI', gets: 'What future ChatGPT models know', feeds: 'future', access: 'blocked' },
  ],
  keepingOut: [{ competitorId: 'wire-belt', name: 'Wire Belt Company', blocked: ['GPTBot'] }],
  ranking: [
    { competitorId: 'martens', name: 'Märtens', role: 'competitor', score: 5 },
    { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', score: 3.5 },
  ],
  sites: 41,
  questions: 20,
  providers: 5,
  readinessChecks: 5,
  checks: [{ check: 'crawlers', scored: true, passing: 30, tribelt: true }],
}

describe('the AI search summary', () => {
  it("shows Tribelt's share against the goal, from the data", async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/ai-search': () => json(summary) }), '/ai-search')
    expect(document.title).toBe('AI search · Findability Explorer')
    const share = wrapper.get('section[aria-labelledby="share-title"]')
    expect(share.text()).toContain('24%')
    expect(share.text()).toContain('of the answers in the newest run, up from 14% on 26 Oct')
    expect(share.get('svg').attributes('aria-label')).toContain('The goal is 40%.')
    expect(share.text()).toContain('Goal 40%')
    expect(share.get('a[href="/ai-search/answers"]').text()).toContain('Each Prompt run asks 5 assistants 20 buyer questions.')
  })

  it('lists the crawlers with what tribelt.nl does with each, and who keeps them out', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/ai-search': () => json(summary) }), '/ai-search')
    const rows = wrapper.findAll('section[aria-labelledby="labs-title"] tbody tr').map((r) => r.findAll('th, td').map((c) => c.text()))
    expect(rows).toEqual([
      ['OAI-SearchBotOpenAI', 'Links in ChatGPT search answers, shows up in answers today', 'Allowed'],
      ['GPTBotOpenAI', 'What future ChatGPT models know, known to future models', 'Blocked'],
    ])
    const labs = wrapper.get('section[aria-labelledby="labs-title"]').text()
    expect(labs).toContain('tribelt.nl keeps GPTBot out.')
    expect(labs).toContain('Wire Belt Company shuts some of them out.')
  })

  it('ranks Tribelt among the best-prepared, and leads on to AI readiness', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/ai-search': () => json(summary) }), '/ai-search')
    const ready = wrapper.get('section[aria-labelledby="ready-title"]')
    expect(ready.findAll('li').map((l) => l.text())).toEqual(['Märtens5', 'Tribelt B.V.3½'])
    expect(ready.get('a[href="/ai-search/readiness"]').text()).toContain('for all 41 sites.')
  })

  it('says so before the first run and scan', async () => {
    const empty = { ...summary, followed: [], trend: [], ranking: [], questions: 0, crawlers: [{ ...summary.crawlers[0], access: undefined }] }
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/ai-search': () => json(empty) }), '/ai-search')
    expect(wrapper.find('svg[role="img"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('No Prompt run has finished yet.')
    expect(wrapper.text()).toContain('No site has been scanned yet.')
    expect(wrapper.text()).toContain('once the first Prompt run is in.')
    expect(wrapper.text()).toContain('Not checked')
  })

  it('says when the summary cannot be loaded', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/ai-search': () => problem(500, 'Internal Server Error', 'Down.') }), '/ai-search')
    expect(wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
