import type { Product } from '../../../infrastructure/api/types.gen'
import { languageLabel } from '../../../shared/vocabulary'
import { inWords } from '../../catching-up/domain/standings'
import { listOf } from '../../terminology/domain/overview'

/** The share of a family's answers naming Tribelt, or undefined before any run asked about it. */
export function aiShare(p: Pick<Product, 'answers' | 'named'>): number | undefined {
  return p.answers === 0 ? undefined : p.named / p.answers
}

/** A share as the cards write it: "30%", or "–" without answers. */
export function shareText(share: number | undefined): string {
  return share === undefined ? '–' : `${String(Math.round(share * 100))}%`
}

const FAMILY_TITLES: Record<Product['family'], { title: string; noun: string }> = {
  eyelink: { title: 'Eyelink', noun: 'eyelink belts' },
  open_spiral: { title: 'Open spiral (balanced weave)', noun: 'open spiral belts' },
  closed_spiral: { title: 'Closed spiral (compound balanced)', noun: 'closed spiral belts' },
  tri_flex_turn_curve: { title: 'Tri-Flex, the curve-going belt', noun: 'Tri-Flex belts' },
  other: { title: 'Other belts', noun: 'other belts' },
}

/** A family's name in the list of families, with what the belt is. */
export function familyTitle(family: Product['family']): string {
  return FAMILY_TITLES[family].title
}

/** What Tribelt calls a family, for the list of families; Tribelt may name none yet. */
export function tribeltCalls(p: Pick<Product, 'tribeltNames'>): string {
  return p.tribeltNames.length === 0 ? 'Tribelt names none for it yet' : `Tribelt calls it ${listOf(p.tribeltNames)}`
}

/** A family page's headline, from how many competitors sell it. */
export function familyHeadline(p: Pick<Product, 'family' | 'sellers'>): string {
  const n = p.sellers.length
  const noun = FAMILY_TITLES[p.family].noun
  if (n === 0) return `No competitor sells ${noun}.`
  const who = inWords(n)
  return `${who.charAt(0).toUpperCase()}${who.slice(1)} ${n === 1 ? 'competitor sells' : 'competitors sell'} ${noun}.`
}

/** The line under a family page's headline: Tribelt's own names for the family. */
export function familyLede(p: Pick<Product, 'tribeltNames'>): string {
  return p.tribeltNames.length === 0 ? 'Tribelt names none for it yet.' : `${tribeltCalls(p)}.`
}

/** How many of a family's terms only competitors use. */
export function termsOnlyOthers(p: Pick<Product, 'terms' | 'termsUsed'>): number {
  return Math.max(0, p.terms.length - p.termsUsed)
}

/** How many sellers a family page shows before "Show all". */
export const SELLERS_SHOWN = 8

/** The sellers to show, the first few unless all are asked for. */
export function visibleSellers<T>(sellers: readonly T[], all: boolean): T[] {
  return all ? [...sellers] : sellers.slice(0, SELLERS_SHOWN)
}

/** The languages a family's terms are in, as a sentence lists them: Dutch, German, and English first. */
export function termLanguages(p: Pick<Product, 'terms'>): string {
  const order = ['nl', 'de', 'en']
  const codes = [...new Set(p.terms.map((t) => (t.language === 'en-US' ? 'en' : t.language)))]
  codes.sort((a, b) => (order.includes(a) ? order.indexOf(a) : 9) - (order.includes(b) ? order.indexOf(b) : 9) || a.localeCompare(b))
  return codes.length === 0 ? '' : `Terms in ${listOf(codes.map(languageLabel))}.`
}
