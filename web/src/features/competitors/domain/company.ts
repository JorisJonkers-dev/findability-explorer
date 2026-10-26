import type { AnalysisSummary, BeltFamily, BeltName, Competitor, Term } from '../../../infrastructure/api/types.gen'
import { FAMILIES, familyLabel, TIERS } from '../../../shared/vocabulary'
import { companyName } from '../../terminology/domain/filters'
import { fold } from '../../terminology/domain/fold'
import { checkScanUrl } from '../../scan/domain/newScan'
import { recentLabel } from '../../scan/domain/recent'
import { isInProgress, progress } from '../../scan/domain/status'
import { shortName } from './matrix'

/** A known company, with the short name the terminology map credits its terms to. */
export type Company = Competitor & { shortName: string }

/** Gives every company its short name, from who the map's terms credit. */
export function withShortNames(competitors: readonly Competitor[], users: readonly string[]): Company[] {
  return competitors.map((c) => ({ ...c, shortName: shortName(c.name, users) }))
}

/** A host as companies are matched by it: lower case, without "www.". */
export function normalizeHost(host: string): string {
  return host.toLowerCase().replace(/^www\./, '')
}

/** The normalized host of an address, or null when it is not one. */
export function hostOf(url: string): string | null {
  try {
    const { hostname } = new URL(url)
    return hostname === '' ? null : normalizeHost(hostname)
  } catch {
    return null
  }
}

/** The company whose website is on host, if any. */
export function companyByHost<T extends Pick<Competitor, 'host'>>(companies: readonly T[], host: string): T | undefined {
  const wanted = normalizeHost(host)
  return companies.find((c) => c.host !== undefined && normalizeHost(c.host) === wanted)
}

// A second-level label that is really part of the country's suffix: example.co.uk, example.com.au.
const SECOND_LEVEL = new Set(['co', 'com', 'org', 'net', 'ac', 'gov', 'edu'])

/** A readable company name from a host: "www.example-belts.co.uk" is "Example Belts". */
export function readableName(host: string): string {
  const labels = normalizeHost(host).split('.')
  if (labels.length > 1) labels.pop()
  if (labels.length > 1 && SECOND_LEVEL.has(String(labels.at(-1)))) labels.pop()
  const name = String(labels.at(-1))
    .split(/[-_]+/)
    .filter((word) => word !== '')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
  return name === '' ? host : name
}

/** The most suggestions the picker shows at once. */
export const MAX_SUGGESTIONS = 8

/**
 * The companies worth suggesting for what is typed in the address field: those with a website
 * whose name, host or website contains the text, folded, ignoring a typed scheme and "www.".
 */
export function suggestions(companies: readonly Company[], text: string): Company[] {
  const needle = fold(text.trim().replace(/^https?:\/\//i, '').replace(/^www\./i, ''))
  if (needle === '') return []
  return companies
    .filter((c) => c.websiteUrl !== undefined)
    .filter((c) => [c.name, c.shortName, c.host ?? '', c.website].some((field) => fold(field).includes(needle)))
    .slice(0, MAX_SUGGESTIONS)
}

/** What the address typed so far means: nothing yet, a known company, or a company to add. */
export type PickerState =
  | { kind: 'none' }
  | { kind: 'known'; company: Company }
  | { kind: 'new'; host: string; name: string }

/** Reads a typed address: once it is a whole address, it is a known company's or a new one. */
export function pickerState(text: string, companies: readonly Company[]): PickerState {
  const check = checkScanUrl(text)
  if (check.kind !== 'ok') return { kind: 'none' }
  const host = String(hostOf(check.url))
  const company = companyByHost(companies, host)
  return company ? { kind: 'known', company } : { kind: 'new', host, name: readableName(host) }
}

/** How often a company was scanned: "Scanned 3×", or "Not scanned yet". */
export function scanCountLabel(scans: number, scanning = false): string {
  if (scanning) {
    if (scans <= 1) return 'First scan running'
    return `${scanCountLabel(scans - 1)}, one more running`
  }
  if (scans === 0) return 'Never scanned'
  return scans === 1 ? 'Scanned once' : `Scanned ${String(scans)} times`
}

/** A link to one of a company's scans: "3 Oct: 15 pages read", or "Reading page 13 of 60". */
export function scanLinkLabel(scan: Pick<AnalysisSummary, 'status' | 'pagesCrawled' | 'maxPages' | 'createdAt' | 'blocked'>, now: Date = new Date()): string {
  if (isInProgress(scan.status)) return progress(scan).label
  return `${recentLabel(scan, now)}: ${scanOutcome(scan)}`
}

const LONG_DATE = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric' })
const CLOCK = new Intl.DateTimeFormat('en-GB', { hour: '2-digit', minute: '2-digit', hour12: false })

function startOfDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime()
}

/** "today", "yesterday" or "3 October 2026", in the viewer's time zone. */
function dayLabel(date: Date, now: Date): string {
  const days = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000)
  if (days === 0) return 'today'
  if (days === 1) return 'yesterday'
  return LONG_DATE.format(date)
}

/** When the last scan started, to the minute, and how it went; null without a scan. */
export function lastScanLabel(
  scan: Pick<AnalysisSummary, 'status' | 'pagesCrawled' | 'maxPages' | 'createdAt' | 'blocked'> | undefined,
  now: Date = new Date(),
): string | null {
  if (!scan) return null
  const at = new Date(scan.createdAt)
  return `Last scan: ${dayLabel(at, now)} at ${CLOCK.format(at)}, ${scanState(scan)}`
}

/** How a scan went, or where it is: "60 pages read", "failed", "reading page 4 of 15". */
function scanState(scan: Pick<AnalysisSummary, 'status' | 'pagesCrawled' | 'maxPages' | 'blocked'>): string {
  return isInProgress(scan.status) ? lowerFirst(progress(scan).label) : scanOutcome(scan).toLowerCase()
}

function lowerFirst(s: string): string {
  return s.charAt(0).toLowerCase() + s.slice(1)
}

/** Whether a scan of the company is waiting or reading pages now. */
export function isScanning(company: Pick<Competitor, 'lastScan'>): boolean {
  return company.lastScan !== undefined && isInProgress(company.lastScan.status)
}

/** A company's scans in a few words for its row in the list: "2 scans · today", "2 scans · reading page 4 of 15". */
export function listScanLabel(company: Pick<Competitor, 'scans' | 'lastScan'>, now: Date = new Date()): string {
  if (company.scans === 0) return 'No scans'
  const count = company.scans === 1 ? '1 scan' : `${String(company.scans)} scans`
  if (!company.lastScan) return count
  return `${count} · ${isInProgress(company.lastScan.status) ? scanState(company.lastScan) : recentLabel(company.lastScan, now)}`
}

const DAY = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'long', year: 'numeric' })

/** Where a company comes from: "Researched", or "Added 22 October 2026". */
export function sourceLabel(company: Pick<Competitor, 'source' | 'addedAt'>): string {
  if (company.source === 'research') return 'Researched'
  return company.addedAt ? `Added ${DAY.format(new Date(company.addedAt))}` : 'Added'
}

export type BeltNameGroup = { family: BeltFamily; label: string; names: string[] }

/** A company's belt names, one group per family, in the families' order. */
export function groupBeltNames(beltNames: readonly BeltName[]): BeltNameGroup[] {
  return FAMILIES.map((family) => ({
    family,
    label: familyLabel(family),
    names: beltNames.filter((b) => b.family === family).map((b) => b.name),
  })).filter((group) => group.names.length > 0)
}

/** Who a scan is of: its company, by the id the server matched or else by host, and its name. */
export function scanCompany(
  scan: { url: string; competitorId?: string },
  companies: readonly Company[],
): { name: string; company: Company | undefined } {
  const host = hostOf(scan.url)
  const company =
    companies.find((c) => c.id === scan.competitorId) ?? (host === null ? undefined : companyByHost(companies, host))
  return { name: company ? company.shortName : (host ?? scan.url), company }
}

/**
 * The terminology map's terms a company uses, the most specific tier first: those whose used-by
 * list names the company, with or without a product line in brackets.
 */
export function companyTerms<T extends Pick<Term, 'term' | 'tier' | 'usedBy'>>(terms: readonly T[], shortName: string): T[] {
  return terms
    .filter((t) => t.usedBy.some((u) => companyName(u) === shortName))
    .sort((a, b) => TIERS.indexOf(a.tier) - TIERS.indexOf(b.tier) || a.term.localeCompare(b.term))
}

/** How a company's scan ended, for its list of scans: "12 pages read", "Scanning", "Blocked by its site", "Failed". */
export function scanOutcome(scan: Pick<AnalysisSummary, 'status' | 'pagesCrawled' | 'blocked'>): string {
  if (isInProgress(scan.status)) return 'Scanning'
  if (scan.blocked === true) return 'Blocked by its site'
  if (scan.status === 'failed') return 'Failed'
  return `${String(scan.pagesCrawled)} ${scan.pagesCrawled === 1 ? 'page' : 'pages'} read`
}
