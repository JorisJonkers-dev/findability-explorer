import { describe, expect, it } from 'vitest'
import { competitor } from '../../../testing/fixtures'
import type { Company } from './company'
import {
  REGIONS,
  beltRows,
  companyKind,
  companySortValue,
  countLabel,
  countryCode,
  familyDots,
  filterCompanies,
  inRegion,
  researchedCompetitors,
  rowLabel,
  selectedCompany,
  termsTitle,
  whereLine,
} from './list'

const company = (name: string, overrides: Parameters<typeof competitor>[1] = {}): Company => ({ ...competitor(name, overrides), shortName: name })

const twentebelt = company('Twentebelt', { country: 'NL (production also in Kudowa-Zdrój, PL)' })
const martens = company('Märtens', { country: 'DE', beltNames: [] })
const ashworth = company('Ashworth', { country: 'US' })
const tribelt = company('Tribelt', { country: 'NL', groupCompany: true })
const all = [twentebelt, martens, ashworth, tribelt]

describe('the company list', () => {
  it('offers five regions', () => {
    expect(REGIONS.map((r) => r.label)).toEqual(['All', 'Netherlands', 'Germany', 'Elsewhere', "Tribelt's group"])
  })

  it('reads the country code at the start of a country', () => {
    expect(countryCode('NL (Panheel)')).toBe('NL')
    expect(countryCode('Netherlands')).toBe('')
  })

  it('puts the group only under All and its own pill', () => {
    expect(inRegion(tribelt, 'all')).toBe(true)
    expect(inRegion(tribelt, 'NL')).toBe(false)
    expect(inRegion(tribelt, 'group')).toBe(true)
    expect(inRegion(twentebelt, 'group')).toBe(false)
    expect(inRegion(twentebelt, 'NL')).toBe(true)
    expect(inRegion(martens, 'DE')).toBe(true)
    expect(inRegion(ashworth, 'other')).toBe(true)
    expect(inRegion(twentebelt, 'other')).toBe(false)
    expect(inRegion(martens, 'other')).toBe(false)
  })

  it('finds companies by name or country, folded', () => {
    expect(filterCompanies(all, '', 'all')).toEqual(all)
    expect(filterCompanies(all, 'MARTENS', 'all')).toEqual([martens])
    expect(filterCompanies(all, 'germany', 'all')).toEqual([martens])
    expect(filterCompanies(all, 'kudowa', 'all')).toEqual([twentebelt])
    expect(filterCompanies(all, '', 'NL')).toEqual([twentebelt])
  })

  it('marks the families a company offers, and says so in words', () => {
    expect(familyDots(twentebelt).filter((d) => d.offers).map((d) => d.family)).toEqual(['tri_flex_turn_curve'])
    expect(rowLabel(twentebelt)).toBe('Twentebelt, Netherlands: Tri-Flex; No scans')
    expect(rowLabel(martens)).toBe('Märtens, Germany: no families recorded; No scans')
  })

  it('counts what is shown', () => {
    expect(countLabel(3, 28)).toBe('3 of 28 companies')
    expect(countLabel(1, 1)).toBe('1 of 1 company')
  })
})

describe('the company card', () => {
  it('names what the company is and where', () => {
    expect(companyKind(tribelt)).toBe("Tribelt's own group")
    expect(companyKind(twentebelt)).toBe('Competitor')
    expect(whereLine(twentebelt)).toBe('Netherlands · website in 2 languages')
    expect(whereLine({ country: 'DE', languages: ['de'] })).toBe('Germany · website in 1 language')
    expect(whereLine({ country: 'DE', languages: ['others unverified'] })).toBe('Germany')
  })

  it("gives its first name for each family, and how many more", () => {
    const rows = beltRows({
      beltNames: [
        { name: 'Flat Wire, Z-Belts', family: 'other' },
        { name: 'TwenteFlex', family: 'tri_flex_turn_curve' },
        { name: 'TwenteFlex Turn', family: 'tri_flex_turn_curve', fromScan: true },
        { name: 'Eyelink belts', family: 'eyelink', fromScan: true },
      ],
    })
    expect(rows.map((r) => [r.label, r.name, r.offered])).toEqual([
      ['Eyelink', 'Eyelink belts, from a scan', true],
      ['Open spiral', 'Not offered', false],
      ['Closed spiral', 'Not offered', false],
      ['Tri-Flex', 'TwenteFlex (and 1 more)', true],
      ['Other belts', 'Flat Wire (and 1 more)', true],
    ])
  })

  it('counts its terms', () => {
    expect(termsTitle(1)).toBe('1 term in the terminology map')
    expect(termsTitle(12)).toBe('12 terms in the terminology map')
  })

  it('shows the company in the address, else the first', () => {
    expect(selectedCompany(all, 'ashworth')).toBe(ashworth)
    expect(selectedCompany(all, 'nobody')).toBe(twentebelt)
    expect(selectedCompany([ashworth, twentebelt], undefined)).toBe(twentebelt)
    expect(selectedCompany([ashworth], undefined)).toBe(ashworth)
    expect(selectedCompany(all, undefined)).toBe(twentebelt)
    expect(selectedCompany<Company>([], undefined)).toBeUndefined()
  })
})

describe('the researched competitors', () => {
  it('leaves out the group and the companies added since', () => {
    const added = company('Example Belts', { source: 'added' })
    expect(researchedCompetitors([...all, added])).toEqual([twentebelt, martens, ashworth])
  })
})

describe('sorting the company list', () => {
  it('sorts by name, by whether a company offers a family, and by country', () => {
    expect(companySortValue(twentebelt, 'name')).toBe('Twentebelt')
    expect(companySortValue(twentebelt, 'tri_flex_turn_curve')).toBe(true)
    expect(companySortValue(twentebelt, 'eyelink')).toBe(false)
    expect(companySortValue(martens, 'country')).toBe('Germany')
  })
})
