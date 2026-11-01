import type { BeltFamily, IntentTier, Language, Term, TermOrigin } from '../../../infrastructure/api/types.gen'
import { BUYER_INTENT } from '../../../shared/tiers'
import { familyLabel, languageLabel } from '../../../shared/vocabulary'

/** The languages a term is added in: those Tribelt's buyers search in. */
export const LANGUAGE_CHOICES: readonly { value: Language; label: string }[] = (['nl', 'de', 'en'] as const).map((value) => ({ value, label: languageLabel(value) }))

/** Every belt family a term can be about, Tribelt's own first. */
export const FAMILY_CHOICES: readonly { value: BeltFamily; label: string }[] = (
  ['eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other', 'all', 'spec_all'] as const
).map((value) => ({ value, label: familyLabel(value) }))

/** Every kind of buyer intent, from the most specific search to the broadest, with what it means. */
export const TIER_CHOICES: readonly { value: IntentTier; label: string; help: string }[] = (['spec', 'product', 'application', 'problem', 'broad'] as const).map((value) => ({
  value,
  ...BUYER_INTENT[value],
}))

/** Where a term comes from, in words. */
export const ORIGIN_WORDS: Record<TermOrigin, string> = { research: 'The desk research', review: 'Accepted from a scan', hand: 'Added by hand' }

/** What is wrong with a term before it is sent; empty when nothing is. */
export function draftError(text: string): string {
  const t = text.trim()
  if (t === '') return 'Write the term.'
  return [...t].length > 200 ? 'A term has at most 200 characters.' : ''
}

/** "oven belt (English)". */
export function termName(t: Pick<Term, 'term' | 'language'>): string {
  return `${t.term} (${languageLabel(t.language)})`
}

/** "Added by Joris Jonkers on 1 November 2026." */
export function handLine(what: 'Added' | 'Removed', h: { by: string; at: string }): string {
  const day = new Date(h.at).toLocaleDateString('en-GB', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })
  return `${what} by ${h.by} on ${day}.`
}
