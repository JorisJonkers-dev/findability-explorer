import { describe, expect, it } from 'vitest'
import { competitor, term, termSearch } from '../../../testing/fixtures'
import { json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const companies = [
  competitor('Märtens Transportbänder GmbH (MARTENS)', {
    country: 'DE (Flensburg)',
    languages: ['de', 'en-GB', 'unverified'],
    beltNames: [
      { name: 'Drahtösengliederband / Loop-joined wire belt', family: 'eyelink' },
      { name: 'Drahtgeflechtband / Wire mesh belt; Stabgeflechtband; Backofenband', family: 'other' },
    ],
  }),
  competitor('Twentebelt', {
    country: 'NL (production also in PL)',
    languages: ['en'],
    beltNames: [
      { name: 'TwenteFlex spiral belts', family: 'tri_flex_turn_curve' },
      { name: 'TwenteSideFlex', family: 'tri_flex_turn_curve' },
    ],
  }),
  competitor('Lesjöfors (metal belts)', { country: 'SE', groupCompany: true, beltNames: [{ name: 'Tri-Flex', family: 'tri_flex_turn_curve' }] }),
  competitor('Tribelt B.V.', { country: 'NL (Haaksbergen)', groupCompany: true }),
]

const api = routes({
  'GET /api/v1/competitors': () => json({ items: companies }),
  'GET /api/v1/terms': () => json(termSearch([term({ usedBy: ['Märtens', 'Twentebelt (TwenteFlex)'] })])),
  'GET /api/v1/analyses': () => json({ items: [] }),
})

describe('the competitors page', () => {
  it("shows each competitor's names in a matrix of Tribelt's families", async () => {
    const { wrapper } = await mountApp(api, '/competitors')
    expect(wrapper.get('h1').text()).toBe('Competitors')
    expect(wrapper.text()).toContain('2 makers of metal belts that sell in Europe')
    const table = wrapper.get('[role="region"][aria-label="Competitors"] table')
    expect(table.findAll('thead th').map((th) => th.text())).toEqual([
      'Company',
      'Eyelink',
      'Open spiral',
      'Closed spiral',
      'Tri-Flex',
      'Other belts',
    ])
    const rows = table.findAll('tbody tr')
    expect(rows.map((r) => r.get('th a[href^="/"]').text())).toEqual(['Märtens', 'Twentebelt'])
    expect(rows[0]?.get('th').text()).toContain('Germany, 2 languages')
    const cells = rows[1]?.findAll('td') ?? []
    expect(cells[3]?.text()).toBe('TwenteFlex spiral belts and 1 more')
    expect(cells[0]?.classes()).toContain('hatched')
    expect(cells[0]?.text()).toBe('None')
    expect(rows[0]?.findAll('td')[4]?.text()).toBe('Drahtgeflechtband and 2 more')
  })

  it("links each company to its page and to its website", async () => {
    const { wrapper, router } = await mountApp(api, '/competitors')
    const row = wrapper.get('tbody tr')
    const visit = row.get('a[target="_blank"]')
    expect(visit.attributes('href')).toBe('https://www.twentebelt.com/')
    expect(visit.attributes('rel')).toBe('noopener noreferrer')
    expect(visit.text()).toBe("Visit Märtens's website (opens in a new tab)")
    const link = row.get('th a[href^="/"]')
    expect(link.attributes('href')).toBe('/competitors/martens-transportbander-gmbh-martens')
    await link.trigger('click')
    await settle()
    expect(router.currentRoute.value.name).toBe('company')
  })

  it('marks an added company and says how often a company was scanned', async () => {
    const more = [...companies, competitor('Example Belts', { source: 'added', scans: 3, websiteUrl: undefined })]
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/competitors': () => json({ items: more }) }), '/competitors')
    const row = wrapper.findAll('[aria-label="Competitors"] tbody tr').at(-1)
    expect(row?.get('th').text()).toContain('Added')
    expect(row?.get('th').text()).toContain('Scanned 3×')
    expect(row?.find('a[target="_blank"]').exists()).toBe(false)
  })

  it("lists Tribelt's own group apart, below the competitors", async () => {
    const { wrapper } = await mountApp(api, '/competitors')
    expect(wrapper.text()).toContain("Lesjöfors belong to Tribelt's own group and are listed apart, below the competitors.")
    const group = wrapper.get('section[aria-labelledby="group-title"]')
    expect(group.findAll('tbody th a[href^="/"]').map((a) => a.text())).toEqual(['Lesjöfors', 'Tribelt'])
  })

  it('finds a company by name or country', async () => {
    const { wrapper } = await mountApp(api, '/competitors')
    const input = wrapper.get('input[type="search"]')
    await input.setValue('netherlands')
    expect(wrapper.findAll('[aria-label="Competitors"] tbody th a[href^="/"]').map((a) => a.text())).toEqual(['Twentebelt'])
    expect(wrapper.text()).toContain('1 of 2 competitors')
    expect(wrapper.get('section[aria-labelledby="group-title"]').findAll('tbody tr')).toHaveLength(1)
    await input.setValue('nowhere')
    expect(wrapper.text()).toContain('No competitor matches “nowhere”.')
    expect(wrapper.find('section[aria-labelledby="group-title"]').exists()).toBe(false)
  })

  it('says it is loading, or why it could not', async () => {
    const slow = await mountApp(routes({ 'GET /api/v1/competitors': () => new Promise<Response>(() => undefined) }), '/competitors')
    expect(slow.wrapper.text()).toContain('Loading the competitors…')
    expect(slow.wrapper.text()).toContain('The makers of metal belts')
    slow.wrapper.unmount()
    const failing = await mountApp(routes({ 'GET /api/v1/competitors': () => problem(500, 'Internal Server Error') }), '/competitors')
    expect(failing.wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
  })

  it('names two or more sister companies in one sentence', async () => {
    const more = [...companies, competitor('De Spiraal B.V.', { groupCompany: true })]
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/competitors': () => json({ items: more }) }), '/competitors')
    expect(wrapper.text()).toContain("Lesjöfors and De Spiraal belong to Tribelt's own group")
  })
})
