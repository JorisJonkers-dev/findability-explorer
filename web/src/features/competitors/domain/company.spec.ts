import { describe, expect, it } from 'vitest'
import type { Term } from '../../../infrastructure/api/types.gen'
import { competitor } from '../../../testing/fixtures'
import {
  companyByHost,
  companyTerms,
  groupBeltNames,
  hostOf,
  MAX_SUGGESTIONS,
  normalizeHost,
  pickerState,
  readableName,
  scanCompany,
  scanCountLabel,
  scanOutcome,
  sourceLabel,
  suggestions,
  withShortNames,
} from './company'

const twentebelt = competitor('Twentebelt B.V.', {
  websiteUrl: 'https://www.twentebelt.com/',
  host: 'twentebelt.com',
  website: 'https://www.twentebelt.com',
})
const martens = competitor('Märtens Transportbänder GmbH', {
  websiteUrl: 'https://www.martens-belts.com/',
  host: 'martens-belts.com',
  website: 'https://www.martens-belts.com',
  scans: 2,
})
const unknownSite = competitor('Cambridge Engineered Solutions', { websiteUrl: undefined, host: undefined, website: 'none found' })
const companies = withShortNames([twentebelt, martens, unknownSite], ['Twentebelt', 'Märtens', 'Cambridge'])

describe('withShortNames', () => {
  it('gives each company the name the terminology credits', () => {
    expect(companies.map((c) => c.shortName)).toEqual(['Twentebelt', 'Märtens', 'Cambridge'])
  })
})

describe('hosts', () => {
  it('compares hosts in lower case, without www.', () => {
    expect(normalizeHost('WWW.Twentebelt.com')).toBe('twentebelt.com')
    expect(hostOf('https://www.twentebelt.com/belts')).toBe('twentebelt.com')
    expect(hostOf('not an address')).toBeNull()
    expect(hostOf('file:///etc/hosts')).toBeNull()
  })

  it('finds a company by its host', () => {
    expect(companyByHost(companies, 'www.Martens-Belts.com')?.id).toBe(martens.id)
    expect(companyByHost(companies, 'example.com')).toBeUndefined()
  })

  it('makes a readable name from a host', () => {
    expect(readableName('www.example-belts.co.uk')).toBe('Example Belts')
    expect(readableName('belts_and_more.de')).toBe('Belts And More')
    expect(readableName('localhost')).toBe('Localhost')
    expect(readableName('-.com')).toBe('-.com')
  })
})

describe('suggestions', () => {
  it('suggests companies with a website by name, host or address, folded', () => {
    expect(suggestions(companies, 'twente').map((c) => c.shortName)).toEqual(['Twentebelt'])
    expect(suggestions(companies, 'MARTENS').map((c) => c.shortName)).toEqual(['Märtens'])
    expect(suggestions(companies, 'https://www.martens-b').map((c) => c.shortName)).toEqual(['Märtens'])
    expect(suggestions(companies, 'cambridge')).toEqual([])
    expect(suggestions(companies, '  ')).toEqual([])
    const hostless = withShortNames([competitor('Wire Belt', { host: undefined, website: 'https://wirebelt.co.uk' })], [])
    expect(suggestions(hostless, 'wirebelt').map((c) => c.name)).toEqual(['Wire Belt'])
  })

  it('suggests at most a handful', () => {
    const many = withShortNames(
      Array.from({ length: 12 }, (_, i) => competitor(`Belts ${String(i)}`, { websiteUrl: `https://belts${String(i)}.com/` })),
      [],
    )
    expect(suggestions(many, 'belts')).toHaveLength(MAX_SUGGESTIONS)
  })
})

describe('pickerState', () => {
  it('waits for a whole address', () => {
    expect(pickerState('twente', companies)).toEqual({ kind: 'none' })
  })

  it('knows a known host, and names a new one after its host', () => {
    const known = pickerState('https://twentebelt.com/x', companies)
    expect(known.kind === 'known' && known.company.id).toBe(twentebelt.id)
    expect(pickerState('www.example-belts.de', companies)).toEqual({
      kind: 'new',
      host: 'example-belts.de',
      name: 'Example Belts',
    })
  })
})

describe('labels', () => {
  it('counts scans', () => {
    expect(scanCountLabel(0)).toBe('Not scanned yet')
    expect(scanCountLabel(3)).toBe('Scanned 3×')
  })

  it('says where a company comes from', () => {
    expect(sourceLabel({ source: 'research' })).toBe('Researched')
    expect(sourceLabel({ source: 'added', addedAt: '2026-10-22T10:00:00Z' })).toBe('Added 22 October 2026')
    expect(sourceLabel({ source: 'added' })).toBe('Added')
  })

  it('says how a scan ended', () => {
    expect(scanOutcome({ status: 'running', pagesCrawled: 2 })).toBe('Scanning')
    expect(scanOutcome({ status: 'failed', pagesCrawled: 0 })).toBe('Failed')
    expect(scanOutcome({ status: 'done', pagesCrawled: 1 })).toBe('1 page read')
    expect(scanOutcome({ status: 'done', pagesCrawled: 12 })).toBe('12 pages read')
  })
})

describe('groupBeltNames', () => {
  it('groups belt names by family, in the families order', () => {
    expect(
      groupBeltNames([
        { name: 'TwenteFlex', family: 'tri_flex_turn_curve' },
        { name: 'Eyelink belt', family: 'eyelink' },
        { name: 'TwenteSideFlex', family: 'tri_flex_turn_curve' },
      ]),
    ).toEqual([
      { family: 'eyelink', label: 'Eyelink', names: ['Eyelink belt'] },
      { family: 'tri_flex_turn_curve', label: 'Tri-Flex', names: ['TwenteFlex', 'TwenteSideFlex'] },
    ])
  })
})

describe('companyTerms', () => {
  const term = (name: string, tier: Term['tier'], usedBy: string[]) => ({ term: name, tier, usedBy })

  it("lists the terms credited to the company, with or without a product line, the most specific first", () => {
    const terms = [
      term('metal belt', 'broad', ['Twentebelt']),
      term('spiral belt', 'product', ['Twentebelt (TwenteFlex)']),
      term('balanced weave', 'product', ['Märtens', 'Twentebelt']),
      term('wire mesh', 'spec', ['Märtens']),
    ]
    expect(companyTerms(terms, 'Twentebelt').map((t) => t.term)).toEqual(['balanced weave', 'spiral belt', 'metal belt'])
  })
})

describe('scanCompany', () => {
  it('finds the company by the id the server matched, else by host, else names the host', () => {
    expect(scanCompany({ url: 'https://example.com', competitorId: martens.id }, companies).name).toBe('Märtens')
    expect(scanCompany({ url: 'https://www.twentebelt.com/' }, companies).name).toBe('Twentebelt')
    expect(scanCompany({ url: 'https://www.example.com/' }, companies)).toEqual({ name: 'example.com', company: undefined })
    expect(scanCompany({ url: 'nonsense' }, companies)).toEqual({ name: 'nonsense', company: undefined })
  })
})
