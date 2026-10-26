import type { VueWrapper } from '@vue/test-utils'
import { flushPromises } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { RescanBatch } from '../../../infrastructure/api/types.gen'
import { competitor, me, summary } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { POLL_INTERVAL_MS } from '../../scan/domain/status'

afterEach(() => {
  vi.useRealTimers()
})

const twentebelt = competitor('Twentebelt', { scans: 2 })
const added = competitor('Example Belts', {
  id: 'example-belts',
  source: 'added',
  scans: 1,
  addedAt: '2026-10-22T09:30:00Z',
  websiteUrl: 'https://www.example-belts.com/',
  host: 'example-belts.com',
})

type Sent = { method: string; path: string; body: unknown }

/** The server as an administrator sees it, recording every change sent. */
function api(sent: Sent[], extra: Record<string, Handler> = {}) {
  const record = (answer: () => Response): Handler => async (request) => {
    const text = await request.text()
    sent.push({ method: request.method, path: new URL(request.url).pathname, body: text === '' ? null : JSON.parse(text) })
    return answer()
  }
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/competitors': () => json({ items: [twentebelt, added] }),
    'GET /api/v1/terms': () => json({ items: [], total: 0, facets: { languages: [], tiers: [], families: [] } }),
    'GET /api/v1/competitors/twentebelt': () => json({ competitor: twentebelt, scans: [summary()] }),
    'GET /api/v1/competitors/example-belts': () => json({ competitor: added, scans: [] }),
    'GET /api/v1/analyses': () => json({ items: [] }),
    'PATCH /api/v1/competitors/example-belts': record(() => json({ ...added, name: 'Example Belts B.V.' })),
    'DELETE /api/v1/competitors/example-belts': record(() => new Response(null, { status: 204 })),
    'DELETE /api/v1/competitors/twentebelt': record(() => new Response(null, { status: 204 })),
    'POST /api/v1/competitors': record(() => json(competitor('New Belts', { id: 'new-belts', source: 'added' }), 201)),
    'POST /api/v1/admin/rescans': record(() =>
      json({
        queued: [summary(), summary({ id: '6c1d2b0f-3e4a-4f6b-9c7d-8e9f0a1b2c3d' })],
        skipped: [{ competitorId: 'martens', name: 'Märtens', reason: 'A scan of it is still queued or running.' }],
      } satisfies RescanBatch, 202),
    ),
    ...extra,
  })
}

const button = (wrapper: VueWrapper, text: string) => {
  const found = wrapper.findAll('button').find((b) => b.text() === text)
  if (!found) throw new Error(`no button ${text}`)
  return found
}

describe('managing a company', () => {
  it('edits an added company, sending only what changed, and refuses a bad address before sending', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/competitors?company=example-belts')
    await button(wrapper, 'Edit').trigger('click')
    await settle()
    const form = wrapper.get('form[aria-label="Edit Example Belts"]')
    expect(document.activeElement).toBe(form.get('input').element)
    const [name, site] = form.findAll('input')
    await site?.setValue('https://intranet')
    await form.trigger('submit')
    expect(form.get('[role="alert"]').text()).toContain('full host name')
    await site?.setValue('https://www.example-belts.com/')
    await name?.setValue('   ')
    await form.trigger('submit')
    expect(form.get('[role="alert"]').text()).toBe('Enter the company’s name.')
    await name?.setValue('Example Belts B.V.')
    await form.trigger('submit')
    await settle()
    expect(sent).toEqual([{ method: 'PATCH', path: '/api/v1/competitors/example-belts', body: { name: 'Example Belts B.V.' } }])
    expect(wrapper.find('form[aria-label="Edit Example Belts"]').exists()).toBe(false)
  })

  it('closes the form without a request when nothing changed, or on cancel', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/competitors?company=example-belts')
    await button(wrapper, 'Edit').trigger('click')
    await settle()
    await wrapper.get('form[aria-label="Edit Example Belts"]').trigger('submit')
    await settle()
    expect(wrapper.find('form[aria-label="Edit Example Belts"]').exists()).toBe(false)
    await button(wrapper, 'Edit').trigger('click')
    await settle()
    await button(wrapper, 'Cancel').trigger('click')
    expect(wrapper.find('form[aria-label="Edit Example Belts"]').exists()).toBe(false)
    expect(sent).toEqual([])
  })

  it('says why the server refused a change', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(
      api(sent, { 'PATCH /api/v1/competitors/example-belts': () => problem(409, 'Conflict', 'Twentebelt already has this website.') }),
      '/competitors?company=example-belts',
    )
    await button(wrapper, 'Edit').trigger('click')
    await settle()
    const form = wrapper.get('form[aria-label="Edit Example Belts"]')
    await form.findAll('input')[1]?.setValue('https://www.twentebelt.com')
    await form.trigger('submit')
    await settle()
    expect(wrapper.get('section[aria-labelledby="company-name"] [role="alert"]').text()).toBe('Twentebelt already has this website.')
  })

  it('deletes an added company with its scans after confirming, and shows the first company again', async () => {
    const sent: Sent[] = []
    const { wrapper, router } = await mountApp(api(sent), '/competitors?company=example-belts')
    await button(wrapper, 'Remove').trigger('click')
    await settle()
    const confirm = wrapper.get('[role="group"][aria-label="Remove Example Belts?"]')
    expect(confirm.text()).toContain('Delete this company with its 1 scan? This cannot be undone.')
    expect(document.activeElement?.textContent.trim()).toBe('Cancel')
    await button(wrapper, 'Cancel').trigger('click')
    expect(sent).toEqual([])
    await button(wrapper, 'Remove').trigger('click')
    await settle()
    await button(wrapper, 'Delete with its 1 scan').trigger('click')
    await settle()
    expect(sent).toEqual([{ method: 'DELETE', path: '/api/v1/competitors/example-belts', body: null }])
    expect(router.currentRoute.value.fullPath).toBe('/competitors')
  })

  it('archives a researched company instead of deleting it, and says when it is refused', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent, { 'DELETE /api/v1/competitors/twentebelt': () => problem(403, 'Forbidden') }), '/competitors')
    await button(wrapper, 'Archive').trigger('click')
    await settle()
    expect(wrapper.text()).toContain('Archive this researched company?')
    await button(wrapper, 'Archive').trigger('click')
    await settle()
    expect(wrapper.get('section[aria-labelledby="company-name"] [role="alert"]').text()).toBe('Only administrators can change the companies.')
  })

  it('restores an archived company', async () => {
    const sent: Sent[] = []
    const archived = { ...added, archived: true }
    const { wrapper } = await mountApp(
      api(sent, {
        'GET /api/v1/competitors': () => json({ items: [twentebelt, archived] }),
        'GET /api/v1/competitors/example-belts': () => json({ competitor: archived, scans: [] }),
      }),
      '/competitors?company=example-belts',
    )
    await button(wrapper, 'Restore').trigger('click')
    await settle()
    expect(sent).toEqual([{ method: 'PATCH', path: '/api/v1/competitors/example-belts', body: { archived: false } }])
  })
})

describe('the administrator tools', () => {
  it('re-scans every company with data at the page budget set, and says what was queued and skipped', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/competitors')
    const budget = wrapper.get('input[type="number"]')
    await budget.setValue('250')
    await budget.trigger('change')
    expect((budget.element as HTMLInputElement).value).toBe('100')
    await budget.setValue('40')
    await budget.trigger('change')
    // Twentebelt is researched and the added company was scanned: both have data.
    await button(wrapper, 'Re-scan all 2 companies with data').trigger('click')
    expect(wrapper.get('#rescan-question').text()).toContain('Re-scan 2 companies, up to 40 pages each?')
    await button(wrapper, 'Cancel').trigger('click')
    expect(wrapper.find('#rescan-question').exists()).toBe(false)
    expect(sent).toEqual([])
    await button(wrapper, 'Re-scan all 2 companies with data').trigger('click')
    await button(wrapper, 'Re-scan 2 companies').trigger('click')
    await settle()
    expect(sent).toEqual([{ method: 'POST', path: '/api/v1/admin/rescans', body: { all: true, maxPages: 40 } }])
    expect(wrapper.text()).toContain('Queued 2 scans. Skipped 1: Märtens (a scan of it is still queued or running).')
  })

  it('follows the queued scans until each has finished', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'] })
    const sent: Sent[] = []
    let finished = false
    const { wrapper } = await mountApp(
      api(sent, {
        'GET /api/v1/analyses': () =>
          json({ items: [summary({ status: finished ? 'done' : 'running' }), summary({ id: '6c1d2b0f-3e4a-4f6b-9c7d-8e9f0a1b2c3d', status: finished ? 'failed' : 'queued' })] }),
      }),
      '/competitors',
    )
    await button(wrapper, 'Re-scan all 2 companies with data').trigger('click')
    await button(wrapper, 'Re-scan 2 companies').trigger('click')
    await settle()
    expect(wrapper.text()).toContain('Re-scanning 2 companies: 0 done, 1 running, 1 waiting.')
    expect(button(wrapper, 'Re-scan all 2 companies with data').attributes('disabled')).toBeDefined()
    finished = true
    await vi.advanceTimersByTimeAsync(POLL_INTERVAL_MS)
    await flushPromises()
    expect(wrapper.text()).toContain('Re-scan finished: 1 done, 1 failed.')
  })

  it('adds a company by name and website, checks both first, and shows it', async () => {
    const sent: Sent[] = []
    const { wrapper, router } = await mountApp(api(sent), '/competitors')
    const toggle = button(wrapper, 'Add a company')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    await toggle.trigger('click')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    const form = wrapper.get('form[aria-label="Add a company"]')
    await form.trigger('submit')
    expect(form.get('[role="alert"]').text()).toBe('Enter the company’s name.')
    expect(form.findAll('input')[0]?.attributes('aria-invalid')).toBe('true')
    await form.findAll('input')[0]?.setValue('New Belts')
    await form.trigger('submit')
    expect(form.get('[role="alert"]').text()).toBe('Enter the address of a website.')
    await form.findAll('input')[1]?.setValue('new-belts.com')
    await form.trigger('submit')
    await settle()
    expect(sent).toEqual([{ method: 'POST', path: '/api/v1/competitors', body: { name: 'New Belts', url: 'https://new-belts.com' } }])
    expect(form.text()).toContain('Added New Belts.')
    expect(router.currentRoute.value.query.company).toBe('new-belts')
  })

  it('says why a company could not be added or the re-scan refused', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(
      api(sent, {
        'POST /api/v1/competitors': () => problem(409, 'Conflict', 'Twentebelt already has this website.'),
        'POST /api/v1/admin/rescans': () => problem(500, 'Internal Server Error'),
      }),
      '/competitors',
    )
    await button(wrapper, 'Add a company').trigger('click')
    const form = wrapper.get('form[aria-label="Add a company"]')
    await form.findAll('input')[0]?.setValue('Twente')
    await form.findAll('input')[1]?.setValue('https://www.twentebelt.com')
    await form.trigger('submit')
    await settle()
    expect(form.get('[role="alert"]').text()).toBe('Twentebelt already has this website.')
    await button(wrapper, 'Re-scan all 2 companies with data').trigger('click')
    await button(wrapper, 'Re-scan 2 companies').trigger('click')
    await settle()
    expect(wrapper.get('section[aria-labelledby="admin-title"] > [role="alert"]').text()).toBe('Internal Server Error')
  })
})
