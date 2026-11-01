import { describe, expect, it } from 'vitest'
import { term as apiTerm } from '../../../testing/fixtures'
import type { MapTerm } from './makers'
import {
  BUYER_INTENT_LIST,
  familyName,
  familyTable,
  findTerm,
  languageTable,
  relatedTerms,
  termId,
  termPath,
  termSentence,
  topMissing,
} from './searchTerms'

const term = (overrides: Partial<MapTerm> = {}): MapTerm => ({ ...apiTerm(), scanMakers: [], ...overrides })

describe('a term address', () => {
  it('is its language and its words, and finds the term again', () => {
    const t = term({ term: 'Drahtösenband, 10 mm', language: 'de' })
    expect(termId(t)).toBe('de--drahtösenband-10-mm')
    expect(termPath(t)).toBe(`/terms/${encodeURIComponent('de--drahtösenband-10-mm')}`)
    expect(termId(term({ term: 'Eyelink', language: 'en-US' }))).toBe('en-us--eyelink')
    expect(termId(term({ term: '/ / /', language: 'en' }))).toBe('en--term')
    const terms = [term({ term: 'a' }), t]
    expect(findTerm(terms, 'de--drahtösenband-10-mm')).toBe(t)
    expect(findTerm(terms, 'nl--nothing')).toBeUndefined()
  })
})

describe('the tables of Search terms', () => {
  const terms = [
    term({ term: 'a', language: 'nl', family: 'eyelink', tribeltUses: true }),
    term({ term: 'b', language: 'nl', family: 'eyelink', usedBy: ['Ashworth'] }),
    term({ term: 'c', language: 'de', family: 'open_spiral', usedBy: [] }),
    term({ term: 'd', language: 'en-US', family: 'all', usedBy: ['Ashworth', 'Twentebelt'] }),
    term({ term: 'e', language: 'en', family: 'spec_all', tribeltUses: true }),
    term({ term: 'f', language: 'fr', family: 'other' }),
  ]

  it('counts each of the three languages, American English as English, and all three', () => {
    expect(languageTable(terms)).toEqual({
      rows: [
        { language: 'nl', label: 'Dutch', total: 2, used: 1, only: 1 },
        { language: 'de', label: 'German', total: 1, used: 0, only: 0 },
        { language: 'en', label: 'English', total: 2, used: 1, only: 1 },
      ],
      all: { total: 5, used: 2, only: 2 },
    })
  })

  it("counts each belt family with Tribelt's own names, and the terms for every belt apart", () => {
    const products = [{ family: 'eyelink' as const, tribeltNames: ['Draadogenbanden', 'Eyelink belts'] }]
    const table = familyTable(terms, products)
    expect(table.rows.map((r) => [r.family, r.label, r.names, r.total, r.used])).toEqual([
      ['eyelink', 'Eyelink belts', 'Draadogenbanden, Eyelink belts', 2, 1],
      ['open_spiral', 'Open spiral belts', 'None recorded', 1, 0],
      ['closed_spiral', 'Closed spiral belts', 'None recorded', 0, 0],
      ['tri_flex_turn_curve', 'Tri-Flex curve and spiral belts', 'None recorded', 0, 0],
      ['other', 'Other metal belts', 'None recorded', 1, 0],
    ])
    expect(table.all).toEqual({ total: 4, used: 1 })
    expect(table.shared).toBe('Not counted above: 2 terms that apply to every belt, 1 of them used')
    expect(familyTable([term({ family: 'all' })], []).shared).toBe('Not counted above: 1 term that applies to every belt, 0 of them used')
  })

  it('ranks the terms competitors use and tribelt.nl does not, by how many use them', () => {
    expect(topMissing(terms, 2).map((t) => t.term)).toEqual(['d', 'b'])
    expect(topMissing(terms).map((t) => t.term)).toEqual(['d', 'b', 'f'])
  })

  it('names every belt family as the pages head them', () => {
    expect(familyName('spec_all')).toBe('Specifications for all belts')
    expect(familyName('tri_flex_turn_curve')).toBe('Tri-Flex curve and spiral belts')
  })

  it('says what each kind of buyer intent means', () => {
    expect(BUYER_INTENT_LIST[0]).toEqual({ tier: 'spec', label: 'Specification', help: 'The buyer already knows the exact belt and its sizes.' })
    expect(BUYER_INTENT_LIST).toHaveLength(5)
  })
})

describe('one search term', () => {
  it('says its language, belt family, competitors, and buyer intent in a sentence', () => {
    expect(termSentence(term({ term: 'x', language: 'en', family: 'eyelink', usedBy: ['Ashworth', 'Twentebelt'] }), 12)).toBe(
      'An English term for eyelink belts. 2 of the 12 competitors use it, and tribelt.nl does not. Its buyer intent is product. The buyer names the type of belt.',
    )
    expect(termSentence(term({ language: 'nl', family: 'spec_all', tier: 'spec', tribeltUses: true }), 1)).toBe(
      'A Dutch term for the specifications of all belts. 1 of the 1 competitors uses it, and tribelt.nl uses it too. Its buyer intent is specification. The buyer already knows the exact belt and its sizes.',
    )
  })

  it('lists other terms of the same family and language, the most used first', () => {
    const t = term({ term: 'x', language: 'nl', family: 'eyelink' })
    const terms = [
      t,
      term({ term: 'b', language: 'nl', family: 'eyelink', usedBy: [] }),
      term({ term: 'a', language: 'nl', family: 'eyelink', usedBy: ['A', 'B'] }),
      term({ term: 'c', language: 'de', family: 'eyelink' }),
      term({ term: 'd', language: 'nl', family: 'other' }),
    ]
    expect(relatedTerms(terms, t).map((r) => r.term)).toEqual(['a', 'b'])
    expect(relatedTerms(terms, t, 1).map((r) => r.term)).toEqual(['a'])
  })
})
