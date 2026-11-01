import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { bots, details, me, readiness, readinessRow, readinessTable } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const tribelt = readinessRow('Tribelt B.V.', {
  competitorId: 'tribelt',
  role: 'tribelt',
  readiness: readiness({ bots: bots({ GPTBot: 'blocked' }), llmsFullTxt: false, checkedAt: '2026-10-28T08:00:00Z' }),
})
const many = Array.from({ length: 18 }, (_, i) => readinessRow(`Belts ${String(i + 1).padStart(2, '0')}`, { readiness: readiness({ markdown: i % 2 === 0, stack: {} }) }))
const table = [
  readinessRow('Twentebelt', { readiness: readiness({ llmsFullTxt: false, markdown: false, llmsTxt: { present: false, bytes: 0, links: 0 } }) }),
  readinessRow('ESFO'),
  readinessRow('Unibelt', { readiness: undefined, scan: undefined }),
  readinessRow('Lesjöfors', { role: 'group' }),
  readinessRow('Mirror of tribelt.nl', { competitorId: 'tribelt-mirror', role: 'mirror' }),
  tribelt,
  ...many,
]

const plan = {
  catalogueVersion: 1,
  improvements: [{ id: 'llms-files', title: 'Add llms.txt and llms-full.txt', moves: 'Improves AI readiness.', measure: 'readinessChecks', auto: true, done: false, ...details, why: 'Because.' }],
  watched: [],
  eyeOn: [],
  readinessChecks: 8,
}

const api = (extra: Record<string, Handler> = {}) =>
  routes({
    'GET /api/v1/ai-readiness': () => json(readinessTable(table)),
    'GET /api/v1/plan': () => json(plan),
    ...extra,
  })

const section = (wrapper: VueWrapper, title: string) => {
  const h2 = wrapper.findAll('h2').find((h) => h.text() === title)
  if (!h2) throw new Error(`no section ${title}`)
  return wrapper.get(`section[aria-labelledby="${h2.attributes('id') ?? ''}"]`)
}

describe('AI readiness', () => {
  it('is framed with its title, its purpose, and where its figures come from', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness')
    expect(document.title).toBe('AI readiness · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('AI readiness')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('How well search engines and AI assistants read, understand, and cite each website, scored out of eight checks.')
    expect(wrapper.get('[data-part="data-line"]').text()).toBe("From each site's newest scan, checked again on 28 October 2026. 23 sites.")
    expect(wrapper.findAll('h2').map((h) => h.text())).toEqual(['Tribelt’s checks', 'How tribelt.nl compares with the competitors', 'Every site'])
    expect(wrapper.find('a[href="/help/websites"]').exists()).toBe(true)
  })

  it("lists Tribelt's eight checks, each passed or not with what it is, and links a failed one to its fix", async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness')
    const checks = section(wrapper, 'Tribelt’s checks')
    expect(checks.text()).toContain('tribelt.nl passes 6½ of the 8 checks.')
    const items = checks.findAll('li')
    expect(items).toHaveLength(8)
    expect(items[0]?.text()).toContain('AI crawlers let in Partly passed, as some crawlers are kept out.')
    expect(items[0]?.get('a').attributes('href')).toBe('/ai-readiness/tribelt')
    expect(items[1]?.text()).toContain('A sitemap Passed.')
    expect(items[1]?.text()).toContain('A sitemap lists every page, so a crawler finds pages that no link leads to.')
    expect(items[1]?.find('a').exists()).toBe(false)
    const full = items[6]
    expect(full?.text()).toContain('llms-full.txt Not passed.')
    expect(full?.get('a').text()).toBe('Recommended fix, add llms.txt and llms-full.txt')
    expect(full?.get('a').attributes('href')).toBe('/recommendations/llms-files')
  })

  it('compares tribelt.nl with the competitors, check by check, with a bar', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness')
    const compare = section(wrapper, 'How tribelt.nl compares with the competitors')
    expect(compare.text()).toContain('For each check, how many of the 20 competitors pass it, and whether tribelt.nl does.')
    const rows = compare.findAll('tbody tr').map((r) => r.findAll('th, td').map((c) => c.text()))
    expect(rows).toHaveLength(8)
    expect(rows[0]).toEqual(['AI crawlers let in', 'Fails', '20 of 20'])
    expect(rows[6]).toEqual(['llms-full.txt', 'Fails', '19 of 20'])
    expect(rows[7]).toEqual(['Markdown pages', 'Passes', '10 of 20'])
    expect(compare.findAll('[data-bar]')).toHaveLength(8)
  })

  it('shows every competitor, Tribelt first, 15 rows until Show all, and every site on request', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness')
    const sites = section(wrapper, 'Every site')
    const names = () => section(wrapper, 'Every site').findAll('tbody th[scope="row"]').map((th) => th.text())
    expect(names()).toHaveLength(15)
    expect(names()[0]).toBe('Tribelt')
    expect(sites.findAll('thead th').map((th) => th.text())).toEqual([
      'Site',
      'Checks passed',
      'AI crawlers let in',
      'A sitemap',
      'A sitemap in every language',
      'Pages in Dutch, German, and English',
      'Structured data about the company',
      'llms.txt',
      'llms-full.txt',
      'Markdown pages',
      'Platform',
    ])
    const first = sites.get('tbody tr')
    expect(first.get('a').attributes('href')).toBe('/ai-readiness/tribelt')
    expect(first.text()).toContain('6½/8')
    expect(first.findAll('[data-mark]').map((m) => m.attributes('data-mark'))).toEqual(['partly', 'yes', 'yes', 'yes', 'yes', 'yes', 'no', 'yes'])
    expect(sites.text()).toContain('Showing 15 of 21.')
    const showAll = sites.findAll('button').find((b) => b.text() === 'Show all')
    await showAll?.trigger('click')
    expect(names()).toHaveLength(21)
    expect(names()).not.toContain('Lesjöfors')
    const all = sites.findAll('button').find((b) => b.text() === 'All sites')
    await all?.trigger('click')
    expect(all?.attributes('aria-pressed')).toBe('true')
    expect(names()).toContain('Lesjöfors')
    expect(names()).toContain('Mirror of tribelt.nl')
    expect(names()).not.toContain('Unibelt')
  })

  it('sorts by any column and finds a site by name', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness')
    const sites = section(wrapper, 'Every site')
    const names = () => sites.findAll('tbody th[scope="row"]').map((th) => th.text())
    await sites.get('button[aria-label^="Site,"]').trigger('click')
    expect(names().slice(0, 3)).toEqual(['Tribelt', 'Belts 01', 'Belts 02'])
    await sites.get('input[type="search"]').setValue('twente')
    expect(names()).toEqual(['Twentebelt'])
    await sites.get('input[type="search"]').setValue('nothing like it')
    expect(sites.text()).toContain('No sites match this filter.')
  })

  it('lets an administrator check every site again, and leads to the scans', async () => {
    let started = false
    const { wrapper } = await mountApp(
      api({
        'GET /api/v1/me': () => json(me({ isAdmin: true })),
        'GET /api/v1/ai-readiness': () => json(readinessTable(table, started ? { running: true, checked: 2, total: 5 } : {})),
        'POST /api/v1/admin/readiness-checks': () => {
          started = true
          return json({ running: true, checked: 0, failed: 0, total: 5 }, 202)
        },
      }),
      '/ai-readiness',
    )
    expect(wrapper.get('a[href="/manage/scans"]').text()).toBe('Manage the scans')
    const check = wrapper.findAll('button').find((b) => b.text() === 'Check AI readiness again')
    await check?.trigger('click')
    await settle()
    expect(wrapper.text()).toContain('Checking 2 of 5 sites again.')
  })

  it('shows no check and no Manage link to an account that is no administrator', async () => {
    const { wrapper } = await mountApp(api(), '/ai-readiness')
    expect(wrapper.findAll('button').some((b) => b.text() === 'Check AI readiness again')).toBe(false)
    expect(wrapper.find('a[href="/manage/scans"]').exists()).toBe(false)
  })

  it('says when tribelt.nl has not been scanned yet', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/ai-readiness': () => json(readinessTable([readinessRow('ESFO')])) }), '/ai-readiness')
    expect(section(wrapper, 'Tribelt’s checks').text()).toContain('tribelt.nl has not been scanned yet.')
    expect(section(wrapper, 'How tribelt.nl compares with the competitors').text()).toContain('tribelt.nl has not been scanned yet.')
  })

  it('opens the page of a site picked in an address of 1.12.0', async () => {
    const { router } = await mountApp(api(), '/ai-readiness?site=esfo')
    expect(router.currentRoute.value.path).toBe('/ai-readiness/esfo')
  })

  it('says when the table cannot load', async () => {
    const { wrapper } = await mountApp(routes({ 'GET /api/v1/ai-readiness': () => problem(500, 'Internal Server Error', 'Down.') }), '/ai-readiness')
    expect(wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
