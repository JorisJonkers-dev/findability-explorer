import { describe, expect, it } from 'vitest'
import type { OverviewTerms, Term } from '../../../infrastructure/api/types.gen'
import { term } from '../../../testing/fixtures'
import { NO_VIEW } from './filters'
import {
  cellBackground,
  chips,
  filterTerms,
  groupOf,
  heatmap,
  heatSortValue,
  languageRows,
  makersOf,
  rankSortValue,
  rankTerms,
  summary,
  termsOf,
  tierRows,
  tribeltTerm,
  tribeltUses,
  usedLine,
} from './terms'

const MAP: Term[] = [
  term({ term: 'Eyelink belt', language: 'en', family: 'eyelink', tier: 'product', usedBy: ['Twentebelt', 'Ashworth', 'Tribelt'], tribeltUses: true }),
  term({ term: 'eye link belt', language: 'en-US', family: 'eyelink', tier: 'broad', usedBy: ['Ashworth'] }),
  term({ term: 'Drahtösengliederband', language: 'de', family: 'eyelink', tier: 'product', usedBy: ['Märtens', 'Tribelt (Drahtösengliederbänder)'], tribeltUses: true }),
  term({ term: 'AISI 304', language: 'de', family: 'spec_all', tier: 'spec', usedBy: ['Märtens', 'Märtens (Typ 2)', 'Twentebelt'] }),
  term({ term: 'belt tracking', language: 'en', family: 'all', tier: 'problem', usedBy: [] }),
]

describe('who uses a term', () => {
  it('knows the terms tribelt.nl uses from the server, whatever the research credits', () => {
    expect(MAP.map(tribeltUses)).toEqual([true, false, true, false, false])
    // The research credits Tribelt, but tribelt.nl's newest scan did not find it.
    expect(tribeltUses(term({ usedBy: ['Tribelt'], tribeltUses: false }))).toBe(false)
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
  const OVERVIEW: OverviewTerms = {
    total: 5,
    tribelt: 2,
    from: 'scan',
    languages: [
      { language: 'en', terms: 3, tribelt: 1 },
      { language: 'de', terms: 2, tribelt: 1 },
    ],
    tiers: [
      { tier: 'product', terms: 2, tribelt: 2 },
      { tier: 'spec', terms: 1, tribelt: 0 },
      { tier: 'problem', terms: 1, tribelt: 0 },
      { tier: 'broad', terms: 1, tribelt: 0 },
    ],
  }

  it('draws the terms per language as the server counts them', () => {
    expect(languageRows(OVERVIEW)).toEqual([
      { value: 'en', label: 'English', count: 3, tribelt: 1, tribeltShare: 1 / 3, otherShare: 2 / 3, description: 'English: 3 terms, tribelt.nl uses 1' },
      { value: 'de', label: 'German', count: 2, tribelt: 1, tribeltShare: 1 / 3, otherShare: 1 / 3, description: 'German: 2 terms, tribelt.nl uses 1' },
    ])
  })

  it('draws the terms per kind of search in the charts order, empty ones too', () => {
    const rows = tierRows(OVERVIEW)
    expect(rows.map((r) => [r.value, r.count, r.tribelt])).toEqual([
      ['broad', 1, 0],
      ['product', 2, 2],
      ['application', 0, 0],
      ['spec', 1, 0],
      ['problem', 1, 0],
    ])
    expect(rows[1]?.tribeltShare).toBe(1)
    expect(rows[2]?.description).toBe('Application: 0 terms, tribelt.nl uses 0')
    expect(rows[0]?.description).toBe('Broad: 1 term, tribelt.nl uses 0')
  })

  it('draws nothing before the figures load', () => {
    expect(languageRows(undefined)).toEqual([])
    expect(tierRows(undefined).every((r) => r.count === 0 && r.otherShare === 0)).toBe(true)
    expect(tierRows({ ...OVERVIEW, tiers: undefined }).every((r) => r.count === 0)).toBe(true)
  })

  it('says how many terms tribelt.nl uses, and how that is known', () => {
    expect(usedLine(OVERVIEW)).toBe('The newest scan of tribelt.nl found 2 of the 5 terms.')
    expect(usedLine({ ...OVERVIEW, from: 'research' })).toBe('Our research found 2 of the 5 terms for Tribelt.')
    expect(usedLine(undefined)).toBe('')
    expect(usedLine({ ...OVERVIEW, total: 0 })).toBe('')
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
      tribeltTerm: 'Uses it',
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

describe('sorting the tables', () => {
  it('sorts the per-company table by company, kind and total, and the ranking by any column', () => {
    const [row] = heatmap(MAP)
    if (!row) throw new Error('no row')
    expect(heatSortValue(row, 'company')).toBe(row.company)
    expect(heatSortValue(row, 'total')).toBe(row.total)
    expect(heatSortValue(row, 'product')).toBe(row.cells.find((c) => c.tier === 'product')?.count)
    expect(heatSortValue({ ...row, cells: [] }, 'spec')).toBe(0)
    const [ranked] = rankTerms(MAP, MAP)
    if (!ranked) throw new Error('no term')
    expect(rankSortValue(ranked, 'makers')).toBe(ranked.makers)
    expect(rankSortValue(ranked, 'tribelt')).toBe(ranked.tribelt)
  })
})

describe("Tribelt's term", () => {
  it('says Tribelt uses a term, names its own term for the same thing, or that it lacks one', () => {
    expect(MAP.map((t) => tribeltTerm(t, MAP))).toEqual(['Uses it', 'Eyelink belt', 'Uses it', 'Lacks it', 'Lacks it'])
    // A term is never paired with Tribelt's term in another language.
    const dutch = term({ term: 'draadogenband', language: 'nl', family: 'eyelink', tier: 'product', usedBy: ['ESFO'] })
    expect(tribeltTerm(dutch, [...MAP, dutch])).toBe('Lacks it')
    const own = term({ term: 'oogjesband', language: 'nl', family: 'eyelink', tier: 'broad', usedBy: ['Tribelt'], tribeltUses: true })
    expect(tribeltTerm(dutch, [...MAP, dutch, own])).toBe('oogjesband')
    // The owner's case: Tribelt's German "kurvengängig" is no stand-in for the English "flex-turn belt".
    const flexTurn = term({ term: 'flex-turn belt', language: 'en', family: 'tri_flex_turn_curve', tier: 'product', usedBy: ['Ashworth'] })
    const kurvengaengig = term({ term: 'kurvengängig', language: 'de', family: 'tri_flex_turn_curve', tier: 'application', usedBy: ['Tribelt'], tribeltUses: true })
    expect(tribeltTerm(flexTurn, [flexTurn, kurvengaengig])).toBe('Lacks it')
    expect(rankTerms([flexTurn], [flexTurn, kurvengaengig])[0]?.tribeltTerm).toBe('Lacks it')
    // A specification is its own thing: Tribelt's other specification is no stand-in for it.
    const pitch = term({ term: 'belt pitch', language: 'en', family: 'spec_all', tier: 'spec', usedBy: ['Ashworth'] })
    const steel = term({ term: 'AISI 316', language: 'en', family: 'spec_all', tier: 'spec', usedBy: ['Tribelt'], tribeltUses: true })
    expect(tribeltTerm(pitch, [pitch, steel])).toBe('Lacks it')
    // Nor is a specification a stand-in for a product name of the same family.
    const ratio = term({ term: 'turn ratio', language: 'en', family: 'tri_flex_turn_curve', tier: 'spec', usedBy: ['Tribelt'], tribeltUses: true })
    const spiral = term({ term: 'spiral belt', language: 'en', family: 'tri_flex_turn_curve', tier: 'product', usedBy: ['Ashworth'] })
    expect(tribeltTerm(spiral, [spiral, ratio])).toBe('Lacks it')
  })
})
