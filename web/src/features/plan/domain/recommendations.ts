import type { PlanImprovement, PlanTarget, StandingMeasure, WatchedNumber } from '../../../infrastructure/api/types.gen'
import { formatValue, MEASURES } from '../../catching-up/domain/standings'

/**
 * The kinds of recommendation. The Recommendation run will write search terms, pages, listings,
 * and fact sheet corrections; until it does, every recommendation is written by hand, and is about
 * AI readiness when a scan decides it is done, or general otherwise.
 */
export type Kind = 'term' | 'page' | 'listing' | 'fact' | 'readiness' | 'general'

const KINDS: readonly Kind[] = ['term', 'page', 'listing', 'fact', 'readiness', 'general']

/** One recommendation's kind, as its row names it. */
export const KIND_LABEL: Record<Kind, string> = {
  term: 'Search term',
  page: 'New page',
  listing: 'Listing',
  fact: 'Fact sheet correction',
  readiness: 'AI readiness',
  general: 'General',
}

/** A kind as the filter names it. */
export const KIND_PLURAL: Record<Kind, string> = {
  term: 'Search terms to add',
  page: 'Pages to add',
  listing: 'Sites to get listed on',
  fact: 'Fact sheet corrections',
  readiness: 'AI readiness',
  general: 'General',
}

/** A recommendation's kind: a scan decides the AI readiness ones; the rest apply in general. */
export function kindOf(it: Pick<PlanImprovement, 'auto'>): Kind {
  return it.auto ? 'readiness' : 'general'
}

/** The kinds the recommendations have, in the filter's order. */
export function kindsIn(items: readonly PlanImprovement[]): Kind[] {
  const there = new Set(items.map(kindOf))
  return KINDS.filter((k) => there.has(k))
}

/** Where a recommendation applies: a hand-written one, to the whole site. */
export function whereText(): string {
  return 'The whole site'
}

/** The list page's data line. */
export function dataLine(catalogueVersion: number): string {
  return `Written by hand, version ${String(catalogueVersion)} of the catalogue. No Recommendation run has written any yet.`
}

/** The recommendations not done yet, best value first, as the server sends them. */
export function open(items: readonly PlanImprovement[]): PlanImprovement[] {
  return items.filter((i) => !i.done)
}

/** What the ranked list can be sorted by. */
export type Sort = 'value' | 'benefit' | 'effort' | 'kind' | 'title'
export type Direction = 'desc' | 'asc'

export const SORTS: readonly { id: Sort; label: string }[] = [
  { id: 'value', label: 'Best value' },
  { id: 'benefit', label: 'Benefit' },
  { id: 'effort', label: 'Effort' },
  { id: 'kind', label: 'Kind' },
  { id: 'title', label: 'Name' },
]

/**
 * The list in the order picked: best value is the server's order, and breaks every other tie; the
 * larger benefit first, the least work first, kinds and names from A. Ascending turns it round.
 */
export function listed(items: readonly PlanImprovement[], sort: Sort, direction: Direction): PlanImprovement[] {
  const value = (a: PlanImprovement, b: PlanImprovement) => items.indexOf(a) - items.indexOf(b)
  const by: Record<Sort, (a: PlanImprovement, b: PlanImprovement) => number> = {
    value,
    benefit: (a, b) => b.benefit - a.benefit || value(a, b),
    effort: (a, b) => a.effort - b.effort || value(a, b),
    kind: (a, b) => KIND_LABEL[kindOf(a)].localeCompare(KIND_LABEL[kindOf(b)]) || value(a, b),
    title: (a, b) => a.title.localeCompare(b.title),
  }
  const sorted = [...items].sort(by[sort])
  return direction === 'asc' ? sorted.reverse() : sorted
}

/** One bubble of the chart: the recommendations with the same benefit and effort. */
export type Cell = { key: string; benefit: number; effort: number; count: number; label: string }

const cellKey = (it: Pick<PlanImprovement, 'benefit' | 'effort'>) => `${String(it.benefit)}|${String(it.effort)}`

/** A bubble for each benefit and effort the recommendations have: the most benefit first, then the least work. */
export function cells(items: readonly PlanImprovement[]): Cell[] {
  const found = new Map<string, Cell>()
  for (const it of items) {
    const key = cellKey(it)
    const cell = found.get(key) ?? { key, benefit: it.benefit, effort: it.effort, count: 0, label: '' }
    cell.count += 1
    found.set(key, cell)
  }
  const order = (a: Cell, b: Cell) => b.benefit - a.benefit || a.effort - b.effort
  return [...found.values()].sort(order).map((c) => ({
    ...c,
    label: `${String(c.count)} ${c.count === 1 ? 'recommendation' : 'recommendations'} with benefit ${String(c.benefit)} of 5 and effort ${String(c.effort)} of 5`,
  }))
}

/** The recommendations of one bubble; all of them when no bubble is picked. */
export function inCell(items: readonly PlanImprovement[], key: string | undefined): PlanImprovement[] {
  return key === undefined ? [...items] : items.filter((i) => cellKey(i) === key)
}

/** The line under a list that shows five rows at a time, when it holds more. */
export function scrollCue(count: number): string {
  return count > 5 ? `Scroll inside the list to see all ${String(count)}.` : ''
}

/** The two scores: "Benefit 4 of 5, effort 1 of 5". */
export function scoreLine(it: Pick<PlanImprovement, 'benefit' | 'effort'>): string {
  return `Benefit ${String(it.benefit)} of 5, effort ${String(it.effort)} of 5`
}

const DAY = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })

/** When a recommendation was marked done, and by whom: an administrator, or the scan of tribelt.nl. */
export function doneLine(it: Pick<PlanImprovement, 'auto' | 'tickedBy' | 'tickedAt'>): string {
  const when = it.tickedAt ? `Marked done on ${DAY.format(new Date(it.tickedAt))}` : 'Marked done, and the date was not recorded'
  const who = it.auto ? 'the scan of tribelt.nl' : (it.tickedBy ?? 'an administrator')
  return `${when}, by ${who}.`
}

/** What each measure is called in a sentence that begins with it. */
const MEASURE_NAME: Record<StandingMeasure, string> = {
  aiShareOfVoice: 'AI responses that name Tribelt',
  citationShare: 'Web responses that cite tribelt.nl',
  readinessChecks: 'AI readiness checks passed',
  sitemapCoverage: 'Pages the sitemaps list',
  termsUsed: 'Search terms tribelt.nl uses',
}

export function measureName(measure: StandingMeasure): string {
  return MEASURE_NAME[measure]
}

/** A name as it reads inside a sentence: the first letter small, unless it starts a capital word such as "AI". */
export function lcFirst(s: string): string {
  return /^[A-Z][a-z]/.test(s) ? s.charAt(0).toLowerCase() + s.slice(1) : s
}

/** The goal that watches each measure, where one does. */
const GOAL_OF: Record<StandingMeasure, PlanTarget | undefined> = {
  aiShareOfVoice: 'aiShareOfVoice',
  citationShare: 'citationShare',
  readinessChecks: 'readinessChecks',
  sitemapCoverage: undefined,
  termsUsed: 'termsUsed',
}

/** The goal on a measure, where one watches it. */
export function goalOf(measure: StandingMeasure, watched: readonly WatchedNumber[]): number | undefined {
  const target = GOAL_OF[measure]
  return target === undefined ? undefined : watched.find((w) => w.target === target)?.goal
}

/** A value as the measure counts it: "15%", "3 of 8", "1,200". */
export function formatMeasure(measure: StandingMeasure, value: number, readinessChecks: number): string {
  const text = formatValue(measure, value)
  return measure === 'readinessChecks' ? `${text} of ${String(readinessChecks)}` : text
}

type OnMeasure = Pick<PlanImprovement, 'measure' | 'tribelt' | 'leader'>

/** Where Tribelt is now, and the goal: "Tribelt now 20%, the goal 40%." */
export function nowSentence(it: Pick<PlanImprovement, 'measure' | 'tribelt'>, goal: number | undefined, readinessChecks: number): string {
  const f = (v: number) => formatMeasure(it.measure, v, readinessChecks)
  const now = it.tribelt === undefined ? 'Tribelt has no figure on this yet' : `Tribelt now ${f(it.tribelt)}`
  if (goal !== undefined) return `${now}, the goal ${f(goal)}.`
  return it.tribelt === undefined ? `${now}.` : `${now}. No goal watches this measure.`
}

/** Tribelt against the best competitor, right whichever is ahead; empty without a best competitor. */
export function goalSentence(it: OnMeasure, readinessChecks: number): string {
  const best = it.leader
  if (!best) return ''
  const f = (v: number) => formatMeasure(it.measure, v, readinessChecks)
  if (it.tribelt === undefined) return `The best competitor, ${best.name}, is at ${f(best.value)}.`
  if (best.value > it.tribelt) return `${best.name} is ahead of Tribelt, with ${f(best.value)} against ${f(it.tribelt)}.`
  if (best.value < it.tribelt) return `Tribelt is ahead of the best competitor, ${best.name}, at ${f(best.value)}.`
  return `The best competitor, ${best.name}, is level with Tribelt at ${f(best.value)}.`
}

/** Tribelt now, the best competitor, and the goal, each from 0 to 1 along one bar. */
export type GoalBar = { now: number; leader: number | undefined; goal: number | undefined }

/**
 * The bar runs to the most a measure can be: 1 for a share, every readiness check for the
 * readiness checks; a count has no most, so it runs a fifth past the goal or to the largest value.
 */
export function goalBar(it: OnMeasure, goal: number | undefined, readinessChecks: number): GoalBar {
  const now = it.tribelt ?? 0
  const best = it.leader?.value
  const scale = MEASURES[it.measure].unit === 'share' ? 1 : it.measure === 'readinessChecks' ? readinessChecks : Math.max((goal ?? 0) * 1.2, best ?? 0, now, 1)
  const at = (v: number) => Math.min(1, Math.max(0, v / scale))
  return { now: at(now), leader: best === undefined ? undefined : at(best), goal: goal === undefined ? undefined : at(goal) }
}

/** Why a recommendation of a kind takes the work it does. */
export function effortWhy(kind: Kind): string {
  switch (kind) {
    case 'term':
      return 'A search term on an existing page is a small change.'
    case 'page':
      return 'A new page is a larger piece of work.'
    case 'listing':
      return 'It needs a request to a site Tribelt does not run.'
    case 'fact':
      return 'It is a change to the fact sheet.'
    case 'readiness':
      return 'It is a change to the site or its files.'
    case 'general':
      return 'It is work for the web agency.'
  }
}

/** A place in words: "1st". */
export function ordinal(n: number): string {
  const rest = n % 100
  if (rest >= 11 && rest <= 13) return `${String(n)}th`
  const last = n % 10
  return `${String(n)}${last === 1 ? 'st' : last === 2 ? 'nd' : last === 3 ? 'rd' : 'th'}`
}

/** An open recommendation's place among the open ones by best value; empty once done. */
export function placeLine(items: readonly PlanImprovement[], id: string): string {
  const still = open(items)
  const i = still.findIndex((it) => it.id === id)
  if (i < 0) return ''
  return `It ranks ${ordinal(i + 1)} of the ${String(still.length)} open recommendations by best value.`
}
