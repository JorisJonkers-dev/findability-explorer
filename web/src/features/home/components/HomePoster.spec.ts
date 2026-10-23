import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it, onTestFinished } from 'vitest'
import { competitor, term, termSearch } from '../../../testing/fixtures'
import { json, mountApp, routes, settle } from '../../../testing/mountApp'

const api = routes({
  'GET /api/v1/competitors': () =>
    json({
      items: [
        competitor('Twentebelt', { beltNames: [{ name: 'Eyelink belts', family: 'eyelink' }] }),
        competitor('Märtens', { beltNames: [] }),
        competitor('Tribelt B.V.', { groupCompany: true }),
      ],
    }),
  'GET /api/v1/terms': () =>
    json(termSearch([term({ language: 'en', usedBy: ['Tribelt'] }), term({ language: 'de', usedBy: ['Märtens'] }), term({ language: 'en-US' })])),
})

const tiles = (wrapper: VueWrapper) => wrapper.findAll('nav[aria-label="Sections"] a')
const poster = (wrapper: VueWrapper) => wrapper.get('main > div > div > div')

describe('the poster', () => {
  it('says what the strategy is about, and credits the course', async () => {
    const { wrapper } = await mountApp(api, '/')
    expect(wrapper.get('h1').text()).toBe('One belt,five names.')
    expect(wrapper.text()).toContain('find Tribelt, on Google and in Gemini.')
    expect(wrapper.find('img[alt="A Tribelt employee at a belt production line"]').exists()).toBe(true)
    const footer = wrapper.get('footer')
    expect(footer.text()).toContain('A student project for Tribelt B.V. in Digital Strategy (201900119)')
    expect(footer.find('img[alt="Tribelt, part of Lesjöfors Group"]').exists()).toBe(true)
    expect(footer.find('img[alt="University of Twente"]').exists()).toBe(true)
    expect(footer.text()).toContain('Signed in as Joris Jonkers')
  })

  it('links to the five sections, each with its figure and line', async () => {
    const { wrapper } = await mountApp(api, '/')
    expect(tiles(wrapper).map((a) => [a.attributes('href'), a.attributes('aria-label')])).toEqual([
      ['/buyers', 'Buyers: 43× Organic visitors contact Tribelt 43 times more often than paid ones. Zoom in.'],
      ['/ai-search', 'AI search: 0 Googlebot visits to our mirror in more than three weeks. AI crawlers read every page. Zoom in.'],
      ['/competitors', 'Competitors: 2 makers sell the same belts in Europe, each under its own names. Zoom in.'],
      ['/terms', "Search terms: 3 search terms in 3 languages. Tribelt's own website uses 1 of them. Zoom in."],
      ['/plan', 'The plan: 12 months in three phases, with a decision at the end of each. Zoom in.'],
    ])
    const competitors = tiles(wrapper)[2]
    expect(competitors?.text()).toContain('Makers offering each Tribelt family, of 2')
    expect(competitors?.text()).toContain('Eyelink1')
    expect(tiles(wrapper)[3]?.text()).toContain('English2')
    expect(tiles(wrapper)[4]?.text()).toContain('Expand and learn')
  })

  it('shows a placeholder until the live figures load', async () => {
    const { wrapper } = await mountApp(routes({}), '/')
    expect(tiles(wrapper)[2]?.attributes('aria-label')).toContain('Competitors: …')
    expect(tiles(wrapper)[3]?.attributes('aria-label')).toContain('Search terms: … search terms in the terminology map.')
  })
})

describe('the zoom', () => {
  it('grows toward a tile, then opens its section; Overview shrinks back out of it', async () => {
    const { wrapper, router } = await mountApp(api, '/')
    await tiles(wrapper)[4]?.trigger('click', { button: 0 })
    expect(poster(wrapper).classes()).toContain('zoom-in')
    expect(router.currentRoute.value.path).toBe('/')
    // A second click while it moves changes nothing.
    await tiles(wrapper)[0]?.trigger('click', { button: 0 })
    await poster(wrapper).trigger('animationend')
    await settle()
    expect(router.currentRoute.value.path).toBe('/plan')
    // Opened from the poster, the section settles in.
    expect(wrapper.get('article').classes()).toContain('detail-in')

    await wrapper.get('article button').trigger('click')
    await settle()
    expect(router.currentRoute.value.path).toBe('/')
    expect(poster(wrapper).classes()).toContain('zoom-out')
    await poster(wrapper).trigger('animationend')
    expect(poster(wrapper).classes()).not.toContain('zoom-out')
  })

  it('leaves a click that opens a new tab to the browser', async () => {
    const { wrapper, router } = await mountApp(api, '/')
    // The browser would open the link itself; jsdom cannot, so the test stops it after the poster has had its say.
    const keep = (event: Event) => {
      event.preventDefault()
    }
    document.addEventListener('click', keep)
    onTestFinished(() => {
      document.removeEventListener('click', keep)
    })
    for (const modifier of [{ metaKey: true }, { ctrlKey: true }, { shiftKey: true }, { altKey: true }, { button: 1 }]) {
      await tiles(wrapper)[0]?.trigger('click', { button: 0, ...modifier })
    }
    expect(poster(wrapper).classes()).not.toContain('zoom-in')
    expect(router.currentRoute.value.path).toBe('/')
  })

  it('shows a section loaded directly without a zoom, and its Overview opens the poster as a new page', async () => {
    const { wrapper, router } = await mountApp(api, '/buyers')
    expect(wrapper.get('article').classes()).not.toContain('detail-in')
    await wrapper.get('article button').trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/')
    // Back from the buyers section, the poster shrinks out of the buyers tile.
    expect(poster(wrapper).classes()).toContain('zoom-out')
  })
})
