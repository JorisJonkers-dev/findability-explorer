import type { AiReadiness, AiReadinessRow } from '../../../infrastructure/api/types.gen'
import { formatValue } from '../../catching-up/domain/standings'

/**
 * The checks a site passes out of five, as the standings count them (internal/insight/domain): its
 * crawlers let in, half when some are kept out, a sitemap, llms.txt, llms-full.txt and Markdown pages.
 */
export function readinessScore(r: AiReadiness): number {
  let crawlers = 0
  if (r.robots !== 'unreachable' && r.bots.length > 0) {
    const blocked = r.bots.filter((b) => b.access === 'blocked').length
    crawlers = blocked === 0 ? 1 : blocked < r.bots.length ? 0.5 : 0
  }
  return crawlers + [r.sitemap.files > 0, r.llmsTxt.present, r.llmsFullTxt, r.markdown].filter(Boolean).length
}

/** The checks the summary counts, and whether a site passes each. */
export const CHECKS: readonly { label: string; passes: (r: AiReadiness) => boolean }[] = [
  { label: 'Lets every AI crawler in', passes: (r) => r.robots !== 'unreachable' && r.bots.length > 0 && r.bots.every((b) => b.access !== 'blocked') },
  { label: 'Has a sitemap', passes: (r) => r.sitemap.files > 0 },
  { label: 'Sitemap in Dutch, German and English', passes: (r) => r.advice.some((a) => a.check === 'sitemapLanguages' && a.ok) },
  { label: 'Has llms.txt', passes: (r) => r.llmsTxt.present },
  { label: 'Has llms-full.txt', passes: (r) => r.llmsFullTxt },
  { label: 'Has Markdown pages', passes: (r) => r.markdown },
  { label: 'Organization structured data', passes: (r) => r.advice.some((a) => a.check === 'organization' && a.ok) },
  { label: 'Product structured data', passes: (r) => r.advice.some((a) => a.check === 'product' && a.ok) },
]

/** The rows a scan has read: the sites the page counts. */
export function scanned(rows: readonly AiReadinessRow[]): (AiReadinessRow & { readiness: AiReadiness })[] {
  return rows.filter((r): r is AiReadinessRow & { readiness: AiReadiness } => r.readiness !== undefined)
}

/** Per check, how many of the scanned sites pass it, and whether Tribelt does (null when not scanned). */
export function checkSummary(rows: readonly AiReadinessRow[]): { label: string; passing: number; sites: number; tribelt: boolean | null }[] {
  const sites = scanned(rows)
  const tribelt = sites.find((r) => r.role === 'tribelt')
  return CHECKS.map((c) => ({
    label: c.label,
    passing: sites.filter((r) => c.passes(r.readiness)).length,
    sites: sites.length,
    tribelt: tribelt ? c.passes(tribelt.readiness) : null,
  }))
}

/** A crawler's access as a mark that differs in shape as well as colour, with what it means. */
export function accessMark(r: AiReadiness | undefined, bot: string): { mark: string; text: string; tone: 'ok' | 'partly' | 'blocked' | 'unknown' } {
  if (!r) return { mark: '–', text: 'Not scanned', tone: 'unknown' }
  if (r.robots === 'unreachable') return { mark: '?', text: 'robots.txt could not be read', tone: 'unknown' }
  const rule = r.bots.find((b) => b.bot === bot)
  if (!rule) return { mark: '?', text: 'Not checked', tone: 'unknown' }
  if (rule.access === 'blocked') return { mark: '✕', text: 'Blocked', tone: 'blocked' }
  if (rule.access === 'partly') return { mark: '◐', text: 'Partly kept out', tone: 'partly' }
  return { mark: '●', text: 'Let in', tone: 'ok' }
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

/** The page's headline: who leads, and where Tribelt stands, by checks passed. */
export function readinessHeadline(rows: readonly AiReadinessRow[]): string {
  const sites = scanned(rows)
  const competitors = sites
    .filter((r) => r.role === 'competitor')
    .sort((a, b) => readinessScore(b.readiness) - readinessScore(a.readiness) || a.name.localeCompare(b.name))
  const leader = competitors[0]
  const tribelt = sites.find((r) => r.role === 'tribelt')
  if (!leader) return 'Who lets search engines and AI assistants in.'
  const lead = `${leader.name} leads with ${formatValue('readinessChecks', readinessScore(leader.readiness))} of five checks.`
  if (!tribelt) return lead
  return `${lead} Tribelt passes ${formatValue('readinessChecks', readinessScore(tribelt.readiness))}.`
}
