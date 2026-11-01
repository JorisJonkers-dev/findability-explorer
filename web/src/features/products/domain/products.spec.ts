import { describe, expect, it } from 'vitest'
import type { Product } from '../../../infrastructure/api/types.gen'
import { responseBars, SELLERS_SHOWN, visibleSellers } from './products'

const sellers = (n: number): Product['sellers'] => Array.from({ length: n }, (_, i) => ({ competitorId: `c${String(i)}`, name: `C${String(i)}`, names: ['x'] }))

describe('a belt family', () => {
  it('shows the first sellers, or all of them when asked', () => {
    const many = sellers(SELLERS_SHOWN + 2)
    expect(visibleSellers(many, false)).toHaveLength(SELLERS_SHOWN)
    expect(visibleSellers(many, true)).toEqual(many)
    expect(visibleSellers(many, true)).not.toBe(many)
  })
})

describe('the AI responses about a family', () => {
  it('bar the share naming Tribelt in each scope a run asked about it', () => {
    const p = {
      byScope: [
        { scope: 'twente' as const, answers: 10, named: 3 },
        { scope: 'german' as const, answers: 0, named: 0 },
        { scope: 'english' as const, answers: 4, named: 4 },
      ],
    }
    expect(responseBars(p)).toEqual([
      { label: 'Twente (Dutch), 10 responses', value: 0.3, display: '30%', tone: 'orange', highlight: true },
      { label: 'English, no region, 4 responses', value: 1, display: '100%', tone: 'orange', highlight: true },
    ])
    expect(responseBars({ byScope: [{ scope: 'dutch', answers: 1, named: 0 }] })[0]?.label).toBe('Dutch, no region, 1 response')
  })
})
