import type { AiReadinessRow, AiSearchSummary, CrawlerAccess, Plan, ReadinessCheckKind, ReadinessRank } from '../../../infrastructure/api/types.gen'
import { formatValue, inWords } from '../../catching-up/domain/standings'
import { CHECK_LABELS } from './overview'

/** What each check means for Tribelt, in a sentence or two. */
export const CHECK_WHAT: Record<ReadinessCheckKind, string> = {
  crawlers: 'robots.txt lets every AI crawler read the site, the search crawlers and the training crawlers (GPTBot, ClaudeBot, Google-Extended, and CCBot). A crawler that is kept out cannot learn about the site or cite it.',
  sitemap: 'A sitemap lists the pages of a site, so crawlers find every page, including new ones.',
  llmsTxt: 'llms.txt is a short map of the site in plain text, written for AI tools.',
  llmsFullTxt: 'llms-full.txt holds the text of the main pages in one file, so an AI tool can read what Tribelt makes without crawling every page.',
  markdown: 'A Markdown version of a page gives AI tools the facts without the menus and scripts around them.',
  languages: 'The site offers its pages in Dutch, German, and English, the three languages Tribelt’s buyers search in.',
  sitemapLanguages: 'The sitemap lists pages in all three languages Tribelt’s buyers search in.',
  organization: 'Structured data on the home page that tells search engines the company’s name, address, and website.',
  product: 'Structured data on a product page that tells search engines and AI tools what the product is and what it does.',
}

/** The recommendation (an improvement of the plan) that fixes each check, by the improvement's id. */
export const CHECK_FIX: Partial<Record<ReadinessCheckKind, string>> = {
  sitemap: 'sitemap-per-language',
  sitemapLanguages: 'sitemap-per-language',
  llmsTxt: 'llms-files',
  llmsFullTxt: 'llms-files',
  markdown: 'markdown-pages',
  product: 'structured-data',
}

/** A link to the recommendation that fixes a check: "See recommendation 2, add llms.txt and llms-full.txt". */
export function fixFor(check: ReadinessCheckKind, plan: Pick<Plan, 'improvements'> | undefined): { to: string; label: string } | undefined {
  const id = CHECK_FIX[check]
  const index = plan?.improvements.findIndex((i) => i.id === id) ?? -1
  const improvement = plan?.improvements[index]
  if (!improvement) return undefined
  return { to: `/recommendations/${improvement.id}`, label: `See recommendation ${String(index + 1)}, ${improvement.title.charAt(0).toLowerCase()}${improvement.title.slice(1)}` }
}

/** One check in the bar list. */
export type CheckRow = { kind: ReadinessCheckKind; label: string; passing: number; sites: number; share: number; tribelt: boolean | null; status: string; aria: string }

/** What is said of tribelt.nl on a check. */
export function tribeltStatus(tribelt: boolean | null): string {
  if (tribelt === null) return 'Not scanned yet'
  return tribelt ? 'Tribelt passes' : 'Tribelt does not pass'
}

/** The checks in two groups: the ones the score counts, then the ones checked beside them. */
export function checkGroups(summary: Pick<AiSearchSummary, 'checks' | 'sites'> | undefined, readinessChecks: number): { title: string; aria: string; checks: CheckRow[] }[] {
  if (!summary) return []
  const row = (c: AiSearchSummary['checks'][number]): CheckRow => {
    const tribelt = c.tribelt ?? null
    const status = tribeltStatus(tribelt)
    return {
      kind: c.check,
      label: CHECK_LABELS[c.check],
      passing: c.passing,
      sites: summary.sites,
      share: summary.sites === 0 ? 0 : c.passing / summary.sites,
      tribelt,
      status,
      aria: `${CHECK_LABELS[c.check]}: ${String(c.passing)} of ${String(summary.sites)} sites pass. ${status}.`,
    }
  }
  return [
    { title: `The ${inWords(readinessChecks)} checks in the score`, aria: `The ${inWords(readinessChecks)} checks in the score`, checks: summary.checks.filter((c) => c.scored).map(row) },
    { title: 'Also checked', aria: 'Also checked, not in the score', checks: summary.checks.filter((c) => !c.scored).map(row) },
  ].filter((g) => g.checks.length > 0)
}

/** A score in words: "three", "three and a half", "none". */
export function scoreWords(score: number): string {
  const whole = Math.floor(score)
  const half = score - whole >= 0.5
  if (!half) return whole === 0 ? 'none' : inWords(whole)
  return whole === 0 ? 'half of one' : `${inWords(whole)} and a half`
}

/** The page's headline: how many checks Tribelt passes, against the best competitors. */
export function readinessTitle(ranking: readonly ReadinessRank[], readinessChecks: number): string {
  const tribelt = ranking.find((r) => r.role === 'tribelt')
  if (!tribelt) return 'How ready the sites are for AI assistants'
  const passes = `Tribelt passes ${scoreWords(tribelt.score)} of the ${inWords(readinessChecks)} readiness checks`
  const best = Math.max(...ranking.filter((r) => r.role === 'competitor').map((r) => r.score), Number.NEGATIVE_INFINITY)
  if (best === Number.NEGATIVE_INFINITY) return passes
  if (tribelt.score > best) return `${passes}, more than any competitor`
  return tribelt.score === best ? `${passes}, matching its best competitors` : `${passes}, fewer than its best competitors`
}

/** The sites with the best score, and Tribelt's if it is not among them. */
export function bestSites(ranking: readonly ReadinessRank[]): { top: number; sites: ReadinessRank[]; tribeltAmong: boolean } {
  const counted = ranking.filter((r) => r.role !== 'mirror')
  const top = Math.max(...counted.map((r) => r.score), 0)
  const best = counted.filter((r) => r.score === top)
  const tribelt = counted.find((r) => r.role === 'tribelt')
  const tribeltAmong = tribelt !== undefined && tribelt.score === top
  return { top, sites: tribelt && !tribeltAmong ? [...best, tribelt] : best, tribeltAmong }
}

/** The three figures under the headline. */
export function readinessFigures(summary: Pick<AiSearchSummary, 'ranking' | 'sites' | 'readinessChecks'>): { big: string; text: string; accent: boolean }[] {
  const tribelt = summary.ranking.find((r) => r.role === 'tribelt')
  const { top, sites, tribeltAmong } = bestSites(summary.ranking)
  const tied = sites.filter((s) => s.score === top).length
  const among = tribeltAmong ? ', Tribelt among them' : ''
  return [
    { big: String(summary.sites), text: summary.sites === 1 ? 'site checked' : 'sites checked', accent: false },
    {
      big: tribelt ? `${formatValue('readinessChecks', tribelt.score)} of ${String(summary.readinessChecks)}` : '–',
      text: 'checks that tribelt.nl passes',
      accent: true,
    },
    { big: String(tied), text: tied === 1 ? `site has the best score${tribeltAmong ? ', and it is Tribelt' : ''}` : `sites share the best score${among}`, accent: false },
  ]
}

/** How a site stands on one check. */
export type SiteCheck = { label: string; state: 'yes' | 'partly' | 'no' | 'unknown' }

/** A site's eight scored checks, read the way the server scores them. */
export function siteChecks(row: Pick<AiReadinessRow, 'readiness'>): SiteCheck[] {
  const r = row.readiness
  const scored: ReadinessCheckKind[] = ['crawlers', 'sitemap', 'llmsTxt', 'llmsFullTxt', 'markdown', 'organization', 'languages', 'sitemapLanguages']
  if (!r) return scored.map((kind) => ({ label: CHECK_LABELS[kind], state: 'unknown' }))
  const blocked = r.bots.filter((b) => b.access === 'blocked').length
  const crawlers: SiteCheck['state'] = r.robots === 'unreachable' ? 'unknown' : blocked === 0 ? 'yes' : blocked === r.bots.length ? 'no' : 'partly'
  const yes = (ok: boolean): SiteCheck['state'] => (ok ? 'yes' : 'no')
  const advised = (check: ReadinessCheckKind) => yes(r.advice.some((a) => a.check === check && a.ok))
  const states = [crawlers, yes(r.sitemap.files > 0), yes(r.llmsTxt.present), yes(r.llmsFullTxt), yes(r.markdown), advised('organization'), advised('languages'), advised('sitemapLanguages')]
  return scored.map((kind, i) => ({ label: CHECK_LABELS[kind], state: states[i] as SiteCheck['state'] }))
}

/** The words for a site's state on a check. */
export const STATE_WORDS: Record<SiteCheck['state'], string> = { yes: 'Yes', partly: 'Partly', no: 'No', unknown: 'Not known' }

/** What tribelt.nl does about a crawler, in the crawler table. */
export const LETS_IN: Record<CrawlerAccess | 'unknown', string> = {
  allowed: 'Lets it in',
  partly: 'Lets it in, except some pages',
  blocked: 'Keeps it out',
  unknown: 'Not checked',
}
