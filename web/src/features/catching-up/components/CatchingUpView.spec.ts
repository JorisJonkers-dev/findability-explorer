import { describe, expect, it } from 'vitest'
import { details } from '../../../testing/fixtures'
import { json, mountApp, problem, routes } from '../../../testing/mountApp'

const standings = {
  measures: [
    { measure: 'aiShareOfVoice', tribelt: 0.24, median: 0.17, leader: { competitorId: 'ashworth', name: 'Ashworth', value: 0.57 }, competitors: 38, trails: true, leads: false },
    { measure: 'termsUsed', tribelt: 38, median: 14, leader: { competitorId: 'martens', name: 'Märtens', value: 30 }, competitors: 20, trails: false, leads: true },
    { measure: 'sitemapCoverage', competitors: 0, trails: false, leads: false },
  ],
  trailing: 1,
  open: [
    { id: 'sitemap-per-language', title: 'A sitemap for each language', moves: 'Helps search engines find every page.', measure: 'sitemapCoverage', auto: true, done: false, ...details, leader: { competitorId: 'ashworth', name: 'Ashworth', value: 0.9 } },
    { id: 'listings', title: 'Get listed where answers find sources', moves: 'Makes web answers cite tribelt.nl more often.', measure: 'citationShare', auto: false, done: false, ...details },
  ],
}

describe('Catching up', () => {
  it('heads the page with the measures Tribelt trails on, and draws three labelled bars per measure', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/standings': () => json(standings) }), '/catching-up')
    expect(wrapper.get('h1').text()).toBe('Tribelt trails the leaders on one of two measures.')
    const figures = wrapper.findAll('figure')
    expect(figures.map((f) => f.get('figcaption').text())).toEqual([
      'AI answers naming the company, in the newest Prompt run Ashworth is ahead.',
      'Search terms from the map found on the site Tribelt leads here.',
      "Share of the site's pages its sitemap lists Not measured on tribelt.nl yet. Scan it again.",
    ])
    expect(figures[0]?.findAll('figcaption ~ div').map((d) => d.text())).toEqual(['Tribelt24%', 'Typical competitor17%', 'Ashworth57%'])
    expect(figures[1]?.text()).toContain('Märtens30')
    expect(wrapper.get('ul[aria-label="Legend"]').text()).toBe('TribeltTypical competitorThe leader')
  })

  it('ends with the open improvements in order, each with its status and the competitor to look at', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/standings': () => json(standings) }), '/catching-up')
    const todo = wrapper.get('section[aria-labelledby="todo-title"]')
    expect(todo.findAll('li').map((l) => l.text())).toEqual([
      '1A sitemap for each languageHelps search engines find every page.Not measured on tribelt.nl yet. Scan it again. Ashworth leads with 90%.Look atAshworth',
      '2Get listed where answers find sourcesMakes web answers cite tribelt.nl more often.No Prompt run has finished yet.',
    ])
    expect(todo.get('li a').attributes('href')).toBe('/competitors/ashworth')
    wrapper.unmount()
    const all = await mountApp(routes({ 'GET /api/v1/standings': () => json({ ...standings, open: [] }) }), '/catching-up')
    expect(all.wrapper.text()).toContain('All planned improvements are done.')
  })

  it('says when the comparison cannot be loaded', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/standings': () => problem(500, 'Internal Server Error', 'Down.') }), '/catching-up')
    expect(wrapper.get('h1').text()).toBe('Where Tribelt trails the leaders.')
    expect(wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
