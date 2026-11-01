import type { BeltFamily, IntentTier, Product } from '../../../infrastructure/api/types.gen'
import { BUYER_INTENT } from '../../../shared/tiers'
import { languageLabel, PRODUCT_FAMILIES, TIERS } from '../../../shared/vocabulary'
import { chartLanguage } from './filters'
import type { MapTerm } from './makers'

/** What the Search terms pages need of a term. */
type TermFacts = Pick<MapTerm, 'term' | 'language' | 'family' | 'tier' | 'usedBy' | 'tribeltUses'>

const slugOf = (words: string): string =>
  words
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, '-')
    .replace(/^-|-$/g, '') || 'term'

/** A term's address: its language, then its words, "de--drahtösenband". The map gives terms no id. */
export function termId(t: Pick<MapTerm, 'term' | 'language'>): string {
  return `${t.language.toLowerCase()}--${slugOf(t.term)}`
}

/** A term's own page. */
export function termPath(t: Pick<MapTerm, 'term' | 'language'>): string {
  return `/terms/${encodeURIComponent(termId(t))}`
}

/** The term an address names, or undefined. */
export function findTerm<T extends Pick<MapTerm, 'term' | 'language'>>(terms: readonly T[], id: string): T | undefined {
  return terms.find((t) => termId(t) === id)
}

/** The pages' name for each belt family. */
const FAMILY_NAMES: Record<BeltFamily, string> = {
  eyelink: 'Eyelink belts',
  open_spiral: 'Open spiral belts',
  closed_spiral: 'Closed spiral belts',
  tri_flex_turn_curve: 'Tri-Flex curve and spiral belts',
  other: 'Other metal belts',
  all: 'All belts',
  spec_all: 'Specifications for all belts',
}

/** How a sentence names the belts a term is for. */
const FAMILY_PHRASES: Record<BeltFamily, string> = {
  eyelink: 'eyelink belts',
  open_spiral: 'open spiral belts',
  closed_spiral: 'closed spiral belts',
  tri_flex_turn_curve: 'Tri-Flex curve and spiral belts',
  other: 'other metal belts',
  all: 'all belts',
  spec_all: 'the specifications of all belts',
}

/** A belt family as the pages head it: "Eyelink belts". */
export function familyName(family: BeltFamily): string {
  return FAMILY_NAMES[family]
}

const used = (t: TermFacts): boolean => t.tribeltUses === true
const onlyOthers = (t: TermFacts): boolean => !used(t) && t.usedBy.length > 0

const THREE = ['nl', 'de', 'en'] as const

/** A row of By language: the terms in the list, those tribelt.nl uses, and those only competitors use. */
export type LanguageRow = { language: (typeof THREE)[number]; label: string; total: number; used: number; only: number }

/** The terms in each of the three languages, American English counted as English, and in all three. */
export function languageTable(terms: readonly TermFacts[]): { rows: LanguageRow[]; all: Omit<LanguageRow, 'language' | 'label'> } {
  const count = (ts: readonly TermFacts[]) => ({ total: ts.length, used: ts.filter(used).length, only: ts.filter(onlyOthers).length })
  const rows = THREE.map((language) => ({ language, label: languageLabel(language), ...count(terms.filter((t) => chartLanguage(t.language) === language)) }))
  return { rows, all: count(terms.filter((t) => (THREE as readonly string[]).includes(chartLanguage(t.language)))) }
}

/** A row of By belt family. */
export type FamilyRow = { family: (typeof PRODUCT_FAMILIES)[number]; label: string; names: string; total: number; used: number }

/** The terms of each of Tribelt's belt families, with its own names; the terms for every belt are counted apart. */
export function familyTable(
  terms: readonly TermFacts[],
  products: readonly Pick<Product, 'family' | 'tribeltNames'>[],
): { rows: FamilyRow[]; all: { total: number; used: number }; shared: string } {
  const isFamily = (t: TermFacts) => (PRODUCT_FAMILIES as readonly string[]).includes(t.family)
  const rows = PRODUCT_FAMILIES.map((family) => {
    const ts = terms.filter((t) => t.family === family)
    const names = products.find((p) => p.family === family)?.tribeltNames ?? []
    return { family, label: familyName(family), names: names.length > 0 ? names.join(', ') : 'None recorded', total: ts.length, used: ts.filter(used).length }
  })
  const inFamilies = terms.filter(isFamily)
  const rest = terms.filter((t) => !isFamily(t))
  const n = rest.length
  return {
    rows,
    all: { total: inFamilies.length, used: inFamilies.filter(used).length },
    shared: `Not counted above: ${String(n)} ${n === 1 ? 'term that applies' : 'terms that apply'} to every belt, ${String(rest.filter(used).length)} of them used`,
  }
}

const byUse = (a: TermFacts, b: TermFacts) => b.usedBy.length - a.usedBy.length || a.term.localeCompare(b.term)

/** The terms competitors use and tribelt.nl does not, the ones the most competitors use first. */
export function topMissing<T extends TermFacts>(terms: readonly T[], count = 15): T[] {
  return terms.filter(onlyOthers).sort(byUse).slice(0, count)
}

/** Other terms of the same belt family and language, the most used first. */
export function relatedTerms<T extends TermFacts>(terms: readonly T[], term: TermFacts, count = 8): T[] {
  return terms
    .filter((t) => t !== term && t.family === term.family && t.language === term.language)
    .sort(byUse)
    .slice(0, count)
}

/** One search term in a sentence: its language, belt family, who uses it, and its buyer intent. */
export function termSentence(t: TermFacts, competitors: number): string {
  const language = languageLabel(t.language)
  const article = /^[AEIOU]/.test(language) ? 'An' : 'A'
  const n = t.usedBy.length
  const tribelt = used(t) ? 'tribelt.nl uses it too' : 'tribelt.nl does not'
  const intent = BUYER_INTENT[t.tier]
  return `${article} ${language} term for ${FAMILY_PHRASES[t.family]}. ${String(n)} of the ${String(competitors)} competitors ${n === 1 ? 'uses' : 'use'} it, and ${tribelt}. Its buyer intent is ${intent.label.toLowerCase()}. ${intent.help}`
}

/** Every kind of buyer intent with what it means, for "What buyer intent means". */
export const BUYER_INTENT_LIST: readonly { tier: IntentTier; label: string; help: string }[] = TIERS.map((tier) => ({ tier, ...BUYER_INTENT[tier] }))
