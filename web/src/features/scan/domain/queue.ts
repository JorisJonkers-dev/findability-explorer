import type { AnalysisSummary, Competitor, LlmBudget, RescanRequest } from '../../../infrastructure/api/types.gen'
import { isInProgress } from './status'

/** The scans in progress: those reading pages now, and those waiting for a free place, oldest first. */
export function scanQueue<T extends Pick<AnalysisSummary, 'status' | 'createdAt'>>(scans: readonly T[]): { running: T[]; waiting: T[] } {
  const oldestFirst = scans.filter((s) => isInProgress(s.status)).sort((a, b) => a.createdAt.localeCompare(b.createdAt))
  return { running: oldestFirst.filter((s) => s.status === 'running'), waiting: oldestFirst.filter((s) => s.status === 'queued') }
}

type Scanned<T> = { company: T; scan: AnalysisSummary }

/** Each company's newest scan, newest first: archived companies too, since their scans stay. */
export function newestScans<T extends Pick<Competitor, 'lastScan'>>(companies: readonly T[]): Scanned<T>[] {
  return companies
    .flatMap((company) => (company.lastScan ? [{ company, scan: company.lastScan }] : []))
    .sort((a, b) => b.scan.createdAt.localeCompare(a.scan.createdAt))
}

/** The company scanned last, with its scan, or undefined before the first scan. */
export function lastScanned<T extends Pick<Competitor, 'lastScan'>>(companies: readonly T[]): Scanned<T> | undefined {
  return newestScans(companies)[0]
}

/** The ways to choose the competitors to rescan. */
export type RescanGroup = 'all' | 'stale' | 'never'

/** A scan older than this many days is due again. */
export const STALE_DAYS = 30

/**
 * The competitors to rescan, by group: every competitor on the list with a website, those whose
 * newest scan is over 30 days old or that were never scanned, and those never scanned.
 */
export function rescanGroups<T extends Pick<Competitor, 'role' | 'archived' | 'websiteUrl' | 'lastScan'>>(companies: readonly T[], now: Date): Record<RescanGroup, T[]> {
  const all = companies.filter((c) => c.role === 'competitor' && !c.archived && c.websiteUrl !== undefined)
  const cut = now.getTime() - STALE_DAYS * 86_400_000
  return {
    all,
    stale: all.filter((c) => !c.lastScan || new Date(c.lastScan.createdAt).getTime() < cut),
    never: all.filter((c) => !c.lastScan),
  }
}

/** The rescan of a group, naming its companies so the scans queued are the ones counted. */
export function rescanRequest(companies: readonly Pick<Competitor, 'id'>[], maxPages: number): RescanRequest {
  return { competitorIds: companies.map((c) => c.id), maxPages }
}

const STAMP = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone: 'UTC' })

/** When a scan started, to the minute: "28 October 2026, 05:48". */
export function scanStamp(iso: string): string {
  return STAMP.format(new Date(iso)).replace(' at ', ', ')
}

const DAY = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })

/** The day a scan started: "28 October 2026". */
export function scanDay(iso: string): string {
  return DAY.format(new Date(iso))
}

/** The pages a scan read of the most it would: "22 of 60". */
export function pagesRead(scan: Pick<AnalysisSummary, 'pagesCrawled' | 'maxPages'>): string {
  return `${String(scan.pagesCrawled)} of ${String(scan.maxPages)}`
}

/** How a scan an administrator stopped ended, or null when nobody stopped it. */
export function stoppedLine(scan: Pick<AnalysisSummary, 'stoppedByHand' | 'pagesCrawled'>): string | null {
  if (scan.stoppedByHand !== true) return null
  if (scan.pagesCrawled === 0) return 'Taken off the queue by hand before it read a page.'
  return `Stopped by hand. It keeps the ${String(scan.pagesCrawled)} ${scan.pagesCrawled === 1 ? 'page' : 'pages'} it read.`
}

const NUMBER = new Intl.NumberFormat('en-GB')

/**
 * What scans cost, from the average tokens one scan's product matching used: "About 58,274
 * tokens for 2 scans. 9,951,789 of this month's tokens are left." Reading pages costs nothing.
 */
export function scanEstimate(budget: Pick<LlmBudget, 'averageTokensPerScan' | 'monthlyTokens' | 'usedThisMonth'> | undefined, scans: number): string {
  if (budget === undefined) return ''
  const left = `${NUMBER.format(Math.max(budget.monthlyTokens - budget.usedThisMonth, 0))} of this month's tokens are left.`
  if (budget.averageTokensPerScan === 0) return `No scan has matched products yet, so there is no estimate. ${left}`
  const what = scans === 1 ? '' : ` for ${NUMBER.format(scans)} scans`
  return `About ${NUMBER.format(budget.averageTokensPerScan * scans)} tokens${what}. ${left}`
}
