import type { AiReadiness, AiReadinessRow, Plan, ReadinessCheck } from '../../../infrastructure/api/types.gen'
import { type Sort, sortRows, type SortValue } from '../../../shared/sort'

/** The eight checks the readiness score counts, in the order the pages list them. */
export const SCORED_CHECKS = ['crawlers', 'sitemap', 'sitemapLanguages', 'languages', 'organization', 'llmsTxt', 'llmsFullTxt', 'markdown'] as const

export type ScoredCheck = (typeof SCORED_CHECKS)[number]

/** Each check's name. */
export const CHECK_NAMES: Record<ScoredCheck, string> = {
  crawlers: 'AI crawlers let in',
  sitemap: 'A sitemap',
  sitemapLanguages: 'A sitemap in every language',
  languages: 'Pages in Dutch, German, and English',
  organization: 'Structured data about the company',
  llmsTxt: 'llms.txt',
  llmsFullTxt: 'llms-full.txt',
  markdown: 'Markdown pages',
}

/** What each check is, and why it matters, in a sentence. */
export const CHECK_WHY: Record<ScoredCheck, string> = {
  crawlers: 'robots.txt lets search and AI crawlers read the site, so assistants can find and learn its pages.',
  sitemap: 'A sitemap lists every page, so a crawler finds pages that no link leads to.',
  sitemapLanguages: 'The sitemap lists the pages in all three languages, so crawlers find each language version.',
  languages: 'The site has pages in Dutch, German, and English, the languages of the buyers in scope.',
  organization: 'The home page says in structured data who the company is, where it is, and what it makes, so an assistant reads it as facts.',
  llmsTxt: 'llms.txt is a short guide to the site for language models, with links to the pages that matter. Having it helps.',
  llmsFullTxt: 'llms-full.txt is the whole site as one plain text file, so an assistant can read it in one go. Having it as well as llms.txt helps more.',
  markdown: 'Each page is also offered as Markdown. A language model reads Markdown more reliably than a designed web page.',
}

/** How a site stands on a check: passed, partly (some crawlers kept out), not passed, or not known. */
export type CheckState = 'yes' | 'partly' | 'no' | 'unknown'

/** The words after a check's name. */
export const STATE_WORDS: Record<CheckState, string> = {
  yes: 'Passed.',
  partly: 'Partly passed, as some crawlers are kept out.',
  no: 'Not passed.',
  unknown: 'Not known, as robots.txt could not be read.',
}

const POINTS: Record<CheckState, number> = { yes: 1, partly: 0.5, no: 0, unknown: 0 }

/** How a site stands on each check, read the way the server scores it. */
export function checkStates(r: AiReadiness): Record<ScoredCheck, CheckState> {
  const blocked = r.bots.filter((b) => b.access === 'blocked').length
  const yes = (ok: boolean): CheckState => (ok ? 'yes' : 'no')
  const advised = (check: ReadinessCheck) => yes(r.advice.some((a) => a.check === check && a.ok))
  return {
    crawlers: r.robots === 'unreachable' || r.bots.length === 0 ? 'unknown' : blocked === 0 ? 'yes' : blocked === r.bots.length ? 'no' : 'partly',
    sitemap: yes(r.sitemap.files > 0),
    sitemapLanguages: advised('sitemapLanguages'),
    languages: advised('languages'),
    organization: advised('organization'),
    llmsTxt: yes(r.llmsTxt.present),
    llmsFullTxt: yes(r.llmsFullTxt),
    markdown: yes(r.markdown),
  }
}

/** How many of the eight checks a site passes, the crawlers counting half when some are kept out. */
export function siteScore(r: AiReadiness): number {
  const states = checkStates(r)
  return SCORED_CHECKS.reduce((sum, c) => sum + POINTS[states[c]], 0)
}

/** The recommendation (an improvement of the plan) that fixes each check, by the improvement's id. */
const CHECK_FIX: Record<Exclude<ScoredCheck, 'crawlers'>, string> = {
  sitemap: 'sitemap-per-language',
  sitemapLanguages: 'sitemap-per-language',
  languages: 'family-pages',
  organization: 'structured-data',
  llmsTxt: 'llms-files',
  llmsFullTxt: 'llms-files',
  markdown: 'markdown-pages',
}

/**
 * Where a failed check of Tribelt's is fixed: the recommendation that fixes it, or for the
 * crawlers, Tribelt's site page, which says which ones robots.txt turns away.
 */
export function checkFix(check: ScoredCheck, plan: Pick<Plan, 'improvements'> | undefined, tribeltId: string): { to: string; label: string } | undefined {
  if (check === 'crawlers') return { to: `/ai-readiness/${tribeltId}`, label: 'See which crawlers tribelt.nl turns away' }
  const improvement = plan?.improvements.find((i) => i.id === CHECK_FIX[check])
  if (!improvement) return undefined
  return { to: `/recommendations/${improvement.id}`, label: `Recommended fix, ${improvement.title.charAt(0).toLowerCase()}${improvement.title.slice(1)}` }
}

/** A row with a recorded readiness. */
export type ReadySite = AiReadinessRow & { readiness: AiReadiness }

function isReady(row: AiReadinessRow): row is ReadySite {
  return row.readiness !== undefined
}

/** tribelt.nl's standing on a check, in the comparison table. */
export const COMPARE_WORDS: Record<CheckState, string> = { yes: 'Passes', partly: 'Partly', no: 'Fails', unknown: 'Not known' }

/** One check, tribelt.nl's standing on it, and how many of the competitors pass it. */
export type CompareRow = { check: ScoredCheck; name: string; tribelt: CheckState; passing: number; share: number }

/** tribelt.nl against the competitors with a recorded readiness, check by check; undefined before tribelt.nl is scanned. */
export function compareChecks(rows: readonly AiReadinessRow[]): { competitors: number; rows: CompareRow[] } | undefined {
  const tribelt = rows.find((r) => r.role === 'tribelt')?.readiness
  if (!tribelt) return undefined
  const mine = checkStates(tribelt)
  const theirs = rows
    .filter((r) => r.role === 'competitor')
    .filter(isReady)
    .map((r) => checkStates(r.readiness))
  return {
    competitors: theirs.length,
    rows: SCORED_CHECKS.map((check) => {
      const passing = theirs.filter((s) => s[check] === 'yes').length
      return { check, name: CHECK_NAMES[check], tribelt: mine[check], passing, share: theirs.length === 0 ? 0 : passing / theirs.length }
    }),
  }
}

/** The sentences over the comparison: where tribelt.nl is ahead of most competitors, and where it is behind some. */
export function compareIntro(c: { competitors: number; rows: readonly CompareRow[] }): string {
  const ahead = c.rows.filter((r) => r.tribelt === 'yes' && r.share < 0.5).length
  const behind = c.rows.filter((r) => (r.tribelt === 'no' || r.tribelt === 'partly') && r.passing > 0).length
  const checks = ahead === 0 ? 'no checks' : ahead === 1 ? '1 check' : `${String(ahead)} checks`
  return [
    `For each check, how many of the ${String(c.competitors)} competitors pass it, and whether tribelt.nl does.`,
    `tribelt.nl passes ${checks} that most competitors fail.`,
    `It fails ${behind === 0 ? 'none' : String(behind)} that some competitors pass.`,
  ].join(' ')
}

/** The sites Every site shows: the competitors, or every site with a recorded readiness. */
export type SiteFilter = 'competitors' | 'all'

/** A column of Every site. */
export type SiteColumn = 'name' | 'score' | ScoredCheck | 'platform'

/** What a site is called on the pages: Tribelt as Tribelt, any other by its name. */
export function siteName(row: Pick<AiReadinessRow, 'name' | 'role'>): string {
  return row.role === 'tribelt' ? 'Tribelt' : row.name
}

/** The content management system a site is built with. */
export function platformOf(r: AiReadiness): string {
  return r.stack.cms ?? 'Not detected'
}

/** What a row sorts by in a column; a check sorts by the points it gives. */
function siteSortValue(row: ReadySite, column: SiteColumn): SortValue {
  if (column === 'name') return siteName(row)
  if (column === 'score') return siteScore(row.readiness)
  if (column === 'platform') return platformOf(row.readiness)
  return POINTS[checkStates(row.readiness)[column]]
}

/**
 * The rows of Every site: the sites with a recorded readiness the filter keeps and whose name has
 * the query, sorted, with Tribelt at the top whatever the sort.
 */
export function everySite(rows: readonly AiReadinessRow[], filter: SiteFilter, sort: Sort<SiteColumn>, query: string): ReadySite[] {
  const q = query.trim().toLowerCase()
  const kept = rows.filter(isReady).filter((r) => (filter === 'all' || r.role === 'competitor' || r.role === 'tribelt') && r.name.toLowerCase().includes(q))
  return [...kept.filter((r) => r.role === 'tribelt'), ...sortRows(kept.filter((r) => r.role !== 'tribelt'), sort, siteSortValue)]
}

const day = (iso: string) => new Date(iso).toLocaleDateString('en-GB', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' })

/** Where the figures of AI readiness come from: each site's newest scan, when it was checked again, and how many sites. */
export function readinessDataLine(rows: readonly AiReadinessRow[]): string {
  const ready = rows.filter(isReady)
  if (ready.length === 0) return 'No site has been scanned yet.'
  const checked = ready
    .flatMap((r) => (r.readiness.checkedAt ? [r.readiness.checkedAt] : []))
    .sort()
    .at(-1)
  return `From each site's newest scan${checked ? `, checked again on ${day(checked)}` : ''}. ${ready.length === 1 ? '1 site' : `${String(ready.length)} sites`}.`
}
