import type { VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Term } from '../../../infrastructure/api/types.gen'
import { term, termSearch } from '../../../testing/fixtures'
import { json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { SEARCH_DELAY_MS } from '../composables/useTermView'

const MAP: Term[] = [
  term({ term: 'Eyelink belt', language: 'en', family: 'eyelink', tier: 'product', usedBy: ['Twentebelt', 'Ashworth', 'Tribelt'] }),
  term({ term: 'eye link belt', language: 'en-US', family: 'eyelink', tier: 'broad', usedBy: ['Ashworth'] }),
  term({ term: 'Drahtösengliederband', language: 'de', family: 'eyelink', tier: 'product', usedBy: ['Märtens', 'Tribelt (Drahtösengliederbänder)'] }),
  term({ term: 'AISI 304', language: 'de', family: 'spec_all', tier: 'spec', usedBy: ['Märtens', 'Twentebelt'] }),
  term({ term: 'belt tracking', language: 'en', family: 'all', tier: 'problem', usedBy: [] }),
]

const api = routes({ 'GET /api/v1/terms': () => json(termSearch(MAP, { total: MAP.length })) })

/** The terms in the ranked list, in order. */
const ranked = (wrapper: VueWrapper) =>
  wrapper.get('ol[aria-label="Ranked terms"]').findAll('li').map((li) => li.get('span.font-medium').text())
const status = (wrapper: VueWrapper) => wrapper.get('[aria-live="polite"]').text()
const button = (wrapper: VueWrapper, label: string) => wrapper.get(`button[aria-label^="${label}"]`)

afterEach(() => {
  vi.useRealTimers()
})

describe('the search terms view', () => {
  it('ranks the whole map, titled by its size', async () => {
    const { wrapper } = await mountApp(api, '/terms')
    expect(wrapper.get('h1').text()).toBe('5 ways to search for a metal belt.')
    expect(document.title).toBe('Search terms · Findability Explorer')
    expect(wrapper.text()).toContain('Terminology map, desk research')
    expect(ranked(wrapper)).toEqual(['AISI 304', 'Eyelink belt', 'Drahtösengliederband', 'eye link belt', 'belt tracking'])
    expect(status(wrapper)).toBe('5 of 5 terms')
    const first = wrapper.get('ol[aria-label="Ranked terms"] li')
    expect(first.text()).toContain('German · Any belt · Märtens, Twentebelt')
    expect(first.text()).toContain('Lacks it')
    expect(first.attributes('tabindex')).toBeUndefined()
    expect(wrapper.get('ol[aria-label="Ranked terms"]').attributes('tabindex')).toBe('0')
  })

  it('charts the languages and kinds, with what Tribelt uses', async () => {
    const { wrapper } = await mountApp(api, '/terms')
    const languages = wrapper.get('[role="group"][aria-label="By language"]').findAll('button')
    expect(languages.map((b) => b.attributes('aria-label'))).toEqual(['English: 3 terms, Tribelt uses 1', 'German: 2 terms, Tribelt uses 1'])
    expect(languages[0]?.findAll('[aria-hidden="true"] span[style]').map((s) => s.attributes('style'))).toEqual([
      'width: 33.33%;',
      'width: 66.67%;',
    ])
    // Above a bar, on hover: what Tribelt uses and what only others still have; at its end, a fraction.
    expect(languages[0]?.get('[data-testid="bar-info"]').text()).toBe('Tribelt uses 1 · only others use 2')
    const tiers = wrapper.get('[role="group"][aria-label="By kind of search"]').findAll('button')
    const ends = (b: (typeof tiers)[number]) => {
      const parts = b.findAll(':scope > span')
      return `${parts[0]?.text() ?? ''} ${parts[2]?.text() ?? ''}`
    }
    expect(tiers.map(ends)).toEqual(['Broad 0/1', 'Product 2/2', 'Application 0/0', 'Specification 0/1', 'Problem 0/1'])
    expect(wrapper.text()).toContain('2/5 terms are used by Tribelt')
  })

  it("filters the ranking by language from above it, and names Tribelt's term for each", async () => {
    const { wrapper, router } = await mountApp(api, '/terms')
    const pills = wrapper.get('[role="group"][aria-label="Language"]')
    expect(pills.findAll('button').map((b) => b.text())).toEqual(['All languages', 'English', 'German'])
    await pills.findAll('button')[2]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ language: 'de' })
    expect(ranked(wrapper)).toEqual(['AISI 304', 'Drahtösengliederband'])
    // The chart on the left agrees.
    expect(button(wrapper, 'German:').attributes('aria-pressed')).toBe('true')
    const cells = wrapper.get('ol[aria-label="Ranked terms"]').findAll('li').map((li) => li.text())
    expect(cells.some((t) => t.includes('Tribelt: Lacks it'))).toBe(true)
    await pills.findAll('button')[0]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({})
  })

  it('filters by a language and a kind, with chips to remove them', async () => {
    const { wrapper, router } = await mountApp(api, '/terms')
    await button(wrapper, 'German:').trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ language: 'de' })
    expect(button(wrapper, 'German:').attributes('aria-pressed')).toBe('true')
    expect(ranked(wrapper)).toEqual(['AISI 304', 'Drahtösengliederband'])
    await button(wrapper, 'Specification:').trigger('click')
    await settle()
    expect(ranked(wrapper)).toEqual(['AISI 304'])
    expect(status(wrapper)).toBe('1 of 5 terms')
    await button(wrapper, 'Remove the filter German').trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ tier: 'spec' })
    // Pressing a chart row again unsets it.
    await button(wrapper, 'Specification:').trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({})
    expect(wrapper.findAll('button[aria-label^="Remove the filter"]')).toHaveLength(0)
  })

  it('lists a company from the heatmap, Tribelt too', async () => {
    const { wrapper, router } = await mountApp(api, '/terms')
    const rows = wrapper.get('ul[aria-label="Terms per company and kind"]').findAll('button')
    expect(rows.map((b) => b.attributes('aria-label'))).toEqual([
      'Ashworth: 2 marketing terms, 0 technical terms',
      'Märtens: 1 marketing terms, 1 technical terms',
      'Tribelt: 2 marketing terms, 0 technical terms',
      'Twentebelt: 1 marketing terms, 1 technical terms',
    ])
    // A cell with terms is shaded; an empty one is grey and blank.
    const ashworth = rows[0]?.findAll('span.h-8')
    expect(ashworth?.[0]?.attributes('style')).toContain('background-color')
    expect(ashworth?.[2]?.text()).toBe('')
    await button(wrapper, 'Tribelt:').trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ user: 'Tribelt' })
    expect(button(wrapper, 'Tribelt:').attributes('aria-pressed')).toBe('true')
    expect(ranked(wrapper)).toEqual(['Eyelink belt', 'Drahtösengliederband'])
    await button(wrapper, 'Remove the filter Tribelt').trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({})
  })

  it('switches between marketing and technical terms, and shows what Tribelt lacks', async () => {
    const { wrapper, router } = await mountApp(api, '/terms')
    const pills = () => wrapper.get('[role="group"][aria-label="Terms"]').findAll('button')
    expect(pills().map((b) => [b.text(), b.attributes('aria-pressed')])).toEqual([
      ['All', 'true'],
      ['Marketing', 'false'],
      ['Technical', 'false'],
    ])
    await pills()[2]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ group: 'technical' })
    expect(ranked(wrapper)).toEqual(['AISI 304', 'belt tracking'])
    // Pressing the pressed one keeps it.
    await pills()[2]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ group: 'technical' })
    await pills()[1]?.trigger('click')
    await settle()
    expect(ranked(wrapper)).toEqual(['Eyelink belt', 'Drahtösengliederband', 'eye link belt'])
    await pills()[0]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({})

    const gap = wrapper.findAll('button').find((b) => b.text() === 'Only terms Tribelt lacks')
    await gap?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ gap: '1' })
    expect(gap?.attributes('aria-pressed')).toBe('true')
    expect(ranked(wrapper)).toEqual(['AISI 304', 'eye link belt', 'belt tracking'])
  })

  it('searches once typing pauses, and marks the match', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const { wrapper, router } = await mountApp(api, '/terms')
    await wrapper.get('input#terms-q').setValue('drahtösen')
    vi.advanceTimersByTime(SEARCH_DELAY_MS)
    await settle()
    expect(router.currentRoute.value.query).toEqual({ q: 'drahtösen' })
    expect(ranked(wrapper)).toEqual(['Drahtösengliederband'])
    expect(wrapper.get('ol[aria-label="Ranked terms"] mark').text()).toBe('Drahtösen')
  })

  it("keeps the old terminology page's links working", async () => {
    const { wrapper, router } = await mountApp(api, '/terminology?q=AISI&user=Twentebelt%20(TwenteFlex)')
    expect(router.currentRoute.value.path).toBe('/terms')
    expect(router.currentRoute.value.query).toEqual({ q: 'AISI', user: 'Twentebelt (TwenteFlex)' })
    expect((wrapper.get('input#terms-q').element as HTMLInputElement).value).toBe('AISI')
    expect(ranked(wrapper)).toEqual(['AISI 304'])
    expect(button(wrapper, 'Twentebelt:').attributes('aria-pressed')).toBe('true')
  })

  it('says it is loading, and when the map cannot be loaded', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/terms': () => new Promise<Response>(() => undefined) }), '/terms')
    expect(wrapper.get('h1').text()).toBe('Ways to search for a metal belt.')
    expect(status(wrapper)).toBe('Loading…')
    wrapper.unmount()
    const failing = await mountApp(routes({ 'GET /api/v1/terms': () => problem(500, 'Internal Server Error', 'The map is not loaded.') }), '/terms')
    expect(failing.wrapper.get('[role="alert"]').text()).toBe('The map is not loaded.')
  })
})
