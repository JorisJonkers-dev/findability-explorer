import { describe, expect, it } from 'vitest'
import type { Term } from '../../../infrastructure/api/types.gen'
import { term } from '../../../testing/fixtures'
import { NO_VIEW } from './filters'
import {
  cellBackground,
  chips,
  filterTerms,
  groupOf,
  heatmap,
  languageRows,
  makersOf,
  rankTerms,
  summary,
  termsOf,
  tierRows,
  tribeltUses,
} from './terms'

const MAP: Term[] = [
  term({ term: 'Eyelink belt', language: 'en', family: 'eyelink', tier: 'product', usedBy: ['Twentebelt', 'Ashworth', 'Tribelt'] }),
  term({ term: 'eye link belt', language: 'en-US', family: 'eyelink', tier: 'broad', usedBy: ['Ashworth'] }),
  term({ term: 'Drahtösengliederband', language: 'de', family: 'eyelink', tier: 'product', usedBy: ['Märtens', 'Tribelt (Drahtösengliederbänder)'] }),
  term({ term: 'AISI 304', language: 'de', family: 'spec_all', tier: 'spec', usedBy: ['Märtens', 'Märtens (Typ 2)', 'Twentebelt'] }),
  term({ term: 'belt tracking', language: 'en', family: 'all', tier: 'problem', usedBy: [] }),
]

describe('who uses a term', () => {
  it("knows Tribelt's own terms, with or without a note in brackets", () => {
    expect(MAP.map(tribeltUses)).toEqual([true, false, true, false, false])
  })

  it('counts each other company once', () => {
    expect(makersOf(MAP[3] as Term)).toEqual(['Märtens', 'Twentebelt'])
    expect(makersOf(MAP[0] as Term)).toEqual(['Twentebelt', 'Ashworth'])
  })

  it('lists the terms of a company, and of Tribelt', () => {
    expect(termsOf(MAP, 'Ashworth').map((t) => t.term)).toEqual(['Eyelink belt', 'eye link belt'])
    expect(termsOf(MAP, 'Tribelt').map((t) => t.term)).toEqual(['Eyelink belt', 'Drahtösengliederband'])
  })

  it('splits the kinds of search into marketing and technical', () => {
    expect(['broad', 'product', 'application', 'spec', 'problem'].map((t) => groupOf(t as Term['tier']))).toEqual([
      'marketing',
      'marketing',
      'marketing',
      'technical',
      'technical',
    ])
  })
})

describe('the charts', () => {
  it('counts terms per language, American English as English, the most first', () => {
    expect(languageRows(MAP)).toEqual([
      { value: 'en', label: 'English', count: 3, tribelt: 1, tribeltShare: 1 / 3, otherShare: 2 / 3, description: 'English: 3 terms, Tribelt uses 1' },
      { value: 'de', label: 'German', count: 2, tribelt: 1, tribeltShare: 1 / 3, otherShare: 1 / 3, description: 'German: 2 terms, Tribelt uses 1' },
    ])
  })

  it('breaks a tie by name', () => {
    const rows = languageRows([term({ language: 'nl' }), term({ language: 'da' })])
    expect(rows.map((r) => r.label)).toEqual(['Danish', 'Dutch'])
  })

  it('counts terms per kind of search, in the charts order, empty ones too', () => {
    const rows = tierRows(MAP)
    expect(rows.map((r) => [r.value, r.count, r.tribelt])).toEqual([
      ['broad', 1, 0],
      ['product', 2, 2],
      ['application', 0, 0],
      ['spec', 1, 0],
      ['problem', 1, 0],
    ])
    expect(rows[1]?.tribeltShare).toBe(1)
    expect(rows[2]?.description).toBe('Application: 0 terms, Tribelt uses 0')
    expect(rows[0]?.description).toBe('Broad: 1 term, Tribelt uses 0')
  })

  it('draws nothing for an empty map', () => {
    expect(languageRows([])).toEqual([])
    expect(tierRows([]).every((r) => r.otherShare === 0)).toBe(true)
  })
})

describe('the heatmap', () => {
  it('lists every company and Tribelt, the most terms first, then by name', () => {
    const rows = heatmap(MAP)
    expect(rows.map((r) => [r.company, r.marketing, r.technical, r.total])).toEqual([
      ['Ashworth', 2, 0, 2],
      ['Märtens', 1, 1, 2],
      ['Tribelt', 2, 0, 2],
      ['Twentebelt', 1, 1, 2],
    ])
    expect(rows[1]?.description).toBe('Märtens: 1 marketing terms, 1 technical terms')
  })

  it('shades each cell against the fullest cell', () => {
    const tribelt = heatmap(MAP).find((r) => r.company === 'Tribelt')
    expect(tribelt?.cells.map((c) => [c.tier, c.count, c.technical, c.intensity])).toEqual([
      ['broad', 0, false, 0],
      ['product', 2, false, 1],
      ['application', 0, false, 0],
      ['spec', 0, true, 0],
      ['problem', 0, true, 0],
    ])
    const martens = heatmap(MAP).find((r) => r.company === 'Märtens')
    expect(martens?.cells[3]?.intensity).toBe(0.625)
  })

  it('has Tribelt alone, unshaded, for an empty map', () => {
    expect(heatmap([]).map((r) => [r.company, r.total])).toEqual([['Tribelt', 0]])
  })

  it('colours marketing blue-grey and technical orange, and an empty cell not at all', () => {
    expect(cellBackground({ intensity: 0.625, technical: true })).toBe('rgba(245, 117, 3, 0.63)')
    expect(cellBackground({ intensity: 1, technical: false })).toBe('rgba(141, 170, 191, 1.00)')
    expect(cellBackground({ intensity: 0, technical: false })).toBeUndefined()
  })
})

describe('filterTerms', () => {
  const names = (view: Partial<typeof NO_VIEW>) => filterTerms(MAP, { ...NO_VIEW, ...view }).map((t) => t.term)

  it('lets everything through without filters', () => {
    expect(names({})).toHaveLength(5)
  })

  it('filters by group, kind, language, company and what Tribelt lacks', () => {
    expect(names({ group: 'technical' })).toEqual(['AISI 304', 'belt tracking'])
    expect(names({ group: 'marketing' })).toEqual(['Eyelink belt', 'eye link belt', 'Drahtösengliederband'])
    expect(names({ tier: 'product' })).toEqual(['Eyelink belt', 'Drahtösengliederband'])
    expect(names({ language: 'en' })).toEqual(['Eyelink belt', 'eye link belt', 'belt tracking'])
    expect(names({ user: 'Märtens' })).toEqual(['Drahtösengliederband', 'AISI 304'])
    expect(names({ user: 'Tribelt' })).toEqual(['Eyelink belt', 'Drahtösengliederband'])
    expect(names({ gap: true })).toEqual(['eye link belt', 'AISI 304', 'belt tracking'])
  })

  it('searches the term as the server folds it', () => {
    expect(names({ q: ' DRAHTÖSEN' })).toEqual(['Drahtösengliederband'])
    expect(names({ q: 'eye-link' })).toEqual(['eye link belt'])
  })

  it('combines filters', () => {
    expect(names({ language: 'de', group: 'technical' })).toEqual(['AISI 304'])
  })
})

describe('rankTerms', () => {
  it('ranks by makers, then the more technical kind, then the alphabet', () => {
    const ranked = rankTerms(MAP, MAP)
    expect(ranked.map((r) => [r.rank, r.term, r.makers])).toEqual([
      [1, 'AISI 304', 2],
      [2, 'Eyelink belt', 2],
      [3, 'Drahtösengliederband', 1],
      [4, 'eye link belt', 1],
      [5, 'belt tracking', 0],
    ])
  })

  it('marks the makers only a scan found', () => {
    const scanned = { ...term({ term: 'spiral belt', usedBy: ['Märtens', 'Pelger'] }), scanMakers: ['Pelger'] }
    expect(rankTerms([scanned], [scanned])[0]?.meta).toBe('English · Tri-Flex · Märtens, Pelger (scan)')
  })

  it('describes each term and measures its bar against the whole map', () => {
    const [first] = rankTerms([MAP[2] as Term], MAP)
    expect(first).toEqual({
      rank: 1,
      term: 'Drahtösengliederband',
      meta: 'German · Eyelink · Märtens',
      tier: 'product',
      kind: 'Product',
      makers: 1,
      share: 0.5,
      tribelt: true,
    })
    expect(rankTerms([MAP[3] as Term], MAP)[0]?.meta).toBe('German · Any belt · Märtens, Twentebelt')
    expect(rankTerms([MAP[4] as Term], MAP)[0]?.meta).toBe('English · All belts')
  })

  it('puts terms of the same makers and kind in alphabetical order', () => {
    const same = [term({ term: 'b', usedBy: ['X'] }), term({ term: 'a', usedBy: ['Y'] })]
    expect(rankTerms(same, same).map((r) => r.term)).toEqual(['a', 'b'])
  })

  it('draws no bars for a map no maker uses', () => {
    expect(rankTerms([MAP[4] as Term], [MAP[4] as Term])[0]?.share).toBe(0)
  })
})

describe('chips and summary', () => {
  it('offers a chip for each chart and heatmap filter in use', () => {
    expect(chips(NO_VIEW)).toEqual([])
    expect(chips({ ...NO_VIEW, language: 'de', tier: 'spec', user: 'Märtens', group: 'technical', gap: true })).toEqual([
      { key: 'language', label: 'German' },
      { key: 'tier', label: 'Specification' },
      { key: 'user', label: 'Märtens' },
    ])
  })

  it('says how many terms are shown of the map', () => {
    expect(summary(35, 146)).toBe('35 of 146 terms')
    expect(summary(146, 146)).toBe('146 of 146 terms')
    expect(summary(0, 1)).toBe('0 of 1 term')
  })
})
