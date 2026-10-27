import type { EyeOn, PlanTarget, WatchedNumber } from '../../../infrastructure/api/types.gen'
import { formatValue } from '../../catching-up/domain/standings'

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
    what: 'Catching up shows where Tribelt trails the leaders. Each competitor has its own page with its website, terms and AI readiness.',
    look: 'Pick the biggest gap, and the competitor who does it best.',
    link: { label: 'Open Catching up', path: '/catching-up' },
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

/** What each number to watch is called, and how its values read. */
export const TARGETS: Record<PlanTarget, { label: string; format: (v: number) => string; step: number; max?: number }> = {
  aiShareOfVoice: { label: 'AI answers that name Tribelt', format: (v) => formatValue('aiShareOfVoice', v), step: 0.01, max: 1 },
  twenteShareOfVoice: { label: 'Answers about Twente that name Tribelt', format: (v) => formatValue('aiShareOfVoice', v), step: 0.01, max: 1 },
  readinessChecks: { label: 'AI readiness checks passed out of five', format: (v) => formatValue('readinessChecks', v), step: 0.5, max: 5 },
  termsUsed: { label: 'Search terms used on tribelt.nl', format: (v) => formatValue('termsUsed', v), step: 1 },
  citations: { label: 'Web answers in a run that cite tribelt.nl', format: (v) => formatValue('termsUsed', v), step: 1 },
}

/** A number to watch as the page draws it: now and the goal as text, and as lengths from 0 to 1. */
export function watchedRow(w: WatchedNumber): { label: string; now: string; goal: string; nowLength: number; goalLength: number } {
  const t = TARGETS[w.target]
  const max = t.max ?? Math.max(w.goal, w.now ?? 0) * 1.15
  return {
    label: t.label,
    now: w.now === undefined ? 'No data yet' : t.format(w.now),
    goal: t.format(w.goal),
    nowLength: Math.min(1, (w.now ?? 0) / max),
    goalLength: Math.min(1, w.goal / max),
  }
}

/** Why to keep an eye on a competitor, in a sentence after its name. */
export function eyeOnText(e: Pick<EyeOn, 'why'>): string {
  switch (e.why) {
    case 'mostNamed':
      return 'is named most in AI answers.'
    case 'bestPrepared':
      return 'has the best prepared website.'
    case 'twente':
      return 'is named most when buyers ask for Twente.'
  }
}
