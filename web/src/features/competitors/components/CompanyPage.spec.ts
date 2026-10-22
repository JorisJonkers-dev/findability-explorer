import { describe, expect, it } from 'vitest'
import { competitor, summary, term, termSearch } from '../../../testing/fixtures'
import { json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const OTHER_ID = '7d2e3c1a-4f5b-4a6c-8d7e-9f0a1b2c3d4e'

const twentebelt = competitor('Twentebelt B.V.', {
  country: 'NL',
  languages: ['en', 'de'],
  scans: 2,
  beltNames: [
    { name: 'TwenteFlex spiral belts', family: 'tri_flex_turn_curve' },
    { name: 'Eyelink belt', family: 'eyelink' },
  ],
})

const terms = [
  term({ term: 'metal conveyor belt', tier: 'broad', usedBy: ['Twentebelt'] }),
  term({ term: 'TwenteFlex', tier: 'product', usedBy: ['Twentebelt (TwenteFlex)'] }),
  term({ term: 'Drahtgeflechtband', usedBy: ['Märtens'] }),
]

function api(detail: unknown = { competitor: twentebelt, scans: [summary(), summary({ id: OTHER_ID, status: 'failed', pagesCrawled: 0 })] }) {
  return routes({
    'GET /api/v1/competitors/twentebelt-b-v': () => json(detail),
    'GET /api/v1/competitors': () => json({ items: [twentebelt] }),
    'GET /api/v1/terms': () => json(termSearch(terms)),
    'GET /api/v1/analyses': () => json({ items: [] }),
  })
}

describe('the company page', () => {
  it("shows the company, its website, its belt names, its terms and its scans", async () => {
    const { wrapper } = await mountApp(api(), '/competitors/twentebelt-b-v')
    expect(document.title).toBe('Competitor · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Twentebelt')
    expect(wrapper.text()).toContain('Twentebelt B.V.')
    expect(wrapper.text()).toContain('Netherlands.')
    expect(wrapper.text()).toContain('Website in English and German.')
    expect(wrapper.text()).toContain('Researched')
    expect(wrapper.text()).toContain('Scanned 2×')

    const visit = wrapper.get('main a[target="_blank"]')
    expect(visit.text()).toContain('Visit website')
    expect(visit.attributes('href')).toBe('https://www.twentebelt.com/')
    expect(visit.attributes('rel')).toBe('noopener noreferrer')

    const names = wrapper.get('section[aria-labelledby="names-title"]')
    expect(names.findAll('dt').map((d) => d.text())).toEqual(['Eyelink', 'Tri-Flex'])

    const used = wrapper.get('section[aria-labelledby="terms-title"]')
    expect(used.findAll('li span.font-semibold').map((s) => s.text())).toEqual(['TwenteFlex', 'metal conveyor belt'])
    expect(used.get('a').attributes('href')).toBe('/terminology?user=Twentebelt')
    expect(used.get('a').text()).toBe('See them in the terminology')

    const scans = wrapper.get('section[aria-labelledby="scans-title"]')
    expect(scans.findAll('a').map((a) => a.attributes('href'))).toEqual(['/scan/5b0c1a9e-2d3f-4e5a-8b6c-7d8e9f0a1b2c', `/scan/${OTHER_ID}`])
    expect(scans.text()).toContain('Failed')
  })

  it('scans the website from the scan form, filled in', async () => {
    const { wrapper, router } = await mountApp(api(), '/competitors/twentebelt-b-v')
    const scan = wrapper.findAll('a').filter((a) => a.text() === 'Scan this website')[0]
    expect(scan?.attributes('href')).toBe('/scan?competitor=twentebelt-b-v')
    await scan?.trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/scan')
    expect((wrapper.get('input[type="url"]').element as HTMLInputElement).value).toBe('https://www.twentebelt.com/')
    expect(wrapper.text()).toContain('known competitor')
  })

  it('says what is not known yet about an added company', async () => {
    const added = competitor('Example Belts', {
      id: 'twentebelt-b-v',
      source: 'added',
      addedAt: '2026-10-22T10:00:00Z',
      websiteUrl: undefined,
      groupCompany: true,
      languages: [],
      beltNames: [],
    })
    const { wrapper } = await mountApp(api({ competitor: added, scans: [] }), '/competitors/twentebelt-b-v')
    expect(wrapper.get('h1').text()).toBe('Example Belts')
    expect(wrapper.text()).toContain('Added 22 October 2026')
    expect(wrapper.text()).toContain("Part of Tribelt's own group")
    expect(wrapper.text()).toContain('The research found no website for this company.')
    expect(wrapper.text()).toContain('No belt names are known for this company yet.')
    expect(wrapper.text()).toContain('The terminology map credits no terms to Example Belts.')
    expect(wrapper.text()).toContain('Its website has not been scanned yet.')
    expect(wrapper.find('main a[target="_blank"]').exists()).toBe(false)
  })

  it('links a long list of terms to the terminology', async () => {
    const many = Array.from({ length: 14 }, (_, i) => term({ term: `term ${String(i)}`, usedBy: ['Twentebelt'] }))
    const handle = routes({
      'GET /api/v1/competitors/twentebelt-b-v': () => json({ competitor: twentebelt, scans: [] }),
      'GET /api/v1/terms': () => json(termSearch(many)),
    })
    const { wrapper } = await mountApp(handle, '/competitors/twentebelt-b-v')
    const used = wrapper.get('section[aria-labelledby="terms-title"]')
    expect(used.findAll('li')).toHaveLength(12)
    expect(used.get('a').text()).toBe('See all 14 terms in the terminology')
  })

  it('says when no company has the id, or why it could not load', async () => {
    const missing = await mountApp(api(), '/competitors/nobody')
    expect(missing.wrapper.get('[role="alert"]').text()).toContain('No company has this address.')
    missing.wrapper.unmount()
    const failing = await mountApp(
      routes({ 'GET /api/v1/competitors/twentebelt-b-v': () => problem(500, 'Internal Server Error') }),
      '/competitors/twentebelt-b-v',
    )
    expect(failing.wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
    failing.wrapper.unmount()
    const slow = await mountApp(
      routes({ 'GET /api/v1/competitors/twentebelt-b-v': () => new Promise<Response>(() => undefined) }),
      '/competitors/twentebelt-b-v',
    )
    expect(slow.wrapper.text()).toContain('Loading the company…')
  })
})
