import { describe, expect, it } from 'vitest'
import type { Plan } from '../../../infrastructure/api/types.gen'
import { competitor, details, overview, product, term, termSearch } from '../../../testing/fixtures'
import { json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const plan = (): Plan => ({
  catalogueVersion: 1,
  improvements: [{ id: 'listings', title: 'Get listed', moves: 'citations', measure: 'citationShare', auto: false, done: false, ...details }],
  watched: [{ target: 'termsUsed', now: 4, goal: 20 }],
  eyeOn: [],
  readinessChecks: 5,
})

const FIGURES = overview({
  terms: {
    total: 10,
    tribelt: 4,
    from: 'scan',
    languages: [
      { language: 'en', terms: 5, tribelt: 2 },
      { language: 'de', terms: 3, tribelt: 1 },
      { language: 'nl', terms: 2, tribelt: 1 },
    ],
    tiers: [],
  },
})

const api = (extra = {}) =>
  routes({
    'GET /api/v1/overview': () => json(FIGURES),
    'GET /api/v1/plan': () => json(plan()),
    'GET /api/v1/terms': () => json(termSearch([term({ language: 'en-US' }), term({ term: 'b', language: 'en' })])),
    'GET /api/v1/competitors': () => json({ items: [competitor('Twentebelt')] }),
    'GET /api/v1/products': () =>
      json({ items: [product(), product({ family: 'open_spiral', tribeltNames: [], termsUsed: 0, terms: [] })], standouts: [] }),
    ...extra,
  })

describe('the Search terms page', () => {
  it('heads itself with the size of the map and what tribelt.nl uses, with three figures', async () => {
    const { wrapper } = await mountApp(api(), '/terms')
    expect(wrapper.get('h1').text()).toBe('Competitors list belts using 10 terms, tribelt.nl uses only 4 of them.')
    const figures = wrapper.get('section[aria-label="Key figures"]').findAll(':scope > div').map((d) => d.text())
    expect(figures).toEqual([
      '10terms in the search term listIn English, German, and Dutch.',
      '4are used on tribelt.nlThe target is 20.',
      '6are used only by competitorsEach one is a search in which Tribelt does not turn up.',
    ])
  })

  it('is neutral before the figures load', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/overview': () => new Promise<Response>(() => undefined) }), '/terms')
    expect(wrapper.get('h1').text()).toBe('Ways to search for a metal belt.')
  })

  it('stresses the language with the most terms, and another when its legend button is pressed', async () => {
    const { wrapper } = await mountApp(api(), '/terms')
    const legend = () => wrapper.get('[role="group"][aria-label="Pick a language"]').findAll('button')
    expect(legend().map((b) => [b.attributes('aria-label'), b.attributes('aria-pressed')])).toEqual([
      ['English: 5 terms', 'true'],
      ['German: 3 terms', 'false'],
      ['Dutch: 2 terms', 'false'],
    ])
    expect(wrapper.get('svg[role="img"]').attributes('aria-label')).toBe('The 10 terms by language: English 5, German 3, Dutch 2.')
    expect(wrapper.text()).toContain('English, 50% of the list')
    expect(wrapper.text()).toContain('5 terms, 1 of them American English. tribelt.nl uses 2 of them.')
    await legend()[1]?.trigger('click')
    await settle()
    expect(legend().map((b) => b.attributes('aria-pressed'))).toEqual(['false', 'true', 'false'])
    expect(wrapper.text()).toContain('German, 30% of the list')
    expect(wrapper.text()).toContain('Germany is where Tribelt wants to grow.')
  })

  it('lists the belt families, each linking to its page, and links to every term', async () => {
    const { wrapper } = await mountApp(api(), '/terms')
    const families = wrapper.get('ul[aria-label="Belt families"]').findAll('a')
    expect(families.map((a) => a.attributes('href'))).toEqual(['/terms/families/eyelink', '/terms/families/open-spiral'])
    expect(families[0]?.text()).toBe('EyelinkTribelt calls it Draadogenbanden1 of 4terms tribelt.nl uses')
    expect(families[1]?.text()).toContain('Tribelt names none for it yet')
    expect(wrapper.get('a[href="/terms/all"]').text()).toBe('its own page')
  })

  it('shows no target when the plan holds none, and says when the families cannot be loaded', async () => {
    const { wrapper } = await mountApp(
      api({ 'GET /api/v1/plan': () => json({ ...plan(), watched: [] }), 'GET /api/v1/products': () => problem(500, 'Internal Server Error', 'Down.') }),
      '/terms',
    )
    expect(wrapper.text()).not.toContain('The target is')
    expect(wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
