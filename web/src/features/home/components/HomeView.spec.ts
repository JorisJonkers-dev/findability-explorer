import { describe, expect, it } from 'vitest'
import { json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { overviewFixture, planFixture, standingsFixture } from '../../../testing/numbersFixtures'

const ok = {
  'GET /api/v1/plan': () => json(planFixture()),
  'GET /api/v1/standings': () => json(standingsFixture()),
  'GET /api/v1/overview': () => json(overviewFixture()),
}

describe('the Overview', () => {
  it('leads with the standings, then the numbers to watch, the sections and the visitors', async () => {
    const { wrapper } = await mountApp(routes(ok), '/')
    expect(document.title).toBe('Overview · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Tribelt trails the leaders on three of four measures.')
    const rows = wrapper.findAll('section[aria-labelledby="watch-title"] > ul:last-child > li a')
    expect(rows.map((a) => a.attributes('href'))).toEqual([
      '/numbers/aiShareOfVoice',
      '/numbers/twenteShareOfVoice',
      '/numbers/citationShare',
      '/numbers/readinessChecks',
      '/numbers/termsUsed',
    ])
    expect(rows[0]?.text()).toContain('AI answers that name Tribelt')
    expect(rows[0]?.text()).toContain('24%')
    expect(rows[0]?.text()).toContain('Target 40%')
    expect(rows[0]?.text()).toContain('Improved by recommendations 1 and 4')
    expect(rows[3]?.text()).toContain('3 of 5')
  })

  it('shows a card for each section, and a way to the Buyers page', async () => {
    const { wrapper } = await mountApp(routes(ok), '/')
    const cards = wrapper.findAll('section[aria-labelledby="sections-title"] li a')
    expect(cards.map((a) => a.attributes('href'))).toEqual(['/ai-search', '/competitors', '/terms', '/recommendations'])
    expect(cards[1]?.text()).toContain('24')
    expect(wrapper.get('a[href="/buyers"]').text()).toContain('See the visitor figures')
    expect(wrapper.get('footer').text()).toContain('Digital Strategy')
  })

  it('says when the numbers fail to load', async () => {
    const { wrapper } = await mountApp(routes({ ...ok, 'GET /api/v1/plan': () => problem(500, 'Internal Server Error', 'Down.') }), '/')
    expect(wrapper.get('[role="alert"]').text()).toContain('Down.')
    expect(wrapper.get('h1').text()).toBe('Overview')
  })

  it('opens a number from its row', async () => {
    const { wrapper, router } = await mountApp(routes(ok), '/')
    await wrapper.get('a[href="/numbers/readinessChecks"]').trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/numbers/readinessChecks')
  })
})
