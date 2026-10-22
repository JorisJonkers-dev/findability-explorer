import type { AnalysisStatus, AnalysisSummary, MatchingStatus } from '../../../infrastructure/api/types.gen'

/** How often a scan in progress is asked for its progress. */
export const POLL_INTERVAL_MS = 2000

/** Whether the scan can still change: it is waiting to start or reading pages. */
export function isInProgress(status: AnalysisStatus): boolean {
  return status === 'queued' || status === 'running'
}

/** How long to wait before asking again, or false once the scan is done or failed. */
export function pollInterval(status: AnalysisStatus | undefined): number | false {
  return status !== undefined && isInProgress(status) ? POLL_INTERVAL_MS : false
}

export type Progress = {
  /** What the scan is doing: "Reading page 4 of 15". */
  label: string
  /** How far it is, from 0 to 1, or null while it waits to start. */
  fraction: number | null
}

/** A scan's progress, for its progress bar. */
export function progress(summary: Pick<AnalysisSummary, 'status' | 'pagesCrawled' | 'maxPages'>): Progress {
  if (summary.status === 'queued') return { label: 'Waiting to start', fraction: null }
  const fraction = Math.min(summary.pagesCrawled / summary.maxPages, 1)
  const next = Math.min(summary.pagesCrawled + 1, summary.maxPages)
  if (summary.status === 'running') {
    return { label: `Reading page ${String(next)} of ${String(summary.maxPages)}`, fraction }
  }
  return { label: `Read ${String(summary.pagesCrawled)} of ${String(summary.maxPages)} pages`, fraction }
}

/**
 * What became of the product matching, in plain words, or null once it is done and the table
 * speaks for itself.
 */
export function matchingText(matching: MatchingStatus): string | null {
  switch (matching) {
    case 'pending':
      return 'Matching starts when the pages are read.'
    case 'disabled':
      return 'Product matching is off: no language model is configured on this server.'
    case 'skipped':
      return "Skipped: this month's language-model budget is used up, or the pages had no text."
    case 'failed':
      return 'The language model did not answer. The rest of the scan is complete.'
    case 'done':
      return null
  }
}
