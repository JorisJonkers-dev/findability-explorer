import type { Product } from '../../../infrastructure/api/types.gen'
import type { Tone } from '../../../shared/ui/charts/tones'
import { scopeName } from '../../prompt-test/domain/results'

/** How many sellers a family page shows before "Show all". */
export const SELLERS_SHOWN = 8

/** The sellers to show, the first few unless all are asked for. */
export function visibleSellers<T>(sellers: readonly T[], all: boolean): T[] {
  return all ? [...sellers] : sellers.slice(0, SELLERS_SHOWN)
}

/** A bar of the AI responses about a family, as the bar list draws it. */
export type ResponseBar = { label: string; value: number; display: string; tone: Tone; highlight: boolean }

/** For each scope a Prompt run asked about a family, the share of its responses that name Tribelt, as bars. */
export function responseBars(p: Pick<Product, 'byScope'>): ResponseBar[] {
  return p.byScope
    .filter((s) => s.answers > 0)
    .map((s) => ({
      label: `${scopeName(s.scope)}, ${String(s.answers)} ${s.answers === 1 ? 'response' : 'responses'}`,
      value: s.named / s.answers,
      display: `${String(Math.round((s.named / s.answers) * 100))}%`,
      tone: 'orange',
      highlight: true,
    }))
}
