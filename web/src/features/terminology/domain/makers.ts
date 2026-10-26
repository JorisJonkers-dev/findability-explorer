import type { Competitor, Term } from '../../../infrastructure/api/types.gen'
import { creditedCompanies, shortName } from '../../competitors/domain/matrix'
import { companyName } from './filters'

/** A term of the map with the makers the scans found on top of the ones the research named. */
export type MapTerm = Term & {
  /** The makers only the scans found, by the short name the map credits them under. */
  scanMakers: string[]
}

/**
 * Adds to each term's makers the companies whose newest scan found it (foundBy), under the short
 * name the map already credits them by, so every view that counts or lists makers counts them too.
 * A company the research names already is not added twice.
 */
export function withScanMakers(terms: readonly Term[], companies: readonly Pick<Competitor, 'id' | 'name'>[]): MapTerm[] {
  const credited = creditedCompanies(terms)
  const names = new Map(companies.map((c) => [c.id, shortName(c.name, credited)]))
  return terms.map((t) => {
    const own = new Set(t.usedBy.map(companyName))
    const scanMakers: string[] = []
    for (const id of t.foundBy ?? []) {
      const name = names.get(id)
      if (name !== undefined && !own.has(name) && !scanMakers.includes(name)) scanMakers.push(name)
    }
    return { ...t, usedBy: [...t.usedBy, ...scanMakers], scanMakers }
  })
}
