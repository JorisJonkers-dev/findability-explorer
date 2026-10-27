import type { CandidateDecisionResult, CandidateReviewState, CompetitorCandidate } from '../../../infrastructure/api/types.gen'
import { fold } from '../../terminology/domain/fold'
import { joinWords } from '../../../shared/words'
import { providerName } from '../../prompt-test/domain/results'
import { candidateWebsite } from '../../prompt-test/domain/settings'
import { atLabel } from '../../settings/domain/time'

/** How often and by which providers' models a company was named: "Named in 37 answers by OpenAI and xAI, last on …". */
export function candidateEvidence(c: CompetitorCandidate): string {
  const answers = c.answers === 1 ? '1 answer' : `${String(c.answers)} answers`
  return `Named in ${answers} by ${joinWords(c.providers.map(providerName))}, last on ${atLabel(c.lastSeenAt)}`
}

const DECIDED = { added: 'Added', dismissed: 'Dismissed' } as const

/** Who decided and when: "Added by joris, 2 Nov 2026, 09:00 UTC", or null while it waits. */
export function candidateDecisionLine(c: CompetitorCandidate): string | null {
  const d = c.decision
  if (!d || d.status === 'pending') return null
  return `${DECIDED[d.status]} by ${d.by}, ${atLabel(d.at)}`
}

/** The fields an administrator checks before a named company joins the competitors. */
export type CandidateDraft = { name: string; website: string }

/** The draft a candidate starts from: its name and the first website the answers gave. */
export function candidateDraft(c: CompetitorCandidate): CandidateDraft {
  return { name: c.name, website: c.website ?? candidateWebsite(c) }
}

/** A draft checked: the name and website to add, or what is wrong with it. */
export function checkCandidateDraft(draft: CandidateDraft): { kind: 'ok'; name: string; website: string } | { kind: 'invalid'; message: string } {
  const name = draft.name.trim()
  const website = draft.website.trim()
  if (name === '') return { kind: 'invalid', message: 'Enter the company’s name.' }
  if (!/^https?:\/\/\S+$/.test(website)) return { kind: 'invalid', message: 'Enter its website, starting with https://.' }
  return { kind: 'ok', name, website }
}

/** The other names a company goes by, the most given first: all but the one it is listed under. */
export function otherNames(c: CompetitorCandidate): CompetitorCandidate['names'] & object {
  return (c.names ?? []).filter((n) => n.joined !== 'lead')
}

/** Whether a company matches what was searched for: in any of its names or websites. */
export function matchesSearch(c: CompetitorCandidate, query: string): boolean {
  const q = fold(query.trim())
  if (q === '') return true
  return [c.name, ...(c.names ?? []).map((n) => n.name), ...(c.sites ?? [])].some((text) => fold(text).includes(q))
}

/** What Haiku made of the companies, to show only some. */
export type CheckFilter = 'all' | 'competitor' | 'not' | 'unsure' | 'unchecked'

export const CHECK_FILTERS: readonly { value: CheckFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'competitor', label: 'Haiku says competitor' },
  { value: 'not', label: 'Haiku says not' },
  { value: 'unsure', label: 'Haiku is unsure' },
  { value: 'unchecked', label: 'Not checked yet' },
]

/** Whether a company is one the filter shows. */
export function keptBy(c: CompetitorCandidate, filter: CheckFilter): boolean {
  const v = c.check
  switch (filter) {
    case 'all':
      return true
    case 'unchecked':
      return !v
    case 'unsure':
      return !!v && !v.sure
    case 'competitor':
      return !!v && v.sure && v.competitor
    case 'not':
      return !!v && v.sure && !v.competitor
  }
}

/** Haiku's check in words, or null before it checked the company. */
export function checkLine(c: CompetitorCandidate): string | null {
  const v = c.check
  if (!v) return null
  if (v.reason === '') return 'Haiku gave no answer for it.'
  if (!v.sure) return `Haiku is unsure. ${v.reason}`
  return `Haiku says ${v.competitor ? 'a competitor' : 'not a competitor'}. ${v.reason}`
}

/** How far Haiku is with the companies. */
export function reviewLine(state: CandidateReviewState, unchecked: number): string {
  const companies = (n: number) => (n === 1 ? '1 company' : `${String(n)} companies`)
  if (state.running) return 'Haiku is merging the duplicates and checking the companies.'
  if (state.problem === 'noKey') return 'No Anthropic key is set up, so Haiku cannot check the companies.'
  if (state.problem === 'failed') return 'Haiku failed on its last try. Try again.'
  if (unchecked > 0) return `Haiku has not checked ${companies(unchecked)} yet.`
  return 'Haiku has checked every company.'
}

/** What adding or dismissing several did, in a sentence or two; the reasons show per company. */
export function decidedLine(action: 'add' | 'dismiss', results: readonly CandidateDecisionResult[]): string {
  const done = results.filter((r) => r.outcome === 'done').length
  const skipped = results.length - done
  const verb = action === 'add' ? 'Added' : 'Dismissed'
  const first = `${verb} ${done === 1 ? '1 company' : `${String(done)} companies`}.`
  if (skipped === 0) return first
  return `${first} Skipped ${String(skipped)}.`
}

/** The ids of the companies a set of rows shows. */
export function idsOf(cs: readonly CompetitorCandidate[]): string[] {
  return cs.map((c) => c.id)
}
