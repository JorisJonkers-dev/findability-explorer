import type { AiReadinessRow, Overview, Plan, PromptRun } from '../../../infrastructure/api/types.gen'
import { localDate } from '../../../shared/dates'
import { isShown } from '../../prompt-test/domain/results'

/** A row of Latest runs and scans: the process, when it last ran, what comes next, and its two pages. */
export type LatestRow = { process: string; to: string; latest: string; next: string; manage: string }

export type LatestFacts = {
  /** The Prompt runs, newest first. */
  runs: readonly PromptRun[]
  /** When the next scheduled Prompt run starts; only administrators may read it. */
  nextRunAt?: string
  readiness: readonly AiReadinessRow[]
  overview: Overview
  plan: Plan
  now: Date
}

const DAY_MS = 86_400_000

/** When each process last ran and what comes next. */
export function latestRows(f: LatestFacts): LatestRow[] {
  // The run the goal figures come from: the newest complete one.
  const run = f.runs.find(isShown)
  const tribelt = f.readiness.find((r) => r.role === 'tribelt')?.scan?.finishedAt
  const competitors = f.readiness.filter((r) => r.role === 'competitor')
  const recent = competitors.filter((r) => r.scan?.finishedAt !== undefined && f.now.getTime() - Date.parse(r.scan.finishedAt) < 30 * DAY_MS).length
  const open = f.plan.improvements.filter((i) => !i.done).length
  const done = f.plan.improvements.length - open
  return [
    {
      process: 'Prompt test',
      to: '/ai-responses',
      manage: '/manage/prompt-test',
      latest: run ? `Last complete run on ${localDate(run.startedAt)}` : 'No Prompt run has finished yet.',
      next: f.nextRunAt ? `The next run is scheduled for ${localDate(f.nextRunAt)}.` : '',
    },
    {
      process: 'Website scans',
      to: '/ai-readiness',
      manage: '/manage/scans',
      latest: tribelt ? `tribelt.nl scanned on ${localDate(tribelt)}` : 'tribelt.nl has not been scanned yet.',
      next: `${String(recent)} of ${String(competitors.length)} competitors were scanned in the last 30 days.`,
    },
    {
      process: 'Search term list',
      to: '/terms',
      manage: '/manage/terms',
      latest: `The list holds ${f.overview.terms.total.toLocaleString('en-GB')} search terms.`,
      next: 'Each scan suggests new search terms for review.',
    },
    {
      process: 'Recommendations',
      to: '/recommendations',
      manage: '/manage/recommendations',
      latest: `${String(open)} recommendations are open and ${done === 0 ? 'none is' : `${String(done)} ${done === 1 ? 'is' : 'are'}`} done.`,
      next: 'A scan of tribelt.nl ticks off the ones it can check, and the rest are ticked off by hand.',
    },
  ]
}

/** Where the Overview's figures come from: the newest Prompt run and the newest scan. */
export function overviewDataLine(runStartedAt: string | undefined, readiness: readonly AiReadinessRow[]): string | undefined {
  const scanned = readiness.flatMap((r) => (r.scan?.finishedAt ? [r.scan.finishedAt] : [])).sort((a, b) => Date.parse(a) - Date.parse(b)).at(-1)
  const run = runStartedAt ? `the Prompt run of ${localDate(runStartedAt)}` : ''
  const scans = scanned ? `the scans up to ${localDate(scanned)}` : ''
  if (!run && !scans) return undefined
  return `From ${[run, scans].filter(Boolean).join(' and ')}.`
}
