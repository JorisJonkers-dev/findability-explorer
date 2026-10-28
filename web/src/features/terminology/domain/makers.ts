import type { Competitor, Term } from '../../../infrastructure/api/types.gen'
import { creditedCompanies, shortName } from '../../competitors/domain/matrix'

/** A term of the map whose companies are its makers, as the server counts them. */
export type MapTerm = Term & {
  /** The makers only the scans found, by the short name the map credits them under. */
  scanMakers: string[]
}

/**
 * Writes each term's makers (the competitors on the list now that use it, as the server counts
 * them: those its credits name and those whose newest scan found it) under the short name the map
 * credits them by, in place of the credits themselves. So every view that counts or lists a term's
 * companies counts the same ones, and a credit to a company no longer on the list counts nowhere.
 */
export function withMakers(terms: readonly Term[], companies: readonly Pick<Competitor, 'id' | 'name'>[]): MapTerm[] {
  const credited = creditedCompanies(terms)
  const names = new Map(companies.map((c) => [c.id, shortName(c.name, credited)]))
  return terms.map((t) => {
    const usedBy: string[] = []
    const scanMakers: string[] = []
    for (const m of t.makers ?? []) {
      const name = names.get(m.competitorId)
      if (name === undefined || usedBy.includes(name)) continue
      usedBy.push(name)
      if (m.fromScan) scanMakers.push(name)
    }
    return { ...t, usedBy, scanMakers }
  })
}
