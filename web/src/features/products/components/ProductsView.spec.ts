import { describe, expect, it } from 'vitest'
import { product } from '../../../testing/fixtures'
import { json, mountApp, problem, routes } from '../../../testing/mountApp'

const overview = {
  items: [
    product(),
    product({
      family: 'open_spiral',
      tribeltNames: [],
      sellers: [],
      terms: [],
      termsUsed: 0,
      answers: 0,
      named: 0,
      byScope: [],
      mostNamed: [],
    }),
  ],
  standouts: [{ finding: 'crowded', family: 'eyelink' }],
}

const api = () =>
  routes({
    'GET /api/v1/products': () => json(overview),
  })

describe('the products', () => {
  it('shows a card per family with its share of answers, sellers, terms and most used name', async () => {
    const { wrapper } = await mountApp(api(), '/products')
    const cards = wrapper.findAll('nav[aria-label="Belt families"] a')
    expect(cards.map((c) => c.attributes('href'))).toEqual(['/products/eyelink', '/products/open-spiral'])
    const eye = cards[0]?.text() ?? ''
    expect(eye).toContain('30%of AI answers about it name Tribelt')
    expect(eye).toContain('Competitors on the list selling it1')
    expect(eye).toContain('Search terms tribelt.nl uses1 of 4')
    expect(eye).toContain('Most used namedraadogenband')
    expect(cards[1]?.text()).toContain('–of AI answers')
    expect(cards[1]?.text()).toContain('Tribelt names none')
    expect(wrapper.get('section[aria-label="What stands out"]').text()).toBe('Most competitors sell Eyelink. 1 competitor sells it. tribelt.nl uses 1 of its 4 search terms.')
  })

  it('opens a family on its own page, with the terms, the sellers and the answers', async () => {
    const one = product({
      sellers: [{ competitorId: 'twentebelt', name: 'Twentebelt', names: ['Oogbanden', 'Draadogenbanden'], websiteUrl: 'https://www.twentebelt.com/' }],
      byScope: [{ scope: 'twente', answers: 10, named: 4 }],
      mostNamed: [
        { competitorId: 'twentebelt', name: 'Twentebelt', role: 'competitor', share: 0.6 },
        { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', share: 0.3 },
      ],
    })
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/products': () => json({ items: [one], standouts: [] }) }), '/products/eyelink')
    expect(wrapper.get('h1').text()).toBe('Eyelink. One competitor sells it, and buyers search for it in four languages.')
    expect(wrapper.get('nav[aria-label="In this section"] a[aria-current="page"]').text()).toBe('Eyelink')
    const terms = wrapper.findAll('section[aria-labelledby="terms-title"] tbody tr').map((r) => r.findAll('th, td').map((c) => c.text()))
    expect(terms[0]).toEqual(['draadogenband, Product', 'Dutch', '9', '✕ No'])
    expect(terms[1]?.[3]).toBe('✓ Yes')
    const sellers = wrapper.findAll('ul[aria-label="Sellers"] li')
    expect(sellers[0]?.text()).toBe('TribeltDraadogenbanden')
    expect(sellers[1]?.get('a').attributes('href')).toBe('/competitors/twentebelt')
    expect(sellers[1]?.get('a[target="_blank"]').attributes('href')).toBe('https://www.twentebelt.com/')
    const answers = wrapper.get('aside[aria-labelledby="answers-title"]')
    expect(answers.text()).toContain('30%of the 40 answers about it name Tribelt')
    expect(answers.text()).toContain('Twente (Dutch)4 of 10')
    expect(answers.findAll('ol li').map((l) => l.text())).toEqual(['Twentebelt 60%', 'Tribelt B.V. 30%'])
  })

  it('says what is missing: no run, no seller, no such family, or no data', async () => {
    const bare = product({ answers: 0, named: 0, sellers: [], tribeltNames: [] })
    const { wrapper, router } = await mountApp(routes({ 'GET /api/v1/products': () => json({ items: [bare], standouts: [] }) }), '/products/eyelink')
    expect(wrapper.text()).toContain('No Prompt run has asked about it yet.')
    expect(wrapper.text()).toContain('No competitor sells it')
    expect(wrapper.text()).toContain('Tribelt names none for it yet.')
    await router.push('/products/nothing')
    expect(wrapper.text()).toContain('Pick a belt family above.')
    wrapper.unmount()
    const down = await mountApp(routes({ 'GET /api/v1/products': () => problem(500, 'Internal Server Error', 'Down.') }), '/products')
    expect(down.wrapper.get('[role="alert"]').text()).toContain('Down.')
    down.wrapper.unmount()
    const downOne = await mountApp(routes({ 'GET /api/v1/products': () => problem(500, 'Internal Server Error', 'Down.') }), '/products/eyelink')
    expect(downOne.wrapper.get('h1').text()).toBe('Eyelink')
    expect(downOne.wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
