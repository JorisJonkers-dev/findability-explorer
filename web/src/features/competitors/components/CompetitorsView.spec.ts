import { type VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { competitor, me, overview, promptResults, readinessRow, readinessTable, summary, term, termSearch } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const twentebelt = competitor('Twentebelt', { scans: 2 })
const martens = competitor('Martens', { country: 'DE', websiteUrl: 'https://www.martens-belts.com/', host: 'martens-belts.com', website: 'https://www.martens-belts.com' })
const ashworth = competitor('Ashworth', { country: 'US', websiteUrl: undefined, host: undefined, website: '', beltNames: [] })
const tribelt = competitor('Tribelt', { role: 'tribelt', groupCompany: true, websiteUrl: 'https://www.tribelt.nl/', host: 'tribelt.nl', website: 'https://www.tribelt.nl' })
const old = competitor('Old Belts', { country: 'BE', archived: true, websiteUrl: 'https://old.example/', host: 'old.example', website: 'https://old.example' })
const sister = competitor('Tribelt Group', { role: 'group', groupCompany: true, websiteUrl: 'https://group.example/', host: 'group.example', website: 'https://group.example' })

/** The server, signed in as an administrator; the scan started answers with each status in line, the last one for good. */
function api(extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/competitors': () => json({ items: [twentebelt, martens, ashworth, tribelt, sister, old] }),
    'GET /api/v1/terms': () => json(termSearch([term({ term: 'TwenteFlex', tier: 'product', usedBy: ['Twentebelt (TwenteFlex)'] })], { tribeltFrom: 'scan' })),
    'GET /api/v1/overview': () => json(overview({ competitors: { makers: 3, families: [{ family: 'eyelink', makers: 2 }, { family: 'open_spiral', makers: 1 }] } })),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'GET /api/v1/ai-readiness': () => json(readinessTable([readinessRow('Twentebelt'), readinessRow('Tribelt', { role: 'tribelt' })])),
    'GET /api/v1/competitors/twentebelt': () => json({ competitor: twentebelt, scans: [summary()] }),
    'GET /api/v1/analyses': () => json({ items: [] }),
    ...extra,
  })
}

const list = (wrapper: VueWrapper) => wrapper.get('section[aria-labelledby="list-title"]')
const names = (wrapper: VueWrapper) => list(wrapper).findAll('tbody tr').map((r) => r.get('th').text())

describe('the competitors page', () => {
  it('is in the frame, with one table, Tribelt pinned on top, and the best competitor named', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/me': () => json(me()) }), '/competitors')
    expect(wrapper.get('h1').text()).toBe('Competitors')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe(
      "The companies that make metal conveyor belts Tribelt's buyers could choose instead, and how each compares with Tribelt on AI responses, AI readiness, and search terms.",
    )
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('3 competitors on the competitor list. No site has been scanned yet.')
    expect(wrapper.get('[data-part="summary"]').text()).toBe('Tribelt is named in 25% of AI responses. The best competitor, Twentebelt, is named in 75%.')
    expect(list(wrapper).findAll('thead th').map((h) => h.text())).toEqual([
      'Name',
      'Country',
      'Belt families sold',
      'Named in AI responses',
      'AI readiness',
      'Search terms used',
      'Last scanned',
    ])
    const rows = list(wrapper).findAll('tbody tr').map((r) => r.findAll('th, td').map((c) => c.text()))
    expect(rows[0]).toEqual(['TribeltFor comparison', 'NL', '1 of 5', '25%', '8 of 8', '0', 'Not scanned'])
    expect(rows[1]).toEqual(['Twentebelt', 'NL', '1 of 5', '75%', '8 of 8', '1', 'Not scanned'])
    expect(names(wrapper)).toEqual(['TribeltFor comparison', 'Twentebelt', 'Ashworth', 'Martens'])
    expect(list(wrapper).findAll('tbody td span[title]').map((c) => c.attributes('title'))).toEqual(['Tri-Flex', 'Tri-Flex', 'Tri-Flex'])
    expect(list(wrapper).findAll('tbody a').map((a) => a.attributes('href'))).toEqual(['/competitors/twentebelt', '/competitors/ashworth', '/competitors/martens'])
    expect(list(wrapper).text()).toContain('Tribelt Group is left out')
    expect(wrapper.find('a[href="/help/competitors"]').exists()).toBe(true)
  })

  it('sorts by any column, filters by name, belt family, and country, and shows the archived companies on request', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/me': () => json(me()) }), '/competitors')
    await list(wrapper).get('button[aria-label="Name, not sorted"]').trigger('click')
    expect(names(wrapper)).toEqual(['TribeltFor comparison', 'Ashworth', 'Martens', 'Twentebelt'])
    await wrapper.get('input[type="search"]').setValue('mar')
    expect(names(wrapper)).toEqual(['TribeltFor comparison', 'Martens'])
    await wrapper.get('input[type="search"]').setValue('')
    await wrapper.get('select[aria-label="Country"]').setValue('US')
    expect(names(wrapper)).toEqual(['TribeltFor comparison', 'Ashworth'])
    await wrapper.get('select[aria-label="Country"]').setValue('')
    await wrapper.get('select[aria-label="Belt family"]').setValue('tri_flex_turn_curve')
    expect(names(wrapper)).toEqual(['TribeltFor comparison', 'Martens', 'Twentebelt'])
    await wrapper.get('select[aria-label="Belt family"]').setValue('')
    const archived = wrapper.get('button[aria-pressed]')
    expect(archived.text()).toBe('Show 1 archived company')
    await archived.trigger('click')
    expect(names(wrapper)).toContain('Old BeltsArchived')
    expect(archived.text()).toBe('Hide 1 archived company')
  })

  it('gives an administrator one way into Manage, and no tools on the page itself', async () => {
    const viewer = await mountApp(api({ 'GET /api/v1/me': () => json(me()) }), '/competitors')
    expect(viewer.wrapper.find('a[href="/manage/competitors"]').exists()).toBe(false)
    viewer.wrapper.unmount()
    const { wrapper } = await mountApp(api(), '/competitors')
    expect(wrapper.get('a[href="/manage/competitors"]').text()).toBe('Manage the competitor list')
    expect(wrapper.find('form[aria-label="Scan a website"]').exists()).toBe(false)
    expect(wrapper.find('#admin-tools-title').exists()).toBe(false)
    expect(wrapper.findAll('button').some((b) => b.text() === 'Add a company')).toBe(false)
  })

  it('says while it loads, and when the companies cannot be loaded', async () => {
    let answer: (response: Response) => void = () => undefined
    const pending = new Promise<Response>((resolve) => {
      answer = resolve
    })
    const loading = await mountApp(api({ 'GET /api/v1/competitors': () => pending }), '/competitors')
    expect(loading.wrapper.text()).toContain('Loading the competitors…')
    answer(json({ items: [twentebelt] }))
    await settle()
    expect(loading.wrapper.text()).not.toContain('Loading the competitors…')
    loading.wrapper.unmount()
    const failing = await mountApp(api({ 'GET /api/v1/competitors': () => problem(500, 'Internal Server Error') }), '/competitors')
    expect(failing.wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
    expect(failing.wrapper.find('section[aria-labelledby="list-title"]').exists()).toBe(false)
  })
})
