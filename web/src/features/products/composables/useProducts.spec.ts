import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useProducts } from './useProducts'

describe('useProducts', () => {
  it('loads the families and what stands out', async () => {
    const { state } = mountComposable(useProducts, routes({ 'GET /api/v1/products': () => json({ items: [], standouts: [{ finding: 'crowded', family: 'other' }] }) }))
    expect(state.standouts.value).toEqual([])
    await flushPromises()
    expect(state.standouts.value).toHaveLength(1)
    expect(state.products.value).toEqual([])
    expect(state.error.value).toBeNull()
  })

  it('reports a failure', async () => {
    const { state } = mountComposable(useProducts, routes({ 'GET /api/v1/products': () => problem(500, 'Internal Server Error', 'Down.') }))
    await flushPromises()
    expect(state.error.value).toContain('Down.')
  })
})
