import { describe, expect, it } from 'vitest'
import type { Product } from '../../../infrastructure/api/types.gen'
import { product } from '../../../testing/fixtures'
import { json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const api = (items: Product[]) => routes({ 'GET /api/v1/products': () => json({ items, standouts: [] }) })

const many = (n: number): Product['sellers'] => Array.from({ length: n }, (_, i) => ({ competitorId: `c${String(i)}`, name: `Seller ${String(i)}`, names: ['Oogbanden', 'Eyelinks'] }))

describe('a belt family page', () => {
  it('is in the frame, with its purpose, data line, and sections in order', async () => {
    const { wrapper } = await mountApp(api([product()]), '/terms/families/eyelink')
    expect(wrapper.get('h1').text()).toBe('Eyelink belts')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('The terms buyers use for these belts, and the companies that sell them.')
    expect(wrapper.get('[data-part="data-line"]').text()).toBe("From the search term list, each site's newest scan, and the newest Prompt run.")
    expect(wrapper.findAll('a[href="/terms"]').map((a) => a.text())).toContain('Back to Search terms')
    expect(wrapper.findAll('h2').map((h) => h.text())).toEqual(['Terms by language', 'Tribelt’s own names', 'Who sells them', 'AI responses about eyelink belts'])
    expect(wrapper.get('section[aria-labelledby="names-title"]').text()).toContain('Draadogenbanden')
  })

  it('bars the share of responses naming Tribelt per scope, and links to them', async () => {
    const p = product({ byScope: [{ scope: 'twente', answers: 10, named: 3 }] })
    const { wrapper } = await mountApp(api([p]), '/terms/families/eyelink')
    const answers = wrapper.get('section[aria-labelledby="responses-title"]')
    expect(answers.get('[role="img"]').attributes('aria-label')).toBe('The share of responses that name Tribelt. Twente (Dutch), 10 responses: 30%.')
    expect(answers.get('a[href="/ai-responses/all"]').text()).toBe('See these responses')
  })

  it('says when no Prompt run has asked about it, when no competitor sells it, and when Tribelt names none', async () => {
    const bare = product({ answers: 0, named: 0, sellers: [], tribeltNames: [] })
    const { wrapper } = await mountApp(api([bare]), '/terms/families/eyelink')
    expect(wrapper.text()).toContain('The newest Prompt run asked no questions about these belts.')
    expect(wrapper.text()).toContain('No competitor on the list sells these belts.')
    expect(wrapper.text()).toContain('No name is recorded.')
  })

  it('tables its terms with their buyer intent, each opening its page, and filters them by language', async () => {
    const { wrapper } = await mountApp(api([product()]), '/terms/families/eyelink')
    const rows = () => wrapper.findAll('section[aria-labelledby="terms-title"] tbody tr').map((r) => r.findAll('th, td').map((c) => c.text()))
    expect(rows()[0]).toEqual(['draadogenband', 'Dutch', 'Product', '9', 'No'])
    expect(rows()[1]).toEqual(['eyelink belt', 'American English', 'Product', '5', 'Yes'])
    expect(wrapper.get('section[aria-labelledby="terms-title"] thead').text()).toContain('Buyer intent')
    expect(wrapper.get('section[aria-labelledby="terms-title"] tbody a').attributes('href')).toBe('/terms/nl--draadogenband')
    expect(wrapper.text()).toContain('4 terms, of which tribelt.nl uses 1.')
    const pills = wrapper.get('[role="group"][aria-label="Language"]').findAll('button')
    await pills.find((b) => b.text() === 'German')?.trigger('click')
    await settle()
    expect(rows()).toEqual([['Drahtösenband', 'German', 'Product', '3', 'No']])
    expect(wrapper.text()).toContain('1 term, of which tribelt.nl uses 0.')
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
    expect(wrapper.get('h1').text()).toBe('Belt family')
    expect(wrapper.text()).toContain('That belt family does not exist.')
    await router.push('/terms/families/open-spiral')
    await settle()
    expect(wrapper.text()).toContain('Loading the belt family…')
    wrapper.unmount()
    const down = await mountApp(routes({ 'GET /api/v1/products': () => problem(500, 'Internal Server Error', 'Down.') }), '/terms/families/eyelink')
    expect(down.wrapper.get('h1').text()).toBe('Eyelink belts')
    expect(down.wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
