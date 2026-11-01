import { describe, expect, it } from 'vitest'
import { BUYER_INTENT, tierBorder, tierFill } from './tiers'
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

describe('buyer intent', () => {
  it('gives every kind a label and one plain sentence', () => {
    expect(TIERS.map((t) => BUYER_INTENT[t])).toEqual([
      { label: 'Specification', help: 'The buyer already knows the exact belt and its sizes.' },
      { label: 'Product', help: 'The buyer names the type of belt.' },
      { label: 'Application', help: 'The buyer describes what the belt is for, such as a bakery oven.' },
      { label: 'Problem', help: 'The buyer describes a problem to solve.' },
      { label: 'Broad', help: 'A general search about conveyor belts.' },
    ])
  })
})
