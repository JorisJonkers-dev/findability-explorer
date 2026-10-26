import { describe, expect, it } from 'vitest'
import type { CompetitorCandidate } from '../../../infrastructure/api/types.gen'
import { me } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const ID = '3a7bd3e2360a9f01'

function candidate(overrides: Partial<CompetitorCandidate> = {}): CompetitorCandidate {
  return {
    id: ID, name: 'Ashworth', status: 'pending', answers: 12, providers: ['openai', 'meta-llama'],
    addresses: ['https://www.ashworth.com/belts'], lastSeenAt: '2026-11-01T06:10:00Z', ...overrides,
  }
}

function server(extra: Record<string, Handler> = {}) {
  const sent: { path: string; body: unknown }[] = []
  const counts = { pending: 1, added: 1, dismissed: 0 }
  const handle = routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/admin/competitor-candidates': (r) => {
      const status = new URL(r.url).searchParams.get('status')
      const items = status === 'added'
        ? [candidate({ id: 'b'.repeat(16), name: 'Wire Belt', answers: 1, status: 'added', decision: { status: 'added', competitorId: 'wire-belt', by: 'joris', at: '2026-11-02T09:00:00Z' } })]
        : status === 'dismissed' ? [] : [candidate()]
      return json({ items, counts })
    },
    [`POST /api/v1/admin/competitor-candidates/${ID}/add`]: async (r) => {
      sent.push({ path: 'add', body: await r.json() })
      return json(candidate({ status: 'added' }))
    },
    [`POST /api/v1/admin/competitor-candidates/${ID}/dismiss`]: () => {
      sent.push({ path: 'dismiss', body: null })
      return json(candidate({ status: 'dismissed' }))
    },
    [`DELETE /api/v1/admin/competitor-candidates/${'b'.repeat(16)}/decision`]: () => {
      sent.push({ path: 'undo', body: null })
      return new Response(null, { status: 204 })
    },
    ...extra,
  })
  return { handle, sent }
}

const button = (wrapper: { findAll: (s: string) => { text: () => string; trigger: (e: string) => Promise<void> }[] }, text: string) => {
  const b = wrapper.findAll('button').find((x) => x.text() === text)
  if (!b) throw new Error(`no button ${text}`)
  return b
}

describe('the competitor candidates', () => {
  it('lists the candidates to review, with who named them and where', async () => {
    const { wrapper } = await mountApp(server().handle, '/settings/candidates')
    expect(document.title).toBe('Competitor candidates · Findability Explorer')
    expect(wrapper.findAll('[aria-label="Show"] button').map((b) => b.text())).toEqual(['To review (1)', 'Added (1)', 'Dismissed (0)'])
    const item = wrapper.get('ul[aria-label="Candidates"] li')
    expect(item.text()).toContain('AshworthNamed in 12 answers')
    expect(item.text()).toContain('By OpenAI, Meta · last on 1 Nov 2026, 06:10 UTC')
    expect(item.text()).toContain('Addresses given: https://www.ashworth.com/belts')
  })

  it('adds a candidate with a name and website, or dismisses it', async () => {
    const { handle, sent } = server()
    const { wrapper } = await mountApp(handle, '/settings/candidates')
    await button(wrapper, 'Add').trigger('click')
    const form = wrapper.get('form[aria-label="Add Ashworth"]')
    const [name, url] = form.findAll('input')
    expect((url?.element as HTMLInputElement).value).toBe('https://www.ashworth.com/')
    await name?.setValue(' Ashworth Bros ')
    await form.trigger('submit')
    await settle()
    expect(sent).toEqual([{ path: 'add', body: { name: 'Ashworth Bros', url: 'https://www.ashworth.com/' } }])
    expect(wrapper.text()).toContain('Added Ashworth Bros to the competitors.')
    await button(wrapper, 'Add').trigger('click')
    await button(wrapper, 'Cancel').trigger('click')
    expect(wrapper.find('form[aria-label="Add Ashworth"]').exists()).toBe(false)
    await button(wrapper, 'Dismiss').trigger('click')
    await settle()
    expect(sent[1]?.path).toBe('dismiss')
    expect(wrapper.text()).toContain('Dismissed Ashworth.')
  })

  it('undoes a decision from its tab', async () => {
    const { handle, sent } = server()
    const { wrapper, router } = await mountApp(handle, '/settings/candidates')
    await wrapper.findAll('[aria-label="Show"] button')[1]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({ status: 'added' })
    expect(wrapper.text()).toContain('Added by joris on 2 Nov 2026, 09:00 UTC.')
    await button(wrapper, 'Undo').trigger('click')
    await settle()
    expect(sent).toEqual([{ path: 'undo', body: null }])
    expect(wrapper.text()).toContain('Wire Belt is back among the candidates to review.')
    await wrapper.findAll('[aria-label="Show"] button')[2]?.trigger('click')
    await settle()
    expect(wrapper.text()).toContain('Nothing here.')
    await wrapper.findAll('[aria-label="Show"] button')[0]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.query).toEqual({})
  })

  it('says what went wrong, and tells a non-administrator', async () => {
    const refused = await mountApp(
      server({ [`POST /api/v1/admin/competitor-candidates/${ID}/dismiss`]: () => problem(409, 'Conflict', 'The candidate was decided already.') }).handle,
      '/settings/candidates',
    )
    await button(refused.wrapper, 'Dismiss').trigger('click')
    await settle()
    expect(refused.wrapper.get('[role="alert"]').text()).toBe('The candidate was decided already.')
    refused.wrapper.unmount()
    const failing = await mountApp(server({ 'GET /api/v1/admin/competitor-candidates': () => problem(500, 'Internal Server Error', 'Down.') }).handle, '/settings/candidates')
    expect(failing.wrapper.get('[role="alert"]').text()).toBe('Down.')
    failing.wrapper.unmount()
    const viewer = await mountApp(server({ 'GET /api/v1/me': () => json(me()), 'GET /api/v1/admin/competitor-candidates': () => problem(403, 'Forbidden') }).handle, '/settings/candidates')
    expect(viewer.wrapper.get('[role="alert"]').text()).toBe('Only administrators can review competitor candidates.')
  })
})
