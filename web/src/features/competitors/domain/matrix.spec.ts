import { describe, expect, it } from 'vitest'
import type { Competitor } from '../../../infrastructure/api/types.gen'
import {
  countryName,
  creditedCompanies,
  familyCell,
  filterRows,
  languageCount,
  languageNames,
  matrixRows,
  moreLabel,
  productNames,
  rowSummary,
  shortName,
  tidyName,
  whereLabel,
} from './matrix'

const company = (name: string, overrides: Partial<Competitor> = {}): Competitor => ({
  id: name.toLowerCase(),
  source: 'research',
  role: 'competitor',
  archived: false,
  scans: 0,
  name,
  country: 'NL',
  website: 'https://example.com',
  languages: ['en'],
  groupCompany: false,
  beltNames: [],
  homepageTitle: 'unverified',
  ...overrides,
})

describe('productNames', () => {
  it('splits a list at commas and semicolons outside brackets', () => {
    expect(productNames('Flat Wire, Z-Baking Belts, Cleatrac; WG Woven Wire')).toEqual([
      'Flat Wire',
      'Z-Baking Belts',
      'Cleatrac',
      'WG Woven Wire',
    ])
    expect(productNames('Eye-Link belts (OGBL, OGBL Plus)')).toEqual(['Eye-Link belts'])
  })

  it('names a product by the first of its translations', () => {
    expect(productNames('Drahtösengliederband / Loop-joined wire belt / Tapis à œillets')).toEqual(['Drahtösengliederband'])
    expect(productNames('Nastri tipo S.A.O. (spirali alternate, traversino) / S.A.L. (liscio)')).toEqual(['Nastri tipo S.A.O.'])
    expect(productNames('Wire link belts Groups 100/200 (flat)')).toEqual(['Wire link belts Groups 100/200'])
  })

  it('tolerates stray brackets and empty parts', () => {
    expect(productNames('Odd) belt, , Other')).toEqual(['Odd) belt', 'Other'])
  })
})

describe('familyCell and moreLabel', () => {
  const names = [
    { name: 'TwenteFlex spiral belts', family: 'tri_flex_turn_curve' as const },
    { name: 'TwenteSideFlex', family: 'tri_flex_turn_curve' as const },
    { name: 'Eyelink belts', family: 'eyelink' as const },
  ]

  it('takes the first name for a family and counts the rest', () => {
    expect(familyCell(names, 'tri_flex_turn_curve')).toEqual({ family: 'tri_flex_turn_curve', first: 'TwenteFlex spiral belts', more: 1 })
    expect(familyCell(names, 'eyelink')).toEqual({ family: 'eyelink', first: 'Eyelink belts', more: 0 })
    expect(familyCell(names, 'closed_spiral')).toEqual({ family: 'closed_spiral', first: null, more: 0 })
  })

  it('says how many more there are', () => {
    expect(moreLabel(0)).toBe('')
    expect(moreLabel(5)).toBe('and 5 more')
  })
})

describe('where a company is', () => {
  it('names the country by its code', () => {
    expect(countryName('NL (Panheel)')).toBe('Netherlands')
    expect(countryName('US (EU units NL + UK)')).toBe('United States')
    expect(countryName('UK (Sittingbourne)')).toBe('United Kingdom')
    expect(countryName('XX (nowhere)')).toBe('XX')
    expect(countryName('Somewhere (unverified)')).toBe('Somewhere')
  })

  it('counts the languages, not the notes about them', () => {
    expect(languageCount(['en', 'en-GB', 'pt', 'it (catalogue)', 'unverified', 'others unverified'])).toBe(4)
    expect(languageNames(['en', 'de', 'unverified'])).toBe('English and German')
  })

  it('puts the two together', () => {
    expect(whereLabel({ country: 'NL', languages: ['en', 'nl'] })).toBe('Netherlands, 2 languages')
    expect(whereLabel({ country: 'NL', languages: ['en'] })).toBe('Netherlands, 1 language')
    expect(whereLabel({ country: 'DE (Jülich)', languages: ['unverified'] })).toBe('Germany')
  })
})

describe('the short names', () => {
  it('tidies a registered name', () => {
    expect(tidyName('Belt Technologies, Inc.')).toBe('Belt Technologies')
    expect(tidyName('De Spiraal B.V.')).toBe('De Spiraal')
    expect(tidyName('IPCO (formerly Sandvik)')).toBe('IPCO')
    expect(tidyName('GmbH')).toBe('GmbH')
  })

  it('counts who the terms credit, most-credited first, without their notes', () => {
    expect(
      creditedCompanies([
        { usedBy: ['Wire Belt', 'Märtens (cinta)'] },
        { usedBy: ['Märtens'] },
        { usedBy: ['Wire Belt Osterloh', 'Märtens'] },
      ]),
    ).toEqual(['Märtens', 'Wire Belt', 'Wire Belt Osterloh'])
  })

  it("picks the longest credited name that fits the registered one", () => {
    const users = ['Märtens', 'Wire Belt', 'Wire Belt Osterloh', 'Wire Belt UK', 'The Furnace Belt Co.', 'Hein Lehmann', '()']
    expect(shortName('Märtens Transportbänder GmbH (MARTENS)', users)).toBe('Märtens')
    expect(shortName('Wire Belt Company Osterloh GmbH', users)).toBe('Wire Belt Osterloh')
    expect(shortName('Wire Belt Company Ltd', users)).toBe('Wire Belt')
    expect(shortName('The Furnace Belt Company Ltd', users)).toBe('The Furnace Belt Co.')
    expect(shortName('HEIN, LEHMANN GmbH', users)).toBe('Hein Lehmann')
    expect(shortName('PELGER GmbH', users)).toBe('PELGER')
  })
})

describe('the matrix', () => {
  const competitors = [
    company('Twentebelt', { beltNames: [{ name: 'TwenteFlex spiral belts', family: 'tri_flex_turn_curve' }] }),
    company('Lesjöfors (product line)', { country: 'SE', groupCompany: true }),
  ]

  it('has a row per company and a cell per Tribelt family', () => {
    const rows = matrixRows(competitors, ['Twentebelt'])
    expect(rows.map((r) => [r.key, r.name, r.where, r.groupCompany])).toEqual([
      ['Twentebelt', 'Twentebelt', 'Netherlands, 1 language', false],
      ['Lesjöfors (product line)', 'Lesjöfors', 'Sweden, 1 language', true],
    ])
    expect(rows[0]?.cells.map((c) => c.first)).toEqual([null, null, null, 'TwenteFlex spiral belts', null])
  })

  it('filters by name or country, folded', () => {
    const rows = matrixRows(competitors, [])
    expect(filterRows(rows, '').map((r) => r.name)).toEqual(['Twentebelt', 'Lesjöfors'])
    expect(filterRows(rows, 'lesjofors').map((r) => r.name)).toEqual(['Lesjöfors'])
    expect(filterRows(rows, 'sweden').map((r) => r.name)).toEqual(['Lesjöfors'])
    expect(filterRows(rows, 'NL').map((r) => r.name)).toEqual(['Twentebelt'])
  })

  it('says how many rows show', () => {
    expect(rowSummary(25, 25, 'competitors')).toBe('25 competitors')
    expect(rowSummary(3, 25, 'competitors')).toBe('3 of 25 competitors')
  })
})
