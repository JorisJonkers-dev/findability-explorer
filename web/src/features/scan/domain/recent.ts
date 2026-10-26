import type { AnalysisSummary } from '../../../infrastructure/api/types.gen'
import { isInProgress } from './status'

const SHORT_DATE = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short' })

function startOfDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
}

/** When a recent scan was made, as the sidebar shows it: "scanning", "blocked", "failed", "today", "3 Oct". */
export function recentLabel(scan: Pick<AnalysisSummary, 'status' | 'createdAt' | 'blocked'>, now: Date = new Date()): string {
  if (isInProgress(scan.status)) return 'scanning'
  if (scan.blocked === true) return 'blocked'
  if (scan.status === 'failed') return 'failed'
  const days = Math.round((startOfDay(now) - startOfDay(new Date(scan.createdAt))) / 86_400_000)
  if (days <= 0) return 'today'
  if (days === 1) return 'yesterday'
  return SHORT_DATE.format(new Date(scan.createdAt))
}
