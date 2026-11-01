import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { analysis, competitor, readinessRow, readinessTable, summary } from '../../../testing/fixtures'
import { buttonNamed, manageApi } from '../../../testing/manageApi'
import { type Handler, json, mountApp, settle } from '../../../testing/mountApp'

const NEW_ID = '6c1d2b0f-3e4a-4f6b-9c7d-8e9f0a1b2c3d'
const companies = [
  competitor('Tribelt', { role: 'tribelt', websiteUrl: 'https://www.tribelt.nl/', scans: 1, lastScan: summary({ createdAt: '2026-10-20T08:00:00Z' }) }),
  competitor('Old Belts', { scans: 1, lastScan: summary({ createdAt: '2026-08-01T08:00:00Z' }) }),
  competitor('New Belts', { scans: 1, lastScan: summary({ createdAt: '2099-10-27T08:00:00Z' }) }),
  competitor('Never Belts', { scans: 0 }),
]
const widget = (wrapper: VueWrapper, title: string) => {
  const found = wrapper.findAll('section').find((s) => s.find('h2').exists() && s.get('h2').text() === title)
  if (!found) throw new Error(`no widget ${title}`)
  return found
}

function server(extra: Record<string, Handler> = {}) {
  const bodies: Record<string, unknown> = {}
  const handle = manageApi(
    {
      'GET /api/v1/analyses': () => json({ items: [summary()] }),
      'GET /api/v1/ai-readiness': () => json(readinessTable([readinessRow('Old Belts'), readinessRow('New Belts', { scan: undefined })])),
      'POST /api/v1/analyses': async (request) => {
        bodies.scan = await request.json()
        return json(analysis({}, { id: NEW_ID, status: 'queued' }), 202)
      },
      [`GET /api/v1/analyses/${NEW_ID}`]: () => json(analysis({}, { id: NEW_ID })),
      'POST /api/v1/admin/rescans': async (request) => {
        bodies.rescan = await request.json()
        return json({ queued: [summary({ status: 'queued' })], skipped: [] }, 202)
      },
      'POST /api/v1/admin/readiness-checks': () => ((bodies.check = true), json({ running: true, checked: 0, failed: 0, total: 1 }, 202)),
      ...extra,
    },
    companies,
  )
  return { handle, bodies }
}

describe('Website scans', () => {
  it('opens on the last scan, the budget, the readiness check, the four starts, and a card per subpage', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage/scans')
    expect(document.title).toBe('Website scans · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Website scans')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('Read the websites of Tribelt and its competitors.')
    expect(wrapper.get('header a[href="/ai-readiness"]').text()).toBe('See AI readiness')
    expect(wrapper.findAll('section > div > h2').map((h) => h.text())).toEqual([
      'Last scan',
      'Language model budget',
      'AI readiness check',
      'Scan tribelt.nl',
      'Scan one site',
      'Rescan competitors',
      'Check AI readiness again',
      'Running now',
      'Scan queue',
      'Recent scans',
      'Scan settings',
    ])
    expect(widget(wrapper, 'Last scan').text()).toContain('27 October 2099')
    expect(widget(wrapper, 'Language model budget').text()).toContain('48,211 of 10,000,000 tokens used this month')
    expect(widget(wrapper, 'AI readiness check').text()).toContain('Not checked again since the app last started.')
    expect(widget(wrapper, 'Scan tribelt.nl').text()).toContain("About 29,137 tokens. 9,951,789 of this month's tokens are left.")
    expect(widget(wrapper, 'Check AI readiness again').text()).toContain('Free. 1 sites.')
    expect(widget(wrapper, 'Running now').text()).toContain('No scan is running.')
    expect(widget(wrapper, 'Recent scans').get('a').attributes('href')).toBe('/manage/scans/recent')
  })

  it('scans tribelt.nl and opens the scan', async () => {
    const { handle, bodies } = server()
    const { wrapper, router } = await mountApp(handle, '/manage/scans')
    await buttonNamed(wrapper, 'Scan tribelt.nl').trigger('click')
    await settle()
    expect(bodies.scan).toEqual({ url: 'https://www.tribelt.nl/', maxPages: 60 })
    expect(router.currentRoute.value.fullPath).toBe(`/manage/scans/${NEW_ID}`)
  })

  it('rescans the competitors of the chosen group, naming them', async () => {
    const { handle, bodies } = server()
    const { wrapper } = await mountApp(handle, '/manage/scans')
    const group = wrapper.get('[role="group"][aria-label="Which competitors"]')
    expect(group.findAll('button').map((b) => [b.text(), b.attributes('aria-pressed')])).toEqual([
      ['All (3)', 'false'],
      ['Over 30 days old (2)', 'true'],
      ['Never scanned (1)', 'false'],
    ])
    await group.findAll('button')[2]?.trigger('click')
    await buttonNamed(wrapper, 'Rescan these competitors').trigger('click')
    await settle()
    expect(bodies.rescan).toEqual({ competitorIds: ['never-belts'], maxPages: 60 })
    expect(widget(wrapper, 'Rescan competitors').text()).toContain('Queued 1 scan.')
  })

  it('checks AI readiness again, and shows the check while it runs', async () => {
    let running = false
    const { handle, bodies } = server({ 'GET /api/v1/ai-readiness': () => json(readinessTable([], running ? { running: true, checked: 1, total: 4 } : {})) })
    const { wrapper } = await mountApp(handle, '/manage/scans')
    running = true
    await buttonNamed(wrapper, 'Check AI readiness again').trigger('click')
    await settle()
    expect(bodies.check).toBe(true)
    expect(widget(wrapper, 'AI readiness check').text()).toContain('Checking, 1 of 4 sites.')
    expect(wrapper.find('[data-part="readiness-check"]').exists()).toBe(true)
  })

  it('says a stopped check stopped, and that an empty group has nobody to rescan', async () => {
    const { wrapper } = await mountApp(
      server({ 'GET /api/v1/ai-readiness': () => json(readinessTable([], { checked: 2, total: 4, stopped: true })) }).handle,
      '/manage/scans',
    )
    expect(widget(wrapper, 'AI readiness check').text()).toContain('2 of 4 sites checked, then stopped by hand.')
    wrapper.unmount()
    const none = await mountApp(manageApi({ 'GET /api/v1/analyses': () => json({ items: [] }) }, []), '/manage/scans')
    expect(widget(none.wrapper, 'Last scan').text()).toContain('None yet.')
    expect(widget(none.wrapper, 'Rescan competitors').text()).toContain('No company is in this group.')
    expect(buttonNamed(none.wrapper, 'Scan tribelt.nl').attributes('disabled')).toBeDefined()
  })
})
