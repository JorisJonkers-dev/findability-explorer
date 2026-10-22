import { describe, expect, it } from 'vitest'
import type { Facets } from '../../../infrastructure/api/types.gen'
import {
  anyLanguageCount,
  companyName,
  familyOptions,
  hasFilters,
  languageOptions,
  MAX_TEXT_LENGTH,
  parseFilters,
  resultSummary,
  splitUsedBy,
  type TermFilters,
  tierOptions,
  toRouteQuery,
  toSearchQuery,
} from './filters'

const none: TermFilters = { q: '', language: null, tier: null, family: null, user: '' }

describe('parseFilters', () => {
  it('reads every filter from the query string', () => {
    expect(
      parseFilters({ q: 'spiral ', language: 'de', tier: 'product', family: 'open_spiral', user: 'Twentebelt' }),
    ).toEqual({ q: 'spiral ', language: 'de', tier: 'product', family: 'open_spiral', user: 'Twentebelt' })
  })

  it('has no filters for an empty query string', () => {
    expect(parseFilters({})).toEqual(none)
  })

  it('drops values the contract refuses', () => {
    expect(parseFilters({ language: 'xx', tier: 'narrow', family: 'spiral', q: null })).toEqual(none)
  })

  it('takes the first of a repeated value, and cuts text to the longest the contract accepts', () => {
    expect(parseFilters({ q: ['a', 'b'], user: [null], language: ['nl'] })).toEqual({ ...none, q: 'a', language: 'nl' })
    expect(parseFilters({ q: 'x'.repeat(150) }).q).toHaveLength(MAX_TEXT_LENGTH)
  })
})

describe('toRouteQuery', () => {
  it('keeps only the filters in use, as typed', () => {
    expect(toRouteQuery(none)).toEqual({})
    expect(toRouteQuery({ ...none, q: 'spiral ', family: 'eyelink' })).toEqual({ q: 'spiral ', family: 'eyelink' })
  })
})

describe('toSearchQuery', () => {
  it('trims text and leaves out empty filters', () => {
    expect(toSearchQuery({ ...none, q: '  ', user: ' ' }, 200)).toEqual({ limit: 200 })
    expect(
      toSearchQuery({ q: ' spiral ', language: 'de', tier: 'spec', family: 'all', user: ' Twentebelt ' }, 50),
    ).toEqual({ q: 'spiral', language: 'de', tier: 'spec', family: 'all', user: 'Twentebelt', limit: 50 })
  })
})

describe('hasFilters', () => {
  it('is true once any filter is set', () => {
    expect(hasFilters(none)).toBe(false)
    expect(hasFilters({ ...none, tier: 'spec' })).toBe(true)
    expect(hasFilters({ ...none, user: 'Ashworth' })).toBe(true)
  })
})

const facets: Facets = {
  languages: [
    { value: 'de', count: 10 },
    { value: 'en', count: 9 },
    { value: 'pl', count: 0 },
  ],
  tiers: [{ value: 'product', count: 24 }],
  families: [{ value: 'eyelink', count: 7 }],
}

describe('the facet options', () => {
  it('lists every tier with its count, zero when the facet has none', () => {
    expect(tierOptions(facets).map((o) => [o.value, o.label, o.count])).toEqual([
      ['spec', 'Specification', 0],
      ['product', 'Product', 24],
      ['application', 'Application', 0],
      ['problem', 'Problem', 0],
      ['broad', 'Broad', 0],
    ])
  })

  it('lists the languages some term is in, and the one picked', () => {
    expect(languageOptions(facets, null).map((o) => o.label)).toEqual(['German', 'English'])
    expect(languageOptions(facets, 'pl')).toEqual([
      { value: 'de', label: 'German', count: 10 },
      { value: 'en', label: 'English', count: 9 },
      { value: 'pl', label: 'Polish', count: 0 },
    ])
  })

  it('adds the language counts up to every matching term', () => {
    expect(anyLanguageCount(facets)).toBe(19)
  })

  it('lists every family with its count', () => {
    const options = familyOptions(facets)
    expect(options).toHaveLength(7)
    expect(options[0]).toEqual({ value: 'eyelink', label: 'Eyelink', count: 7 })
    expect(options[3]).toEqual({ value: 'tri_flex_turn_curve', label: 'Tri-Flex', count: 0 })
  })
})

describe('resultSummary', () => {
  it('says how many terms match of how many the map holds', () => {
    expect(resultSummary(0, 146)).toBe('No terms match')
    expect(resultSummary(1, 0)).toBe('1 term')
    expect(resultSummary(146, 146)).toBe('146 terms')
    expect(resultSummary(35, 146)).toBe('35 of 146 terms')
    expect(resultSummary(1, 1)).toBe('1 term')
  })
})

describe('companyName', () => {
  it('drops a trailing bracketed product line or division', () => {
    expect(companyName('Twentebelt (TwenteFlex)')).toBe('Twentebelt')
    expect(companyName('Campbelt (CMG) ')).toBe('Campbelt')
  })

  it('keeps a name without one, and a name that is only brackets', () => {
    expect(companyName(' Märtens ')).toBe('Märtens')
    expect(companyName('Wire Belt (UK) Ltd')).toBe('Wire Belt (UK) Ltd')
    expect(companyName('(unknown)')).toBe('(unknown)')
  })
})

describe('splitUsedBy', () => {
  it('splits off the note in brackets', () => {
    expect(splitUsedBy('Twentebelt (TwenteFlex)')).toEqual({ company: 'Twentebelt', note: 'TwenteFlex' })
    expect(splitUsedBy('Ashworth')).toEqual({ company: 'Ashworth', note: '' })
    expect(splitUsedBy('(unknown)')).toEqual({ company: '(unknown)', note: '' })
  })
})
