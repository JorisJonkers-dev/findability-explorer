import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it, onTestFinished } from 'vitest'
import { competitor, promptResults, term, termSearch } from '../../../testing/fixtures'
import { json, mountApp, routes, settle } from '../../../testing/mountApp'

const api = routes({
  'GET /api/v1/standings': () =>
    json({
      measures: [
        { measure: 'aiShareOfVoice', tribelt: 0.24, median: 0.1, leader: { competitorId: 'a', name: 'Ashworth', value: 0.57 }, competitors: 3, trails: true, leads: false },
        { measure: 'termsUsed', tribelt: 38, median: 10, leader: { competitorId: 'b', name: 'B', value: 30 }, competitors: 3, trails: false, leads: true },
      ],
      trailing: 1,
      readinessChecks: 5,
      open: [],
    }),
  'GET /api/v1/overview': () =>
    json({
      terms: { total: 3, tribelt: 1, languages: [{ language: 'en', terms: 2, tribelt: 1 }, { language: 'de', terms: 1, tribelt: 0 }] },
      competitors: { makers: 2, families: [{ family: 'eyelink', makers: 1 }, { family: 'open_spiral', makers: 0 }] },
      aiSearch: {
        answers: 4,
        companies: [
          { competitorId: 'twentebelt', name: 'Twentebelt', role: 'competitor', answers: 3, share: 0.75 },
          { competitorId: 'tribelt', name: 'Tribelt', role: 'tribelt', answers: 1, share: 0.25 },
        ],
      },
    }),
  // The sections the zoom opens load their own data.
  'GET /api/v1/prompt-test': () => json(promptResults()),
  'GET /api/v1/competitors': () =>
    json({
      items: [
        competitor('Twentebelt', { beltNames: [{ name: 'Eyelink belts', family: 'eyelink' }] }),
        competitor('Märtens', { beltNames: [] }),
        competitor('Tribelt B.V.', { groupCompany: true }),
      ],
    }),
  'GET /api/v1/terms': () =>
    json(
      termSearch([term({ language: 'en', usedBy: ['Tribelt'] }), term({ language: 'de', usedBy: ['Märtens'] }), term({ language: 'en-US' })], {
        facets: { languages: [{ value: 'en', count: 1 }, { value: 'de', count: 1 }, { value: 'en-US', count: 1 }], tiers: [], families: [] },
      }),
    ),
})

const tiles = (wrapper: VueWrapper) => wrapper.findAll('nav[aria-label="Sections"] a')
const overviewLink = (wrapper: VueWrapper) => {
  const link = wrapper.findAll('nav[aria-label="Main"] a').find((a) => a.text() === 'Overview')
  if (!link) throw new Error('no Overview link')
  return link
}
const poster = (wrapper: VueWrapper) => wrapper.get('main > div > div > div')

describe('the poster', () => {
  it('says what the strategy is about, and credits the course', async () => {
    const { wrapper } = await mountApp(api, '/')
    expect(wrapper.get('h1').text()).toBe('One belt,many names.')
    expect(wrapper.text()).toContain('find Tribelt, on Google and in Gemini.')
    expect(wrapper.find('img[alt="A Tribelt employee at a belt production line"]').exists()).toBe(true)
    const footer = wrapper.get('footer')
    expect(footer.text()).toContain('A student project for Tribelt B.V. in Digital Strategy (201900119)')
    expect(footer.find('img[alt="Tribelt, part of Lesjöfors Group"]').exists()).toBe(true)
    expect(footer.find('img[alt="University of Twente"]').exists()).toBe(true)
    expect(footer.text()).toContain('Signed in as Joris Jonkers')
  })

  it('links to the four sections, each with its figure and line', async () => {
    const { wrapper } = await mountApp(api, '/')
    expect(tiles(wrapper).map((a) => [a.attributes('href'), a.attributes('aria-label')])).toEqual([
      ['/ai-search', 'AI search: 25% of the AI answers in the newest Prompt run name Tribelt. Twentebelt is named in 75%. Zoom in.'],
      ['/competitors', 'Competitors: 2 competitors on our list sell the same belts in Europe, each under its own names. Zoom in.'],
      ['/terms', 'Search terms: 3 search terms in 2 languages. The newest scan of tribelt.nl found 1 of them. Zoom in.'],
      ['/recommendations', 'Recommendations: 52 weekly checks a year. Every fix shows up in the numbers the week after. Zoom in.'],
    ])
    expect(tiles(wrapper)[0]?.text()).toContain('Tribelt25%')
    const competitors = tiles(wrapper)[1]
    expect(competitors?.text()).toContain('Competitors offering each Tribelt family, of 2')
    expect(competitors?.text()).toContain('Eyelink1')
    expect(tiles(wrapper)[2]?.text()).toContain('English2')
    expect(tiles(wrapper)[3]?.text()).toContain('Read the Monday report')
  })

  it('asks the server for its figures, not for the whole map, the companies or the Prompt runs', async () => {
    const { requests } = await mountApp(api, '/')
    expect(requests()).toContain('GET /api/v1/overview')
    expect(requests().filter((r) => /^GET \/api\/v1\/(terms|competitors|prompt-test)\b/.test(r))).toEqual([])
  })

  it('shows a placeholder until the live figures load', async () => {
    const { wrapper } = await mountApp(routes({}), '/')
    expect(tiles(wrapper)[0]?.attributes('aria-label')).toContain('AI search: … of AI answers name Tribelt, once the first Prompt run is in.')
    expect(tiles(wrapper)[1]?.attributes('aria-label')).toContain('Competitors: …')
    expect(tiles(wrapper)[2]?.attributes('aria-label')).toContain('Search terms: … search terms in the terminology map.')
  })
})

describe('the zoom', () => {
  it('grows toward a tile, then opens its section; Overview links back', async () => {
    const { wrapper, router } = await mountApp(api, '/')
    await tiles(wrapper)[3]?.trigger('click', { button: 0 })
    expect(poster(wrapper).classes()).toContain('zoom-in')
    expect(router.currentRoute.value.path).toBe('/')
    // A second click while it moves changes nothing.
    await tiles(wrapper)[0]?.trigger('click', { button: 0 })
    await poster(wrapper).trigger('animationend')
    await settle()
    expect(router.currentRoute.value.path).toBe('/recommendations')
    // Opened from the poster, the section settles in.
    expect(wrapper.find('.detail-in').exists()).toBe(true)

    await wrapper.get('nav[aria-label="Main"] a[href="/"]:not([aria-label])').trigger('click')
    await settle()
    expect(router.currentRoute.value.path).toBe('/')
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
    const { wrapper, router } = await mountApp(api, '/terms')
    expect(wrapper.find('.detail-in').exists()).toBe(false)
    expect(wrapper.find('nav[aria-label="Main"] button.cursor-zoom-out').exists()).toBe(false)
    await overviewLink(wrapper).trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/')
    // Back from the search terms section, the poster shrinks out of its tile.
    expect(poster(wrapper).classes()).toContain('zoom-out')
  })

  it('shows the poster without a zoom back from a section it has no tile for', async () => {
    const { wrapper, router } = await mountApp(api, '/buyers')
    await router.push('/')
    await settle()
    expect(poster(wrapper).classes()).not.toContain('zoom-out')
  })
})
