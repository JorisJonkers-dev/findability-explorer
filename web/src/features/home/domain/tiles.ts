import type { Competitor, PromptTestResults, Standing, Term } from '../../../infrastructure/api/types.gen'
import { formatValue, MEASURES } from '../../catching-up/domain/standings'
import { latestRun, tally } from '../../prompt-test/domain/results'
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

/** The competitors the desk research studied, which the poster counts: not the group or the mirror, not added since. */
function researched<T extends Pick<Competitor, 'source' | 'groupCompany' | 'role'>>(competitors: readonly T[]): T[] {
  return competitors.filter((c) => c.source === 'research' && !c.groupCompany && c.role === 'competitor')
}

/** How many makers the research found. */
export function makerCount(competitors: readonly Pick<Competitor, 'source' | 'groupCompany' | 'role'>[]): number {
  return researched(competitors).length
}

/** For each of Tribelt's families, how many of the researched makers offer it, of all of them. */
export function familyBars(competitors: readonly Pick<Competitor, 'source' | 'groupCompany' | 'role' | 'beltNames'>[]): TileBar[] {
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

/** The map's term count in one language, as the server counts it over the whole map. */
export type LanguageCount = { value: string; count: number }

/**
 * The terms per language, the most first, against the largest: the part Tribelt uses first, the
 * part only others use after it. American English counts as English. With the server's counts,
 * the totals cover the whole map, even terms not loaded.
 */
export function languageBars(terms: readonly Pick<Term, 'language' | 'usedBy'>[], languages?: readonly LanguageCount[]): TileBar[] {
  const counts = new Map<string, { total: number; tribelt: number }>()
  const add = (raw: string, total: number, tribelt: number) => {
    const language = raw === 'en-US' ? 'en' : raw
    const count = counts.get(language) ?? { total: 0, tribelt: 0 }
    count.total += total
    count.tribelt += tribelt
    counts.set(language, count)
  }
  for (const term of terms) add(term.language, languages ? 0 : 1, usedByTribelt(term) ? 1 : 0)
  for (const l of languages ?? []) add(l.value, l.count, 0)
  for (const [language, count] of counts) if (count.total === 0) counts.delete(language)
  // Every language with a term shows; none is cut off.
  const rows = [...counts.entries()].sort(([a, m], [b, n]) => n.total - m.total || a.localeCompare(b))
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

/** Per measure, Tribelt's value against the leader's: Tribelt's part, then the gap to the leader. */
export function gapBars(measures: readonly Standing[]): TileBar[] {
  return measures.map((m) => {
    const tribelt = m.tribelt ?? 0
    const best = Math.max(tribelt, m.leader?.value ?? 0)
    return {
      label: MEASURES[m.measure].short,
      value: m.tribelt === undefined ? '–' : formatValue(m.measure, m.tribelt),
      first: best === 0 ? 0 : tribelt / best,
      second: best === 0 ? 0 : (best - tribelt) / best,
      zero: false,
    }
  })
}

/** The Catching up tile's figure: how many measures Tribelt trails on. */
export function catchUpFigure(measures: readonly Standing[], trailing: number): { big: string; text: string } {
  if (measures.length === 0) return { big: '…', text: 'measures where Tribelt trails the leaders.' }
  return { big: String(trailing), text: trailing === 1 ? 'measure where Tribelt trails the leader.' : 'measures where Tribelt trails the leaders.' }
}

/** The shares of the newest run's answers naming Tribelt and the four competitors named most, against the largest. */
export function voiceBars(results: Pick<PromptTestResults, 'runs' | 'mentions'> | undefined): TileBar[] {
  const run = latestRun(results)
  const answers = run ? tally(run.cells).answers : 0
  if (!results || answers === 0) return []
  const tribelt = results.mentions.find((m) => m.role === 'tribelt')
  const rows = [
    ...results.mentions.filter((m) => m.role === 'competitor').slice(0, 4),
    ...(tribelt ? [tribelt] : [{ name: 'Tribelt', answers: 0 }]),
  ].sort((a, b) => b.answers - a.answers)
  const most = Math.max(1, ...rows.map((r) => r.answers))
  return rows.map((r) => ({
    label: r === tribelt || r.name === 'Tribelt' ? 'Tribelt' : r.name,
    value: `${String(Math.round((r.answers / answers) * 100))}%`,
    first: r.answers / most,
    second: 0,
    zero: false,
  }))
}

/** The AI search tile's figure: the share of the newest run's answers naming Tribelt. */
export function aiFigure(results: Pick<PromptTestResults, 'runs' | 'mentions'> | undefined): { big: string; text: string } {
  const bars = voiceBars(results)
  const us = bars.find((b) => b.label === 'Tribelt')
  if (!us) return { big: '…', text: 'of AI answers name Tribelt, once the first Prompt run is in.' }
  const top = bars.find((b) => b.label !== 'Tribelt')
  return { big: us.value, text: top ? `of AI answers name Tribelt. ${top.label} is named in ${top.value}.` : 'of AI answers name Tribelt.' }
}

/** The point a zoom grows from: the centre of a box, in the poster's own pixels. */
export function originOf(box: { left: number; top: number; width: number; height: number }): string {
  return `${String(box.left + box.width / 2)}px ${String(box.top + box.height / 2)}px`
}

/**
 * How many languages the map's terms are in, each code once (American English is English): from
 * the server's counts over the whole map when given, else from the terms loaded.
 */
export function languageCount(terms: readonly Pick<Term, 'language'>[], languages?: readonly LanguageCount[]): number {
  const codes = languages ? languages.filter((l) => l.count > 0).map((l) => l.value) : terms.map((t) => t.language)
  return new Set(codes.map(chartLanguage)).size
}

/** The search terms tile's line: "search terms in 3 languages. Tribelt's own website uses 30 of them." */
export function termsText(terms: readonly Pick<Term, 'language' | 'usedBy'>[], languages?: readonly LanguageCount[]): string {
  return `search terms in ${String(languageCount(terms, languages))} languages. Tribelt's own website uses ${String(tribeltTermCount(terms))} of them.`
}
