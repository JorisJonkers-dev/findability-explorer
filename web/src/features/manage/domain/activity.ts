import type { AnalysisSummary, PromptRun, ReadinessCheckState } from '../../../infrastructure/api/types.gen'

/** How a piece of work ended, in the words of the spec. */
export type Ended = 'Done' | 'Done, with failed responses' | 'Stopped by hand' | 'Stopped by the budget' | 'Failed'

/** The tone of an ending's pill. */
export type Tone = 'good' | 'warn' | 'bad'

/** The start of the failure text of a run an administrator stopped (the backend's StoppedByHand). */
const STOPPED_BY_HAND = 'Stopped by hand'

const NUMBER = new Intl.NumberFormat('en-GB')
const WHEN = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone: 'UTC' })

/** A moment in words: "28 October 2026, 09:37" (UTC). */
export function whenLabel(iso: string): string {
  return WHEN.format(new Date(iso)).replace(' at ', ', ')
}

/** A site by its host, without www: "tribelt.nl". */
export function hostOf(url: string): string {
  try {
    return new URL(url).host.replace(/^www\./, '')
  } catch {
    return url
  }
}

/** How a Prompt run ended, or null while it runs. A stopped run says in its failure why it stopped. */
export function endedOf(run: Pick<PromptRun, 'status' | 'failure' | 'failed'>): Ended | null {
  switch (run.status) {
    case 'running':
      return null
    case 'done':
      return run.failed > 0 ? 'Done, with failed responses' : 'Done'
    case 'stopped':
      return run.failure?.startsWith(STOPPED_BY_HAND) ? 'Stopped by hand' : 'Stopped by the budget'
    case 'failed':
      return 'Failed'
  }
}

/** How a scan ended, or null while it waits or runs. */
export function scanEnded(scan: Pick<AnalysisSummary, 'status' | 'stoppedByHand'>): Ended | null {
  switch (scan.status) {
    case 'queued':
    case 'running':
      return null
    case 'done':
      return 'Done'
    case 'failed':
      return scan.stoppedByHand ? 'Stopped by hand' : 'Failed'
  }
}

/** The pill's tone for an ending. */
export function endedTone(ended: Ended): Tone {
  switch (ended) {
    case 'Done':
      return 'good'
    case 'Failed':
      return 'bad'
    case 'Done, with failed responses':
    case 'Stopped by hand':
    case 'Stopped by the budget':
      return 'warn'
  }
}

/** One line of Recent work. */
export type WorkLine = {
  id: string
  /** When it ended, or started when that is all there is. */
  at: string
  work: string
  ended: Ended
  did: string
  /** What it cost; absent when the API does not say. */
  costUsd?: number
  to: string
}

const RUNS_PAGE = '/manage/prompt-test/runs'

/** How many pieces of work Recent work lists. */
export const RECENT_LIMIT = 20

/** The last pieces of work that ended, newest first: Prompt runs and scans. */
export function recentWork(runs: readonly PromptRun[], scans: readonly AnalysisSummary[], limit = RECENT_LIMIT): WorkLine[] {
  const lines: WorkLine[] = []
  for (const r of runs) {
    const ended = endedOf(r)
    if (ended === null) continue
    lines.push({
      id: r.id,
      at: r.finishedAt ?? r.startedAt,
      work: 'Prompt run',
      ended,
      did: `${NUMBER.format(r.answered)} of ${NUMBER.format(r.tasks)} responses, Prompt set version ${String(r.promptSetVersion)}`,
      costUsd: r.costUsd,
      to: RUNS_PAGE,
    })
  }
  for (const s of scans) {
    const ended = scanEnded(s)
    if (ended === null) continue
    lines.push({
      id: s.id,
      at: s.finishedAt ?? s.createdAt,
      work: `Scan of ${hostOf(s.url)}`,
      ended,
      did: `${NUMBER.format(s.pagesCrawled)} ${s.pagesCrawled === 1 ? 'page' : 'pages'} read`,
      to: `/manage/scans/${s.id}`,
    })
  }
  return lines.sort((a, b) => (a.at < b.at ? 1 : a.at > b.at ? -1 : 0)).slice(0, limit)
}

/** A piece of work that runs now, with its progress and its own Stop. */
export type RunningWork = {
  kind: 'promptRun' | 'scan' | 'check'
  id: string
  label: string
  done: number
  total: number
  unit: string
  costSoFarUsd?: number
  costToGoUsd?: number
  startedAt?: string
  to: string
  stopLabel: string
  /** What stopping keeps, said before the second click. */
  keeps: string
}

/**
 * Everything that runs now: Prompt runs, scans that read pages (not the ones waiting), and the AI
 * readiness check. runEstimateUsd, what a run costs, gives a run its cost to go.
 */
export function runningWork(
  runs: readonly PromptRun[],
  scans: readonly AnalysisSummary[],
  check: ReadinessCheckState | undefined,
  runEstimateUsd?: number,
): RunningWork[] {
  const out: RunningWork[] = []
  for (const r of runs.filter((x) => x.status === 'running')) {
    out.push({
      kind: 'promptRun',
      id: r.id,
      label: 'Prompt run',
      done: r.answered + r.failed,
      total: r.tasks,
      unit: 'responses',
      costSoFarUsd: r.costUsd,
      ...(runEstimateUsd === undefined ? {} : { costToGoUsd: Math.max(0, Math.round((runEstimateUsd - r.costUsd) * 100) / 100) }),
      startedAt: r.startedAt,
      to: RUNS_PAGE,
      stopLabel: 'Stop the Prompt run',
      keeps: 'Stopping keeps the responses so far. You can finish the run later with a retry.',
    })
  }
  for (const s of scans.filter((x) => x.status === 'running')) {
    out.push({
      kind: 'scan',
      id: s.id,
      label: `Scan of ${hostOf(s.url)}`,
      done: s.pagesCrawled,
      total: s.maxPages,
      unit: 'pages',
      startedAt: s.createdAt,
      to: `/manage/scans/${s.id}`,
      stopLabel: 'Stop the scan',
      keeps: 'Stopping keeps the pages read so far.',
    })
  }
  if (check?.running) {
    out.push({
      kind: 'check',
      id: 'check',
      label: 'AI readiness check',
      done: check.checked,
      total: check.total,
      unit: 'sites',
      to: '/ai-readiness',
      stopLabel: 'Stop the check',
      keeps: 'Stopping keeps what the check recorded for the sites it read. The others keep what their scan recorded.',
    })
  }
  return out
}
