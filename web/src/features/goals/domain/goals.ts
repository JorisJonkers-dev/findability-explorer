import type {
  EyeOnReason,
  MeasureLeader,
  Overview,
  OverviewLanguage,
  Plan,
  PlanImprovement,
  PlanTarget,
  PromptCell,
  PromptRun,
  ReadinessCheckSummary,
  Standing,
  StandingMeasure,
  Standings,
} from '../../../infrastructure/api/types.gen'
import { localDate } from '../../../shared/dates'
import type { Tone } from '../../../shared/ui/charts/tones'
import { languageLabel } from '../../../shared/vocabulary'
import { formatValue, inWords } from '../../catching-up/domain/standings'
import type { Measure } from '../../prompt-test/domain/history'
import { FAMILIES, providerName, SCOPES } from '../../prompt-test/domain/results'
import { CHECK_NAMES, CHECK_WHY, SCORED_CHECKS } from '../../readiness/domain/checks'

/** How a goal is measured: by the Prompt test, or by the scans of the sites. */
type Process = 'prompt-test' | 'scans'

/** What the pages know about each goal. The server decides which goals there are and their values. */
type Definition = {
  name: string
  /** One sentence on what the goal measures; the readiness checks say how many there are. */
  purpose: (readinessChecks: number) => string
  /** How its values are written. */
  unit: StandingMeasure
  process: Process
  /** The standing that compares Tribelt with the competitors in the goal's own unit. */
  compare?: StandingMeasure
  /** The competitor to look at, when the plan names one instead of the standings. */
  eyeOn?: EyeOnReason
  /** The recommendations aimed at it are the ones for this measure. */
  improvedBy: StandingMeasure
  /** Its line over time, from the Prompt test history. */
  history?: Measure
}

const DEFINITIONS: Record<PlanTarget, Definition> = {
  aiShareOfVoice: {
    name: 'AI responses that name Tribelt',
    purpose: () => "The share of a Prompt run's responses that name Tribelt.",
    unit: 'aiShareOfVoice',
    process: 'prompt-test',
    compare: 'aiShareOfVoice',
    improvedBy: 'aiShareOfVoice',
    history: { label: 'Tribelt', tone: 'orange', strong: true, pick: (r) => r.all },
  },
  twenteShareOfVoice: {
    name: 'AI responses about the Twente region name Tribelt',
    purpose: () => "The share of a Prompt run's responses about the Twente region that name Tribelt.",
    unit: 'aiShareOfVoice',
    process: 'prompt-test',
    eyeOn: 'twente',
    improvedBy: 'aiShareOfVoice',
    history: { label: 'Tribelt', tone: 'orange', strong: true, pick: (r) => r.twente },
  },
  citationShare: {
    name: 'Web responses that cite tribelt.nl',
    purpose: () => "The share of a Prompt run's responses with web search that link to a tribelt.nl page.",
    unit: 'citationShare',
    process: 'prompt-test',
    compare: 'citationShare',
    improvedBy: 'citationShare',
    history: { label: 'Tribelt', tone: 'orange', strong: true, pick: (r) => r.citation },
  },
  readinessChecks: {
    name: 'AI readiness checks passed',
    purpose: (n) => `How many of the ${inWords(n)} AI readiness checks tribelt.nl passes.`,
    unit: 'readinessChecks',
    process: 'scans',
    compare: 'readinessChecks',
    improvedBy: 'readinessChecks',
  },
  termsUsed: {
    name: 'Search terms tribelt.nl uses',
    purpose: () => 'How many search terms on the list appear on tribelt.nl.',
    unit: 'termsUsed',
    process: 'scans',
    compare: 'termsUsed',
    improvedBy: 'termsUsed',
  },
}

/** The five goals, in the order the spec names them. */
export const GOAL_ORDER: readonly PlanTarget[] = ['aiShareOfVoice', 'twenteShareOfVoice', 'citationShare', 'readinessChecks', 'termsUsed']

/** The goal with this address, or undefined when there is none. */
export function targetOfSlug(slug: string): PlanTarget | undefined {
  return GOAL_ORDER.find((t) => t === slug)
}

/** A goal page's name in the browser tab: "Goal, AI responses that name Tribelt", or "Goal" for an address no goal has. */
export function goalTabTitle(slug: string): string {
  const target = targetOfSlug(slug)
  return target ? `Goal, ${DEFINITIONS[target].name}` : 'Goal'
}

/** The address of a goal's page. */
export function pathOfTarget(target: PlanTarget): string {
  return `/goals/${target}`
}

/** Whether the Prompt test measures the goal, so its page draws the history with the Prompt set filter. */
export function isPromptGoal(target: PlanTarget): boolean {
  return DEFINITIONS[target].process === 'prompt-test'
}

/** The goal's line over time, from the Prompt test history; undefined for a goal the scans measure. */
export function historyMeasure(target: PlanTarget): Measure | undefined {
  return DEFINITIONS[target].history
}

/** What the pages have to work with: the plan and the standings. */
export type Facts = { plan: Plan; standings: Standings }

/** A labelled bar, as the shared bar list draws it. */
export type Bar = { label: string; value: number; display: string; tone: Tone; highlight?: boolean }

/** A value as the pages write it: "24%", "3 of 8", "45". */
export function valueText(target: PlanTarget, value: number, readinessChecks: number): string {
  const def = DEFINITIONS[target]
  const text = formatValue(def.unit, value)
  return def.unit === 'readinessChecks' ? `${text} of ${String(readinessChecks)}` : text
}

function standingOf(standings: Standings, measure: StandingMeasure | undefined): Standing | undefined {
  return measure === undefined ? undefined : standings.measures.find((s) => s.measure === measure)
}

/** The best competitor: the leader of a standing, or the one the plan names. */
function bestOf({ plan, standings }: Facts, def: Definition): MeasureLeader | undefined {
  if (def.eyeOn) return plan.eyeOn.find((e) => e.why === def.eyeOn)?.leader
  return standingOf(standings, def.compare)?.leader
}

/** Where a bar ends: a share at 100%, the checks at all of them, a count at its largest value. */
function scaleOf(def: Definition, readinessChecks: number, values: readonly number[]): number {
  switch (def.unit) {
    case 'aiShareOfVoice':
    case 'citationShare':
    case 'sitemapCoverage':
      return 1
    case 'readinessChecks':
      return readinessChecks
    case 'termsUsed':
      return Math.max(...values, 0.0001)
  }
}

/** One row of the Overview's Goals table. */
export type GoalRow = {
  target: PlanTarget
  path: string
  name: string
  now: string
  measured: boolean
  goal: string
  /** "Ashworth, 57%", or "None yet". */
  best: string
  /** Tribelt, the goal, and the best competitor on the progress bar, each from 0 to 100. */
  bar: { now: number; goal: number; best: number | undefined }
  /** What the progress bar shows, for a screen reader. */
  barLabel: string
  aria: string
}

const at = (v: number, scale: number): number => Math.round((Math.max(0, v) / scale) * 10_000) / 100

/** The Goals table: a row per goal the plan holds, in the spec's order. */
export function goalRows(facts: Facts): GoalRow[] {
  const checks = facts.plan.readinessChecks
  return GOAL_ORDER.flatMap((target) => {
    const w = facts.plan.watched.find((x) => x.target === target)
    if (!w) return []
    const def = DEFINITIONS[target]
    const best = bestOf(facts, def)
    const scale = scaleOf(def, checks, [w.goal * 1.2, w.now ?? 0, best?.value ?? 0])
    const now = w.now === undefined ? 'Not measured' : valueText(target, w.now, checks)
    const goal = valueText(target, w.goal, checks)
    const bestValue = best ? valueText(target, best.value, checks) : ''
    return [
      {
        target,
        path: pathOfTarget(target),
        name: def.name,
        now,
        measured: w.now !== undefined,
        goal,
        best: best ? `${best.name}, ${bestValue}` : 'None yet',
        bar: { now: w.now === undefined ? 0 : at(w.now, scale), goal: at(w.goal, scale), best: best ? at(best.value, scale) : undefined },
        barLabel: `Tribelt now ${w.now === undefined ? 'not measured' : now}${best ? `, the best competitor ${bestValue}` : ''}, the goal ${goal}.`,
        aria: `${def.name}: Tribelt ${w.now === undefined ? 'not measured' : now}, the goal ${goal}, ${best ? `the best competitor ${best.name} with ${bestValue}` : 'no best competitor yet'}. Open its page.`,
      },
    ]
  })
}

/** The Overview's sentence: how AI responses name Tribelt, the readiness checks, and the search terms. */
export function overviewSummary(facts: Facts, overview: Overview): string {
  const companies = overview.aiSearch.companies
  const mine = companies.find((c) => c.role === 'tribelt')?.share
  let first = 'No Prompt run has finished yet'
  if (mine !== undefined) {
    const ahead = companies.filter((c) => c.role === 'competitor' && c.share !== undefined && c.share > mine).length
    if (ahead === 0) first = 'AI responses name Tribelt more often than any competitor'
    else first = `AI responses name ${ahead === 1 ? 'one competitor' : `${String(ahead)} competitors`} more often than Tribelt`
  }
  const score = facts.plan.watched.find((w) => w.target === 'readinessChecks')?.now
  const checks = facts.plan.readinessChecks
  const ready =
    score === undefined ? 'tribelt.nl has not been checked for AI readiness yet' : `tribelt.nl passes ${formatValue('readinessChecks', score)} of the ${String(checks)} AI readiness checks`
  const terms = `tribelt.nl uses ${formatValue('termsUsed', overview.terms.tribelt)} of the ${formatValue('termsUsed', overview.terms.total)} search terms buyers use`
  return `${first}, ${ready}, and ${terms}.`
}

/** Everything a goal's page shows from the plan and the standings. */
export type GoalPage = {
  target: PlanTarget
  name: string
  purpose: string
  /** The goal's value, for the line over time. */
  goal: number
  /** Whether Tribelt or the best competitor is ahead, and the goal. */
  summary: string
  /** Tribelt, the typical competitor (where there is one), the best competitor, and the goal. */
  bars: Bar[]
  barsMax: number
  /** How the competitors are compared on this goal. */
  comparison: string
  leader: MeasureLeader | undefined
  /** The open recommendations aimed at it, in the plan's order. */
  improvements: PlanImprovement[]
}

function summaryOf(f: (v: number) => string, now: number | undefined, goalText: string, leader: MeasureLeader | undefined): string {
  if (now === undefined) return `Tribelt is not measured on this goal yet. The goal is ${goalText}.`
  if (leader && leader.value > now) return `${leader.name} is ahead of Tribelt, with ${f(leader.value)} against ${f(now)}. The goal is ${goalText}.`
  const against = leader ? ` against ${leader.name}'s ${f(leader.value)}` : ''
  return `Tribelt leads every competitor, with ${f(now)}${against}. The goal is ${goalText}.`
}

function comparisonOf(def: Definition, standing: Standing | undefined, leader: MeasureLeader | undefined): string {
  if (def.eyeOn) return leader ? `${leader.name} is named most when buyers ask about Twente. The other competitors are not compared on this goal.` : 'No competitor is named for Twente yet.'
  if (standing?.median === undefined) return 'No competitor has data on this goal yet.'
  return 'The typical competitor is the middle figure across all competitors on the list.'
}

/** A goal's page; undefined when the plan does not hold that goal. */
export function goalPage(facts: Facts, target: PlanTarget): GoalPage | undefined {
  const w = facts.plan.watched.find((x) => x.target === target)
  if (!w) return undefined
  const def = DEFINITIONS[target]
  const checks = facts.plan.readinessChecks
  const standing = standingOf(facts.standings, def.compare)
  const leader = bestOf(facts, def)
  const f = (v: number) => valueText(target, v, checks)
  const goalText = f(w.goal)

  const bars: Bar[] = [{ label: 'Tribelt', value: w.now ?? 0, display: w.now === undefined ? 'Not measured' : f(w.now), tone: 'orange', highlight: true }]
  if (def.compare !== undefined) {
    const typical = standing?.median
    bars.push({ label: 'The typical competitor', value: typical ?? 0, display: typical === undefined ? 'No data yet' : f(typical), tone: 'blue-grey' })
  }
  bars.push(
    leader
      ? { label: `${leader.name}, the best competitor`, value: leader.value, display: f(leader.value), tone: 'ink' }
      : { label: 'The best competitor', value: 0, display: 'None yet', tone: 'ink' },
  )
  bars.push({ label: 'The goal', value: w.goal, display: goalText, tone: 'silver' })

  return {
    target,
    name: def.name,
    purpose: def.purpose(checks),
    goal: w.goal,
    summary: summaryOf(f, w.now, goalText, leader),
    bars,
    barsMax: scaleOf(def, checks, bars.map((b) => b.value * 1.1)),
    comparison: comparisonOf(def, standing, leader),
    leader,
    improvements: facts.plan.improvements.filter((i) => i.measure === def.improvedBy && !i.done),
  }
}

/** Where a Prompt test goal's figures come from: the newest run, its questions, models, and responses. */
export function promptDataLine(run: PromptRun | undefined): string {
  if (!run) return 'No Prompt run has finished yet.'
  const models = run.models.length
  const responses = `${run.answered.toLocaleString('en-GB')} ${run.answered === 1 ? 'response' : 'responses'} in all`
  const asked = models > 0 ? `${String(Math.round(run.tasks / (models * 2)))} questions to ${String(models)} models, each from the model’s own knowledge and with web search, ` : ''
  return `From the Prompt run of ${localDate(run.startedAt)}: ${asked}${responses}. Prompt set version ${String(run.promptSetVersion)}.`
}

/** One list of Broken down: a title, and a bar per part with the total last. */
export type BreakdownGroup = { title: string; rows: Bar[] }

/** A Prompt test goal broken down by model, scope (except for Twente), and belt family, from the newest run. */
export function promptBreakdown(target: PlanTarget, cells: readonly PromptCell[]): BreakdownGroup[] {
  if (!isPromptGoal(target)) return []
  const twente = target === 'twenteShareOfVoice'
  const citing = target === 'citationShare'
  const base = cells.filter((c) => (!twente || c.scope === 'twente') && (!citing || c.mode === 'web'))
  const hit = (c: PromptCell) => (citing ? c.cited : c.named)
  const row = (label: string, xs: readonly PromptCell[], total = false): Bar[] => {
    const answers = xs.reduce((n, c) => n + c.answers, 0)
    if (answers === 0) return []
    const share = xs.reduce((n, c) => n + hit(c), 0) / answers
    const bar: Bar = { label, value: share, display: `${String(Math.round(share * 100))}%`, tone: total ? 'ink' : 'orange' }
    return [total ? { ...bar, highlight: true } : bar]
  }
  const all = (label: string) => row(label, base, true)
  if (all('All').length === 0) return []
  const providers = [...new Set(base.map((c) => c.provider))]
  const byModel = providers.flatMap((p) => row(providerName(p), base.filter((c) => c.provider === p))).sort((a, b) => b.value - a.value)
  const groups: BreakdownGroup[] = [{ title: 'By model', rows: [...byModel, ...all('All models')] }]
  if (!twente) groups.push({ title: 'By scope', rows: [...SCOPES.flatMap((s) => row(s.label, base.filter((c) => c.scope === s.value))), ...all('All scopes')] })
  groups.push({ title: 'By belt family', rows: [...FAMILIES.flatMap((f) => row(f.label, base.filter((c) => c.family === f.value))), ...all('All belt families')] })
  return groups
}

/** The readiness checks tribelt.nl is scored on, in the pages' order, each passed, failed, or not known. */
export function readinessBreakdown(checks: readonly ReadinessCheckSummary[]): { label: string; what: string; passed: boolean | null }[] {
  return SCORED_CHECKS.flatMap((check) => {
    const c = checks.find((x) => x.check === check && x.scored)
    return c ? [{ label: CHECK_NAMES[check], what: CHECK_WHY[check], passed: c.tribelt ?? null }] : []
  })
}

/** The search terms tribelt.nl uses, per language, with where the bars end. */
export function termsBreakdown(languages: readonly OverviewLanguage[]): { rows: Bar[]; max: number } {
  const rows: Bar[] = languages.map((l) => ({ label: `${languageLabel(l.language)}, of ${formatValue('termsUsed', l.terms)}`, value: l.tribelt, display: formatValue('termsUsed', l.tribelt), tone: 'orange' }))
  return { rows, max: rows.length ? Math.max(...rows.map((r) => r.value)) * 1.1 : 1 }
}
