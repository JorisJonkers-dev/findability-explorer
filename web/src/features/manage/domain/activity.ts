import type { AnalysisSummary, LlmCredential, PromptRun, SpendLine, Work, WorkKind } from '../../../infrastructure/api/types.gen'
import type { Ending } from '../../../shared/endings'

/** The start of the failure text of a run an administrator stopped (the backend's StoppedByHand). */
const STOPPED_BY_HAND = 'Stopped by hand'

const NUMBER = new Intl.NumberFormat('en-GB')
const WHEN = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone: 'UTC' })
const DAY = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })
const TIME = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit', timeZone: 'UTC' })

/** A moment in words: "28 October 2026, 09:37" (UTC). */
export function whenLabel(iso: string): string {
  return WHEN.format(new Date(iso)).replace(' at ', ', ')
}

/** A day in words: "27 October 2026" (UTC). */
export function dayLabel(iso: string): string {
  return DAY.format(new Date(iso))
}

/** A time of day: "06:00" (UTC). */
export function timeLabel(iso: string): string {
  return TIME.format(new Date(iso))
}

/** How a Prompt run ended, as the API names endings, or null while it runs. A stopped run says in its failure why it stopped. */
export function endedOf(run: Pick<PromptRun, 'status' | 'failure' | 'failed'>): Ending | null {
  switch (run.status) {
    case 'running':
      return null
    case 'done':
      return run.failed > 0 ? 'doneWithFailures' : 'done'
    case 'stopped':
      return run.failure?.startsWith(STOPPED_BY_HAND) ? 'stoppedByHand' : 'stoppedByBudget'
    case 'failed':
      return 'failed'
  }
}

/** How a running piece of work stops, and what stopping keeps, said before the second click. */
export type Stop = { label: string; keeps: string }

/** What a kind of work is called, what its parts are, where it is managed, and how it stops. */
type KindInfo = { label: string; unit: string; to: (id: string) => string; stop: Stop | null }

const KINDS: Record<WorkKind, KindInfo> = {
  promptRun: {
    label: 'Prompt run',
    unit: 'responses',
    to: () => '/manage/prompt-test/runs',
    stop: { label: 'Stop the Prompt run', keeps: 'Stopping keeps the responses so far. You can finish the run later with a retry.' },
  },
  scan: { label: 'Scan', unit: 'pages', to: (id) => `/manage/scans/${id}`, stop: { label: 'Stop the scan', keeps: 'Stopping keeps the pages read so far.' } },
  rescan: {
    label: 'Rescan of competitors',
    unit: 'scans',
    to: () => '/manage/scans',
    stop: { label: 'Stop the rescan', keeps: 'Stopping keeps the scans done so far, and takes the waiting ones off the queue.' },
  },
  readinessCheck: {
    label: 'AI readiness check',
    unit: 'sites',
    to: () => '/ai-readiness',
    stop: { label: 'Stop the check', keeps: 'Stopping keeps what the check recorded for the sites it read. The others keep what their scan recorded.' },
  },
  termReview: {
    label: 'Term review',
    unit: 'search terms',
    to: () => '/manage/terms',
    stop: { label: 'Stop the term review', keeps: 'Stopping keeps what the review decided so far.' },
  },
  candidateReview: {
    label: 'Review of the competitor candidates',
    unit: 'companies',
    to: () => '/manage/competitors/candidates',
    stop: { label: 'Stop the review', keeps: 'Stopping keeps what the review decided so far.' },
  },
  competitorCheck: {
    label: 'Check of the competitor list',
    unit: 'companies',
    to: () => '/manage/competitors',
    stop: { label: 'Stop the check', keeps: 'Stopping keeps what the check decided so far.' },
  },
  recommendationRun: { label: 'Recommendation run', unit: 'steps', to: () => '/manage/recommendations', stop: null },
}

/** What a piece of work did: "200 of 200 responses, 3 failed", or "12 search terms" when the total is not known. */
export function didLabel(work: Pick<Work, 'kind' | 'done' | 'total' | 'failed'>): string {
  const unit = KINDS[work.kind].unit
  const count = work.total > 0 ? `${NUMBER.format(work.done)} of ${NUMBER.format(work.total)} ${unit}` : `${NUMBER.format(work.done)} ${unit}`
  return work.failed > 0 ? `${count}, ${NUMBER.format(work.failed)} failed` : count
}

/** What a piece of work cost on both keys together. */
export function costOf(work: Pick<Work, 'openRouterCostUsd' | 'anthropicCostUsd'>): number {
  return Math.round((work.openRouterCostUsd + work.anthropicCostUsd) * 10000) / 10000
}

/** One line of Recent work. */
export type WorkLine = { id: string; at: string; work: string; ended: Ending; did: string; costUsd: number; to: string }

/** The work that ended, as Recent work lists it; work without an ending is left out. */
export function recentWork(recent: readonly Work[]): WorkLine[] {
  return recent.flatMap((w) => {
    const ended = w.ending
    if (ended === undefined) return []
    const kind = KINDS[w.kind]
    return [{ id: w.id, at: w.finishedAt ?? w.startedAt, work: kind.label, ended, did: didLabel(w), costUsd: costOf(w), to: kind.to(w.id) }]
  })
}

/** A piece of work that runs now, with its progress and, when it has one, its own Stop. */
export type RunningWork = {
  kind: WorkKind
  id: string
  label: string
  done: number
  total: number
  unit: string
  costSoFarUsd: number
  costToGoUsd?: number
  startedAt: string
  to: string
  /** How it stops; null when it cannot be stopped. */
  stop: Stop | null
}

/** The work that runs now. runEstimateUsd, what a Prompt run costs, gives a Prompt run its cost to go. */
export function runningWork(running: readonly Work[], runEstimateUsd?: number): RunningWork[] {
  return running.map((w) => {
    const kind = KINDS[w.kind]
    const soFar = costOf(w)
    return {
      kind: w.kind,
      id: w.id,
      label: kind.label,
      done: w.done,
      total: w.total,
      unit: kind.unit,
      costSoFarUsd: soFar,
      ...(w.kind === 'promptRun' && runEstimateUsd !== undefined ? { costToGoUsd: Math.max(0, Math.round((runEstimateUsd - soFar) * 100) / 100) } : {}),
      startedAt: w.startedAt,
      to: kind.to(w.id),
      stop: kind.stop,
    }
  })
}

/** A Prompt run as a piece of work, for the Prompt test's pages, which read the runs themselves. */
export function runAsWork(run: PromptRun): Work {
  const ended = endedOf(run)
  return {
    id: run.id,
    kind: 'promptRun',
    startedAt: run.startedAt,
    ...(run.finishedAt ? { finishedAt: run.finishedAt } : {}),
    ...(ended === null ? {} : { ending: ended }),
    done: run.answered + run.failed,
    total: run.tasks,
    failed: run.failed,
    openRouterCostUsd: run.costUsd,
    anthropicCostUsd: 0,
  }
}

const SPEND_KINDS: Record<SpendLine['kind'], string> = {
  promptRuns: 'Prompt runs',
  scans: 'Scans',
  readingResponses: 'Reading responses',
  termReview: 'Term review',
  candidateChecks: 'Candidate checks',
  recommendationRuns: 'Recommendation runs',
}
const KEYS: Record<SpendLine['key'], string> = { openRouter: 'OpenRouter', anthropic: 'Anthropic' }

/** This month's spending as rows, with the total of all work. */
export function spendingRows(lines: readonly SpendLine[]): { rows: { kind: string; paidFrom: string; usd: number }[]; totalUsd: number } {
  return {
    rows: lines.map((l) => ({ kind: SPEND_KINDS[l.kind], paidFrom: KEYS[l.key], usd: l.costUsd })),
    totalUsd: Math.round(lines.reduce((sum, l) => sum + l.costUsd, 0) * 10000) / 10000,
  }
}

/** Whether a run that ended still lacks responses, so a retry would finish it. */
export function needsFinishing(run: Pick<PromptRun, 'status' | 'answered' | 'tasks'>): boolean {
  return run.status !== 'running' && run.answered < run.tasks
}

/** The newest run that has ended. */
export function lastEnded<T extends Pick<PromptRun, 'status'>>(runs: readonly T[]): T | undefined {
  return runs.find((r) => r.status !== 'running')
}

/** How many scans wait in the queue. */
export function queuedScans(scans: readonly Pick<AnalysisSummary, 'status'>[]): number {
  return scans.filter((s) => s.status === 'queued').length
}

/** The keys that are missing: no readable OpenRouter key stops Prompt runs, no Anthropic key the reviews. */
export function missingKeys(credentials: readonly Pick<LlmCredential, 'provider' | 'readable'>[]): ('openrouter' | 'anthropic')[] {
  const has = (provider: 'openrouter' | 'anthropic') => credentials.some((c) => c.provider === provider && c.readable)
  return (['openrouter', 'anthropic'] as const).filter((p) => !has(p))
}

/** "1 search term waits" or "6,072 search terms wait". */
export function waitLine(n: number, one: string, many: string): string {
  return n === 1 ? `1 ${one} waits` : `${NUMBER.format(n)} ${many} wait`
}
