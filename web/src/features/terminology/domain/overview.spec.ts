import { describe, expect, it } from 'vitest'
import type { OverviewTerms } from '../../../infrastructure/api/types.gen'
import { listOf, termsFigures } from './overview'

const terms = (overrides: Partial<OverviewTerms> = {}): OverviewTerms => ({
  total: 10,
  tribelt: 4,
  from: 'scan',
  languages: [
    { language: 'nl', terms: 5, tribelt: 2 },
    { language: 'de', terms: 3, tribelt: 1 },
    { language: 'en', terms: 2, tribelt: 1 },
  ],
  tiers: [],
  ...overrides,
})

describe('listOf', () => {
  it('lists words as a sentence does, with the Oxford comma', () => {
    expect(listOf([])).toBe('')
    expect(listOf(['a'])).toBe('a')
    expect(listOf(['a', 'b'])).toBe('a and b')
    expect(listOf(['a', 'b', 'c'])).toBe('a, b, and c')
  })
})

describe('the figures', () => {
  it('count the map, what tribelt.nl uses against its goal, and what only competitors use', () => {
    expect(termsFigures(terms(), 20)).toEqual([
      { big: '10', text: 'terms in the search term list', note: 'In Dutch, German, and English.' },
      { big: '4', text: 'used on tribelt.nl', note: 'Goal 20' },
      { big: '6', text: 'used only by competitors', note: 'Each one is a search in which Tribelt does not turn up.' },
    ])
  })

  it('have no notes without languages or a goal, and are zero before the figures load', () => {
    const [map, used, others] = termsFigures(undefined, undefined)
    expect(map).toEqual({ big: '0', text: 'terms in the search term list', note: '' })
    expect(used?.note).toBe('')
    expect(others?.big).toBe('0')
    expect(termsFigures(terms({ total: 2, tribelt: 5 }), undefined)[2]?.big).toBe('0')
  })
})
