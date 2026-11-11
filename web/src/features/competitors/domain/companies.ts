import type { Competitor } from '../../../infrastructure/api/types.gen'
import { MODEL_DECIDER } from './candidates'
import { archivedByHaiku } from './checks'

/** The companies on the competitor list, and those archived from it: competitors by role, A to Z. */
export function listCompanies<T extends Pick<Competitor, 'role' | 'name'>>(companies: readonly T[]): T[] {
  return companies.filter((c) => c.role === 'competitor').sort((a, b) => a.name.localeCompare(b.name))
}

/** The line above the list: "61 companies count as competitors and 4 are archived." */
export function listLine(companies: readonly Pick<Competitor, 'archived'>[]): string {
  const archived = companies.filter((c) => c.archived).length
  const active = companies.length - archived
  const count = active === 1 ? '1 company counts as a competitor' : `${String(active)} companies count as competitors`
  return archived === 0 ? `${count}.` : `${count} and ${String(archived)} ${archived === 1 ? 'is' : 'are'} archived.`
}

/** A company's status on the list. */
export function statusOf(company: Pick<Competitor, 'archived' | 'archivedBy'>): string {
  if (!company.archived) return 'Competitor'
  return archivedByHaiku(company) ? 'Archived by Haiku' : 'Archived'
}

/** Who archived a company, and when, or null while it is on the list. */
export function archivedLine(company: Pick<Competitor, 'archived' | 'archivedBy' | 'archivedAt'>, day: (iso: string) => string): string | null {
  if (!company.archived) return null
  const by = company.archivedBy === undefined ? '' : company.archivedBy === MODEL_DECIDER ? ' by Claude Haiku' : ` by ${company.archivedBy}`
  const on = company.archivedAt === undefined ? '' : ` on ${day(company.archivedAt)}`
  return `Archived${by}${on}. It no longer counts as a competitor, and its scans stay.`
}
