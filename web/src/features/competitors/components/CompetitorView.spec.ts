import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { competitor, me, overview, promptResults, readinessRow, readinessTable, summary, term, termSearch } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const SCAN_ID = '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d'
const tribelt = competitor('Tribelt', { role: 'tribelt' })
const twentebelt = competitor('Twentebelt', { scans: 2, lastScan: summary() })
const martens = competitor('Martens', { scans: 0, websiteUrl: undefined, host: undefined, website: '', beltNames: [], lastScan: undefined })

function api(extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/competitors': () => json({ items: [tribelt, twentebelt, martens] }),
    'GET /api/v1/competitors/twentebelt': () => json({ competitor: twentebelt, scans: [summary({ id: SCAN_ID, status: 'running' }), summary()] }),
    'GET /api/v1/competitors/martens': () => json({ competitor: martens, scans: [] }),
    'GET /api/v1/terms': () => json(termSearch([term({ term: 'TwenteFlex', usedBy: ['Twentebelt'] })])),
    'GET /api/v1/overview': () => json(overview()),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'GET /api/v1/ai-readiness': () => json(readinessTable([readinessRow('Tribelt', { role: 'tribelt' }), readinessRow('Twentebelt')])),
    ...extra,
  })
}

const link = (wrapper: VueWrapper, text: string) => wrapper.findAll('a').find((a) => a.text() === text)

describe('a competitor’s page', () => {
  it('sets the competitor against Tribelt, with its belts and the ways to look closer', async () => {
    const { wrapper } = await mountApp(api(), '/competitors/twentebelt')
    expect(document.title).toBe('Competitor · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Twentebelt is named in three times as many AI answers as Tribelt.')
    expect(wrapper.text()).toContain('Netherlands. Its website is in 2 languages. Last scan')
    expect(wrapper.get('section[aria-label="Key figures"]').text()).toContain('75%of AI answers name itTribelt: 25%.')
    expect(wrapper.get('section[aria-labelledby="compare-title"] h2').text()).toBe('Twentebelt against Tribelt')
    const belts = wrapper.get('section[aria-labelledby="belts-title"]')
    expect(belts.findAll('li').map((li) => li.text())).toContain('Tri-FlexTwenteFlex spiral belts')
    expect(belts.get('a[href="/terms/families/eyelink"]').text()).toBe('Eyelink')
    const visit = wrapper.get('a[target="_blank"]')
    expect(visit.attributes('href')).toBe('https://www.twentebelt.com/')
    expect(visit.attributes('rel')).toBe('noopener noreferrer')
    expect(wrapper.get('a[href="/competitors/twentebelt/history"]').text()).toBe('See its scan history')
    expect(wrapper.text()).toContain('Back to Competitors')
  })

  it('opens its newest finished scan', async () => {
    const { wrapper, router } = await mountApp(api(), '/competitors/twentebelt')
    await link(wrapper, 'Open its latest scan')?.trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/scan/5b0c1a9e-2d3f-4e5a-8b6c-7d8e9f0a1b2c')
  })

  it('says when no scan has read the website, and has no latest scan to open', async () => {
    const { wrapper } = await mountApp(api(), '/competitors/martens')
    expect(wrapper.text()).toContain('No scan of its website has run yet.')
    expect(wrapper.find('a[target="_blank"]').exists()).toBe(false)
    expect(link(wrapper, 'Open its latest scan')).toBeUndefined()
  })

  it('says when no company has the id, or the company cannot be loaded', async () => {
    const missing = await mountApp(api({ 'GET /api/v1/competitors/nobody': () => problem(404, 'Not Found') }), '/competitors/nobody')
    expect(missing.wrapper.get('[role="alert"]').text()).toBe('No company has this id.')
    missing.wrapper.unmount()
    const failing = await mountApp(api({ 'GET /api/v1/competitors/twentebelt': () => problem(500, 'Internal Server Error') }), '/competitors/twentebelt')
    expect(failing.wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
  })

  it('shows no controls to an account that is no administrator', async () => {
    const { wrapper } = await mountApp(api(), '/competitors/twentebelt')
    expect(wrapper.findAll('button').some((b) => b.text() === 'Scan again')).toBe(false)
    expect(wrapper.find('[aria-label="Manage this company"]').exists()).toBe(false)
  })

  it('lets an administrator scan it again, and says what was queued', async () => {
    let sent: unknown = null
    const { wrapper } = await mountApp(
      api({
        'GET /api/v1/me': () => json(me({ isAdmin: true })),
        'POST /api/v1/admin/rescans': async (request) => {
          sent = await request.json()
          return json({ queued: [summary()], skipped: [] }, 202)
        },
      }),
      '/competitors/twentebelt',
    )
    await wrapper.findAll('button').find((b) => b.text() === 'Scan again')?.trigger('click')
    await settle()
    expect(sent).toEqual({ competitorIds: ['twentebelt'] })
    expect(wrapper.get('[aria-live="polite"]').text()).toBe('Queued 1 scan.')
    expect(wrapper.find('[aria-label="Manage this company"]').exists()).toBe(true)
  })

  it('offers an administrator to scan a website no scan has read, and says when the request fails', async () => {
    const withSite = { ...martens, websiteUrl: 'https://www.martens-belts.com/' }
    const { wrapper } = await mountApp(
      api({
        'GET /api/v1/me': () => json(me({ isAdmin: true })),
        'GET /api/v1/competitors/martens': () => json({ competitor: withSite, scans: [] }),
        'GET /api/v1/competitors': () => json({ items: [tribelt, twentebelt, withSite] }),
        'POST /api/v1/admin/rescans': () => problem(500, 'Internal Server Error'),
      }),
      '/competitors/martens',
    )
    await wrapper.findAll('button').find((b) => b.text() === 'Scan this website')?.trigger('click')
    await settle()
    expect(wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
  })

  it('shows what a measure means when it is picked', async () => {
    const { wrapper } = await mountApp(api(), '/competitors/twentebelt')
    const chart = wrapper.get('section[aria-labelledby="compare-title"]')
    await chart.findAll('button').find((b) => b.text().startsWith('AI readiness checks passed'))?.trigger('click')
    expect(chart.get('[aria-live="polite"]').text()).toContain('Both sites pass 8 of 8 of the AI readiness checks.')
  })
})
