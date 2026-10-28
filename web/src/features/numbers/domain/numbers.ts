import type { EyeOnReason, MeasureLeader, Plan, PlanImprovement, PlanTarget, Standing, StandingMeasure, Standings, WatchedNumber } from '../../../infrastructure/api/types.gen'
import { formatValue, inWords } from '../../catching-up/domain/standings'

/** A list as a sentence writes it, with the Oxford comma: "1", "1 and 5", "1, 2, and 5". */
export function oxford(items: readonly string[]): string {
  if (items.length <= 2) return items.join(' and ')
  return `${items.slice(0, -1).join(', ')}, and ${items[items.length - 1] ?? ''}`
}

/** What the page knows about each number to watch. The server decides which numbers there are and their targets. */
type Definition = {
  /** The address, `/numbers/:slug`. */
  slug: string
  label: string
  /** What the number counts, as a sentence. */
  counts: string
  /** How its values are written. */
  unit: StandingMeasure
  /** The standing that compares Tribelt with the competitors in the number's own unit. */
  compare?: StandingMeasure
  /** The standing whose leader is shown beside a number that is not in the same unit, such as a count against a share. */
  context?: StandingMeasure
  /** The competitor to look at, when the plan names one instead of the standings. */
  eyeOn?: EyeOnReason
  /** The improvements that move it are the ones for this measure. */
  improvedBy: StandingMeasure
}

const DEFINITIONS: Record<PlanTarget, Definition> = {
  aiShareOfVoice: {
    slug: 'ai-answers',
    label: 'AI answers that name Tribelt',
    counts: "the share of the newest Prompt run's answers that name Tribelt at least once.",
    unit: 'aiShareOfVoice',
    compare: 'aiShareOfVoice',
    improvedBy: 'aiShareOfVoice',
  },
  twenteShareOfVoice: {
    slug: 'twente-answers',
    label: 'Answers about Twente that name Tribelt',
    counts: "the share of the newest Prompt run's answers to questions about Twente that name Tribelt at least once.",
    unit: 'aiShareOfVoice',
    eyeOn: 'twente',
    improvedBy: 'aiShareOfVoice',
  },
  citations: {
    slug: 'web-citations',
    label: 'Web answers that cite tribelt.nl',
    counts: "how many of the newest Prompt run's web answers cite tribelt.nl.",
    unit: 'termsUsed',
    context: 'citationShare',
    improvedBy: 'citationShare',
  },
  readinessChecks: {
    slug: 'ai-readiness',
    label: 'AI readiness checks passed',
    counts: 'how many of the AI readiness checks the newest scan of tribelt.nl passes.',
    unit: 'readinessChecks',
    compare: 'readinessChecks',
    improvedBy: 'readinessChecks',
  },
  termsUsed: {
    slug: 'search-terms',
    label: 'Search terms tribelt.nl uses',
    counts: "how many of the map's search terms the newest scan of tribelt.nl found.",
    unit: 'termsUsed',
    compare: 'termsUsed',
    improvedBy: 'termsUsed',
  },
}

/** The number to watch with this address, or undefined when there is none. */
export function targetOfSlug(slug: string): PlanTarget | undefined {
  return (Object.keys(DEFINITIONS) as PlanTarget[]).find((t) => DEFINITIONS[t].slug === slug)
}

/** The address of a number's page. */
export function pathOfTarget(target: PlanTarget): string {
  return `/numbers/${DEFINITIONS[target].slug}`
}

/** What the page has to work with: the plan, the standings and how many readiness checks there are. */
export type Facts = { plan: Plan; standings: Standings }

/** A value as the page writes it: "24%", "3 of 5", "45". */
export function valueText(target: PlanTarget, value: number, readinessChecks: number): string {
  const def = DEFINITIONS[target]
  const text = formatValue(def.unit, value)
  return def.unit === 'readinessChecks' ? `${text} of ${String(readinessChecks)}` : text
}

function standingOf(standings: Standings, measure: StandingMeasure | undefined): Standing | undefined {
  return measure === undefined ? undefined : standings.measures.find((s) => s.measure === measure)
}

/** The recommendation numbers that improve a number: the places in the plan's order, from 1. */
export function improvementRanks(plan: Plan, target: PlanTarget): number[] {
  const measure = DEFINITIONS[target].improvedBy
  return plan.improvements.flatMap((it, index) => (it.measure === measure ? [index + 1] : []))
}

/** The competitor to look at: the leader of a standing, or the one the plan names. */
function leaderOf({ plan, standings }: Facts, def: Definition): MeasureLeader | undefined {
  if (def.eyeOn) return plan.eyeOn.find((e) => e.why === def.eyeOn)?.leader
  return standingOf(standings, def.compare ?? def.context)?.leader
}

/** Whether the leader's value is in the number's own unit, so that a bar can set the two side by side. */
function sameUnit(def: Definition): boolean {
  return def.compare !== undefined || def.eyeOn !== undefined
}

/** The share of the way to the target, as a whole percentage. */
function wayThere(now: number, goal: number): number {
  return goal > 0 ? Math.round(Math.min(1, now / goal) * 100) : 100
}

/** One number to watch as the overview draws it: a row with a bar, the target and the leader. */
export type NumberRow = {
  target: PlanTarget
  path: string
  label: string
  /** Tribelt's value, or what is missing. */
  now: string
  measured: boolean
  goal: string
  /** Where the bar ends, the target mark and the leader's dot, each from 0 to 100. */
  fill: number
  goalAt: number
  leaderAt: number | undefined
  /** A sentence on how far along it is. */
  progress: string
  /** "Improved by recommendations 1 and 5", or empty. */
  improvedBy: string
  aria: string
}

/** The sentence under a number's bar: how far it is, and who leads. */
function progressLine(def: Definition, w: WatchedNumber, leader: MeasureLeader | undefined): string {
  if (w.now === undefined) return 'Not measured yet.'
  const head = w.met === true ? 'The target is met.' : `${String(wayThere(w.now, w.goal))}% of the way to the target.`
  if (leader === undefined || !sameUnit(def)) return head
  return w.now >= leader.value ? `${head} Tribelt leads the field.` : `${head} ${leader.name} is at ${formatValue(def.unit, leader.value)}.`
}

/** The rows of the numbers to watch, in the order the server gives them. */
export function numberRows(facts: Facts): NumberRow[] {
  const { plan, standings } = facts
  const checks = standings.readinessChecks
  return plan.watched.map((w) => {
    const def = DEFINITIONS[w.target]
    const leader = sameUnit(def) ? leaderOf(facts, def) : undefined
    const scale = Math.max(w.goal, w.now ?? 0, leader?.value ?? 0, 0.0001) * 1.08
    const at = (v: number) => (Math.max(0, v) / scale) * 100
    const goal = valueText(w.target, w.goal, checks)
    const now = w.now === undefined ? 'Not measured' : valueText(w.target, w.now, checks)
    const progress = progressLine(def, w, leader)
    const ranks = improvementRanks(plan, w.target)
    return {
      target: w.target,
      path: pathOfTarget(w.target),
      label: def.label,
      now,
      measured: w.now !== undefined,
      goal,
      fill: w.now === undefined ? 0 : at(w.now),
      goalAt: at(w.goal),
      leaderAt: leader === undefined ? undefined : at(leader.value),
      progress,
      improvedBy: ranks.length === 0 ? '' : `Improved by recommendation${ranks.length > 1 ? 's' : ''} ${oxford(ranks.map(String))}`,
      aria: `${def.label}: ${now}, target ${goal}. ${progress} Open its page.`,
    }
  })
}

export type BarTone = 'orange' | 'blue-grey' | 'ink' | 'silver'

/** Everything the page of one number shows. */
export type NumberDetail = {
  target: PlanTarget
  label: string
  headline: string
  counts: string
  figures: { big: string; title: string; note: string; tone: 'tribelt' | 'plain' }[]
  /** Tribelt, the typical competitor, the leader and the target, where the number's unit allows. */
  bars: { label: string; value: number; display: string; tone: BarTone; highlight?: boolean }[]
  barsMax: number
  /** Who the typical competitor is, or how competitors are compared on this number. */
  comparison: string
  leader: MeasureLeader | undefined
  improvements: { rank: number; improvement: PlanImprovement }[]
}

/** The headline: who is ahead, or how far Tribelt is. */
function headlineOf(def: Definition, w: WatchedNumber, nowText: string, goalText: string, leader: MeasureLeader | undefined): string {
  if (w.now === undefined) return `${def.label} is not measured yet.`
  if (leader === undefined || !sameUnit(def)) return `Tribelt is at ${nowText}, against a target of ${goalText}.`
  if (w.now >= leader.value) return `Tribelt leads on this number, with ${nowText}.`
  return `${leader.name} is ahead with ${formatValue(def.unit, leader.value)}, against Tribelt's ${nowText}.`
}

/** How the competitors are compared on a number, in a sentence. */
function comparisonOf(def: Definition, standing: Standing | undefined, leader: MeasureLeader | undefined): string {
  if (def.eyeOn) return leader ? `${leader.name} is named most when buyers ask for Twente. The other competitors are not compared on this number.` : 'No competitor is named for Twente yet.'
  if (def.context) return 'Competitors are compared by the share of web answers that cite their site, not by the count.'
  if (standing?.median === undefined) return 'No competitor has data on this number yet.'
  return `The typical competitor is the middle value over the ${inWords(standing.competitors)} competitors with data.`
}

/** The page of one number to watch, from the plan and the standings; undefined when the plan has no target for it. */
export function numberDetail(facts: Facts, target: PlanTarget): NumberDetail | undefined {
  const { plan, standings } = facts
  const w = plan.watched.find((x) => x.target === target)
  if (!w) return undefined
  const def = DEFINITIONS[target]
  const checks = standings.readinessChecks
  const standing = standingOf(standings, def.compare)
  const leader = leaderOf(facts, def)
  const nowText = w.now === undefined ? 'No data yet' : valueText(target, w.now, checks)
  const goalText = valueText(target, w.goal, checks)
  const together = sameUnit(def) && leader !== undefined

  const bars: NumberDetail['bars'] = [{ label: 'Tribelt', value: w.now ?? 0, display: nowText, tone: 'orange', highlight: true }]
  if (def.compare !== undefined) {
    const typical = standing?.median
    bars.push({ label: 'Typical competitor', value: typical ?? 0, display: typical === undefined ? 'No data yet' : valueText(target, typical, checks), tone: 'blue-grey' })
  }
  if (together) bars.push({ label: leader.name, value: leader.value, display: valueText(target, leader.value, checks), tone: 'ink' })
  bars.push({ label: 'Target', value: w.goal, display: goalText, tone: 'silver' })

  let way = 'The first scan or Prompt run fills this in.'
  if (w.now !== undefined) way = w.met === true ? 'Tribelt has met the target.' : `Tribelt is ${String(wayThere(w.now, w.goal))}% of the way there.`
  const leaderBig = leader === undefined ? 'None' : together ? valueText(target, leader.value, checks) : formatValue(def.context ?? def.unit, leader.value)

  return {
    target,
    label: def.label,
    headline: headlineOf(def, w, nowText, goalText, leader),
    counts: `This number is ${def.counts}`,
    figures: [
      { big: w.now === undefined ? 'No data' : nowText, title: 'Tribelt now', note: way, tone: 'tribelt' },
      {
        big: leaderBig,
        title: leader === undefined ? 'No leader yet' : `${leader.name}, the leader`,
        note: together || leader === undefined ? 'The competitor with the best value now.' : 'By the share of web answers that cite its site.',
        tone: 'plain',
      },
      { big: goalText, title: 'Target for the year', note: w.met === true ? 'The target is met.' : 'The goal for this number by the end of the year.', tone: 'plain' },
    ],
    bars,
    barsMax: Math.max(...bars.map((b) => b.value), 0.0001) * 1.08,
    comparison: comparisonOf(def, standing, leader),
    leader,
    improvements: plan.improvements.flatMap((improvement, index) => (improvement.measure === def.improvedBy ? [{ rank: index + 1, improvement }] : [])),
  }
}
