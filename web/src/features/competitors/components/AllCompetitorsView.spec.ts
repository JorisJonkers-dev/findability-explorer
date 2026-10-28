import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { competitor, me, promptResults, readiness, readinessRow, summary, readinessTable } from '../../../testing/fixtures'
import { type Handler, json, mountApp as mountRouted, problem, routes, settle } from '../../../testing/mountApp'

/** The list is no longer routed, so the test gives it a route of its own. */
async function mountApp(handle: Handler, path: string) {
  const app = await mountRouted(handle, '/')
  app.router.addRoute({ path: '/all-competitors', component: () => import('./AllCompetitorsView.vue') })
  await app.router.push(path.replace('/competitors/all', '/all-competitors'))
  await settle()
  return app
}

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
    'GET /api/v1/ai-readiness': () => json(readinessTable([readinessRow('Twentebelt', { readiness: readiness({ markdown: false }) })])),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'GET /api/v1/terms': () => json({ items: [], total: 0, facets: { languages: [], families: [], tiers: [] } }),
    'GET /api/v1/admin/competitor-checks': () => json({ items: [], unchecked: 0, review: { running: false } }),
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
      '✓crawlers: Lets every AI crawler in',
      '✓sitemap: Has a sitemap',
      '✓llms.txt: Has llms.txt',
      '✓llms-full: Has llms-full.txt',
      '✕Markdown: No Markdown pages',
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

describe("Haiku's check of the list", () => {
  const rubber = competitor('Rubber Co', { id: 'rubber-co', archived: true, archivedBy: 'language model', archivedAt: '2026-10-27T09:00:00Z' })
  const checks = {
    items: [
      { competitorId: 'twentebelt', competitor: true, sure: true, reason: 'Makes eyelink belts for ovens.', model: 'haiku', at: '2026-10-27T09:00:00Z' },
      { competitorId: 'unibelt', competitor: false, sure: false, reason: 'Makes plastic modular belts.', model: 'haiku', at: '2026-10-27T09:00:00Z' },
      { competitorId: 'rubber-co', competitor: false, sure: true, reason: 'Makes rubber belting for quarries.', model: 'haiku', at: '2026-10-27T09:00:00Z' },
    ],
    unchecked: 2,
    review: { running: false },
  }

  function admin(extra: Record<string, Handler> = {}) {
    return api(true, {
      'GET /api/v1/competitors': () =>
        json({
          items: [
            competitor('Twentebelt', { scans: 2, lastScan: summary() }),
            competitor('Unibelt', { scans: 0, websiteUrl: 'https://unibelt.nl/' }),
            competitor('Kept Belts', { scans: 1, keptBy: 'admin-1' }),
            competitor('Fresh Belts', { scans: 1 }),
            competitor('Tribelt B.V.', { id: 'tribelt', role: 'tribelt', scans: 1 }),
            rubber,
            competitor('Gone', { archived: true, archivedBy: 'admin-1' }),
          ],
        }),
      'GET /api/v1/admin/competitor-checks': () => json(checks),
      ...extra,
    })
  }

  const filterButton = (wrapper: VueWrapper, text: string) => wrapper.findAll('[aria-label="Show"] button').find((b) => b.text() === text)

  it("shows an administrator each company's verdict and reason, and flags the unsure ones", async () => {
    const { wrapper } = await mountApp(admin(), '/competitors/all')
    const section = wrapper.get('section[aria-label="Haiku\'s check of the list"]')
    expect(section.text()).toContain('Haiku has not checked 2 companies on the list yet.')
    const row = (name: string) => wrapper.findAll('tbody tr').find((r) => r.get('th a').text() === name)
    expect(row('Twentebelt')?.text()).toContain('Haiku says competitorMakes eyelink belts for ovens.')
    expect(row('Unibelt')?.text()).toContain('Haiku is unsureMakes plastic modular belts.')
    expect(row('Unibelt')?.find('[role="group"][aria-label="Decide on Unibelt"]').exists()).toBe(true)
    expect(row('Twentebelt')?.find('[role="group"]').exists()).toBe(false)
    expect(row('Kept Belts')?.text()).toContain('Kept on the list')
    expect(row('Fresh Belts')?.findAll('td')[4]?.text()).toBe('Not checked')
    // The fifth cell is Haiku's: empty for Tribelt, which it never checks.
    expect(row('Tribelt B.V.')?.findAll('td')[4]?.text()).toBe('')
    expect(row('Twentebelt')?.findAll('td')[4]?.text()).toContain('Haiku says competitor')
    expect(filterButton(wrapper, 'Haiku is unsure (1)')).toBeDefined()
    await filterButton(wrapper, 'Haiku is unsure (1)')?.trigger('click')
    await settle()
    expect(names(wrapper)).toEqual(['Unibelt'])
  })

  it('keeps or archives a company Haiku is unsure of', async () => {
    const sent: unknown[] = []
    const { wrapper } = await mountApp(
      admin({
        'PATCH /api/v1/competitors/unibelt': async (r) => {
          sent.push(await r.json())
          return json(competitor('Unibelt'))
        },
      }),
      '/competitors/all',
    )
    const decide = wrapper.get('[aria-label="Decide on Unibelt"]')
    await decide.findAll('button')[0]?.trigger('click')
    await settle()
    expect(wrapper.text()).toContain('Unibelt stays on the list, and Haiku will not archive it.')
    await wrapper.get('[aria-label="Decide on Unibelt"]').findAll('button')[1]?.trigger('click')
    await settle()
    expect(sent).toEqual([{ kept: true }, { archived: true }])
  })

  it('lists what Haiku archived and why, each to restore', async () => {
    const sent: unknown[] = []
    const { wrapper, router } = await mountApp(
      admin({
        'PATCH /api/v1/competitors/rubber-co': async (r) => {
          sent.push(await r.json())
          return json({ ...rubber, archived: false, keptBy: 'admin-1' })
        },
      }),
      '/competitors/all',
    )
    await filterButton(wrapper, 'Archived by Haiku (1)')?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ show: 'archived' })
    expect(wrapper.find('table').exists()).toBe(false)
    const list = wrapper.get('ul[aria-label="Archived by Haiku"]')
    expect(list.findAll('li').map((l) => l.find('span').text())).toEqual(['Rubber CoMakes rubber belting for quarries.'])
    await list.get('button').trigger('click')
    await settle()
    expect(sent).toEqual([{ archived: false }])
    expect(wrapper.text()).toContain('Rubber Co is back on the list, and Haiku will not archive it again.')
  })

  it('says when Haiku archived nothing', async () => {
    const { wrapper } = await mountApp(admin({ 'GET /api/v1/competitors': () => json({ items: [competitor('Twentebelt')] }) }), '/competitors/all?show=archived')
    expect(wrapper.text()).toContain('Haiku has archived no company.')
  })

  it('starts a check of the list', async () => {
    let started = false
    const { wrapper } = await mountApp(
      admin({ 'POST /api/v1/admin/competitor-checks': () => ((started = true), json({ running: true }, 202)) }),
      '/competitors/all',
    )
    await wrapper.findAll('button').find((b) => b.text() === 'Check the list with Haiku')?.trigger('click')
    await settle()
    expect(started).toBe(true)
  })

  it('sends an administrator without a key to the settings, and says when the check is refused', async () => {
    const { wrapper } = await mountApp(admin({ 'GET /api/v1/admin/competitor-checks': () => json({ ...checks, review: { running: false, problem: 'noKey' } }) }), '/competitors/all')
    expect(wrapper.get('section[aria-label="Haiku\'s check of the list"] a').attributes('href')).toBe('/settings')
    wrapper.unmount()
    const refused = await mountApp(admin({ 'GET /api/v1/admin/competitor-checks': () => problem(500, 'Internal Server Error') }), '/competitors/all')
    expect(refused.wrapper.text()).toContain('Internal Server Error')
  })

  it('is not there for a viewer, nor its filters', async () => {
    const { wrapper } = await mountApp(api(), '/competitors/all?show=archived')
    expect(wrapper.find('section[aria-label="Haiku\'s check of the list"]').exists()).toBe(false)
    expect(wrapper.find('table').exists()).toBe(true)
    expect(filterButton(wrapper, 'Archived by Haiku (0)')).toBeUndefined()
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
