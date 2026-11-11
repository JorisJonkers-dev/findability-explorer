import type { OverviewTerms } from '../../../infrastructure/api/types.gen'
import { languageLabel } from '../../../shared/vocabulary'

/** Words listed as a sentence lists them, with the Oxford comma: "a", "a and b", "a, b, and c". */
export function listOf(words: readonly string[]): string {
  if (words.length <= 2) return words.join(' and ')
  return `${words.slice(0, -1).join(', ')}, and ${words.slice(-1).join('')}`
}

/** A figure on the main page: a big number, what it counts and a note under it. */
export type TermsFigure = { big: string; text: string; note: string }

/** The three figures: the terms in the search term list, those tribelt.nl uses (against its goal) and those only competitors use. */
export function termsFigures(terms: OverviewTerms | undefined, goal: number | undefined): TermsFigure[] {
  const total = terms?.total ?? 0
  const used = terms?.tribelt ?? 0
  const languages = (terms?.languages ?? []).map((l) => languageLabel(l.language))
  return [
    { big: String(total), text: 'terms in the search term list', note: languages.length > 0 ? `In ${listOf(languages)}.` : '' },
    { big: String(used), text: 'used on tribelt.nl', note: goal === undefined ? '' : `Goal ${String(goal)}` },
    {
      big: String(Math.max(0, total - used)),
      text: 'used only by competitors',
      note: 'Each one is a search in which Tribelt does not turn up.',
    },
  ]
}
