import { describe, expect, it } from 'vitest'
import type { Plan } from '../../../infrastructure/api/types.gen'
import { competitor, details, me, overview, product, term, termSearch } from '../../../testing/fixtures'
import { json, mountApp, problem, routes } from '../../../testing/mountApp'

const plan = (): Plan => ({
  catalogueVersion: 1,
  improvements: [{ id: 'listings', title: 'Get listed', moves: 'citations', measure: 'citationShare', auto: false, done: false, ...details }],
  watched: [{ target: 'termsUsed', now: 4, goal: 20 }],
  eyeOn: [],
  readinessChecks: 8,
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

const TERMS = [
  term({ term: 'draadogenband', language: 'nl', family: 'eyelink', usedBy: ['Twentebelt', 'Ashworth'] }),
  term({ term: 'eyelink belt', language: 'en-US', family: 'eyelink', tribeltUses: true }),
  term({ term: 'Spiralband', language: 'de', family: 'open_spiral', usedBy: ['Twentebelt'] }),
  term({ term: 'mesh belt', language: 'en', family: 'all', usedBy: [] }),
]

const api = (extra = {}) =>
  routes({
    'GET /api/v1/overview': () => json(FIGURES),
    'GET /api/v1/plan': () => json(plan()),
    'GET /api/v1/terms': () => json(termSearch(TERMS)),
    'GET /api/v1/competitors': () => json({ items: [competitor('Twentebelt'), competitor('Ashworth')] }),
    'GET /api/v1/products': () => json({ items: [product(), product({ family: 'open_spiral', tribeltNames: [], termsUsed: 0, terms: [] })], standouts: [] }),
    ...extra,
  })

const rows = (table: ReturnType<Awaited<ReturnType<typeof mountApp>>['wrapper']['get']>) =>
  table.findAll('tbody tr, tfoot tr').map((r) => r.findAll('th, td').map((c) => c.text()))

describe('the Search terms page', () => {
  it('is in the frame, with its purpose, its data line, and its sections in order', async () => {
    const { wrapper } = await mountApp(api(), '/terms')
    expect(wrapper.get('h1').text()).toBe('Search terms')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe(
      'The words buyers use to look for metal conveyor belts, and how many of them tribelt.nl uses. Pages with these words are easier to find.',
    )
    expect(wrapper.get('[data-part="data-line"]').text()).toBe("From the search term list (4 terms) and each site's newest scan.")
    expect(wrapper.findAll('main h2, h2').map((h) => h.text())).toEqual([
      'Search term adoption over time',
      'By language',
      'By belt family',
      'Terms competitors use and tribelt.nl does not',
    ])
    expect(wrapper.get('details summary').text()).toBe('What buyer intent means')
    expect(wrapper.get('a[href="/help/terms"]').text()).toBe('How this is measured, in Help')
    expect(wrapper.find('a[href="/manage/terms"]').exists()).toBe(false)
  })

  it('gives an administrator the way to manage the list', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/me': () => json(me({ isAdmin: true })) }), '/terms')
    expect(wrapper.get('a[href="/manage/terms"]').text()).toBe('Manage the search term list')
  })

  it('shows three figures, the second against its goal', async () => {
    const { wrapper } = await mountApp(api(), '/terms')
    const figures = wrapper.get('[aria-label="Key figures"]').findAll(':scope > div').map((d) => d.text())
    expect(figures).toEqual([
      '10terms in the search term listIn English, German, and Dutch.',
      '4used on tribelt.nlGoal 20',
      '6used only by competitorsEach one is a search in which Tribelt does not turn up.',
    ])
  })

  it('says what adoption over time is still missing', async () => {
    const { wrapper } = await mountApp(api(), '/terms')
    expect(wrapper.get('section[aria-labelledby="adoption-title"]').text()).toContain(
      'The share over time follows once the app keeps which terms each scan of tribelt.nl found. Today it knows the newest scan only.',
    )
  })

  it('counts the terms by language and by belt family, each row opening its terms', async () => {
    const { wrapper } = await mountApp(api(), '/terms')
    const languages = wrapper.get('section[aria-labelledby="by-language-title"] table')
    expect(rows(languages)).toEqual([
      ['Dutch', '1', '0', '1'],
      ['German', '1', '0', '1'],
      ['English', '2', '1', '0'],
      ['All three languages', '4', '1', '2'],
    ])
    expect(languages.findAll('tbody a').map((a) => a.attributes('href'))).toEqual(['/terms/all?language=nl', '/terms/all?language=de', '/terms/all?language=en'])
    const families = wrapper.get('section[aria-labelledby="by-family-title"] table')
    expect(rows(families)[0]).toEqual(['Eyelink belts', 'Draadogenbanden', '2', '1'])
    expect(rows(families)[1]).toEqual(['Open spiral belts', 'None recorded', '1', '0'])
    expect(rows(families).at(-1)).toEqual(['All belt families', 'Not counted above: 1 term that applies to every belt, 0 of them used', '3', '1'])
    expect(families.get('tbody a').attributes('href')).toBe('/terms/families/eyelink')
  })

  it('folds what each kind of buyer intent means', async () => {
    const { wrapper } = await mountApp(api(), '/terms')
    const fold = wrapper.get('details')
    expect(fold.attributes('open')).toBeUndefined()
    expect(fold.findAll('li').map((l) => l.text())).toEqual([
      'Specification. The buyer already knows the exact belt and its sizes.',
      'Product. The buyer names the type of belt.',
      'Application. The buyer describes what the belt is for, such as a bakery oven.',
      'Problem. The buyer describes a problem to solve.',
      'Broad. A general search about conveyor belts.',
    ])
  })

  it('lists the terms competitors use and tribelt.nl does not, each opening its page', async () => {
    const { wrapper } = await mountApp(api(), '/terms')
    const top = wrapper.get('section[aria-labelledby="missing-title"]')
    expect(top.text()).toContain('The 15 terms that the most competitors use.')
    expect(rows(top.get('table'))).toEqual([
      ['draadogenband', 'Dutch', 'Eyelink belts', 'Product', '2'],
      ['Spiralband', 'German', 'Open spiral belts', 'Product', '1'],
    ])
    expect(top.findAll('tbody a').map((a) => a.attributes('href'))).toEqual(['/terms/nl--draadogenband', '/terms/de--spiralband'])
    expect(top.get('a[href="/terms/all"]').text()).toBe('See every term')
  })

  it('shows no goal when the plan holds none, and says when the families cannot be loaded', async () => {
    const { wrapper } = await mountApp(
      api({ 'GET /api/v1/plan': () => json({ ...plan(), watched: [] }), 'GET /api/v1/products': () => problem(500, 'Internal Server Error', 'Down.') }),
      '/terms',
    )
    expect(wrapper.text()).not.toContain('Goal 20')
    expect(wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
