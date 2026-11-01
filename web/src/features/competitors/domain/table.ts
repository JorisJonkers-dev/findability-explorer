import type { BeltFamily } from '../../../infrastructure/api/types.gen'
import { type Sort, type SortValue, sortRows } from '../../../shared/sort'
import { familyLabel, PRODUCT_FAMILIES } from '../../../shared/vocabulary'
import { oxford } from '../../../shared/words'
import type { Company } from './company'
import { longDate } from '../../../shared/dates'

/** One row of the Competitors table: Tribelt or a competitor, with what each column shows. */
export type CompetitorRow = {
  id: string
  name: string
  tribelt: boolean
  archived: boolean
  country: string
  /** The belt families of Tribelt's five it sells, by their short names. */
  families: string[]
  familyIds: BeltFamily[]
  /** The share of AI responses that name it; none before a Prompt run. */
  share: number | undefined
  /** Its AI readiness score; none before a scan read the site. */
  ready: number | undefined
  /** How many terms of the search term list it uses. */
  terms: number
  /** When its newest scan finished, or started while it runs. */
  scannedAt: string | undefined
}

/** What the table measures of a company, by its id. */
export type Measures = { share: (id: string) => number | undefined; ready: (id: string) => number | undefined; terms: (id: string) => number }

/** Tribelt and every competitor, archived ones marked; the group's own companies and the mirror are left out. */
export function competitorRows(companies: readonly Company[], measures: Measures): CompetitorRow[] {
  return companies
    .filter((c) => c.role === 'tribelt' || c.role === 'competitor')
    .map((c) => {
      const familyIds = PRODUCT_FAMILIES.filter((f) => c.beltNames.some((b) => b.family === f))
      return {
        id: c.id,
        name: c.shortName,
        tribelt: c.role === 'tribelt',
        archived: c.archived,
        country: c.country.trim(),
        families: familyIds.map(familyLabel),
        familyIds,
        share: measures.share(c.id),
        ready: measures.ready(c.id),
        terms: measures.terms(c.id),
        scannedAt: c.lastScan ? (c.lastScan.finishedAt ?? c.lastScan.createdAt) : undefined,
      }
    })
}

/** The belt families a row sells, as "n of 5", with their names for the cell's title. */
export function familiesCell(row: Pick<CompetitorRow, 'families'>): { text: string; title: string } {
  if (row.families.length === 0) return { text: 'None recorded', title: '' }
  return { text: `${String(row.families.length)} of ${String(PRODUCT_FAMILIES.length)}`, title: oxford(row.families) }
}

export type CompetitorColumn = 'name' | 'country' | 'families' | 'share' | 'ready' | 'terms' | 'scan'

/** What a column sorts a row by; a figure the row lacks sorts below every other. */
export function columnValue(row: CompetitorRow, key: CompetitorColumn): SortValue {
  switch (key) {
    case 'name':
      return row.name
    case 'country':
      return row.country
    case 'families':
      return row.families.length
    case 'share':
      return row.share ?? -1
    case 'ready':
      return row.ready ?? -1
    case 'terms':
      return row.terms
    case 'scan':
      return row.scannedAt ?? ''
  }
}

/** What the table is narrowed to: a name, a belt family, a country, and whether archived companies show. */
export type CompetitorFilter = { q: string; family: BeltFamily | ''; country: string; archived: boolean }

export const NO_FILTER: CompetitorFilter = { q: '', family: '', country: '', archived: false }

/** The table's rows: Tribelt pinned on top, then the competitors that pass the filter, sorted; the most named first unless asked otherwise. */
export function competitorTable(rows: readonly CompetitorRow[], filter: CompetitorFilter, sort: Sort<CompetitorColumn>): CompetitorRow[] {
  const q = filter.q.trim().toLowerCase()
  const shown = rows.filter(
    (r) =>
      !r.tribelt &&
      (filter.archived || !r.archived) &&
      (q === '' || r.name.toLowerCase().includes(q)) &&
      (filter.family === '' || r.familyIds.includes(filter.family)) &&
      (filter.country === '' || r.country === filter.country),
  )
  const byName = sortRows(shown, { key: 'name', direction: 'asc' }, columnValue)
  const sorted = sortRows(byName, sort ?? { key: 'share', direction: 'desc' }, columnValue)
  return [...rows.filter((r) => r.tribelt), ...sorted]
}

/** The countries of the companies, for the country filter. */
export function countriesOf(rows: readonly CompetitorRow[]): string[] {
  return [...new Set(rows.map((r) => r.country).filter((c) => c !== ''))].sort()
}

/** A row with its share of AI responses. */
export type NamedRow = CompetitorRow & { share: number }

/** The competitor named in the most AI responses; a tie goes to the name first alphabetically. */
export function bestCompetitor(rows: readonly CompetitorRow[]): NamedRow | undefined {
  return rows
    .filter((r): r is NamedRow => !r.tribelt && !r.archived && r.share !== undefined)
    .sort((a, b) => b.share - a.share || a.name.localeCompare(b.name))[0]
}

const pct = (share: number) => `${String(Math.round(share * 100))}%`

/** The sentence over the table: how often Tribelt is named, and the best competitor. */
export function summaryLine(tribelt: CompetitorRow | undefined, best: NamedRow | undefined): string {
  if (tribelt?.share === undefined) return 'No Prompt run has counted the AI responses yet.'
  const own = `Tribelt is named in ${pct(tribelt.share)} of AI responses.`
  return best === undefined ? own : `${own} The best competitor, ${best.name}, is named in ${pct(best.share)}.`
}

/** A scan's date as the pages write it: "20 October 2026"; "Not scanned" without one. */
export function scanDate(iso: string | undefined): string {
  return iso === undefined ? 'Not scanned' : longDate(iso)
}
