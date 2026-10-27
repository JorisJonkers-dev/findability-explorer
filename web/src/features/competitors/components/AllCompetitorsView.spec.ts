import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { competitor, me, promptResults, readiness, readinessRow, summary } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

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

describe('adding competitors', () => {
  const pending = {
    items: [
      { id: '3a7bd3e2360a9f01', name: 'Intralox', answers: 23, providers: ['openai'], models: [], addresses: ['https://www.intralox.com/en'], lastSeenAt: '2026-11-01T06:00:00Z', status: 'pending' },
      { id: '4b8ce4f3471b0a12', name: 'Habasit', answers: 14, providers: ['google'], models: [], addresses: [], lastSeenAt: '2026-11-01T06:00:00Z', status: 'pending' },
    ],
    counts: { pending: 2, added: 0, dismissed: 0, unchecked: 0 },
    review: { running: false },
  }

  function admin(extra: Record<string, Handler> = {}) {
    return api(true, { 'GET /api/v1/admin/competitor-candidates': () => json(pending), ...extra })
  }

  const tray = (wrapper: VueWrapper) => wrapper.findAll('ul[aria-label="To add"] li').map((l) => [l.get('span').text(), (l.get('input').element as HTMLInputElement).value])

  it('suggests as you type, marks a competitor already added, and picks by keyboard', async () => {
    const { wrapper } = await mountApp(admin(), '/competitors/all')
    const field = wrapper.get('input[role="combobox"]')
    await field.setValue('twente')
    await field.trigger('input')
    const options = () => wrapper.findAll('[role="option"]')
    expect(options()[0]?.text()).toBe('TwentebeltAlready a competitorAdded')
    expect(options().at(-1)?.text()).toBe('Add “twente” as a new companyAdd its website before adding itAdd')
    await options()[0]?.trigger('mousedown')
    expect(tray(wrapper)).toEqual([])
    await field.setValue('intra')
    await field.trigger('input')
    expect(options()[0]?.text()).toContain('Named in 23 AI answers')
    await field.trigger('keydown', { key: 'ArrowDown' })
    await field.trigger('keydown', { key: 'ArrowUp' })
    expect(field.attributes('aria-activedescendant')).toMatch(/-s0$/)
    await field.trigger('keydown', { key: 'Enter' })
    expect(tray(wrapper)).toEqual([['Intralox', 'https://www.intralox.com/']])
    await field.setValue('x')
    await field.trigger('input')
    await field.trigger('keydown', { key: 'Escape' })
    expect(field.attributes('aria-expanded')).toBe('false')
  })

  it('takes a pasted list, skipping competitors already added, and sends the tray in one request', async () => {
    let sent: unknown = null
    const { wrapper } = await mountApp(
      admin({
        'POST /api/v1/admin/competitors/batch': async (r) => {
          sent = await r.json()
          return json({
            items: [
              { name: 'Berndorf Band', outcome: 'added', competitorId: 'berndorf-band' },
              { name: 'Lumsden Belting', outcome: 'rejected', reason: 'Add its website first.' },
            ],
            scans: 1,
          })
        },
      }),
      '/competitors/all',
    )
    await wrapper.findAll('button').find((b) => b.text() === 'Paste a list instead')?.trigger('click')
    await wrapper.get('textarea').setValue('Berndorf Band, berndorfband-group.com\nTwentebelt, twentebelt.com\nLumsden Belting')
    await wrapper.findAll('button').find((b) => b.text() === 'Put them in the tray')?.trigger('click')
    expect(wrapper.text()).toContain('Left out as competitors already: Twentebelt (Twentebelt).')
    expect(tray(wrapper)).toEqual([
      ['Berndorf Band', 'https://berndorfband-group.com/'],
      ['Lumsden Belting', ''],
    ])
    await wrapper.findAll('button').find((b) => b.text() === 'Add 2 competitors')?.trigger('click')
    await settle()
    expect(sent).toEqual({ items: [{ name: 'Berndorf Band', url: 'https://berndorfband-group.com/' }, { name: 'Lumsden Belting' }], scan: true })
    expect(wrapper.text()).toContain('Added 1 competitor. Could not add 1. Started 1 scan.')
    expect(wrapper.get('ul[aria-label="Not added"]').text()).toBe('Lumsden Belting. Add its website first.')
    // What was refused stays to fix.
    expect(tray(wrapper)).toEqual([['Lumsden Belting', '']])
    await wrapper.findAll('ul[aria-label="To add"] button')[0]?.trigger('click')
    expect(wrapper.text()).toContain('Nothing picked yet.')
  })

  it('adds a named company in one click, or marks it as no competitor', async () => {
    let dismissed = false
    const { wrapper } = await mountApp(
      admin({ 'POST /api/v1/admin/competitor-candidates/4b8ce4f3471b0a12/dismiss': () => ((dismissed = true), json({ ...pending.items[1], status: 'dismissed' })) }),
      '/competitors/all',
    )
    const named = wrapper.get('ul[aria-label="Named in AI answers"]')
    expect(named.findAll('li').map((l) => l.find('span').text())).toEqual(['IntraloxNamed in 23 AI answers. intralox.com', 'HabasitNamed in 14 AI answers'])
    await named.findAll('li')[0]?.findAll('button')[0]?.trigger('click')
    await named.findAll('li')[0]?.findAll('button')[0]?.trigger('click')
    expect(tray(wrapper)).toEqual([['Intralox', 'https://www.intralox.com/']])
    await wrapper.findAll('button').find((b) => b.text() === 'Search one at a time instead' || b.text() === 'Paste a list instead')?.trigger('click')
    await named.findAll('li')[1]?.findAll('button')[1]?.trigger('click')
    await settle()
    expect(dismissed).toBe(true)
  })

  it('says when the batch is refused', async () => {
    const { wrapper } = await mountApp(
      admin({ 'POST /api/v1/admin/competitors/batch': () => problem(422, 'Unprocessable Entity', 'Add 1 to 100 companies at once.') }),
      '/competitors/all',
    )
    await wrapper.get('ul[aria-label="Named in AI answers"] li button').trigger('click')
    await wrapper.findAll('button').find((b) => b.text() === 'Add 1 competitor')?.trigger('click')
    await settle()
    expect(wrapper.get('[role="alert"]').text()).toBe('Add 1 to 100 companies at once.')
  })

  it('is not there for a viewer', async () => {
    const { wrapper } = await mountApp(api(), '/competitors/all')
    expect(wrapper.find('#add-title').exists()).toBe(false)
  })
})
