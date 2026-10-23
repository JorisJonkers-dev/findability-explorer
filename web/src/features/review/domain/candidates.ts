import type {
  BeltFamily,
  IntentTier,
  Language,
  TermAcceptance,
  TermCandidate,
  TermCandidateCounts,
  TermCandidateStatus,
} from '../../../infrastructure/api/types.gen'
import { zProblem } from '../../../infrastructure/api/zod.gen'
import { describeError } from '../../../shared/describeError'
import { familyLabel, isLanguage, languageLabel, tierLabel } from '../../../shared/vocabulary'
import { atLabel } from '../../settings/domain/time'

const STATUS_LABELS: Record<TermCandidateStatus, string> = { pending: 'To review', accepted: 'Accepted', rejected: 'Rejected' }

/** The review's tabs, in order. */
export const STATUSES = Object.keys(STATUS_LABELS) as TermCandidateStatus[]

/** A tab's label with its count: "To review (41)". */
export function statusTab(status: TermCandidateStatus, counts: TermCandidateCounts | undefined): string {
  return counts ? `${STATUS_LABELS[status]} (${String(counts[status])})` : STATUS_LABELS[status]
}

/** A status read from the address, or pending. */
export function parseStatus(raw: unknown): TermCandidateStatus {
  return typeof raw === 'string' && Object.hasOwn(STATUS_LABELS, raw) ? (raw as TermCandidateStatus) : 'pending'
}

/** The fields an administrator edits before a candidate joins the map. */
export type AcceptDraft = { term: string; language: Language; family: BeltFamily; tier: IntentTier; usedBy: string }

/**
 * The draft a candidate starts from: its phrase, its first language the map knows (else English),
 * the classifier's family (else other belts) and tier, and the companies that use it.
 */
export function draftOf(candidate: TermCandidate): AcceptDraft {
  const language = candidate.languages.find(isLanguage) ?? 'en'
  return {
    term: candidate.phrase,
    language,
    family: candidate.suggestedFamily ?? 'other',
    tier: candidate.suggestedTier,
    usedBy: candidate.companies.map((c) => c.name).join(', '),
  }
}

export const MAX_TERM_LENGTH = 200
export const MAX_USED_BY = 20

/** A draft checked: the body to send, or what is wrong with it. */
export function checkDraft(draft: AcceptDraft): { kind: 'ok'; body: TermAcceptance } | { kind: 'invalid'; message: string } {
  const term = draft.term.trim()
  if (term === '') return { kind: 'invalid', message: 'Enter the term as the map should show it.' }
  if (term.length > MAX_TERM_LENGTH) return { kind: 'invalid', message: `A term has at most ${String(MAX_TERM_LENGTH)} characters.` }
  const usedBy = [...new Set(draft.usedBy.split(',').map((u) => u.trim()).filter((u) => u !== ''))]
  if (usedBy.length > MAX_USED_BY) return { kind: 'invalid', message: `Name at most ${String(MAX_USED_BY)} companies.` }
  return { kind: 'ok', body: { term, language: draft.language, family: draft.family, tier: draft.tier, usedBy } }
}

/** The evidence for a candidate, in one line: "3 companies, 4 scans, in English and German". */
export function evidenceLine(candidate: TermCandidate): string {
  const companies = candidate.companies.length === 1 ? '1 company' : `${String(candidate.companies.length)} companies`
  const scans = candidate.scans === 1 ? '1 scan' : `${String(candidate.scans)} scans`
  const languages = candidate.languages.map(languageLabel)
  const where = languages.length === 0 ? '' : `, in ${languages.length === 1 ? String(languages[0]) : `${languages.slice(0, -1).join(', ')} and ${String(languages.at(-1))}`}`
  return `${companies}, ${scans}${where}`
}

/** The classifier's suggestion: "Product, Tri-Flex; closest map term: spiral belt". */
export function suggestionLine(candidate: TermCandidate): string {
  const family = candidate.suggestedFamily ? `, ${familyLabel(candidate.suggestedFamily)}` : ''
  const closest = candidate.closestTerm ? `; closest map term: ${candidate.closestTerm}` : ''
  return `${tierLabel(candidate.suggestedTier)}${family}${closest}`
}

/** Who decided and when: "Accepted by joris, 23 Oct 2026, 09:30 UTC", or null while pending. */
export function decisionLine(candidate: TermCandidate): string | null {
  const d = candidate.decision
  if (!d) return null
  return `${STATUS_LABELS[d.status]} by ${d.by}, ${atLabel(d.at)}`
}

/** A refused review action in words. */
export function reviewProblem(error: unknown): string {
  const problem = zProblem.safeParse(error)
  if (problem.success && problem.data.status === 403) return 'Only administrators can review terms.'
  if (problem.success && problem.data.status === 404) return 'This candidate is gone. Reload the page.'
  return describeError(error)
}

/** Whether to ask again after a failure: never when the server refused (a 4xx), at most three times otherwise. */
export function shouldRetry(failures: number, error: unknown): boolean {
  const problem = zProblem.safeParse(error)
  const refused = problem.success && problem.data.status >= 400 && problem.data.status < 500
  return !refused && failures < 3
}
