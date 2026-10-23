import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { TermCandidate, TermCandidateList } from '../../../infrastructure/api/types.gen'
import { me, term } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const ID = '3a7bd3e2360a9f01'

function candidate(overrides: Partial<TermCandidate> = {}): TermCandidate {
  return {
    id: ID,
    phrase: 'spiral freezer belt',
    status: 'pending',
    languages: ['en', 'pt'],
    companies: [
      { competitorId: 'twentebelt', name: 'Twentebelt' },
      { competitorId: 'martens', name: 'Märtens' },
    ],
    scans: 3,
    bestScore: 42.5,
    maxPages: 6,
    suggestedTier: 'application',
    suggestedFamily: 'tri_flex_turn_curve',
    closestTerm: 'spiral belt',
    lastSeenAt: '2026-10-23T09:30:00Z',
    ...overrides,
  }
}

const counts = { pending: 1, accepted: 1, rejected: 0 }
const accepted = candidate({
  id: 'ffffffffffffffff',
  phrase: 'wire mesh belt',
  status: 'accepted',
  decision: { status: 'accepted', by: 'joris', at: '2026-10-23T10:00:00Z', term: term({ term: 'wire mesh belt', origin: 'review', family: 'other', tier: 'product' }) },
})

type Sent = { method: string; path: string; body: unknown }

function api(sent: Sent[], extra: Record<string, Handler> = {}) {
  const record = (answer: () => Response): Handler => async (request) => {
    const text = await request.text()
    sent.push({ method: request.method, path: new URL(request.url).pathname, body: text === '' ? null : JSON.parse(text) })
    return answer()
  }
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/admin/term-candidates': (request) => {
      const status = new URL(request.url).searchParams.get('status')
      const items = status === 'accepted' ? [accepted] : status === 'rejected' ? [] : [candidate()]
      return json({ items, counts } satisfies TermCandidateList)
    },
    [`POST /api/v1/admin/term-candidates/${ID}/accept`]: record(() => json(candidate({ status: 'accepted' }))),
    [`POST /api/v1/admin/term-candidates/${ID}/reject`]: record(() => json(candidate({ status: 'rejected' }))),
    'DELETE /api/v1/admin/term-candidates/ffffffffffffffff/decision': record(() => json(candidate({ status: 'pending' }))),
    ...extra,
  })
}

const button = (wrapper: VueWrapper, text: string) => {
  const found = wrapper.findAll('button').find((b) => b.text() === text)
  if (!found) throw new Error(`no button ${text}`)
  return found
}

describe('the term review', () => {
  it('lists the pending candidates with their evidence and the suggestion', async () => {
    const { wrapper } = await mountApp(api([]), '/settings/terms')
    expect(document.title).toBe('Term review · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Term review')
    expect(wrapper.get('[role="group"][aria-label="Show"]').findAll('button').map((b) => [b.text(), b.attributes('aria-pressed')])).toEqual([
      ['To review (1)', 'true'],
      ['Accepted (1)', 'false'],
      ['Rejected (0)', 'false'],
    ])
    const item = wrapper.get('ul[aria-label="Candidates"] li')
    expect(item.get('h3').text()).toBe('spiral freezer belt')
    expect(item.text()).toContain('2 companies, 3 scans, in English and Portuguese')
    expect(item.text()).toContain('last seen 23 Oct 2026, 09:30 UTC')
    expect(item.text()).toContain('Found at Twentebelt, Märtens')
    expect(item.text()).toContain('Suggested: Application, Tri-Flex; closest map term: spiral belt')
    expect(item.text()).toContain('Best score 42.5 · up to 6 pages')
  })

  it('accepts a candidate with the fields as edited, and checks them first', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/settings/terms')
    await button(wrapper, 'Accept…').trigger('click')
    await settle()
    const form = wrapper.get('form[aria-label="Accept spiral freezer belt"]')
    const [termInput, usedBy] = form.findAll('input')
    expect(document.activeElement).toBe(termInput?.element)
    expect((usedBy?.element as HTMLInputElement).value).toBe('Twentebelt, Märtens')
    const [language, family, tier] = form.findAll('select')
    expect((language?.element as HTMLSelectElement).value).toBe('en')
    expect((family?.element as HTMLSelectElement).value).toBe('tri_flex_turn_curve')
    expect((tier?.element as HTMLSelectElement).value).toBe('application')
    await termInput?.setValue(' ')
    await form.trigger('submit')
    expect(form.get('[role="alert"]').text()).toBe('Enter the term as the map should show it.')
    await termInput?.setValue('Spiral freezer belt')
    await usedBy?.setValue('Twentebelt, , Twentebelt, Märtens')
    await language?.setValue('en-US')
    await form.trigger('submit')
    await settle()
    expect(sent).toEqual([
      {
        method: 'POST',
        path: `/api/v1/admin/term-candidates/${ID}/accept`,
        body: { term: 'Spiral freezer belt', language: 'en-US', family: 'tri_flex_turn_curve', tier: 'application', usedBy: ['Twentebelt', 'Märtens'] },
      },
    ])
    expect(wrapper.text()).toContain('Added “Spiral freezer belt” to the terminology map.')
  })

  it('rejects a candidate, and cancels an acceptance', async () => {
    const sent: Sent[] = []
    const { wrapper } = await mountApp(api(sent), '/settings/terms')
    await button(wrapper, 'Accept…').trigger('click')
    await settle()
    await button(wrapper, 'Cancel').trigger('click')
    expect(wrapper.find('form[aria-label="Accept spiral freezer belt"]').exists()).toBe(false)
    await button(wrapper, 'Reject').trigger('click')
    await settle()
    expect(sent).toEqual([{ method: 'POST', path: `/api/v1/admin/term-candidates/${ID}/reject`, body: null }])
    expect(wrapper.text()).toContain('Rejected “spiral freezer belt”.')
  })

  it('shows who decided, and undoes a decision', async () => {
    const sent: Sent[] = []
    const { wrapper, router } = await mountApp(api(sent), '/settings/terms')
    await button(wrapper, 'Accepted (1)').trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/settings/terms?status=accepted')
    const item = wrapper.get('ul[aria-label="Candidates"] li')
    expect(item.text()).toContain('Accepted by joris, 23 Oct 2026, 10:00 UTC')
    expect(item.text()).toContain('In the map as “wire mesh belt”, English, Product, Other belts.')
    await button(wrapper, 'Undo').trigger('click')
    await settle()
    expect(sent).toEqual([{ method: 'DELETE', path: '/api/v1/admin/term-candidates/ffffffffffffffff/decision', body: null }])
    expect(wrapper.text()).toContain('“wire mesh belt” is back in the queue.')

    await button(wrapper, 'Rejected (0)').trigger('click')
    await settle()
    expect(wrapper.text()).toContain('Nothing here.')
    await button(wrapper, 'To review (1)').trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/settings/terms')
  })

  it('says why a decision failed', async () => {
    const { wrapper } = await mountApp(
      api([], { [`POST /api/v1/admin/term-candidates/${ID}/reject`]: () => problem(409, 'Conflict', 'This candidate was decided already.') }),
      '/settings/terms',
    )
    await button(wrapper, 'Reject').trigger('click')
    await settle()
    expect(wrapper.get('[role="alert"]').text()).toBe('This candidate was decided already.')
  })

  it('is closed to an account that is no administrator, and says when the list fails', async () => {
    const viewer = await mountApp(
      api([], { 'GET /api/v1/me': () => json(me()), 'GET /api/v1/admin/term-candidates': () => problem(403, 'Forbidden') }),
      '/settings/terms',
    )
    expect(viewer.wrapper.get('[role="alert"]').text()).toContain('Only administrators can review terms.')
    expect(viewer.wrapper.find('ul[aria-label="Candidates"]').exists()).toBe(false)
    viewer.wrapper.unmount()
    // A refusal is not asked again, so it shows at once.
    const failing = await mountApp(api([], { 'GET /api/v1/admin/term-candidates': () => problem(400, 'Bad Request') }), '/settings/terms')
    expect(failing.wrapper.get('[role="alert"]').text()).toBe('Bad Request')
    expect(failing.wrapper.get('[role="group"][aria-label="Show"]').findAll('button')[0]?.text()).toBe('To review')
  })
})
