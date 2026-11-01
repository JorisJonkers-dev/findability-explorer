import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { overviewRoutes } from '../../../testing/homeFixtures'
import { mountApp, problem, routes, settle } from '../../../testing/mountApp'

/** The section under the heading title. */
function region(w: VueWrapper, title: string) {
  const found = w.findAll('section').find((s) => s.find('h2').exists() && s.get('h2').text() === title)
  if (!found) throw new Error(`no section ${title}`)
  return found
}

describe('a goal page', () => {
  it('names the goal, says what it measures and who is ahead, with the way back', async () => {
    const { wrapper } = await mountApp(routes(overviewRoutes({}, true)), '/goals/aiShareOfVoice')
    expect(wrapper.get('h1').text()).toBe('AI responses that name Tribelt')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe("The share of a Prompt run's responses that name Tribelt.")
    expect(wrapper.get('[data-part="data-line"]').text()).toMatch(/^From the Prompt run of 1 November 2026: 90 questions to 2 models/)
    expect(wrapper.get('[data-part="summary"]').text()).toBe('Ashworth is ahead of Tribelt, with 57% against 24%. The goal is 40%.')
    expect(wrapper.findAll('a').find((a) => a.text().includes('Back to Overview'))?.attributes('href')).toBe('/')
    expect(wrapper.findAll('a').find((a) => a.text() === 'Set the goals')?.attributes('href')).toBe('/manage/recommendations/goals')
  })

  it('sets Tribelt against the competitors beside Over time, drawn from the history with the Prompt set filter', async () => {
    const { wrapper } = await mountApp(routes(overviewRoutes()), '/goals/aiShareOfVoice')
    const compare = region(wrapper, 'Tribelt against the competitors')
    expect(compare.get('h2').text()).toBe('Tribelt against the competitors')
    expect(compare.text()).toContain('The typical competitor is the middle figure across all competitors on the list.')
    expect(compare.get('[role="img"]').attributes('aria-label')).toBe(
      'AI responses that name Tribelt. Tribelt: 24%. The typical competitor: 10%. Ashworth, the best competitor: 57%. The goal: 40%.',
    )
    const time = region(wrapper, 'Over time')
    expect(time.text()).toContain('One point for each Prompt run. A dashed ring marks a partial run.')
    expect(time.findAll('[role="group"][aria-label="Prompt set"] button').map((b) => b.text())).toEqual(['All versions', 'Version 2, current', 'Version 1'])
    expect(time.findAll('circle[data-part="dot"]')).toHaveLength(2)
    expect(time.findAll('[data-part="end"] text').map((t) => t.text())).toEqual(['Tribelt 25%', 'Goal 40%'])
    expect(time.get('[data-part="notes"]').text()).toContain('a partial run of version 1')
  })

  it('breaks the goal down by model, scope, and belt family, and lists what improves it', async () => {
    const { wrapper } = await mountApp(routes(overviewRoutes()), '/goals/aiShareOfVoice')
    const broken = region(wrapper, 'Broken down')
    expect(broken.findAll('h3').map((h) => h.text())).toEqual(['By model', 'By scope', 'By belt family'])
    expect(broken.get('a[href="/ai-responses/all"]').text()).toBe('See every response')
    const improves = region(wrapper, 'What improves it')
    expect(improves.findAll('a').map((a) => [a.attributes('href'), a.text()])).toEqual([
      ['/recommendations/submit-search-engines', 'Submit tribelt.nl to Google and Bing'],
      ['/recommendations/answer-questions', 'Answer buyers questions'],
    ])
    expect(improves.text()).toContain('Benefit 3, effort 2')
  })

  it('lists the readiness checks, without a line over time yet', async () => {
    const { wrapper } = await mountApp(routes(overviewRoutes()), '/goals/readinessChecks')
    expect(wrapper.get('h1').text()).toBe('AI readiness checks passed')
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('From the newest scan of each site.')
    const time = region(wrapper, 'Over time')
    expect(time.text()).toContain('Only the newest scan of tribelt.nl is kept for this goal, so there is no line yet.')
    expect(time.find('[aria-label="Prompt set"]').exists()).toBe(false)
    const broken = region(wrapper, 'Broken down')
    expect(broken.findAll('li').map((l) => l.text())).toEqual([
      expect.stringMatching(/^✓PassedAI crawlers let in/),
      expect.stringMatching(/^✕Not passedllms\.txt/),
      expect.stringMatching(/^\?Not knownMarkdown pages/),
    ])
    expect(region(wrapper, 'What improves it').text()).toContain('No open recommendation aims at this goal.')
  })

  it('counts the search terms tribelt.nl uses per language', async () => {
    const { wrapper } = await mountApp(routes(overviewRoutes()), '/goals/termsUsed')
    expect(region(wrapper, 'Broken down').get('[role="img"]').attributes('aria-label')).toBe('Search terms tribelt.nl uses, per language. English, of 200: 20.')
  })

  it('says when a breakdown cannot be loaded, when there is no such goal, and when the goals fail', async () => {
    const runs = await mountApp(routes(overviewRoutes({ 'GET /api/v1/prompt-test': () => problem(500, 'Internal Server Error', 'Runs down.') })), '/goals/citationShare')
    expect(region(runs.wrapper, 'Broken down').get('[role="alert"]').text()).toContain('Runs down.')
    runs.wrapper.unmount()
    const none = await mountApp(routes(overviewRoutes()), '/goals/nothing')
    expect(none.wrapper.text()).toContain('There is no goal at this address. Pick one from the Overview.')
    none.wrapper.unmount()
    const down = await mountApp(routes(overviewRoutes({ 'GET /api/v1/plan': () => problem(500, 'Internal Server Error', 'Down.') })), '/goals/aiShareOfVoice')
    await settle()
    expect(down.wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
