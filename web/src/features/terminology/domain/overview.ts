import type { OverviewLanguage, OverviewTerms, Term } from '../../../infrastructure/api/types.gen'
import { languageLabel } from '../../../shared/vocabulary'
import type { Tone } from '../../../shared/ui/charts/tones'

/** Words listed as a sentence lists them, with the Oxford comma: "a", "a and b", "a, b, and c". */
export function listOf(words: readonly string[]): string {
  if (words.length <= 2) return words.join(' and ')
  return `${words.slice(0, -1).join(', ')}, and ${words.slice(-1).join('')}`
}

/** The main page's headline, from the map's count and the terms tribelt.nl uses; neutral until the figures load. */
export function termsHeadline(terms: OverviewTerms | undefined): string {
  if (!terms || terms.total === 0) return 'Ways to search for a metal belt.'
  const only = terms.tribelt * 2 < terms.total ? 'only ' : ''
  return `Competitors list belts using ${String(terms.total)} terms, tribelt.nl uses ${only}${String(terms.tribelt)} of them.`
}

/** A figure on the main page: a big number, what it counts and a note under it. */
export type TermsFigure = { big: string; text: string; note: string }

/** The three figures: the terms in the search term list, those tribelt.nl uses (against its target) and those only competitors use. */
export function termsFigures(terms: OverviewTerms | undefined, goal: number | undefined): TermsFigure[] {
  const total = terms?.total ?? 0
  const used = terms?.tribelt ?? 0
  const languages = (terms?.languages ?? []).map((l) => languageLabel(l.language))
  return [
    { big: String(total), text: 'terms in the search term list', note: languages.length > 0 ? `In ${listOf(languages)}.` : '' },
    { big: String(used), text: 'are used on tribelt.nl', note: goal === undefined ? '' : `The target is ${String(goal)}.` },
    {
      big: String(Math.max(0, total - used)),
      text: 'are used only by competitors',
      note: 'Each one is a search in which Tribelt does not turn up.',
    },
  ]
}

/** One language of the map, as a slice of the donut and a legend button. */
export type LanguageSlice = { value: OverviewLanguage['language']; label: string; count: number; tribelt: number; share: number; tone: Tone }

const SLICE_TONES: readonly Tone[] = ['blue-grey', 'silver', 'ink']

/** Every language with a term, the most terms first, each with its share of the map. */
export function languageSlices(terms: OverviewTerms | undefined): LanguageSlice[] {
  const total = terms?.total ?? 0
  return (terms?.languages ?? []).map((l, i) => ({
    value: l.language,
    label: languageLabel(l.language),
    count: l.terms,
    tribelt: l.tribelt,
    share: total > 0 ? l.terms / total : 0,
    tone: SLICE_TONES[i % SLICE_TONES.length] as Tone,
  }))
}

/** How many of the map's terms are American English, which the language counts count as English. */
export function americanEnglishCount(terms: readonly Pick<Term, 'language'>[]): number {
  return terms.filter((t) => t.language === 'en-US').length
}

export const DONUT_RADIUS = 76
const CIRCUMFERENCE = 2 * Math.PI * DONUT_RADIUS

/** One arc of the language donut; the picked language is drawn wider and in orange. */
export type Arc = { value: string; dash: string; offset: number; width: number; picked: boolean }

/** The arcs of the slices, a small gap between neighbours. */
export function donutArcs(slices: readonly LanguageSlice[], picked: string | null): Arc[] {
  let start = 0
  return slices.map((s) => {
    const len = s.share * CIRCUMFERENCE
    const arc = {
      value: s.value,
      dash: `${Math.max(0, len - 2).toFixed(1)} ${CIRCUMFERENCE.toFixed(1)}`,
      offset: -start,
      width: s.value === picked ? 34 : 26,
      picked: s.value === picked,
    }
    start += len
    return arc
  })
}

/** What the picked language says under the donut. */
export function pickedLanguage(slice: LanguageSlice | undefined, americanEnglish: number): { title: string; text: string } | undefined {
  if (!slice) return undefined
  const terms = `${String(slice.count)} ${slice.count === 1 ? 'term' : 'terms'}`
  const american = slice.value === 'en' && americanEnglish > 0 ? `, ${String(americanEnglish)} of them American English` : ''
  const grow = slice.value === 'de' ? ' Germany is where Tribelt wants to grow.' : ''
  return {
    title: `${slice.label}, ${String(Math.round(slice.share * 100))}% of the list`,
    text: `${terms}${american}.${grow} tribelt.nl uses ${String(slice.tribelt)} of them.`,
  }
}
