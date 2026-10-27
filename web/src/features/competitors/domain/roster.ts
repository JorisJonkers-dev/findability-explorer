import type { AiReadiness, Competitor, PromptTestResults } from '../../../infrastructure/api/types.gen'
import { latestRun, tally } from '../../prompt-test/domain/results'

/** One readiness check as a chip: a tick, a cross or "Some" before its name, and what it says. */
export type Chip = { label: string; state: 'yes' | 'partly' | 'no'; mark: string; tip: string }

function chip(label: string, state: Chip['state'], yes: string, no: string, partly = ''): Chip {
  return { label, state, mark: state === 'yes' ? '✓' : state === 'partly' ? 'Some' : '✕', tip: state === 'yes' ? yes : state === 'partly' ? partly : no }
}

/** A site's five readiness chips: crawlers, sitemap, llms.txt, llms-full.txt and Markdown; none before a scan read it. */
export function chipsOf(r: AiReadiness | undefined): Chip[] {
  if (!r) return []
  const unreadable = r.robots === 'unreachable' || r.bots.length === 0
  const blocked = r.bots.filter((b) => b.access === 'blocked').length
  const crawlers: Chip['state'] = unreadable || blocked === r.bots.length ? 'no' : blocked > 0 ? 'partly' : 'yes'
  return [
    chip('crawlers', crawlers, 'Lets every AI crawler in', unreadable ? 'robots.txt could not be read' : 'Keeps every AI crawler out', 'Keeps some AI crawlers out'),
    chip('sitemap', r.sitemap.files > 0 ? 'yes' : 'no', 'Has a sitemap', 'No sitemap'),
    chip('llms.txt', r.llmsTxt.present ? 'yes' : 'no', 'Has llms.txt', 'No llms.txt'),
    chip('llms-full', r.llmsFullTxt ? 'yes' : 'no', 'Has llms-full.txt', 'No llms-full.txt'),
    chip('Markdown', r.markdown ? 'yes' : 'no', 'Has Markdown pages', 'No Markdown pages'),
  ]
}

/** A company's share of the answers of the newest run with answers, or undefined before any. */
export function shareOf(results: Pick<PromptTestResults, 'runs' | 'mentions'> | undefined, id: string): number | undefined {
  const run = latestRun(results)
  const answers = run ? tally(run.cells).answers : 0
  if (!results || answers === 0) return undefined
  return (results.mentions.find((m) => m.competitorId === id)?.answers ?? 0) / answers
}

/** What the list can be narrowed to. */
export type RosterFilter = 'all' | 'competitor' | 'ours' | 'never'

export const ROSTER_FILTERS: readonly { value: RosterFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'competitor', label: 'Competitors' },
  { value: 'ours', label: 'Tribelt and group' },
  { value: 'never', label: 'Never scanned' },
]

export function parseRosterFilter(raw: unknown): RosterFilter {
  return ROSTER_FILTERS.find((f) => f.value === raw)?.value ?? 'all'
}

/** Whether a company stays in the list under a filter. */
export function keeps(c: Pick<Competitor, 'role' | 'scans'>, filter: RosterFilter): boolean {
  switch (filter) {
    case 'all':
      return true
    case 'competitor':
      return c.role === 'competitor'
    case 'ours':
      return c.role === 'tribelt' || c.role === 'group'
    case 'never':
      return c.scans === 0
  }
}

/** The order of the list: Tribelt, then the competitors the answers name most, then by name, then the group. */
export function rosterOrder<T extends Pick<Competitor, 'role' | 'name'>>(companies: readonly T[], share: (c: T) => number | undefined): T[] {
  const band = (c: T) => (c.role === 'tribelt' ? 0 : c.role === 'competitor' ? 1 : 2)
  // A company no run has named yet sorts below every share.
  const ofShare = (c: T) => share(c) ?? -1
  return [...companies].sort((a, b) => band(a) - band(b) || ofShare(b) - ofShare(a) || a.name.localeCompare(b.name))
}

const REGIONS = new Intl.DisplayNames(['en'], { type: 'region', fallback: 'code' })

/** Where a company is: its country's name, or nothing the research did not note. */
export function placeOf(c: Pick<Competitor, 'country'>): string {
  if (!/^[A-Z]{2}$/.test(c.country)) return c.country
  return String(REGIONS.of(c.country))
}

/** Its role, as the list writes it. */
export function roleLabel(c: Pick<Competitor, 'role'>): string {
  return { tribelt: 'Us', mirror: 'Mirror', group: 'Group company', competitor: 'Competitor' }[c.role]
}

/** When it was last scanned: "24 Oct", "Blocked", "Scanning" or "Never". */
export function lastScanOf(c: Pick<Competitor, 'scans' | 'lastScan'>): { text: string; attention: boolean } {
  const scan = c.lastScan
  if (c.scans === 0 || !scan) return { text: 'Never', attention: true }
  if (scan.status === 'queued' || scan.status === 'running') return { text: 'Scanning', attention: false }
  if (scan.blocked) return { text: 'Blocked', attention: true }
  const at = scan.finishedAt ?? scan.createdAt
  return { text: new Date(at).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', timeZone: 'UTC' }), attention: scan.status === 'failed' }
}
