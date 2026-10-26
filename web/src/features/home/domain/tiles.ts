import type { Competitor, Term } from '../../../infrastructure/api/types.gen'
import { familyLabel, languageLabel, PRODUCT_FAMILIES } from '../../../shared/vocabulary'
import { chartLanguage } from '../../terminology/domain/filters'
import { companyName } from '../../terminology/domain/filters'

/** One bar of a tile's chart: a filled part, optionally a second part after it, and its value. */
export type TileBar = {
  label: string
  value: string
  /** The first part's length, from 0 to 1. */
  first: number
  /** The second part's length, from 0 to 1. */
  second: number
  /** Whether the value is a zero worth pointing out. */
  zero: boolean
}

/** The competitors the desk research studied, which the poster counts: not the group, not added since. */
function researched<T extends Pick<Competitor, 'source' | 'groupCompany'>>(competitors: readonly T[]): T[] {
  return competitors.filter((c) => c.source === 'research' && !c.groupCompany)
}

/** How many makers the research found. */
export function makerCount(competitors: readonly Pick<Competitor, 'source' | 'groupCompany'>[]): number {
  return researched(competitors).length
}

/** For each of Tribelt's families, how many of the researched makers offer it, of all of them. */
export function familyBars(competitors: readonly Pick<Competitor, 'source' | 'groupCompany' | 'beltNames'>[]): TileBar[] {
  const makers = researched(competitors)
  return PRODUCT_FAMILIES.map((family) => {
    const count = makers.filter((c) => c.beltNames.some((b) => b.family === family)).length
    return { label: familyLabel(family), value: String(count), first: makers.length > 0 ? count / makers.length : 0, second: 0, zero: count === 0 }
  })
}

/** Whether Tribelt's own website uses a term: the map credits Tribelt with it. */
export function usedByTribelt(term: Pick<Term, 'usedBy'>): boolean {
  return term.usedBy.some((u) => companyName(u) === 'Tribelt')
}

/** How many of the map's terms Tribelt's own website uses. */
export function tribeltTermCount(terms: readonly Pick<Term, 'usedBy'>[]): number {
  return terms.filter(usedByTribelt).length
}

/** The languages the poster charts: the eight with the most terms. */
export const LANGUAGES_SHOWN = 8

/**
 * The terms per language, the most first, against the largest: the part Tribelt uses first, the
 * part only others use after it. American English counts as English.
 */
export function languageBars(terms: readonly Pick<Term, 'language' | 'usedBy'>[]): TileBar[] {
  const counts = new Map<string, { total: number; tribelt: number }>()
  for (const term of terms) {
    const language = term.language === 'en-US' ? 'en' : term.language
    const count = counts.get(language) ?? { total: 0, tribelt: 0 }
    count.total += 1
    if (usedByTribelt(term)) count.tribelt += 1
    counts.set(language, count)
  }
  const rows = [...counts.entries()].sort(([a, m], [b, n]) => n.total - m.total || a.localeCompare(b)).slice(0, LANGUAGES_SHOWN)
  // Every language listed has a term, so the largest is at least one.
  const max = rows[0]?.[1].total ?? 1
  return rows.map(([language, { total, tribelt }]) => ({
    label: languageLabel(language),
    value: String(total),
    first: tribelt / max,
    second: (total - tribelt) / max,
    zero: false,
  }))
}

/** Pages each crawler fetched, against the most any did. */
export function crawlerBars(crawlers: readonly { name: string; pages: number }[]): TileBar[] {
  const max = Math.max(0, ...crawlers.map((c) => c.pages))
  return crawlers.map((c) => ({ label: c.name, value: String(c.pages), first: max > 0 ? c.pages / max : 0, second: 0, zero: c.pages === 0 }))
}

/** The point a zoom grows from: the centre of a box, in the poster's own pixels. */
export function originOf(box: { left: number; top: number; width: number; height: number }): string {
  return `${String(box.left + box.width / 2)}px ${String(box.top + box.height / 2)}px`
}

/** How many languages the map's terms are in, each code once: American English counts on its own. */
export function languageCount(terms: readonly Pick<Term, 'language'>[]): number {
  return new Set(terms.map((t) => chartLanguage(t.language))).size // American English is English
}

/** The search terms tile's line: "search terms in 3 languages. Tribelt's own website uses 30 of them." */
export function termsText(terms: readonly Pick<Term, 'language' | 'usedBy'>[]): string {
  return `search terms in ${String(languageCount(terms))} languages. Tribelt's own website uses ${String(tribeltTermCount(terms))} of them.`
}
