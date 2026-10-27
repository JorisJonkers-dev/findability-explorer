import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { competitor, me, promptResults, readiness, readinessRow, summary } from '../../../testing/fixtures'
import { type Handler, json, mountApp, routes, settle } from '../../../testing/mountApp'

function api(admin = false, extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: admin })),
    'GET /api/v1/competitors': () =>
      json({
        items: [
          competitor('Twentebelt', { scans: 2, lastScan: summary(), websiteUrl: 'https://www.twentebelt.com/', country: 'NL' }),
          competitor('Unibelt', { scans: 0, websiteUrl: 'https://unibelt.nl/' }),
          competitor('Tribelt B.V.', { id: 'tribelt', role: 'tribelt', scans: 1, lastScan: summary() }),
          competitor('De Spiraal', { role: 'group', scans: 1, lastScan: summary() }),
          competitor('Mirror of tribelt.nl', { role: 'mirror' }),
          competitor('Gone', { archived: true }),
        ],
      }),
    'GET /api/v1/ai-readiness': () => json({ items: [readinessRow('Twentebelt', { readiness: readiness({ markdown: false }) })] }),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'GET /api/v1/terms': () => json({ items: [], total: 0, facets: { languages: [], families: [], tiers: [] } }),
    ...extra,
  })
}

const names = (wrapper: VueWrapper) => wrapper.findAll('tbody th[scope="row"] a').map((a) => a.text())

describe('All competitors', () => {
  it('lists every company with its website, readiness, share of answers and last scan, Tribelt first', async () => {
    const { wrapper } = await mountApp(api(), '/competitors/all')
    expect(names(wrapper)).toEqual(['Tribelt B.V.', 'Twentebelt', 'Unibelt', 'De Spiraal'])
    const twente = wrapper.findAll('tbody tr')[1]
    expect(twente?.get('th a').attributes('href')).toBe('/competitors/twentebelt')
    expect(twente?.get('td a').attributes('target')).toBe('_blank')
    expect(twente?.get('td a').text()).toBe('twentebelt.com')
    expect(twente?.findAll('li').map((l) => l.text())).toEqual([
      '●crawlers: Lets every AI crawler in',
      '●sitemap: Has a sitemap',
      '●llms.txt: Has llms.txt',
      '●llms-full: Has llms-full.txt',
      '–Markdown: No Markdown pages',
    ])
    expect(twente?.text()).toContain('75%')
    expect(twente?.text()).toContain('Competitor. Netherlands')
    expect(wrapper.findAll('tbody tr')[2]?.text()).toContain('Never')
    expect(wrapper.find('tbody button').exists()).toBe(false)
  })

  it('filters, keeping the filter in the address', async () => {
    const { wrapper, router } = await mountApp(api(), '/competitors/all')
    await wrapper.findAll('[aria-label="Show"] button').find((b) => b.text() === 'Never scanned')?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ show: 'never' })
    expect(names(wrapper)).toEqual(['Unibelt'])
    await wrapper.findAll('[aria-label="Show"] button').find((b) => b.text() === 'All')?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({})
  })

  it('lets an administrator scan one, the never scanned, or all again', async () => {
    const sent: unknown[] = []
    const { wrapper } = await mountApp(
      api(true, {
        'POST /api/v1/admin/rescans': async (r) => {
          sent.push(await r.json())
          return json({ queued: [], skipped: [] }, 202)
        },
      }),
      '/competitors/all',
    )
    const button = (text: string) => wrapper.findAll('button').find((b) => b.text().startsWith(text))
    expect(button('Scan the')?.text()).toBe('Scan the 1 never scanned')
    await button('Scan the')?.trigger('click')
    await settle()
    await button('Scan all again')?.trigger('click')
    await settle()
    await wrapper.findAll('tbody tr')[2]?.get('button').trigger('click')
    await settle()
    expect(sent).toEqual([{ unscanned: true }, { all: true }, { competitorIds: ['unibelt'] }])
    expect(wrapper.text()).toContain('Queued 0 scans.')
  })
})
