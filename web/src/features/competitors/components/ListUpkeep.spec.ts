import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { competitor, me, overview, termSearch } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const fenner = competitor('Fenner', { id: 'fenner' })
const intralox = competitor('Intralox', { id: 'intralox', keptBy: 'joris' })
const rubber = competitor('Rubber Co', { id: 'rubber-co', archived: true, archivedBy: 'language model', archivedAt: '2026-10-27T09:00:00Z' })
const unsure = { competitorId: 'fenner', competitor: false, sure: false, reason: 'Makes rubber belting.', model: 'haiku', at: '2026-10-27T09:00:00Z' }
const sure = { competitorId: 'intralox', competitor: true, sure: true, reason: '', model: 'haiku', at: '2026-10-27T09:00:00Z' }
const archivedCheck = { competitorId: 'rubber-co', competitor: false, sure: true, reason: 'Sells rubber.', model: 'haiku', at: '2026-10-27T09:00:00Z' }

type Body = { method: string; path: string; body: unknown }

function api(sent: Body[] = [], extra: Record<string, Handler> = {}) {
  const record = (answer: Handler): Handler => async (request) => {
    const text = await request.text()
    sent.push({ method: request.method, path: new URL(request.url).pathname, body: text === '' ? null : JSON.parse(text) })
    return answer(request)
  }
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/competitors': () => json({ items: [fenner, intralox, rubber] }),
    'GET /api/v1/terms': () => json(termSearch([])),
    'GET /api/v1/overview': () => json(overview()),
    'GET /api/v1/admin/competitor-checks': () => json({ items: [unsure, sure, archivedCheck], unchecked: 2, review: { running: false } }),
    'POST /api/v1/admin/competitor-checks': record(() => json({ running: true }, 202)),
    'PATCH /api/v1/competitors/fenner': record(() => json(fenner)),
    'PATCH /api/v1/competitors/rubber-co': record(() => json(rubber)),
    ...extra,
  })
}

const upkeep = (wrapper: VueWrapper) => wrapper.get('section[aria-labelledby="upkeep-title"]')
const button = (section: ReturnType<typeof upkeep>, text: string) => {
  const found = section.findAll('button').find((b) => b.text().startsWith(text))
  if (!found) throw new Error(`no button ${text}`)
  return found
}

describe('the upkeep of the list', () => {
  it('is for administrators only', async () => {
    const { wrapper } = await mountApp(api([], { 'GET /api/v1/me': () => json(me()) }), '/competitors')
    expect(wrapper.find('#upkeep-title').exists()).toBe(false)
  })

  it('says how far Haiku is, lists the companies it is unsure of and the ones it archived', async () => {
    const { wrapper } = await mountApp(api(), '/competitors')
    const section = upkeep(wrapper)
    expect(section.get('h3').text()).toBe('Haiku’s check of the list')
    expect(section.text()).toContain('Haiku has not checked 2 companies on the list yet.')
    const unsureList = section.get('ul[aria-label="Companies Haiku is unsure of"]')
    expect(unsureList.findAll('li').map((li) => li.text())).toEqual(['FennerHaiku is unsure. Makes rubber belting.Keep FennerArchive Fenner'])
    expect(unsureList.get('a').attributes('href')).toBe('/competitors/fenner')
    const archived = section.get('ul[aria-label="Archived by Haiku"]')
    expect(archived.text()).toContain('Rubber Co')
    expect(archived.text()).toContain('Sells rubber.')
  })

  it('starts a check of the list', async () => {
    const sent: Body[] = []
    const { wrapper } = await mountApp(api(sent), '/competitors')
    await button(upkeep(wrapper), 'Check the list with Haiku').trigger('click')
    await settle()
    expect(sent.map((s) => `${s.method} ${s.path}`)).toEqual(['POST /api/v1/admin/competitor-checks'])
  })

  it('disables the check while Haiku checks, and sends to the settings when there is no key', async () => {
    const running = await mountApp(api([], { 'GET /api/v1/admin/competitor-checks': () => json({ items: [], unchecked: 0, review: { running: true } }) }), '/competitors')
    expect(running.wrapper.text()).toContain('Haiku is checking the companies on the list.')
    expect(button(upkeep(running.wrapper), 'Check the list').attributes('disabled')).toBeDefined()
    running.wrapper.unmount()
    const noKey = await mountApp(api([], { 'GET /api/v1/admin/competitor-checks': () => json({ items: [], unchecked: 0, review: { running: false, problem: 'noKey' } }) }), '/competitors')
    const settings = upkeep(noKey.wrapper).get('a[href="/settings"]')
    expect(settings.text()).toBe('Add one in Language models')
    expect(noKey.wrapper.findAll('button').some((b) => b.text() === 'Check the list with Haiku')).toBe(false)
  })

  it('keeps, archives and restores a company, and says so', async () => {
    const sent: Body[] = []
    const { wrapper } = await mountApp(api(sent), '/competitors')
    const section = upkeep(wrapper)
    await button(section, 'Keep').trigger('click')
    await settle()
    expect(section.text()).toContain('Fenner stays on the list, and Haiku will not archive it.')
    await button(section, 'Archive').trigger('click')
    await settle()
    expect(section.text()).toContain('Archived Fenner. Its scans stay.')
    await button(section, 'Restore').trigger('click')
    await settle()
    expect(section.text()).toContain('Rubber Co is back on the list, and Haiku will not archive it again.')
    expect(sent.map((s) => [s.method, s.path, s.body])).toEqual([
      ['PATCH', '/api/v1/competitors/fenner', { kept: true }],
      ['PATCH', '/api/v1/competitors/fenner', { archived: true }],
      ['PATCH', '/api/v1/competitors/rubber-co', { archived: false }],
    ])
  })

  it('says why a check or a decision was refused', async () => {
    const { wrapper } = await mountApp(
      api([], { 'POST /api/v1/admin/competitor-checks': () => problem(403, 'Forbidden'), 'PATCH /api/v1/competitors/fenner': () => problem(403, 'Forbidden') }),
      '/competitors',
    )
    const section = upkeep(wrapper)
    await button(section, 'Check the list with Haiku').trigger('click')
    await button(section, 'Keep').trigger('click')
    await settle()
    expect(section.findAll('[role="alert"]').map((a) => a.text())).toEqual(['Only administrators can change the companies.', 'Only administrators can change the companies.'])
  })

  it('shows nothing of the lists when Haiku is sure of everything and archived nothing', async () => {
    const { wrapper } = await mountApp(api([], { 'GET /api/v1/admin/competitor-checks': () => json({ items: [sure], unchecked: 0, review: { running: false } }), 'GET /api/v1/competitors': () => json({ items: [fenner, intralox] }) }), '/competitors')
    const section = upkeep(wrapper)
    expect(section.text()).toContain('Haiku has checked every company on the list.')
    expect(section.find('ul').exists()).toBe(false)
  })
})
