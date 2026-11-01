import { describe, expect, it } from 'vitest'
import type { Plan } from '../../../infrastructure/api/types.gen'
import { competitor, details, me, overview, term, termSearch } from '../../../testing/fixtures'
import { json, mountApp, routes } from '../../../testing/mountApp'

const plan = (): Plan => ({
  catalogueVersion: 1,
  improvements: [
    { id: 'names', title: 'Name the competitor equivalents', moves: 'terms', measure: 'termsUsed', auto: false, done: false, ...details },
    { id: 'listings', title: 'Get listed', moves: 'citations', measure: 'citationShare', auto: false, done: false, ...details },
  ],
  watched: [],
  eyeOn: [],
  readinessChecks: 8,
})

const TERMS = [
  term({ term: 'Spiralband', language: 'de', family: 'open_spiral', usedBy: ['Twentebelt', 'Ashworth'], source: 'https://www.twentebelt.com/de/' }),
  term({ term: 'Spiralgurt', language: 'de', family: 'open_spiral', usedBy: ['Twentebelt'], tribeltUses: true, source: '' }),
  term({ term: 'Gliederband', language: 'de', family: 'open_spiral', usedBy: [] }),
  term({ term: 'spiral belt', language: 'en', family: 'open_spiral' }),
]

const PAGES = [
  { competitorId: 'twentebelt', name: 'Twentebelt', role: 'competitor', pages: [{ url: 'https://www.twentebelt.com/de/spiral', title: 'Spiralbänder' }] },
  { competitorId: 'ashworth', name: 'Ashworth', role: 'competitor', pages: [] },
]

const api = (extra = {}) =>
  routes({
    'GET /api/v1/term-pages': () => json({ items: PAGES }),
    'GET /api/v1/overview': () => json(overview({ competitors: { makers: 12, families: [] } })),
    'GET /api/v1/plan': () => json(plan()),
    'GET /api/v1/terms': () => json(termSearch(TERMS)),
    'GET /api/v1/competitors': () => json({ items: [competitor('Twentebelt'), competitor('Ashworth')] }),
    ...extra,
  })

describe('one search term', () => {
  it('says what it is in a sentence, in the frame, with its sections in two columns', async () => {
    const { wrapper } = await mountApp(api(), '/terms/de--spiralband')
    expect(wrapper.get('h1').text()).toBe('Spiralband')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe(
      'A German term for open spiral belts. 2 of the 12 competitors use it, and tribelt.nl does not. Its buyer intent is product. The buyer names the type of belt.',
    )
    expect(wrapper.get('[data-part="data-line"]').text()).toBe("From the search term list and each site's newest scan.")
    expect(wrapper.get('a[href="/terms/all"]').text()).toBe('Back to Every term')
    expect(wrapper.findAll('h2').map((h) => h.text())).toEqual(['Where competitors use it', 'On tribelt.nl', 'Related terms', 'The recommendation'])
    expect(wrapper.get('[data-part="columns"]').classes()).toContain('min-[1100px]:grid-cols-2')
    expect(wrapper.find('a[href="/help/terms"]').exists()).toBe(true)
  })

  it('lists, side by side, the pages where competitors use it and where tribelt.nl does', async () => {
    const { wrapper, requests } = await mountApp(api(), '/terms/de--spiralband')
    expect(requests()).toContain('GET /api/v1/term-pages?term=Spiralband&language=de')
    const where = wrapper.get('section[aria-labelledby="term-pages-competitors"]')
    expect(where.findAll('a[href^="/competitors/"]').map((a) => a.attributes('href'))).toEqual(['/competitors/twentebelt', '/competitors/ashworth'])
    expect(where.get('a[href="https://www.twentebelt.com/de/spiral"]').text()).toBe('Spiralbänder')
    expect(wrapper.get('section[aria-labelledby="term-pages-tribelt"]').text()).toContain('tribelt.nl’s newest scan did not find this term.')
  })

  it('lists related terms of its family and language, and the recommendations for search terms', async () => {
    const { wrapper } = await mountApp(api(), '/terms/de--spiralband')
    const related = wrapper.get('section[aria-labelledby="related-title"]')
    expect(related.findAll('li').map((l) => l.text())).toEqual(['✓Used on tribelt.nlSpiralgurt1 competitor', '·Not used on tribelt.nlGliederband0 competitors'])
    expect(related.get('a').attributes('href')).toBe('/terms/de--spiralgurt')
    const rec = wrapper.get('section[aria-labelledby="recommendation-title"]')
    expect(rec.findAll('a').map((a) => [a.text(), a.attributes('href')])).toEqual([['Name the competitor equivalents', '/recommendations/names']])
  })

  it('shows the tribelt.nl pages that use a term, and an administrator may manage the list', async () => {
    const tribelt = { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', pages: [{ url: 'https://www.tribelt.nl/de/spiral', title: 'Spiralgurte' }] }
    const { wrapper } = await mountApp(
      api({ 'GET /api/v1/me': () => json(me({ isAdmin: true })), 'GET /api/v1/term-pages': () => json({ items: [tribelt] }) }),
      '/terms/de--spiralgurt',
    )
    expect(wrapper.get('section[aria-labelledby="term-pages-tribelt"] a').text()).toBe('Spiralgurte')
    expect(wrapper.get('a[href="/manage/terms"]').text()).toBe('Manage the search term list')
  })

  it('says when no competitor uses it, nothing is related, and no recommendation is open', async () => {
    const { wrapper } = await mountApp(
      api({
        'GET /api/v1/terms': () => json(termSearch([term({ term: 'solo', language: 'nl', usedBy: [], source: '' })])),
        'GET /api/v1/plan': () => json({ ...plan(), improvements: [] }),
        'GET /api/v1/term-pages': () => json({ items: [] }),
      }),
      '/terms/nl--solo',
    )
    expect(wrapper.text()).toContain('No competitor’s newest scan found this term.')
    expect(wrapper.text()).toContain('No other term of this belt family and language is in the list.')
    expect(wrapper.text()).toContain('No recommendation for search terms is open.')
  })

  it('says when the address names no term', async () => {
    const { wrapper } = await mountApp(api(), '/terms/xx--nothing')
    expect(wrapper.get('h1').text()).toBe('Search term')
    expect(wrapper.get('[role="alert"]').text()).toBe('There is no term with this address.')
  })
})
