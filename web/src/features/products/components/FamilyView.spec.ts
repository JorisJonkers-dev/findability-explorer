import { describe, expect, it } from 'vitest'
import type { Product } from '../../../infrastructure/api/types.gen'
import { product } from '../../../testing/fixtures'
import { json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const api = (items: Product[]) => routes({ 'GET /api/v1/products': () => json({ items, standouts: [] }) })

const many = (n: number): Product['sellers'] => Array.from({ length: n }, (_, i) => ({ competitorId: `c${String(i)}`, name: `Seller ${String(i)}`, names: ['Oogbanden', 'Eyelinks'] }))

describe('a belt family page', () => {
  it('heads itself with how many competitors sell it, and links back', async () => {
    const { wrapper } = await mountApp(api([product()]), '/terms/families/eyelink')
    expect(wrapper.get('h1').text()).toBe('One competitor sells eyelink belts.')
    expect(wrapper.text()).toContain('Back to Search terms')
    expect(wrapper.text()).toContain('Tribelt calls it Draadogenbanden.')
  })

  it('has three figures: sellers, the terms used, and the share of AI answers naming Tribelt', async () => {
    const { wrapper } = await mountApp(api([product()]), '/terms/families/eyelink')
    const figures = wrapper.get('section[aria-label="Key figures"]').findAll(':scope > div').map((d) => d.text())
    expect(figures[0]).toBe('1competitors sell itNot counting Tribelt or its group companies.')
    expect(figures[1]).toBe('1 of 4of its search terms are used on tribelt.nlTerms in Dutch, German, English, and French.')
    expect(figures[2]).toBe('30%of AI answers about Eyelink belts name TribeltRead those answers')
    expect(wrapper.get('a[href="/ai-search/answers"]').text()).toBe('Read those answers')
  })

  it('says when no Prompt run has asked about it, and when no competitor sells it', async () => {
    const bare = product({ answers: 0, named: 0, sellers: [], tribeltNames: [] })
    const { wrapper } = await mountApp(api([bare]), '/terms/families/eyelink')
    expect(wrapper.get('h1').text()).toBe('No competitor sells eyelink belts.')
    expect(wrapper.text()).toContain('No Prompt run has asked about it yet')
    expect(wrapper.text()).toContain('–')
    expect(wrapper.text()).toContain('No competitor sells it.')
    expect(wrapper.text()).toContain('Tribelt names none for it yet.')
  })

  it('tables its terms with the kind, and filters them by language', async () => {
    const { wrapper } = await mountApp(api([product()]), '/terms/families/eyelink')
    const rows = () => wrapper.findAll('section[aria-labelledby="terms-title"] tbody tr').map((r) => r.findAll('th, td').map((c) => c.text()))
    expect(rows()[0]).toEqual(['draadogenband', 'Dutch', 'Product', '9', 'No'])
    expect(rows()[1]).toEqual(['eyelink belt', 'American English', 'Product', '5', 'Yes'])
    expect(wrapper.text()).toContain('4 terms, of which tribelt.nl uses 1.')
    const pills = wrapper.get('[role="group"][aria-label="Language"]').findAll('button')
    await pills.find((b) => b.text() === 'German')?.trigger('click')
    await settle()
    expect(rows()).toEqual([['Drahtösenband', 'German', 'Product', '3', 'No']])
    expect(wrapper.text()).toContain('1 term, of which tribelt.nl uses 0.')
    expect(wrapper.get('svg[role="img"]').attributes('aria-label')).toContain('1 of 4')
  })

  it('lists its sellers, linking each to its company, and shows the first eight until asked for all', async () => {
    const { wrapper } = await mountApp(api([product({ sellers: many(10) })]), '/terms/families/eyelink')
    const list = () => wrapper.get('ul[aria-label="Sellers"]').findAll('li')
    expect(list()).toHaveLength(8)
    expect(list()[0]?.text()).toBe('Seller 0Calls it Oogbanden, Eyelinks')
    expect(list()[0]?.get('a').attributes('href')).toBe('/competitors/c0')
    const toggle = wrapper.get('button[aria-expanded]')
    expect(toggle.text()).toBe('Show all 10')
    await toggle.trigger('click')
    expect(list()).toHaveLength(10)
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(toggle.text()).toBe('Show the first 8')
    await toggle.trigger('click')
    expect(list()).toHaveLength(8)
  })

  it('has no toggle for few sellers', async () => {
    const { wrapper } = await mountApp(api([product()]), '/terms/families/eyelink')
    expect(wrapper.find('button[aria-expanded]').exists()).toBe(false)
  })

  it('says that a family does not exist, is loading, or cannot be loaded', async () => {
    const { wrapper, router } = await mountApp(api([product()]), '/terms/families/eyelink')
    await router.push('/terms/families/nothing')
    await settle()
    expect(wrapper.get('h1').text()).toBe('No such belt family.')
    expect(wrapper.text()).toContain('That belt family does not exist.')
    await router.push('/terms/families/open-spiral')
    await settle()
    expect(wrapper.text()).toContain('Loading the belt family…')
    wrapper.unmount()
    const down = await mountApp(routes({ 'GET /api/v1/products': () => problem(500, 'Internal Server Error', 'Down.') }), '/terms/families/eyelink')
    expect(down.wrapper.get('h1').text()).toBe('Eyelink')
    expect(down.wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
