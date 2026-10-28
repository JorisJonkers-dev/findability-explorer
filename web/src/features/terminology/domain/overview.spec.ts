import { describe, expect, it } from 'vitest'
import type { OverviewTerms } from '../../../infrastructure/api/types.gen'
import { americanEnglishCount, donutArcs, languageSlices, listOf, pickedLanguage, termsFigures, termsHeadline } from './overview'

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

describe('the headline', () => {
  it('is neutral until there are terms', () => {
    expect(termsHeadline(undefined)).toBe('Ways to search for a metal belt.')
    expect(termsHeadline(terms({ total: 0, tribelt: 0 }))).toBe('Ways to search for a metal belt.')
  })

  it('counts the terms, and says "only" when tribelt.nl uses fewer than half', () => {
    expect(termsHeadline(terms())).toBe('Competitors list belts using 10 terms, tribelt.nl uses only 4 of them.')
    expect(termsHeadline(terms({ tribelt: 5 }))).toBe('Competitors list belts using 10 terms, tribelt.nl uses 5 of them.')
  })
})

describe('the figures', () => {
  it('count the map, what tribelt.nl uses against its target, and what only competitors use', () => {
    expect(termsFigures(terms(), 20)).toEqual([
      { big: '10', text: 'terms in the map', note: 'In Dutch, German, and English.' },
      { big: '4', text: 'are used on tribelt.nl', note: 'The target is 20.' },
      { big: '6', text: 'are used only by competitors', note: 'Each one is a search in which Tribelt does not turn up.' },
    ])
  })

  it('have no notes without languages or a target, and are zero before the figures load', () => {
    const [map, used, others] = termsFigures(undefined, undefined)
    expect(map).toEqual({ big: '0', text: 'terms in the map', note: '' })
    expect(used?.note).toBe('')
    expect(others?.big).toBe('0')
    expect(termsFigures(terms({ total: 2, tribelt: 5 }), undefined)[2]?.big).toBe('0')
  })
})

describe('the language slices', () => {
  it('take the share of the map and a tone each, cycling', () => {
    const slices = languageSlices(terms({ languages: [...terms().languages, { language: 'fr', terms: 0, tribelt: 0 }] }))
    expect(slices.map((s) => [s.value, s.label, s.count, s.tribelt, s.share, s.tone])).toEqual([
      ['nl', 'Dutch', 5, 2, 0.5, 'blue-grey'],
      ['de', 'German', 3, 1, 0.3, 'silver'],
      ['en', 'English', 2, 1, 0.2, 'ink'],
      ['fr', 'French', 0, 0, 0, 'blue-grey'],
    ])
  })

  it('are none before the figures load, and have no share in an empty map', () => {
    expect(languageSlices(undefined)).toEqual([])
    expect(languageSlices(terms({ total: 0 }))[0]?.share).toBe(0)
  })
})

describe('American English', () => {
  it('counts the terms in en-US', () => {
    expect(americanEnglishCount([{ language: 'en-US' }, { language: 'en' }, { language: 'en-US' }])).toBe(2)
    expect(americanEnglishCount([])).toBe(0)
  })
})

describe('the donut arcs', () => {
  it('follow each other round the ring, the picked one wider', () => {
    const arcs = donutArcs(languageSlices(terms()), 'de')
    expect(arcs.map((a) => [a.value, a.width, a.picked])).toEqual([
      ['nl', 26, false],
      ['de', 34, true],
      ['en', 26, false],
    ])
    expect(arcs[0]?.offset).toBeCloseTo(0)
    expect(arcs[1]?.offset).toBeCloseTo(-0.5 * 2 * Math.PI * 76)
    expect(arcs[0]?.dash).toMatch(/^\d+\.\d \d+\.\d$/)
  })

  it('never has a negative dash, and picks none without a pick', () => {
    const arcs = donutArcs(languageSlices(terms({ languages: [{ language: 'nl', terms: 0, tribelt: 0 }] })), null)
    expect(arcs[0]?.dash.startsWith('0.0 ')).toBe(true)
    expect(arcs[0]?.picked).toBe(false)
  })
})

describe('the picked language', () => {
  const [nl, de, en] = languageSlices(terms())

  it('says nothing without a slice', () => {
    expect(pickedLanguage(undefined, 0)).toBeUndefined()
  })

  it('says its share and what tribelt.nl uses', () => {
    expect(pickedLanguage(nl, 0)).toEqual({ title: 'Dutch, 50% of the map', text: '5 terms. tribelt.nl uses 2 of them.' })
  })

  it('says that Germany is where Tribelt wants to grow', () => {
    expect(pickedLanguage(de, 0)?.text).toBe('3 terms. Germany is where Tribelt wants to grow. tribelt.nl uses 1 of them.')
  })

  it('counts American English within English, when there is any', () => {
    expect(pickedLanguage(en, 1)?.text).toBe('2 terms, 1 of them American English. tribelt.nl uses 1 of them.')
    expect(pickedLanguage(en, 0)?.text).toBe('2 terms. tribelt.nl uses 1 of them.')
  })

  it('writes one term in the singular', () => {
    const one = languageSlices(terms({ total: 1, languages: [{ language: 'nl', terms: 1, tribelt: 0 }] }))[0]
    expect(pickedLanguage(one, 0)?.text).toBe('1 term. tribelt.nl uses 0 of them.')
  })
})
