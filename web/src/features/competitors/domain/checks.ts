import type { CandidateReviewState, Competitor, CompetitorCheck } from '../../../infrastructure/api/types.gen'
import { MODEL_DECIDER } from './candidates'

/** Haiku's check of each company on the list, by the company's id. */
export function checksById(checks: readonly CompetitorCheck[]): Map<string, CompetitorCheck> {
  return new Map(checks.map((c) => [c.competitorId, c]))
}

/** Whether Claude Haiku archived the company, rather than a person. */
export function archivedByHaiku(c: Pick<Competitor, 'archived' | 'archivedBy'>): boolean {
  return c.archived && c.archivedBy === MODEL_DECIDER
}

/** Whether a company waits for a person: on the list, kept by nobody, and Haiku is unsure of it. */
export function isFlagged(c: Pick<Competitor, 'archived' | 'keptBy'>, check: CompetitorCheck | undefined): boolean {
  return !c.archived && c.keptBy === undefined && check !== undefined && !check.sure
}

/** Haiku's verdict in a few words, or null before it checked the company. */
export function verdictLabel(check: CompetitorCheck | undefined): string | null {
  if (!check) return null
  if (!check.sure) return 'Haiku is unsure'
  return check.competitor ? 'Haiku says competitor' : 'Haiku says not a competitor'
}

/** Why Haiku said what it said, as a sentence. */
export function verdictReason(check: CompetitorCheck | undefined): string {
  return check === undefined || check.reason === '' ? 'Haiku gave no reason for it.' : check.reason
}

/** How far Haiku is with the companies on the list. */
export function listCheckLine(state: CandidateReviewState, unchecked: number): string {
  if (state.running) return 'Haiku is checking the companies on the list.'
  if (state.problem === 'noKey') return 'No Anthropic key is set up, so Haiku cannot check the list.'
  if (state.problem === 'budget') return "This month's language-model budget is used up, so Haiku checks the rest of the list next month."
  if (state.problem === 'failed') return 'Haiku failed on its last try. Try again.'
  if (unchecked === 1) return 'Haiku has not checked 1 company on the list yet.'
  if (unchecked > 0) return `Haiku has not checked ${String(unchecked)} companies on the list yet.`
  return 'Haiku has checked every company on the list.'
}

/** The companies Claude Haiku archived, the latest first. */
export function haikuArchived<T extends Pick<Competitor, 'archived' | 'archivedBy' | 'archivedAt' | 'name'>>(companies: readonly T[]): T[] {
  return companies
    .filter(archivedByHaiku)
    .sort((a, b) => (b.archivedAt ?? '').localeCompare(a.archivedAt ?? '') || a.name.localeCompare(b.name))
}
