import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { TermSearch } from '../../../infrastructure/api/types.gen'
import { term, termSearch } from '../../../testing/fixtures'
import { json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { SEARCH_DELAY_MS } from '../composables/useTermFilters'

const terms = [
  term({ term: 'spiral belt / spiral conveyor belt', usedBy: ['Twentebelt (TwenteFlex)', 'Ashworth'] }),
  term({ term: 'Spiralgurte', language: 'de', family: 'open_spiral', tier: 'spec', usedBy: ['Wire Belt Osterloh'] }),
]

/** Answers every search with result, and the whole map with 146 terms. */
function api(result: (search: URLSearchParams) => TermSearch | Response = () => termSearch(terms, { total: 24 })) {
  return routes({
    'GET /api/v1/terms': (request) => {
      const search = new URL(request.url).searchParams
      if ([...search.keys()].join() === 'limit') return json(termSearch(terms, { total: 146 }))
      const answer = result(search)
      return answer instanceof Response ? answer : json(answer)
    },
    'GET /api/v1/analyses': () => json({ items: [] }),
  })
}

describe('the terminology page', () => {
  it('counts the terms that match of all in the map', async () => {
    const { wrapper } = await mountApp(api(), '/terminology?q=spiral')
    expect(wrapper.get('h1').text()).toBe('Terminology')
    expect(wrapper.text()).toContain('24 of 146 terms')
    expect(wrapper.get('caption').text()).toBe('Terms matching spiral')
  })

  it('shows each term with its tier, the match marked, its language, family and users', async () => {
    const { wrapper } = await mountApp(api(), '/terminology?q=spiral')
    const rows = wrapper.findAll('[role="region"][aria-label="Matching terms"] tbody tr')
    expect(rows).toHaveLength(2)
    expect(rows[0]?.findAll('mark').map((m) => m.text())).toEqual(['spiral', 'spiral'])
    expect(rows[0]?.text()).toContain('(Product)')
    const cells = rows[1]?.findAll('td') ?? []
    expect(cells[1]?.text()).toBe('German')
    expect(cells[2]?.text()).toBe('Open spiral')
    expect(rows[0]?.findAll('button').map((b) => b.text())).toEqual(['Twentebelt (TwenteFlex)', 'Ashworth'])
  })

  it('picks a company by its button, and unpicks it the same way', async () => {
    const { wrapper, router } = await mountApp(api(), '/terminology?q=TwenteFlex')
    await wrapper.get('tbody button').trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ q: 'TwenteFlex', user: 'Twentebelt' })
    expect(wrapper.text()).toContain('Terms used by Twentebelt.')
    expect(wrapper.get('tbody button').attributes('aria-pressed')).toBe('true')
    await wrapper.get('tbody button').trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ q: 'TwenteFlex' })
  })

  it('shows every company again, or clears every filter', async () => {
    const { wrapper, router } = await mountApp(api(), '/terminology?user=Ashworth&tier=spec')
    const button = (text: string) => wrapper.findAll('button').find((b) => b.text() === text)
    await button('Show every company')?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ tier: 'spec' })
    await button('Clear filters')?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({})
    expect(wrapper.text()).not.toContain('Clear filters')
    expect(wrapper.get('caption').text()).toBe('Terms in the terminology map')
  })

  it('filters by kind of search, language and family, with their counts', async () => {
    const facets = {
      languages: [
        { value: 'de', count: 10 },
        { value: 'en', count: 9 },
      ],
      tiers: [{ value: 'product', count: 24 }],
      families: [{ value: 'eyelink', count: 7 }],
    }
    const { wrapper, router } = await mountApp(api(() => termSearch(terms, { total: 24, facets })), '/terminology?q=belt')
    const tiers = wrapper.get('[role="group"][aria-label="Kind of search"]')
    expect(tiers.findAll('button').map((b) => b.text())).toEqual([
      'Specification 0',
      'Product 24',
      'Application 0',
      'Problem 0',
      'Broad 0',
    ])
    await tiers.findAll('button')[1]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ q: 'belt', tier: 'product' })
    expect(tiers.findAll('button')[1]?.attributes('aria-pressed')).toBe('true')
    expect(tiers.findAll('button')[1]?.classes()).toContain('border-tier-product')

    const languages = wrapper.get('[role="group"][aria-label="Language"]')
    expect(languages.findAll('button').map((b) => b.text())).toEqual(['All languages 19', 'German 10', 'English 9'])
    await languages.findAll('button')[1]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ q: 'belt', tier: 'product', language: 'de' })
    await languages.findAll('button')[0]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ q: 'belt', tier: 'product' })

    const family = wrapper.get('select')
    expect(family.findAll('option')[1]?.text()).toBe('Eyelink (7)')
    await family.setValue('eyelink')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ q: 'belt', tier: 'product', family: 'eyelink' })
  })

  it('says when nothing matches, or the search failed', async () => {
    const none = await mountApp(api(() => termSearch([])), '/terminology?q=zzz')
    expect(none.wrapper.text()).toContain('No terms match. Try a shorter search or fewer filters.')
    expect(none.wrapper.text()).toContain('No terms match')
    none.wrapper.unmount()
    const failing = await mountApp(api(() => problem(400, 'Bad Request', 'The query is too long.')), '/terminology?q=x')
    expect(failing.wrapper.get('[role="alert"]').text()).toBe('The query is too long.')
  })

  it('says it is searching before the first answer', async () => {
    const { wrapper } = await mountApp(
      routes({ 'GET /api/v1/terms': () => new Promise<Response>(() => undefined) }),
      '/terminology',
    )
    expect(wrapper.text()).toContain('Searching…')
  })
})

describe('typing a search', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('puts the search in the address once typing pauses', async () => {
    const { wrapper, router } = await mountApp(api(), '/terminology')
    await wrapper.get('input[type="search"]').setValue('TwenteFlex')
    vi.advanceTimersByTime(SEARCH_DELAY_MS)
    await settle()
    expect(router.currentRoute.value.query).toEqual({ q: 'TwenteFlex' })
  })
})
