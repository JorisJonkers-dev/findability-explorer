import type { CompetitorCandidate } from '../../../infrastructure/api/types.gen'
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
  return { name: c.name, website: candidateWebsite(c) }
}

/** A draft checked: the name and website to add, or what is wrong with it. */
export function checkCandidateDraft(draft: CandidateDraft): { kind: 'ok'; name: string; website: string } | { kind: 'invalid'; message: string } {
  const name = draft.name.trim()
  const website = draft.website.trim()
  if (name === '') return { kind: 'invalid', message: 'Enter the company’s name.' }
  if (!/^https?:\/\/\S+$/.test(website)) return { kind: 'invalid', message: 'Enter its website, starting with https://.' }
  return { kind: 'ok', name, website }
}
