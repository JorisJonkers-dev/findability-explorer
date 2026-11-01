import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { term } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useHandTerms } from './useHandTerms'

const at = '2026-11-01T09:00:00Z'
const list = { added: [{ term: term({ term: 'oven belt', origin: 'hand' }), by: 'Joris', at }], removed: [] }

describe('useHandTerms', () => {
  it('lists the terms added and removed by hand', async () => {
    const { state } = mountComposable(() => useHandTerms(), routes({ 'GET /api/v1/admin/terms/hand': () => json(list) }))
    expect(state.added.value).toEqual([])
    expect(state.removed.value).toEqual([])
    await flushPromises()
    expect(state.added.value[0]?.term.term).toBe('oven belt')
    expect(state.removed.value).toEqual([])
    expect(state.isLoading.value).toBe(false)
    expect(state.error.value).toBeNull()
  })

  it('adds, removes, and puts back a term, saying what it did, and asks for the lists again', async () => {
    const { state, sent } = mountComposable(
      () => useHandTerms(),
      routes({
        'GET /api/v1/admin/terms/hand': () => json(list),
        'POST /api/v1/admin/terms/hand': () => json(term({ term: 'oven belt', origin: 'hand' }), 201),
        'POST /api/v1/admin/terms/removed': () => new Response(null, { status: 204 }),
        'DELETE /api/v1/admin/terms/removed': () => new Response(null, { status: 204 }),
      }),
    )
    await flushPromises()
    const adding = state.add({ term: 'oven belt', language: 'en', family: 'eyelink', tier: 'application' })
    expect(state.isChanging.value).toBe(true)
    expect(await adding).toBe(true)
    expect(state.notice.value).toBe('oven belt (English) is on the list. Scans look for it from the next scan on.')
    expect(await state.remove({ term: 'spiral belt', language: 'en' })).toBe(true)
    expect(state.notice.value).toBe('spiral belt (English) is off the list. It is no longer counted or looked for.')
    expect(await state.putBack({ term: 'spiral belt', language: 'en' })).toBe(true)
    expect(state.notice.value).toBe('spiral belt (English) is back on the list.')
    expect(sent).toContain('DELETE /api/v1/admin/terms/removed?term=spiral%20belt&language=en')
    expect(sent.filter((s) => s === 'GET /api/v1/admin/terms/hand').length).toBeGreaterThan(1)
    expect(state.isChanging.value).toBe(false)
  })

  it('says why a change or the lists failed', async () => {
    const { state } = mountComposable(
      () => useHandTerms(),
      routes({
        'GET /api/v1/admin/terms/hand': () => problem(500, 'Internal Server Error', 'Down.'),
        'POST /api/v1/admin/terms/hand': () => problem(409, 'Conflict', 'The terminology map has this term in this language already.'),
      }),
    )
    await flushPromises()
    expect(state.error.value).toContain('Down.')
    expect(await state.add({ term: 'spiral belt', language: 'en', family: 'other', tier: 'broad' })).toBe(false)
    expect(state.failure.value).toContain('has this term')
    expect(state.notice.value).toBe('')
  })
})
