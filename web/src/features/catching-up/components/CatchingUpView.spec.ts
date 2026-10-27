import { describe, expect, it } from 'vitest'
import { json, mountApp, problem, routes } from '../../../testing/mountApp'

const standings = {
  measures: [
    { measure: 'aiShareOfVoice', tribelt: 0.24, median: 0.17, leader: { competitorId: 'ashworth', name: 'Ashworth', value: 0.57 }, competitors: 38, trails: true, leads: false },
    { measure: 'termsUsed', tribelt: 38, median: 14, leader: { competitorId: 'martens', name: 'Märtens', value: 30 }, competitors: 20, trails: false, leads: true },
    { measure: 'sitemapPages', competitors: 0, trails: false, leads: false },
  ],
  trailing: 1,
}

describe('Catching up', () => {
  it('heads the page with the measures Tribelt trails on, and draws three labelled bars per measure', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/standings': () => json(standings) }), '/catching-up')
    expect(wrapper.get('h1').text()).toBe('Tribelt trails the leaders on one of two measures.')
    const figures = wrapper.findAll('figure')
    expect(figures.map((f) => f.get('figcaption').text())).toEqual([
      'How often AI answers name the company Ashworth is ahead.',
      'Search terms from the map used on the site Tribelt leads here.',
      'Pages in the sitemap Tribelt has no data here yet.',
    ])
    expect(figures[0]?.findAll('figcaption ~ div').map((d) => d.text())).toEqual(['Tribelt24%', 'Typical competitor17%', 'Ashworth57%'])
    expect(figures[1]?.text()).toContain('Märtens30')
    expect(wrapper.get('ul[aria-label="Legend"]').text()).toBe('TribeltTypical competitorThe leader')
  })

  it('says when the comparison cannot be loaded', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/standings': () => problem(500, 'Internal Server Error', 'Down.') }), '/catching-up')
    expect(wrapper.get('h1').text()).toBe('Where Tribelt trails the leaders.')
    expect(wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
