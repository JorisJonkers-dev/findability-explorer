import type { OverviewAiSearch, OverviewCompetitors, OverviewTerms, Standing } from '../../../infrastructure/api/types.gen'
import { formatValue, MEASURES } from '../../catching-up/domain/standings'
import { familyLabel, languageLabel, PRODUCT_FAMILIES } from '../../../shared/vocabulary'

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

/** For each of Tribelt's families, how many of the researched makers offer it, of all of them; every family at none until the figures load. */
export function familyBars(competitors: OverviewCompetitors | undefined): TileBar[] {
  const makers = competitors?.makers ?? 0
  const families = competitors?.families ?? PRODUCT_FAMILIES.map((family) => ({ family, makers: 0 }))
  return families.map(({ family, makers: count }) => ({
    label: familyLabel(family),
    value: String(count),
    first: makers > 0 ? count / makers : 0,
    second: 0,
    zero: count === 0,
  }))
}

/** The languages the poster charts: the eight with the most terms. */
export const LANGUAGES_SHOWN = 8

/** The terms per language, the most first, against the largest: the part Tribelt uses first, the part only others use after it. */
export function languageBars(terms: OverviewTerms | undefined): TileBar[] {
  const rows = (terms?.languages ?? []).slice(0, LANGUAGES_SHOWN)
  // Every language listed has a term, so the largest is at least one.
  const max = rows[0]?.terms ?? 1
  return rows.map((row) => ({
    label: languageLabel(row.language),
    value: String(row.terms),
    first: row.tribelt / max,
    second: (row.terms - row.tribelt) / max,
    zero: false,
  }))
}

/** The search terms tile's figure: "387 search terms in 3 languages. Tribelt's own website uses 30 of them." */
export function termsFigure(terms: OverviewTerms | undefined): { big: string; text: string } {
  if (!terms || terms.total === 0) return { big: '…', text: 'search terms in the terminology map.' }
  return {
    big: String(terms.total),
    text: `search terms in ${String(terms.languages.length)} languages. Tribelt's own website uses ${String(terms.tribelt)} of them.`,
  }
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
export function voiceBars(ai: OverviewAiSearch | undefined): TileBar[] {
  if (!ai || ai.answers === 0) return []
  const most = Math.max(1, ...ai.companies.map((c) => c.answers))
  return ai.companies.map((c) => ({
    label: c.name,
    value: `${String(Math.round((c.answers / ai.answers) * 100))}%`,
    first: c.answers / most,
    second: 0,
    zero: false,
  }))
}

/** The AI search tile's figure: the share of the newest run's answers naming Tribelt. */
export function aiFigure(ai: OverviewAiSearch | undefined): { big: string; text: string } {
  const bars = voiceBars(ai)
  const us = bars.find((b) => b.label === 'Tribelt')
  if (!us) return { big: '…', text: 'of AI answers name Tribelt, once the first Prompt run is in.' }
  const top = bars.find((b) => b.label !== 'Tribelt')
  return { big: us.value, text: top ? `of AI answers name Tribelt. ${top.label} is named in ${top.value}.` : 'of AI answers name Tribelt.' }
}

/** The point a zoom grows from: the centre of a box, in the poster's own pixels. */
export function originOf(box: { left: number; top: number; width: number; height: number }): string {
  return `${String(box.left + box.width / 2)}px ${String(box.top + box.height / 2)}px`
}
