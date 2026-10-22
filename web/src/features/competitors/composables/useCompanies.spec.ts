import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { competitor, term, termSearch } from '../../../testing/fixtures'
import { json, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useCompanies } from './useCompanies'

describe('useCompanies', () => {
  it('gives every company the short name the map credits, and finds one by id', async () => {
    const { state } = mountComposable(
      useCompanies,
      routes({
        'GET /api/v1/competitors': () => json({ items: [competitor('Twentebelt B.V.')] }),
        'GET /api/v1/terms': () => json(termSearch([term({ usedBy: ['Twentebelt'] })])),
      }),
    )
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(state.companies.value.map((c) => c.shortName)).toEqual(['Twentebelt'])
    expect(state.byId('twentebelt-b-v')?.name).toBe('Twentebelt B.V.')
    expect(state.byId(undefined)).toBeUndefined()
    expect(state.error.value).toBeNull()
  })
})
