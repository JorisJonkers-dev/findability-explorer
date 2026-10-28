import { flushPromises } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { product } from '../../../testing/fixtures'
import { json, problem, routes } from '../../../testing/mountApp'
import { mountComposable } from '../../../testing/mountComposable'
import { useProducts } from './useProducts'

describe('useProducts', () => {
  it('loads the families', async () => {
    const { state } = mountComposable(useProducts, routes({ 'GET /api/v1/products': () => json({ items: [product()], standouts: [] }) }))
    expect(state.products.value).toEqual([])
    await flushPromises()
    expect(state.products.value).toHaveLength(1)
    expect(state.error.value).toBeNull()
  })

  it('reports a failure', async () => {
    const { state } = mountComposable(useProducts, routes({ 'GET /api/v1/products': () => problem(500, 'Internal Server Error', 'Down.') }))
    await flushPromises()
    expect(state.error.value).toContain('Down.')
  })
})
