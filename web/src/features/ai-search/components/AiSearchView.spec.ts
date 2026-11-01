import { describe, expect, it } from 'vitest'
import type { AiSearchSummary, Plan } from '../../../infrastructure/api/types.gen'
import { promptResults, promptRun } from '../../../testing/fixtures'
import { promptHistory } from '../../../testing/historyFixtures'
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
    'GET /api/v1/prompt-test/history': () => json(promptHistory()),
    ...extra,
  })

describe('the AI search page', () => {
  it('heads the page with where Tribelt ranks among the companies the answers name, and says what the newest run asked', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    expect(document.title).toBe('AI responses · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Tribelt ranks second among the companies AI answers name.')
    expect(wrapper.text()).toContain("In the newest Prompt run, AI assistants from two providers gave 12 answers to buyers' questions. OpenAI names Tribelt most often, in 3 of its 6 answers.")
  })

  it('shows three key figures that lead to the numbers and to AI readiness', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    const figures = wrapper.findAll('section[aria-label="Key figures"] a')
    expect(figures.map((f) => [f.attributes('href'), f.text()])).toEqual([
      ['/goals/aiShareOfVoice', '25%of AI answers name TribeltTarget 50%.'],
      ['/goals/citationShare', '17%of web answers cite tribelt.nlTarget 50%.'],
      ['/ai-readiness', '3½ of 8AI readiness checks passedSee how AI readiness is checked.'],
    ])
  })

  it('shows the share of answers naming Tribelt, for all answers or those about Twente', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
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
    expect(donut.get('a').attributes('href')).toBe('/ai-responses/all')
  })

  it('lists who the answers name, the most named first, with a page for each competitor', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    const named = wrapper.get('section[aria-labelledby="named-title"]')
    expect(named.text()).toContain('Share of the 12 answers that name each company')
    expect(named.findAll('li').map((l) => l.text())).toEqual(['1Ashworth57%', '2Tribelt B.V.Target 40%24%', '3LesjöforsA group company of Tribelt, not a competitor10%'])
    expect(named.findAll('li a').map((a) => a.attributes('href'))).toEqual(['/competitors/ashworth'])
    expect(named.text()).toContain('the shares add up to more than 100%')
  })

  it('leads on to AI readiness by site, saying how Tribelt fares', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    const card = wrapper.findAll('a[href="/ai-readiness"]').at(-1)
    if (!card) throw new Error('no card')
    expect(card.text()).toContain('AI readiness by site')
    expect(card.text()).toContain('Tribelt fails one of the eight checks. See which ones.')
  })

  it('says so before the first Prompt run', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/ai-search': () => json({ ...summary, trend: [], ranking: [] }), 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [promptRun({ cells: [] })] })) }), '/ai-responses')
    await settle()
    expect(wrapper.get('h1').text()).toBe('AI assistants have not been asked about Tribelt yet.')
    expect(wrapper.text()).toContain('No Prompt run has finished yet.')
  })

  it('says when the summary or the results cannot be loaded', async () => {
    const summaryDown = await mountApp(api({ 'GET /api/v1/ai-search': () => problem(500, 'Internal Server Error', 'Down.') }), '/ai-responses')
    expect(summaryDown.wrapper.get('[role="alert"]').text()).toContain('Down.')
    summaryDown.wrapper.unmount()
    const resultsDown = await mountApp(api({ 'GET /api/v1/prompt-test': () => problem(500, 'Internal Server Error', 'Gone.') }), '/ai-responses')
    expect(resultsDown.wrapper.get('[role="alert"]').text()).toContain('Gone.')
  })
})

describe('Over time on the AI responses page', () => {
  const BREAK_NOTE = '27 October 2026: The questions changed here (prompt set version 2), so the figures before and after are not the same measure.'
  const PARTIAL_NOTE =
    '27 October 2026, 06:03: a partial run of version 1. Only 62 of its 1,260 requests got a response (the rest failed because the OpenRouter key was refused), and 47 of those could be read, so its figure rests on few responses.'

  it('shows every version by default: a line per language, the goal, the break, the partial run, and the figures as a table', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    const section = wrapper.get('section[aria-labelledby="over-time-title"]')
    expect(section.get('h2').text()).toBe('Over time')
    expect(section.text()).toContain('How often AI responses name Tribelt in each Prompt run. The lines show each language the questions were asked in. Twente counts as Dutch.')
    const buttons = section.findAll('[role="group"][aria-label="Prompt set"] button')
    expect(buttons.map((b) => [b.text(), b.attributes('aria-pressed')])).toEqual([
      ['All versions', 'true'],
      ['Version 2, current', 'false'],
      ['Version 1', 'false'],
    ])
    expect(section.text()).toContain('The questions changed with each version, so the line breaks between versions. Version 1 asked 90 questions, and version 2 asks 20 questions.')
    expect(section.get('[data-part="legend"]').text()).toBe('AllDutchGermanEnglishGoalA dashed ring marks a partial run.')
    expect(section.findAll('[data-part="end"] text').map((t) => t.text())).toEqual(['All 25%', 'Dutch 41%', 'English 10%', 'Goal 50%'])
    expect(section.get('[data-part="band"]').text()).toBe('Prompt set version 2')
    expect(section.findAll('[data-part="notes"] li').map((l) => l.text())).toEqual([BREAK_NOTE, PARTIAL_NOTE])
    const table = section.get('details')
    expect(table.get('summary').text()).toBe('Show the figures as a table')
    expect(table.findAll('th[scope="col"]').map((t) => t.text())).toEqual(['Date', 'All', 'Dutch', 'German', 'English'])
    expect(table.findAll('tbody tr').map((r) => r.findAll('th, td').map((c) => c.text()))).toEqual([
      ['27 October 2026, 06:03 (partial run)', '40%', '48%', '', ''],
      ['27 October 2026, 08:36', '25%', '41%', '', '10%'],
    ])
  })

  it('keeps the choice of version in the address, and says what that version asks', async () => {
    const { wrapper, router } = await mountApp(api(), '/ai-responses')
    const section = wrapper.get('section[aria-labelledby="over-time-title"]')
    await section.findAll('[role="group"][aria-label="Prompt set"] button')[2]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/ai-responses?version=1')
    expect(section.get('[aria-pressed="true"]').text()).toBe('Version 1')
    expect(section.text()).toContain(
      'Version 1, published 26 October 2026: 90 questions. A supplier, an application, and a problem question for each belt family, in six scopes. It has one run so far, so there is no line yet.',
    )
    expect(section.findAll('[data-part="notes"] li').map((l) => l.text())).toEqual([PARTIAL_NOTE])
    expect(section.find('[data-part="band"]').exists()).toBe(false)
    expect(section.findAll('polyline')).toHaveLength(0)
  })

  it('opens at the version the address names', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses?version=2')
    const section = wrapper.get('section[aria-labelledby="over-time-title"]')
    expect(section.get('[aria-pressed="true"]').text()).toBe('Version 2, current')
    expect(section.text()).toContain('Version 2, published 27 October 2026: 20 questions.')
    expect(section.find('[data-part="notes"]').exists()).toBe(false)
  })

  it('says so before a run has a figure, and when the history cannot be loaded', async () => {
    const empty = await mountApp(api({ 'GET /api/v1/prompt-test/history': () => json(promptHistory({ runs: [] })) }), '/ai-responses')
    expect(empty.wrapper.get('section[aria-labelledby="over-time-title"]').text()).toContain('No complete runs yet.')
    empty.wrapper.unmount()
    const down = await mountApp(api({ 'GET /api/v1/prompt-test/history': () => problem(500, 'Internal Server Error', 'History down.') }), '/ai-responses')
    expect(down.wrapper.get('section[aria-labelledby="over-time-title"] [role="alert"]').text()).toContain('History down.')
  })
})
