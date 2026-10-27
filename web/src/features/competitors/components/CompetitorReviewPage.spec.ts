import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { CompetitorCandidate } from '../../../infrastructure/api/types.gen'
import { me } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const ID = '3a7bd3e2360a9f01'

const named: CompetitorCandidate = {
  id: ID,
  name: 'Ashworth',
  status: 'pending',
  answers: 37,
  providers: ['openai', 'x-ai'],
  models: ['openai/gpt-oss-20b'],
  addresses: ['https://www.ashworth.com/products'],
  lastSeenAt: '2026-11-01T06:10:00Z',
}
const added: CompetitorCandidate = {
  ...named,
  id: 'ffffffffffffffff',
  name: 'Wire Belt Company',
  status: 'added',
  addresses: [],
  decision: { status: 'added', competitorId: 'wire-belt', by: 'joris', at: '2026-11-02T09:00:00Z' },
}
const counts = { pending: 1, added: 1, dismissed: 0 }

type Sent = { method: string; path: string; body: unknown }

function api(sent: Sent[], extra: Record<string, Handler> = {}) {
  const record = (answer: () => Response): Handler => async (request) => {
    const text = await request.text()
    sent.push({ method: request.method, path: new URL(request.url).pathname, body: text === '' ? null : JSON.parse(text) })
    return answer()
  }
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/admin/competitor-candidates': (request) => {
      const status = new URL(request.url).searchParams.get('status')
      return json({ items: status === 'added' ? [added] : status === 'dismissed' ? [] : [named], counts })
    },
    [`POST /api/v1/admin/competitor-candidates/${ID}/add`]: record(() => json({ ...named, status: 'added' })),
    [`POST /api/v1/admin/competitor-candidates/${ID}/dismiss`]: record(() => json({ ...named, status: 'dismissed' })),
    'DELETE /api/v1/admin/competitor-candidates/ffffffffffffffff/decision': record(() => new Response(null, { status: 204 })),
    ...extra,
  })
}

const button = (wrapper: VueWrapper, text: string) => {
  const found = wrapper.findAll('button').find((b) => b.text() === text)
  if (!found) throw new Error(`no button ${text}`)
  return found
}

describe('the competitor review', () => {
  it('lists the companies the answers name, with their evidence, under its own settings tab', async () => {
    const { wrapper } = await mountApp(api([]), '/settings/competitors')
    expect(document.title).toBe('Competitor review · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Competitor review')
    const tab = wrapper.get('nav[aria-label="Settings"] a[href="/settings/competitors"]')
    expect(tab.attributes('aria-current')).toBe('page')
    expect(tab.text()).toBe('Competitor review 1')
    expect(wrapper.get('[role="group"][aria-label="Show"]').findAll('button').map((b) => b.text())).toEqual(['To review (1)', 'Added (1)', 'Dismissed (0)'])
    const item = wrapper.get('ul[aria-label="Companies"] li')
    expect(item.get('h3').text()).toBe('Ashworth')
    expect(item.text()).toContain('Named in 37 answers by OpenAI and xAI, last on 1 Nov 2026, 06:10 UTC')
    expect(item.get('a').attributes('href')).toBe('https://www.ashworth.com/products')
  })

  it('adds a company with its website, checking it first, and dismisses another', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/settings/competitors')
    await button(wrapper, 'Add…').trigger('click')
    const form = wrapper.get('form[aria-label="Add Ashworth"]')
    expect((form.get('input[type="url"]').element as HTMLInputElement).value).toBe('https://www.ashworth.com/')
    await form.get('input[type="url"]').setValue('ashworth.com')
    await form.trigger('submit')
    expect(form.get('[role="alert"]').text()).toBe('Enter its website, starting with https://.')
    expect(sent).toEqual([])
    await form.get('input[type="url"]').setValue('https://www.ashworth.com/')
    await form.trigger('submit')
    await settle()
    expect(sent).toEqual([{ method: 'POST', path: `/api/v1/admin/competitor-candidates/${ID}/add`, body: { name: 'Ashworth', url: 'https://www.ashworth.com/' } }])
    expect(wrapper.text()).toContain('Added Ashworth to the competitors.')
    wrapper.unmount()

    const other: Sent[] = []
    const again = await mountApp(api(other), '/settings/competitors')
    await button(again.wrapper, 'Not a competitor').trigger('click')
    await settle()
    expect(other.map((s) => `${s.method} ${s.path}`)).toEqual([`POST /api/v1/admin/competitor-candidates/${ID}/dismiss`])
  })

  it('shows who decided, links an added company, and undoes a decision', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/settings/competitors?status=added')
    const item = wrapper.get('ul[aria-label="Companies"] li')
    expect(item.text()).toContain('Added by joris, 2 Nov 2026, 09:00 UTC')
    expect(item.text()).toContain('The answers gave no website for it.')
    expect(item.get('a').attributes('href')).toBe('/competitors/wire-belt')
    await button(wrapper, 'Undo').trigger('click')
    await settle()
    expect(sent.map((s) => `${s.method} ${s.path}`)).toEqual(['DELETE /api/v1/admin/competitor-candidates/ffffffffffffffff/decision'])
  })

  it('is closed to an account that is no administrator', async () => {
    const { wrapper } = await mountApp(api([], { 'GET /api/v1/admin/competitor-candidates': () => problem(403, 'Forbidden') }), '/settings/competitors')
    expect(wrapper.text()).toContain('Only administrators can review competitors.')
  })
})
