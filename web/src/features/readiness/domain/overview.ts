import type { AiReadiness, AiReadinessRow, ReadinessCheckState, ReadinessRank } from '../../../infrastructure/api/types.gen'
import { nameList } from '../../ai-search/domain/summary'
import { formatValue } from '../../catching-up/domain/standings'
import { joinWords } from '../../../shared/words'

/**
 * The checks the summary lists, and whether a site passes each. The five counted ones are the
 * checks the ranking and the headline score (the crawlers count half when only some are kept out).
 */
export const CHECKS: readonly { label: string; counted: boolean; passes: (r: AiReadiness) => boolean }[] = [
  { label: 'Lets every AI crawler in', counted: true, passes: (r) => r.robots !== 'unreachable' && r.bots.length > 0 && r.bots.every((b) => b.access !== 'blocked') },
  { label: 'Has a sitemap', counted: true, passes: (r) => r.sitemap.files > 0 },
  { label: 'Has llms.txt', counted: true, passes: (r) => r.llmsTxt.present },
  { label: 'Has llms-full.txt', counted: true, passes: (r) => r.llmsFullTxt },
  { label: 'Has Markdown pages', counted: true, passes: (r) => r.markdown },
  { label: 'Sitemap in Dutch, German and English', counted: false, passes: (r) => r.advice.some((a) => a.check === 'sitemapLanguages' && a.ok) },
  { label: 'Organization structured data', counted: false, passes: (r) => r.advice.some((a) => a.check === 'organization' && a.ok) },
  { label: 'Product structured data', counted: false, passes: (r) => r.advice.some((a) => a.check === 'product' && a.ok) },
]

/** The rows a scan has read: the sites the page counts. */
export function scanned(rows: readonly AiReadinessRow[]): (AiReadinessRow & { readiness: AiReadiness })[] {
  return rows.filter((r): r is AiReadinessRow & { readiness: AiReadiness } => r.readiness !== undefined)
}

/** Per check, how many of the scanned sites pass it, and whether Tribelt does (null when not scanned). */
export function checkSummary(rows: readonly AiReadinessRow[]): { label: string; counted: boolean; passing: number; sites: number; tribelt: boolean | null }[] {
  const sites = scanned(rows)
  const tribelt = sites.find((r) => r.role === 'tribelt')
  return CHECKS.map((c) => ({
    label: c.label,
    counted: c.counted,
    passing: sites.filter((r) => c.passes(r.readiness)).length,
    sites: sites.length,
    tribelt: tribelt ? c.passes(tribelt.readiness) : null,
  }))
}

/** How a mark in the table looks: allowed or present, partly, blocked, absent, or unknown. */
export type MarkTone = 'ok' | 'partly' | 'blocked' | 'no' | 'unknown'

/** A symbol in the table, with the sentence a screen reader and a hover read instead. */
export type Mark = { mark: string; text: string; tone: MarkTone }

/** What each symbol in the crawler columns means, for the legend. */
export const ACCESS_LEGEND: readonly { mark: string; tone: MarkTone; meaning: string }[] = [
  { mark: '✓', tone: 'ok', meaning: 'robots.txt lets the crawler read every page.' },
  { mark: '~', tone: 'partly', meaning: 'robots.txt closes some pages to it, often only admin, login or checkout pages.' },
  { mark: '✕', tone: 'blocked', meaning: 'robots.txt closes the whole site to it.' },
  { mark: '–', tone: 'unknown', meaning: 'The site is not scanned yet, or its robots.txt could not be read.' },
]

/** A crawler's access as a symbol, with a sentence saying what it means for that crawler. */
export function accessMark(r: AiReadiness | undefined, bot: string): Mark {
  if (!r) return { mark: '–', text: 'Not scanned yet.', tone: 'unknown' }
  if (r.robots === 'unreachable') return { mark: '–', text: 'robots.txt could not be read.', tone: 'unknown' }
  const rule = r.bots.find((b) => b.bot === bot)
  if (!rule) return { mark: '–', text: `${bot} was not checked.`, tone: 'unknown' }
  if (rule.access === 'blocked') return { mark: '✕', text: `${bot} may not read any page.`, tone: 'blocked' }
  if (rule.access === 'partly') return { mark: '~', text: `${bot} may read the site, except some pages robots.txt closes.`, tone: 'partly' }
  return { mark: '✓', text: `${bot} may read every page.`, tone: 'ok' }
}

/** Whether a site has a file or a kind of page, as the same ticks and crosses the crawler columns use. */
export function hasMark(present: boolean, what: string): Mark {
  return present ? { mark: '✓', text: `Has ${what}.`, tone: 'ok' } : { mark: '✕', text: `No ${what}.`, tone: 'no' }
}

/** The five checks the headline scores, as one sentence. */
export function fiveChecksLine(): string {
  const checks = CHECKS.filter((c) => c.counted).map((c) => c.label.charAt(0).toLowerCase() + c.label.slice(1))
  return `The five checks ask whether a site ${joinWords(checks)}.`
}

/** How many sites the page counts, when they were scanned and when they were checked again, as sentences. */
export function scanNote(rows: readonly AiReadinessRow[]): string {
  const sites = scanned(rows).length
  return [sites === 1 ? '1 site' : `${String(sites)} sites`, scanRange(rows), checkedAgain(rows)]
    .filter((s): s is string => s !== null)
    .map((s) => `${s}.`)
    .join(' ')
}

/** The first and last day the scans shown were made, or null before any scan. */
export function scanRange(rows: readonly AiReadinessRow[]): string | null {
  const days = rows
    .flatMap((r) => (r.scan?.finishedAt ? [r.scan.finishedAt] : []))
    .map((iso) => iso.slice(0, 10))
    .sort()
  const first = days[0]
  const last = days.at(-1)
  if (first === undefined || last === undefined) return null
  const day = (d: string) => new Date(`${d}T00:00:00Z`).toLocaleDateString('en-GB', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })
  return first === last ? `Scanned on ${day(first)}` : `Last scans from ${day(first)} to ${day(last)}`
}

/** The newest day an administrator checked a site's AI readiness again, as a line; null when none was. */
export function checkedAgain(rows: readonly AiReadinessRow[]): string | null {
  const last = rows
    .flatMap((r) => (r.readiness?.checkedAt ? [r.readiness.checkedAt] : []))
    .sort()
    .at(-1)
  if (last === undefined) return null
  const day = new Date(last).toLocaleDateString('en-GB', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })
  return `AI readiness checked again on ${day}`
}

/** How far checking every site's AI readiness again has come, or how it ended; empty when none ran. */
export function readinessCheckLine(check: ReadinessCheckState | undefined): string {
  if (!check || check.total === 0) return ''
  const sites = (n: number) => (n === 1 ? '1 site' : `${String(n)} sites`)
  if (check.running) return `Checking ${String(check.checked)} of ${sites(check.total)} again.`
  const read = check.total - check.failed
  const done = `Checked ${sites(read)} again.`
  if (check.failed === 0) return done
  return check.failed === 1
    ? `${done} 1 site could not be read and keeps its previous result.`
    : `${done} ${String(check.failed)} sites could not be read and keep their previous result.`
}

/**
 * The page's headline, from the readiness ranking the AI search summary serves: who leads on the
 * five checks, the first two companies tied for the lead and how many more, and where Tribelt
 * stands. Tribelt tied with the best competitors shares the lead.
 */
export function readinessHeadline(ranking: readonly ReadinessRank[]): string {
  const competitors = ranking.filter((r) => r.role === 'competitor')
  const tribelt = ranking.find((r) => r.role === 'tribelt')
  if (competitors.length === 0) return 'Who lets search engines and AI assistants in.'
  const top = Math.max(...competitors.map((r) => r.score))
  const leaders = nameList(competitors.filter((r) => r.score === top).map((r) => r.name), 2)
  const checks = (score: number) => formatValue('readinessChecks', score)
  const many = competitors.filter((r) => r.score === top).length > 1
  if (tribelt && tribelt.score > top) return `Tribelt leads with ${checks(tribelt.score)} of five checks. Next is ${leaders} with ${checks(top)}.`
  if (tribelt && tribelt.score === top) return `Tribelt shares the lead with ${leaders}, at ${checks(top)} of five checks.`
  const lead = `${leaders} ${many ? 'lead' : 'leads'} with ${checks(top)} of five checks.`
  return tribelt ? `${lead} Tribelt passes ${checks(tribelt.score)}.` : lead
}
