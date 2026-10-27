import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { CandidateReviewState, CompetitorCandidate, CompetitorCandidateList } from '../../../infrastructure/api/types.gen'
import { me } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const METIZ = '3a7bd3e2360a9f01'
const KIENZLE = '4b8ce4f3471b0a12'

const metiz: CompetitorCandidate = {
  id: METIZ,
  name: 'Taiwan Metiz Alliance',
  status: 'pending',
  answers: 8,
  providers: ['openai', 'x-ai'],
  models: ['openai/gpt-oss-20b'],
  addresses: ['https://metiz.tw/product_conveyor_belt.htm', 'metiz.tw'],
  lastSeenAt: '2026-10-27T08:41:00Z',
  sites: ['metiz.tw'],
  website: 'https://metiz.tw/',
  names: [
    { id: METIZ, name: 'Taiwan Metiz Alliance', answers: 6, joined: 'lead' },
    { id: '3a7bd3e2360a9f02', name: 'Metiz', answers: 1, joined: 'site' },
    { id: '3a7bd3e2360a9f03', name: 'Metiz Wire', answers: 1, joined: 'model' },
  ],
  check: { competitor: true, sure: true, reason: 'Makes wire mesh belts.', model: 'anthropic: claude-haiku-5-5', at: '2026-10-27T09:00:00Z' },
}
const kienzle: CompetitorCandidate = {
  id: KIENZLE,
  name: 'Kienzle GmbH',
  status: 'pending',
  answers: 5,
  providers: ['anthropic'],
  models: [],
  addresses: [],
  lastSeenAt: '2026-10-27T08:46:00Z',
  check: { competitor: false, sure: true, reason: 'Builds furnaces.', model: 'anthropic: claude-haiku-5-5', at: '2026-10-27T09:00:00Z' },
}
const added: CompetitorCandidate = {
  ...kienzle,
  id: 'ffffffffffffffff',
  name: 'Wire Belt Company',
  status: 'added',
  check: undefined,
  decision: { status: 'added', competitorId: 'wire-belt', by: 'joris', at: '2026-11-02T09:00:00Z' },
}

type Sent = { method: string; path: string; body: unknown }

function api(sent: Sent[], review: CandidateReviewState = { running: false }, extra: Record<string, Handler> = {}) {
  const record = (answer: () => Response): Handler => async (request) => {
    const text = await request.text()
    sent.push({ method: request.method, path: new URL(request.url).pathname, body: text === '' ? null : JSON.parse(text) })
    return answer()
  }
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/admin/competitor-candidates': (request) => {
      const status = new URL(request.url).searchParams.get('status')
      const items = status === 'added' ? [added] : status === 'dismissed' ? [] : [metiz, kienzle]
      return json({ items, counts: { pending: 2, added: 1, dismissed: 0, unchecked: 3 }, review } satisfies CompetitorCandidateList)
    },
    [`POST /api/v1/admin/competitor-candidates/${METIZ}/add`]: record(() => json({ ...metiz, status: 'added' })),
    [`POST /api/v1/admin/competitor-candidates/${KIENZLE}/add`]: record(() => json({ ...kienzle, status: 'added' })),
    [`POST /api/v1/admin/competitor-candidates/${KIENZLE}/dismiss`]: record(() => json({ ...kienzle, status: 'dismissed' })),
    'DELETE /api/v1/admin/competitor-candidates/ffffffffffffffff/decision': record(() => new Response(null, { status: 204 })),
    'POST /api/v1/admin/competitor-candidates/3a7bd3e2360a9f03/split': record(() => new Response(null, { status: 204 })),
    'POST /api/v1/admin/competitor-candidates/review': record(() => json({ running: true }, 202)),
    'POST /api/v1/admin/competitor-candidates/decisions': record(() =>
      json({ items: [{ id: METIZ, name: 'Taiwan Metiz Alliance', outcome: 'done' }, { id: KIENZLE, name: 'Kienzle GmbH', outcome: 'skipped', reason: 'The answers gave no website for it.' }] }),
    ),
    ...extra,
  })
}

const button = (wrapper: VueWrapper, text: string) => {
  const found = wrapper.findAll('button').find((b) => b.text() === text)
  if (!found) throw new Error(`no button ${text}`)
  return found
}
const rows = (wrapper: VueWrapper) => wrapper.findAll('ul[aria-label="Companies"] > li')
const calls = (sent: Sent[]) => sent.map((s) => `${s.method} ${s.path}`)

describe('the competitor review', () => {
  it('lists one row per company, with its other names, website and Haiku’s check, under its own settings tab', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/settings/competitors')
    expect(document.title).toBe('Competitor review · Findability Explorer')
    expect(wrapper.get('nav[aria-label="Settings"] a[href="/settings/competitors"]').text()).toBe('Competitor review 2')
    const review = wrapper.get('section[aria-label="Haiku\'s review"]')
    expect(review.text()).toContain('Haiku has not checked 3 companies yet.')
    const first = rows(wrapper)[0]
    expect(first?.get('h3').text()).toBe('Taiwan Metiz Alliance')
    expect(first?.text()).toContain('Named in 8 answers by OpenAI and xAI.')
    expect(first?.text()).toContain('Also named')
    expect(first?.text()).toContain('Haiku says a competitor. Makes wire mesh belts.')
    expect(first?.get('a[target="_blank"]').attributes('href')).toBe('https://metiz.tw/')
    // Only a merged name can be split off; one on the same website cannot.
    expect(first?.findAll('button').filter((b) => b.text().startsWith('Split off')).map((b) => b.text())).toEqual(['Split off Metiz Wire'])
    await button(wrapper, 'Review with Haiku').trigger('click')
    await settle()
    expect(calls(sent)).toEqual(['POST /api/v1/admin/competitor-candidates/review'])
  })

  it('searches names and websites, and filters on Haiku’s check', async () => {
    const { wrapper } = await mountApp(api([]), '/settings/competitors')
    await wrapper.get('input[type="search"]').setValue('metiz wire')
    expect(rows(wrapper).map((r) => r.get('h3').text())).toEqual(['Taiwan Metiz Alliance'])
    await wrapper.get('input[type="search"]').setValue('')
    await button(wrapper, 'Haiku says not').trigger('click')
    expect(rows(wrapper).map((r) => r.get('h3').text())).toEqual(['Kienzle GmbH'])
    await button(wrapper, 'Not checked yet').trigger('click')
    expect(wrapper.text()).toContain('Nothing here.')
  })

  it('adds a company in one click when its website is known, and asks for one when not', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/settings/competitors')
    await button(wrapper, 'Add').trigger('click')
    await settle()
    expect(sent[0]).toEqual({ method: 'POST', path: `/api/v1/admin/competitor-candidates/${METIZ}/add`, body: { name: 'Taiwan Metiz Alliance', url: 'https://metiz.tw/' } })
    await button(wrapper, 'Add…').trigger('click')
    const form = wrapper.get('form[aria-label="Add Kienzle GmbH"]')
    await form.trigger('submit')
    expect(form.get('[role="alert"]').text()).toBe('Enter its website, starting with https://.')
    await form.get('input[type="url"]').setValue('https://www.kienzle.com/')
    await form.trigger('submit')
    await settle()
    expect(sent[1]?.body).toEqual({ name: 'Kienzle GmbH', url: 'https://www.kienzle.com/' })
  })

  it('moves, selects and dismisses by keyboard', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/settings/competitors')
    const first = rows(wrapper)[0]
    if (!first) throw new Error('no rows')
    ;(first.element as HTMLElement).focus()
    await first.trigger('keydown', { key: 'ArrowDown' })
    await settle()
    const second = rows(wrapper)[1]
    expect(document.activeElement).toBe(second?.element)
    await second?.trigger('keydown', { key: 'x' })
    expect(wrapper.text()).toContain('1 selected')
    await second?.trigger('keydown', { key: 'd' })
    await settle()
    expect(calls(sent)).toEqual([`POST /api/v1/admin/competitor-candidates/${KIENZLE}/dismiss`])
    await second?.trigger('keydown', { key: 'k' })
    await settle()
    expect(document.activeElement).toBe(rows(wrapper)[0]?.element)
  })

  it('adds or dismisses several at once, and says which it skipped', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/settings/competitors')
    await wrapper.get('input[type="checkbox"]').setValue(true)
    expect(wrapper.text()).toContain('2 selected')
    await button(wrapper, 'Add with their websites').trigger('click')
    await settle()
    expect(sent[0]).toEqual({ method: 'POST', path: '/api/v1/admin/competitor-candidates/decisions', body: { action: 'add', ids: [METIZ, KIENZLE] } })
    expect(wrapper.text()).toContain('Added 1 company. Skipped 1.')
    expect(wrapper.get('ul[aria-label="Skipped"]').text()).toBe('Kienzle GmbH. The answers gave no website for it.')
    expect(wrapper.text()).not.toContain('selected')
  })

  it('splits a merged name off its company', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/settings/competitors')
    await button(wrapper, 'Split off Metiz Wire').trigger('click')
    await settle()
    expect(calls(sent)).toEqual(['POST /api/v1/admin/competitor-candidates/3a7bd3e2360a9f03/split'])
    expect(wrapper.text()).toContain('Metiz Wire is a company of its own again.')
  })

  it('shows who decided, links an added company, and undoes a decision', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/settings/competitors?status=added')
    const item = rows(wrapper)[0]
    expect(item?.text()).toContain('Added by joris, 2 Nov 2026, 09:00 UTC')
    expect(item?.get('a[href="/competitors/wire-belt"]').text()).toBe('Open its page')
    await button(wrapper, 'Undo').trigger('click')
    await settle()
    expect(calls(sent)).toEqual(['DELETE /api/v1/admin/competitor-candidates/ffffffffffffffff/decision'])
  })

  it('says when no Anthropic key is set up, and is closed to an account that is no administrator', async () => {
    const none = await mountApp(api([], { running: false, problem: 'noKey' }), '/settings/competitors')
    expect(none.wrapper.get('section[aria-label="Haiku\'s review"] a').attributes('href')).toBe('/settings')
    none.wrapper.unmount()
    const { wrapper } = await mountApp(api([], { running: false }, { 'GET /api/v1/admin/competitor-candidates': () => problem(403, 'Forbidden') }), '/settings/competitors')
    expect(wrapper.text()).toContain('Only administrators can review competitors.')
  })
})
