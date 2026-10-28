import type { PlanImprovement, PlanTarget, StandingMeasure, WatchedNumber } from '../../../infrastructure/api/types.gen'
import { formatValue, inWords } from '../../catching-up/domain/standings'

/** How the list can be ranked: best value is the server's order. */
export type Sort = 'value' | 'benefit' | 'effort'

export const SORTS: readonly { id: Sort; label: string }[] = [
  { id: 'value', label: 'Best value' },
  { id: 'benefit', label: 'Biggest benefit' },
  { id: 'effort', label: 'Least work' },
]

const BENEFIT_WORDS = ['', 'Very small benefit', 'Small benefit', 'Medium benefit', 'Large benefit', 'Very large benefit']
const EFFORT_WORDS = ['', 'A few hours', 'About a day', 'A few days', 'A few weeks', 'Several weeks']

/** The benefit score in words: "Large benefit". */
export function benefitText(benefit: number): string {
  return BENEFIT_WORDS[benefit] ?? ''
}

/** The effort score in words: "A few days". */
export function effortText(effort: number): string {
  return EFFORT_WORDS[effort] ?? ''
}

/** Five squares, the first `n` filled. */
export function scoreDots(n: number): boolean[] {
  return [1, 2, 3, 4, 5].map((i) => i <= n)
}

/** A recommendation as the page ranks it: its place among those still to do. */
export type Ranked = { item: PlanImprovement; rank: number }

/**
 * The recommendations not done yet, in the order of the sort, each with its place in that order.
 * The server sends them best value first, and a tie keeps that order.
 */
export function rankOpen(items: readonly PlanImprovement[], sort: Sort): Ranked[] {
  const open = items.filter((i) => !i.done)
  const by: Record<Sort, (a: PlanImprovement, b: PlanImprovement) => number> = {
    value: () => 0,
    benefit: (a, b) => b.benefit - a.benefit || a.effort - b.effort,
    effort: (a, b) => a.effort - b.effort || b.benefit - a.benefit,
  }
  return open.sort(by[sort]).map((item, i) => ({ item, rank: i + 1 }))
}

/** The sentence a chart dot carries for a screen reader and a tooltip. */
export function dotTip(r: Ranked): string {
  return `${String(r.rank)}. ${r.item.title}. ${benefitText(r.item.benefit)}, ${effortText(r.item.effort).toLowerCase()}.`
}

/** The target that watches a measure, where one does. */
const TARGET_OF: Partial<Record<StandingMeasure, PlanTarget>> = {
  aiShareOfVoice: 'aiShareOfVoice',
  readinessChecks: 'readinessChecks',
  termsUsed: 'termsUsed',
}

/** What each measure is called in a sentence that begins with the measure. */
export const MEASURE_NAME: Record<StandingMeasure, string> = {
  aiShareOfVoice: 'AI answers that name Tribelt',
  citationShare: 'Web answers that cite tribelt.nl',
  readinessChecks: 'AI readiness checks passed',
  sitemapCoverage: 'Share of pages the sitemaps list',
  termsUsed: 'Search terms tribelt.nl uses',
}

/** One bar of the comparison: who, the value written out, and its length from 0 to 1. */
export type Row = { who: string; text: string; length: number; tone: 'tribelt' | 'leader' | 'target' }

/**
 * Tribelt against the leader, and the target when one watches the measure, on one scale that runs
 * to the largest of them. A value the server has not got says so, and has no bar.
 */
export function comparison(it: Pick<PlanImprovement, 'measure' | 'tribelt' | 'leader'>, watched: readonly WatchedNumber[], readinessChecks: number): Row[] {
  const target = TARGET_OF[it.measure]
  const goal = target ? watched.find((w) => w.target === target)?.goal : undefined
  const max = Math.max(it.tribelt ?? 0, it.leader?.value ?? 0, goal ?? 0)
  const row = (who: string, v: number | undefined, tone: Row['tone']): Row => ({
    who,
    text: v === undefined ? 'No data yet' : it.measure === 'readinessChecks' ? `${formatValue(it.measure, v)} of ${inWords(readinessChecks)}` : formatValue(it.measure, v),
    length: v === undefined || max === 0 ? 0 : v / max,
    tone,
  })
  const rows = [row('Tribelt', it.tribelt, 'tribelt')]
  if (it.leader) rows.push(row(it.leader.name, it.leader.value, 'leader'))
  if (goal !== undefined) rows.push(row('Target', goal, 'target'))
  return rows
}

/** A recommendation's place in words: "1st". */
export function ordinal(n: number): string {
  const rest = n % 100
  if (rest >= 11 && rest <= 13) return `${String(n)}th`
  const last = n % 10
  return `${String(n)}${last === 1 ? 'st' : last === 2 ? 'nd' : last === 3 ? 'rd' : 'th'}`
}

/** The recommendations that improve the same measure, other than this one. */
export function alsoImproving(items: readonly PlanImprovement[], it: PlanImprovement): PlanImprovement[] {
  return items.filter((o) => o.measure === it.measure && o.id !== it.id)
}

/** The comparison in one line: "AI answers that name Tribelt: 5%, Habasit 22%, target 40%". */
export function compareLine(it: Pick<PlanImprovement, 'measure' | 'tribelt' | 'leader'>, watched: readonly WatchedNumber[], readinessChecks: number): string {
  const parts = comparison(it, watched, readinessChecks).map((r) => (r.tone === 'tribelt' ? r.text : r.tone === 'target' ? `target ${r.text}` : `${r.who} ${r.text}`))
  return `${MEASURE_NAME[it.measure]}: ${parts.join(', ')}`
}

/** A recommendation's place among those still to do by best value, and how many there are; undefined once done. */
export function placeOf(items: readonly PlanImprovement[], id: string): { rank: number; of: number } | undefined {
  const open = rankOpen(items, 'value')
  const found = open.find((r) => r.item.id === id)
  return found ? { rank: found.rank, of: items.length } : undefined
}

/** A recommendation's effort compared with the others, in a sentence. */
export function effortNote(items: readonly PlanImprovement[], it: PlanImprovement): string {
  const others = items.filter((o) => o.id !== it.id)
  if (others.every((o) => o.effort > it.effort)) return `The least work of all ${inWords(items.length)} recommendations.`
  if (others.every((o) => o.effort < it.effort)) return `The most work of all ${inWords(items.length)} recommendations.`
  return `${String(others.filter((o) => o.effort < it.effort).length)} of the other ${inWords(others.length)} take less work.`
}

/** What comes before a recommendation in the ranking, in a sentence. */
export function placeNote(rank: number, of: number): string {
  if (rank === 1) return `It comes before the other ${inWords(of - 1)}.`
  return `${String(rank - 1)} of the other ${inWords(of - 1)} come before it.`
}
