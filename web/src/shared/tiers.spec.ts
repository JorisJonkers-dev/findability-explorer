import { describe, expect, it } from 'vitest'
import { tierBorder, tierFill } from './tiers'
import { TIERS } from './vocabulary'

describe('the tier colours', () => {
  it('gives every tier its own fill and border', () => {
    expect(TIERS.map(tierFill)).toEqual([
      'bg-tier-spec',
      'bg-tier-product',
      'bg-tier-application',
      'bg-tier-problem',
      'bg-tier-broad',
    ])
    expect(tierBorder('problem')).toBe('border-tier-problem')
  })
})
