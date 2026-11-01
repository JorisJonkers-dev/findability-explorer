import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useTermPages } from './useTermPages'

const items = [{ competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', pages: [{ url: 'https://www.tribelt.nl/', title: 'Home' }] }]

describe('useTermPages', () => {
  it('asks for the pages that use the term, and asks again for another', async () => {
    const term = ref('eyelink belt')
    const { state, sent } = mountComposable(() => useTermPages(term, ref('en')), routes({ 'GET /api/v1/term-pages': () => json({ items }) }))
    await flushPromises()
    expect(state.split.value.tribelt?.pages[0]?.title).toBe('Home')
    expect(state.isLoading.value).toBe(false)
    expect(sent).toContain('GET /api/v1/term-pages?term=eyelink%20belt&language=en')
    term.value = 'spiral belt'
    await flushPromises()
    expect(sent).toContain('GET /api/v1/term-pages?term=spiral%20belt&language=en')
  })

  it('reports a failure', async () => {
    const { state } = mountComposable(() => useTermPages(ref('x'), ref('en')), routes({ 'GET /api/v1/term-pages': () => problem(500, 'Internal Server Error', 'Down.') }))
    await flushPromises()
    expect(state.error.value).toContain('Down.')
  })
})
