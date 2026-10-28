import type { EyeOn, PlanImprovement, PlanPart, PlanTarget, StandingMeasure, TermsSource, WatchedNumber } from '../../../infrastructure/api/types.gen'
import { formatValue, inWords } from '../../catching-up/domain/standings'
import { joinWords } from '../../../shared/words'

/** One step of the weekly routine: what to do, when, what to look for, and the page to open. */
export type Step = { number: string; title: string; when: string; what: string; look: string; link: { label: string; path: string } }

/** The weekly routine: about an hour a week in the tool. */
export const ROUTINE: readonly Step[] = [
  {
    number: '1',
    title: 'Read the Monday report',
    when: 'Monday morning, 15 minutes',
    what: 'On a weekly schedule the Prompt test asks the buyer questions again early every Monday.',
    look: 'Did more answers name Tribelt than last week? Which questions still leave Tribelt out?',
    link: { label: 'Open AI search', path: '/ai-search' },
  },
  {
    number: '2',
    title: 'See who is ahead',
    when: 'Monday, 15 minutes',
    what: 'The overview shows where Tribelt trails the leaders. Each competitor has its own page with its website, terms and AI readiness.',
    look: 'Pick the biggest gap, and the competitor who does it best.',
    link: { label: 'Open the overview', path: '/' },
  },
  {
    number: '3',
    title: 'Make one improvement',
    when: 'During the week, about half an hour',
    what: 'Take the next improvement on this page. Most are small changes to tribelt.nl that marketing or the web agency can make in a day.',
    look: 'Copy what the leader does, in Tribelt’s own words and products.',
    link: { label: 'See the improvements', path: '/plan' },
  },
  {
    number: '4',
    title: 'Scan tribelt.nl again',
    when: 'Friday, one click',
    what: 'A new scan updates the AI readiness and the search terms straight away. The answers follow in next Monday’s Prompt test.',
    look: 'Check the change shows up. Then tick it off.',
    link: { label: 'Open Tribelt’s page', path: '/competitors/tribelt' },
  },
]

/** The year in three phases, by month from November. */
export const YEAR: readonly { label: string; start: number; span: number; tone: 'orange' | 'black' | 'blue-grey' }[] = [
  { label: 'Fix the basics', start: 1, span: 2, tone: 'orange' },
  { label: 'Write for buyers', start: 3, span: 4, tone: 'black' },
  { label: 'Get cited and keep checking', start: 7, span: 6, tone: 'blue-grey' },
]

export const MONTHS = ['Nov', 'Dec', 'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct'] as const

/** What each number to watch is called, and how its values read; the server says how high each can go. */
export const TARGETS: Record<PlanTarget, { label: string; format: (v: number) => string; step: number }> = {
  aiShareOfVoice: { label: 'AI answers in the newest Prompt run that name Tribelt', format: (v) => formatValue('aiShareOfVoice', v), step: 0.01 },
  twenteShareOfVoice: {
    label: 'Answers about Twente in the newest Prompt run that name Tribelt',
    format: (v) => formatValue('aiShareOfVoice', v),
    step: 0.01,
  },
  readinessChecks: { label: "AI readiness checks tribelt.nl's newest scan passed", format: (v) => formatValue('readinessChecks', v), step: 0.5 },
  termsUsed: { label: "Search terms from the map found on tribelt.nl's newest scan", format: (v) => formatValue('termsUsed', v), step: 1 },
  citations: { label: 'Web answers in the newest Prompt run that cite tribelt.nl', format: (v) => formatValue('termsUsed', v), step: 1 },
}

/**
 * The label of a number to watch: the readiness checks say how many there are, and the search
 * terms say so when the desk research counts them, before a scan of tribelt.nl.
 */
export function targetLabel(target: PlanTarget, readinessChecks: number, termsFrom?: TermsSource): string {
  if (target === 'termsUsed' && termsFrom === 'research') return 'Search terms from the map found for Tribelt by our research'
  if (target === 'readinessChecks') return `${TARGETS[target].label}, out of ${inWords(readinessChecks)}`
  return TARGETS[target].label
}

/** A number to watch as the page draws it: a met target says so and asks to be raised, and has no target mark behind the bar. */
export type WatchedRow = {
  label: string
  now: string
  goal: string
  /** The line beside the label: "24% to 40%", or "Met: 45, target was 35". */
  status: string
  met: boolean
  nowLength: number
  /** Where the target mark sits, from 0 to 1; undefined when the target is met. */
  goalLength: number | undefined
}

/**
 * A number to watch, now against its target. The bar's scale reaches the target, or past the value
 * when the value has passed it, so a met target fills the bar rather than drawing a target behind it.
 */
export function watchedRow(w: WatchedNumber, readinessChecks: number, termsFrom?: TermsSource): WatchedRow {
  const t = TARGETS[w.target]
  const met = w.met === true && w.now !== undefined
  const max = w.max ?? Math.max(w.goal, w.now ?? 0) * 1.15
  const now = w.now === undefined ? 'No data yet' : t.format(w.now)
  const goal = t.format(w.goal)
  return {
    label: targetLabel(w.target, readinessChecks, termsFrom),
    now,
    goal,
    status: met ? `Met: ${now}, target was ${goal}` : `${now} to ${goal}`,
    met,
    nowLength: Math.min(1, (w.now ?? 0) / max),
    goalLength: met ? undefined : Math.min(1, w.goal / max),
  }
}

/** The line under the numbers when a target is met: what to do about it, for those who can. */
export function metLine(watched: readonly WatchedNumber[], isAdmin: boolean): string {
  if (!watched.some((w) => w.met === true && w.now !== undefined)) return ''
  return isAdmin
    ? 'A target you have met is done. Raise it under Change the targets.'
    : 'A target you have met is done. An administrator can raise it.'
}

/** Why to keep an eye on a competitor, in a sentence after its name. */
export function eyeOnText(e: Pick<EyeOn, 'why'>): string {
  switch (e.why) {
    case 'mostNamed':
      return 'is named most in AI answers.'
    case 'bestPrepared':
      return 'is best prepared for AI.'
    case 'twente':
      return 'is named most when buyers ask for Twente.'
  }
}

/** Tribelt's value on each measure in a sentence, and what is missing without one. */
const NOW: Record<StandingMeasure, { says: (value: string, readinessChecks: number) => string; missing: string }> = {
  aiShareOfVoice: { says: (v) => `${v} of the AI answers in the newest Prompt run name Tribelt.`, missing: 'No Prompt run has finished yet.' },
  citationShare: { says: (v) => `${v} of the web answers in the newest Prompt run cite tribelt.nl.`, missing: 'No Prompt run has finished yet.' },
  readinessChecks: { says: (v, checks) => `tribelt.nl passes ${v} of ${inWords(checks)} AI readiness checks.`, missing: 'No scan of tribelt.nl yet.' },
  sitemapCoverage: {
    says: (v) => `The sitemaps of tribelt.nl list ${v} of the pages its newest scan read.`,
    missing: 'Not measured on tribelt.nl yet. Scan it again.',
  },
  termsUsed: { says: (v) => `The newest scan of tribelt.nl found ${v} search terms from the map.`, missing: 'No scan of tribelt.nl yet.' },
}

/** Where Tribelt stands on the measure an improvement moves, and who leads, in sentences. */
export function standingLine(it: Pick<PlanImprovement, 'measure' | 'tribelt' | 'leader'>, readinessChecks: number, termsFrom?: TermsSource): string {
  const now = NOW[it.measure]
  const credited = it.measure === 'termsUsed' && termsFrom === 'research'
  const says = (v: string) => (credited ? `Our research found ${v} search terms from the map for Tribelt.` : now.says(v, readinessChecks))
  const tribelt = it.tribelt === undefined ? now.missing : says(formatValue(it.measure, it.tribelt))
  return it.leader ? `${tribelt} ${it.leader.name} leads with ${formatValue(it.measure, it.leader.value)}.` : tribelt
}

/** An improvement's status now: what the latest scan found when a scan checks it, else where Tribelt stands. */
export function statusLine(it: Pick<PlanImprovement, 'measure' | 'tribelt' | 'leader' | 'parts'>, readinessChecks: number, termsFrom?: TermsSource): string {
  return foundLine(it.parts) || standingLine(it, readinessChecks, termsFrom)
}

/** What the latest scan of tribelt.nl found of an improvement it checks, and what it did not. */
export function foundLine(parts: readonly PlanPart[]): string {
  const found = parts.filter((p) => p.present).map((p) => p.name)
  const missing = parts.filter((p) => !p.present).map((p) => p.name)
  if (found.length === 0 && missing.length === 0) return ''
  if (found.length === 0) return `The latest scan did not find ${joinWords(missing, 'or')}.`
  if (missing.length === 0) return `The latest scan found ${joinWords(found)}.`
  return `The latest scan found ${joinWords(found)}, but not ${joinWords(missing, 'or')}.`
}
