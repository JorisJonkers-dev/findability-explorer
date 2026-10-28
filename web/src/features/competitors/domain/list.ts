import type { BeltFamily, Competitor, TermsSource } from '../../../infrastructure/api/types.gen'
import { familyLabel, PRODUCT_FAMILIES } from '../../../shared/vocabulary'
import { fold } from '../../terminology/domain/fold'
import { type Company, listScanLabel } from './company'
import { countryName, familyCell, languageCount } from './matrix'
import type { SortValue } from '../../../shared/sort'

/** Which companies the list shows: all, those in one country, elsewhere, or Tribelt's own group. */
export type Region = 'all' | 'NL' | 'DE' | 'other' | 'group'

export const REGIONS: readonly { value: Region; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'NL', label: 'Netherlands' },
  { value: 'DE', label: 'Germany' },
  { value: 'other', label: 'Elsewhere' },
  { value: 'group', label: "Tribelt's group" },
]

/** The country code a company's country starts with: "NL (Panheel)" is NL; none is ''. */
export function countryCode(country: string): string {
  return /^([A-Z]{2})\b/.exec(country)?.[1] ?? ''
}

/** Whether a company belongs in a region. The group's companies show only under All and the group. */
export function inRegion(company: Pick<Competitor, 'country' | 'groupCompany'>, region: Region): boolean {
  if (region === 'group') return company.groupCompany
  if (company.groupCompany || region === 'all') return region === 'all'
  const code = countryCode(company.country)
  return region === 'other' ? code !== 'NL' && code !== 'DE' : code === region
}

/** The companies in region whose name or country contains query, folded as the server folds. */
export function filterCompanies<T extends Company>(companies: readonly T[], query: string, region: Region): T[] {
  const needle = fold(query)
  return companies
    .filter((c) => inRegion(c, region))
    .filter((c) => needle === '' || [c.shortName, c.name, c.country, countryName(c.country)].some((f) => fold(f).includes(needle)))
}

/** Whether a company offers a belt of each of Tribelt's five families, in the families' order. */
export function familyDots(company: Pick<Competitor, 'beltNames'>): { family: BeltFamily; offers: boolean }[] {
  return PRODUCT_FAMILIES.map((family) => ({ family, offers: company.beltNames.some((b) => b.family === family) }))
}

/** A list row in words: "Twentebelt, Netherlands: Eyelink, Open spiral, Tri-Flex". */
export function rowLabel(company: Company): string {
  const offered = familyDots(company).filter((d) => d.offers).map((d) => familyLabel(d.family))
  const families = offered.length > 0 ? offered.join(', ') : 'no families recorded'
  return `${company.shortName}, ${countryName(company.country)}: ${families}; ${listScanLabel(company)}`
}

/** How many companies are shown of how many there are: "3 of 28 companies". */
export function countLabel(shown: number, total: number): string {
  return `${String(shown)} of ${String(total)} ${total === 1 ? 'company' : 'companies'}`
}

/** What the company is to Tribelt. */
export function companyKind(company: Pick<Competitor, 'groupCompany'>): string {
  return company.groupCompany ? "Tribelt's own group" : 'Competitor'
}

/** The line under a company's name: "Netherlands. Its website is in 2 languages." */
export function whereLine(company: Pick<Competitor, 'country' | 'languages'>): string {
  const count = languageCount(company.languages)
  const site = count > 0 ? ` Its website is in ${String(count)} ${count === 1 ? 'language' : 'languages'}.` : ''
  return `${countryName(company.country)}.${site}`
}

export type BeltRow = { family: BeltFamily; label: string; name: string; offered: boolean }

/** The company's name for each of Tribelt's families: its first, and how many more it has. */
export function beltRows(company: Pick<Competitor, 'beltNames'>): BeltRow[] {
  return PRODUCT_FAMILIES.map((family) => {
    const cell = familyCell(company.beltNames, family)
    const more = cell.more > 0 ? ` (and ${String(cell.more)} more)` : ''
    // A family only a scan found the company offering says so.
    const scanOnly = cell.first !== null && !company.beltNames.some((b) => b.family === family && b.fromScan !== true)
    const name = cell.first === null ? 'Not offered' : `${cell.first}${more}${scanOnly ? ', from a scan' : ''}`
    return { family, label: familyLabel(family), name, offered: cell.first !== null }
  })
}

/**
 * How many of the map's terms a company uses, and how that is known: "45 terms from the search term list, found
 * by the newest scan of tribelt.nl" for Tribelt, "12 terms from the search term list, from our research or found
 * by its newest scan" for another company.
 */
export function termsTitle(count: number, tribelt?: TermsSource): string {
  const terms = `${String(count)} ${count === 1 ? 'term' : 'terms'} from the search term list`
  if (tribelt === 'scan') return `${terms}, found by the newest scan of tribelt.nl`
  if (tribelt === 'research') return `${terms}, from our research until tribelt.nl is scanned`
  return `${terms}, from our research or found by its newest scan`
}

/** The company the view opens on: Twentebelt, the Dutch maker closest to Tribelt. */
export const DEFAULT_COMPANY = 'twentebelt'

/** The company the view shows: the one in the address, else the default, else the first in the list. */
export function selectedCompany<T extends { id: string }>(companies: readonly T[], id: string | undefined): T | undefined {
  return companies.find((c) => c.id === id) ?? companies.find((c) => c.id === DEFAULT_COMPANY) ?? companies[0]
}

/** A column of the company list to sort by: the name, a belt family, or the country. */
export type CompanyColumn = 'name' | (typeof PRODUCT_FAMILIES)[number] | 'country'

/**
 * What a company sorts by in a column: its short name, whether it offers the family (offering
 * companies first when descending), or its country's name.
 */
export function companySortValue(company: Company, column: CompanyColumn): SortValue {
  if (column === 'name') return company.shortName
  if (column === 'country') return countryName(company.country)
  return company.beltNames.some((b) => b.family === column)
}
