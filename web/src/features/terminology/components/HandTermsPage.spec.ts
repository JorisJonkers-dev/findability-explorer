import { describe, expect, it } from 'vitest'
import { me, term, termSearch } from '../../../testing/fixtures'
import { manageCounts } from '../../../testing/manageFixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

const at = '2026-11-01T09:00:00Z'
const MAP = [
  term({ term: 'spiral belt', language: 'en', family: 'open_spiral', tier: 'product', origin: 'research' }),
  term({ term: 'Spiralband', language: 'de', family: 'open_spiral', tier: 'product', origin: 'research' }),
  term({ term: 'oven belt', language: 'en', family: 'eyelink', tier: 'application', usedBy: [], origin: 'hand' }),
]

function api(state: { added: unknown[]; removed: unknown[] }, extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    ...manageCounts(),
    'GET /api/v1/terms': () => json(termSearch(MAP, { total: MAP.length })),
    'GET /api/v1/competitors': () => json({ items: [] }),
    'GET /api/v1/admin/terms/hand': () => json(state),
    ...extra,
  })
}

const button = (wrapper: { findAll: (s: string) => { text: () => string; trigger: (e: string) => Promise<void> }[] }, text: string) => {
  const b = wrapper.findAll('button').find((x) => x.text() === text)
  if (!b) throw new Error(`no button ${text}`)
  return b
}

describe('Add or remove terms', () => {
  it('lists the terms added by hand, marked, with the list’s size', async () => {
    const state = { added: [{ term: MAP[2], by: 'Joris Jonkers', at }], removed: [] }
    const { wrapper } = await mountApp(api(state), '/manage/terms/edit')
    expect(document.title).toBe('Add or remove terms · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Add or remove terms')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('Add a search term by hand, or take one off the list.')
    expect(wrapper.findAll('h2').map((h) => h.text())).toEqual(['Added by hand', 'Add a search term'])
    const added = wrapper.get('table[aria-label="Search terms added by hand"]')
    expect(added.text()).toContain('oven belt')
    expect(added.text()).toContain('Added by hand')
    expect(added.text()).toContain('English')
    expect(added.text()).toContain('Eyelink')
    expect(added.text()).toContain('Application')
    expect(wrapper.text()).toContain('3 search terms are on the list')
  })

  it('adds a search term, and says scans look for it from the next scan on', async () => {
    const state = { added: [] as unknown[], removed: [] }
    let sent: unknown = null
    const { wrapper } = await mountApp(
      api(state, {
        'POST /api/v1/admin/terms/hand': async (r) => {
          sent = await r.json()
          return json(term({ term: 'roestvrijstalen transportband', language: 'nl', origin: 'hand' }), 201)
        },
      }),
      '/manage/terms/edit',
    )
    expect(wrapper.text()).toContain('No search term was added by hand yet.')
    await button(wrapper, 'Add the search term').trigger('click')
    expect(wrapper.get('[role="alert"]').text()).toBe('Write the term.')
    await wrapper.get('input[name="term"]').setValue('roestvrijstalen transportband')
    await wrapper.get('select[name="language"]').setValue('nl')
    await wrapper.get('select[name="family"]').setValue('closed_spiral')
    await wrapper.get('select[name="tier"]').setValue('product')
    await button(wrapper, 'Add the search term').trigger('click')
    await settle()
    expect(sent).toEqual({ term: 'roestvrijstalen transportband', language: 'nl', family: 'closed_spiral', tier: 'product' })
    expect(wrapper.text()).toContain('roestvrijstalen transportband (Dutch) is on the list. Scans look for it from the next scan on.')
    expect((wrapper.get('input[name="term"]').element as HTMLInputElement).value).toBe('')
    expect(wrapper.text()).toContain('What buyer intent means')
  })

  it('says why a term was not added', async () => {
    const { wrapper } = await mountApp(
      api({ added: [], removed: [] }, { 'POST /api/v1/admin/terms/hand': () => problem(409, 'Conflict', 'The terminology map has this term in this language already.') }),
      '/manage/terms/edit',
    )
    await wrapper.get('input[name="term"]').setValue('spiral belt')
    await button(wrapper, 'Add the search term').trigger('click')
    await settle()
    expect(wrapper.get('[role="alert"]').text()).toContain('has this term in this language already')
  })

  it('finds a term on the list and removes it, and puts a removed one back', async () => {
    let removed: unknown = null
    let back = ''
    const state = { added: [], removed: [{ term: MAP[1], by: 'Joris Jonkers', at }] }
    const { wrapper } = await mountApp(
      api(state, {
        'POST /api/v1/admin/terms/removed': async (r) => {
          removed = await r.json()
          return new Response(null, { status: 204 })
        },
        'DELETE /api/v1/admin/terms/removed': (r) => {
          back = new URL(r.url).search
          return new Response(null, { status: 204 })
        },
      }),
      '/manage/terms/edit',
    )
    await wrapper.get('input[type="search"][name="find"]').setValue('spiral b')
    const hits = wrapper.get('ul[aria-label="Search terms on the list that match"]')
    expect(hits.findAll('li').map((l) => l.text().split('English')[0]?.trim())).toEqual(['spiral belt'])
    await hits.get('button').trigger('click')
    await settle()
    expect(removed).toEqual({ term: 'spiral belt', language: 'en' })
    expect(wrapper.text()).toContain('spiral belt (English) is off the list. It is no longer counted or looked for.')
    const gone = wrapper.get('ul[aria-label="Removed search terms"]')
    expect(gone.text()).toContain('Spiralband')
    expect(gone.text()).toContain('Removed by Joris Jonkers on 1 November 2026.')
    await gone.get('button').trigger('click')
    await settle()
    expect(back).toBe('?term=Spiralband&language=de')
    expect(wrapper.text()).toContain('Spiralband (German) is back on the list.')
  })

  it('removes a term added by hand from its row', async () => {
    let removed: unknown = null
    const { wrapper } = await mountApp(
      api({ added: [{ term: MAP[2], by: 'Joris Jonkers', at }], removed: [] }, {
        'POST /api/v1/admin/terms/removed': async (r) => {
          removed = await r.json()
          return new Response(null, { status: 204 })
        },
      }),
      '/manage/terms/edit',
    )
    await wrapper.get('table[aria-label="Search terms added by hand"]').get('button[aria-label="Remove oven belt from the list"]').trigger('click')
    await settle()
    expect(removed).toEqual({ term: 'oven belt', language: 'en' })
  })

  it('says when the lists cannot load', async () => {
    const { wrapper } = await mountApp(api({ added: [], removed: [] }, { 'GET /api/v1/admin/terms/hand': () => problem(500, 'Internal Server Error', 'Down.') }), '/manage/terms/edit')
    expect(wrapper.get('[role="alert"]').text()).toContain('Down.')
  })
})
