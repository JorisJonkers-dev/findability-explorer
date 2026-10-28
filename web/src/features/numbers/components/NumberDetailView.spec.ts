import { describe, expect, it } from 'vitest'
import { json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { planFixture, standingsFixture } from '../../../testing/numbersFixtures'

const ok = { 'GET /api/v1/plan': () => json(planFixture()), 'GET /api/v1/standings': () => json(standingsFixture()) }

describe('the page of a number', () => {
  it('shows the headline, the key figures, the comparison and what improves it', async () => {
    const { wrapper } = await mountApp(routes(ok), '/numbers/aiShareOfVoice')
    expect(wrapper.get('h1').text()).toBe("Ashworth is ahead with 57%, against Tribelt's 24%.")
    expect(wrapper.findAll('a').some((a) => a.text() === 'Back to Overview')).toBe(true)
    expect(wrapper.text()).toContain("This number is the share of the newest Prompt run's answers that name Tribelt at least once.")
    expect(wrapper.findAll('section[aria-label="Key figures"] > div').map((d) => d.text())).toEqual([
      '24%Tribelt nowTribelt is 60% of the way there.',
      '57%Ashworth, the leaderThe competitor with the best value now.',
      '40%Target for the yearThe goal for this number by the end of the year.',
    ])
    expect(wrapper.get('#compare-title').text()).toBe('Tribelt against the field')
    expect(wrapper.text()).toContain('The typical competitor is the middle value over the four competitors with data.')
    const improves = wrapper.findAll('section[aria-labelledby="improves-title"] li a')
    expect(improves.map((a) => a.attributes('href'))).toEqual(['/recommendations/submit-search-engines', '/recommendations/answer-questions'])
    expect(improves[0]?.text()).toContain('Submit tribelt.nl to Google and Bing')
    expect(wrapper.get('a[href="/competitors/ashworth"]').text()).toBe('See what Ashworth does differently')
  })

  it('says when nothing improves a number, and marks a done improvement', async () => {
    const { wrapper } = await mountApp(routes(ok), '/numbers/termsUsed')
    expect(wrapper.text()).toContain('No recommendation improves this number yet.')
    const readiness = await mountApp(routes(ok), '/numbers/readinessChecks')
    expect(readiness.wrapper.get('section[aria-labelledby="improves-title"] li').text()).toContain('Done. Improves AI readiness.')
  })

  it('goes back to the Overview', async () => {
    const { wrapper, router } = await mountApp(routes(ok), '/numbers/aiShareOfVoice')
    await wrapper.findAll('a').find((a) => a.text() === 'Back to Overview')?.trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/')
  })

  it('says so at an address that is not a number', async () => {
    const { wrapper } = await mountApp(routes(ok), '/numbers/nothing')
    expect(wrapper.get('h1').text()).toBe('Number to watch')
    expect(wrapper.text()).toContain('There is no number to watch at this address.')
  })

  it('says when the numbers fail to load', async () => {
    const { wrapper } = await mountApp(routes({ ...ok, 'GET /api/v1/standings': () => problem(500, 'Internal Server Error', 'Down.') }), '/numbers/aiShareOfVoice')
    expect(wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
