import type { AiReadiness, AiReadinessRow, BotRule, CrawlerAccess, TribeltCrawler } from '../../../infrastructure/api/types.gen'
import { languageLabel } from '../../../shared/vocabulary'
import { oxford } from '../../../shared/words'
import { formatValue } from '../../catching-up/domain/standings'
import { hostOf } from '../../prompt-test/domain/grid'
import { type ScoredCheck, siteName, siteScore } from './checks'
import { BOTS } from './readiness'
import { longDate } from '../../../shared/dates'
import { plural } from '../../../shared/numbers'

const int = (n: number) => n.toLocaleString('en-US')
const day = longDate
const languages = (codes: readonly string[]) => oxford(codes.map(languageLabel))

/** The site's page title: "Tribelt’s AI readiness". */
export function siteTitle(row: Pick<AiReadinessRow, 'name' | 'role'>): string {
  return `${siteName(row)}’s AI readiness`
}

/** The host of a site's newest scan, or its name before any scan. */
export function siteHost(row: Pick<AiReadinessRow, 'name' | 'scan'>): string {
  return row.scan ? hostOf(row.scan.url) : row.name
}

/** The sentence under the title: how many checks the site passes. */
export function sitePurpose(row: AiReadinessRow): string {
  const host = siteHost(row)
  if (!row.readiness) return `No scan has recorded the AI readiness of ${host} yet.`
  return `${host} passes ${formatValue('readinessChecks', siteScore(row.readiness))} of the 8 checks.`
}

/** Where the site's figures come from: "From scan 3 of 22 October 2026, checked again on 28 October 2026." */
export function scanDataLine(row: AiReadinessRow): string {
  if (!row.scan) return ''
  const scan = row.scan.number === undefined ? 'the scan' : `scan ${String(row.scan.number)}`
  const again = row.readiness?.checkedAt ? `, checked again on ${day(row.readiness.checkedAt)}` : ''
  return `From ${scan} of ${day(row.scan.finishedAt ?? row.scan.createdAt)}${again}.`
}

/** "Built with WordPress, Yoast SEO, and Cloudflare."; empty when the scan saw none. */
export function builtWith(r: AiReadiness): string {
  const stack = [r.stack.cms, r.stack.seoPlugin, r.stack.botProtection].filter((s) => s !== undefined)
  return stack.length > 0 ? `Built with ${oxford(stack)}.` : ''
}

const ok = (r: AiReadiness, check: AiReadiness['advice'][number]['check']) => r.advice.some((a) => a.check === check && a.ok)
const HOME_UNREAD = 'The home page could not be read, so its structured data is not known.'

function crawlersEvidence(r: AiReadiness): string {
  if (r.robots === 'missing') return 'The site has no robots.txt, so nothing keeps a crawler out.'
  if (r.robots === 'unreachable') return 'robots.txt could not be reached, so the crawler policy is not known.'
  const n = (access: CrawlerAccess) => r.bots.filter((b) => b.access === access).length
  const partly = n('partly')
  const blocked = n('blocked')
  const kept = partly > 0 ? `, ${String(partly)} of them with some paths kept out` : ''
  return `${String(n('allowed') + partly)} of the ${String(r.bots.length)} crawlers are allowed in${kept}, and ${String(blocked)} ${blocked === 1 ? 'is' : 'are'} blocked.`
}

function sitemapEvidence(r: AiReadiness): string {
  const s = r.sitemap
  if (s.files === 0) return s.declared ? 'robots.txt points to a sitemap, but the scan could not read it.' : 'robots.txt does not point to a sitemap.'
  const covers = s.coverage
    ? s.coverage.share >= 1
      ? ', which covers every page the scan read'
      : `, which covers ${formatValue('sitemapCoverage', s.coverage.share)} of the pages the scan read`
    : ''
  return `A sitemap${s.files > 1 ? ` in ${String(s.files)} files` : ''} lists ${int(s.addresses)} addresses${covers}.`
}

function organizationEvidence(r: AiReadiness): string {
  if (!r.homeRead) return HOME_UNREAD
  if (ok(r, 'organization')) return `The home page says who the company is in structured data (${oxford(r.structuredData)}).`
  return r.structuredData.length > 0 ? `The home page carries structured data (${oxford(r.structuredData)}), but none says who the company is.` : 'The home page carries no structured data.'
}

/** What the scan found that a check is judged on, in a sentence. */
export function evidence(check: ScoredCheck, r: AiReadiness): string {
  switch (check) {
    case 'crawlers':
      return crawlersEvidence(r)
    case 'sitemap':
      return sitemapEvidence(r)
    case 'sitemapLanguages':
      return r.sitemap.languages.length > 0 ? `The sitemap lists pages in ${languages(r.sitemap.languages)}.` : 'The sitemap lists no language for its pages.'
    case 'languages':
      if (!r.homeRead) return 'The home page could not be read, so its languages are not known.'
      return r.languages.length > 0 ? `The site has pages in ${languages(r.languages)}.` : 'The home page names no language versions.'
    case 'organization':
      return organizationEvidence(r)
    case 'llmsTxt':
      return r.llmsTxt.present ? `llms.txt is ${int(r.llmsTxt.bytes)} bytes and links to ${int(r.llmsTxt.links)} pages.` : 'The site has no llms.txt.'
    case 'llmsFullTxt':
      return r.llmsFullTxt ? 'The site has an llms-full.txt.' : 'The site has no llms-full.txt.'
    case 'markdown':
      return r.markdown ? 'Its pages are also offered as Markdown.' : 'Its pages are not offered as Markdown.'
  }
}

/** The structured data on the home page, and whether Product data, which the score leaves out, was found. */
export function structuredData(r: AiReadiness): { found: string; product: string } {
  const found = !r.homeRead ? HOME_UNREAD : r.structuredData.length > 0 ? `${oxford(r.structuredData)}.` : 'No structured data was found on the home page.'
  return { found, product: `Product structured data: ${ok(r, 'product') ? 'found' : 'not found'}. It is shown here, but it does not count toward the score.` }
}

/** Each file the scan looks for, and what it found of it. */
export function filesAsFound(r: AiReadiness): { name: string; text: string }[] {
  const robots = { found: 'Found.', missing: 'Not found.', unreachable: 'The scan could not reach it.' }[r.robots]
  return [
    { name: 'robots.txt', text: robots },
    { name: 'llms.txt', text: r.llmsTxt.present ? `Found, ${int(r.llmsTxt.bytes)} bytes, ${int(r.llmsTxt.links)} links.` : 'Not found.' },
    { name: 'llms-full.txt', text: r.llmsFullTxt ? 'Found.' : 'Not found.' },
    { name: 'Sitemap', text: r.sitemap.files > 0 ? `Found, ${plural(r.sitemap.files, 'file', 'files')} with ${plural(r.sitemap.addresses, 'address', 'addresses')}.` : 'Not found.' },
  ]
}

const ACCESS_WORDS: Record<CrawlerAccess, string> = { allowed: 'Allowed', partly: 'Allowed, with some paths kept out', blocked: 'Blocked' }

/** One crawler in the crawler policy: who runs it, what it does, and what robots.txt says to it. */
export type CrawlerRow = { bot: string; lab: string; gets: string; access: string; note: string; blocked: boolean }

function accessOf(r: AiReadiness, b: BotRule): Pick<CrawlerRow, 'access' | 'note'> {
  if (r.robots === 'missing') return { access: 'Not mentioned', note: 'The site has no robots.txt.' }
  if (r.robots === 'unreachable') return { access: 'Not known', note: 'robots.txt could not be reached.' }
  return { access: ACCESS_WORDS[b.access], note: b.named ? '' : 'Not named, so the general rules apply.' }
}

/** Each crawler the scan checked, described by the AI search summary, or by who runs it when that does not know it. */
export function crawlerRows(r: AiReadiness, described: readonly TribeltCrawler[]): CrawlerRow[] {
  return r.bots.map((b) => {
    const known = described.find((c) => c.bot === b.bot)
    return {
      bot: b.bot,
      lab: known?.lab ?? BOTS.find((x) => x.token === b.bot)?.owner ?? '',
      gets: known?.gets ?? '',
      ...accessOf(r, b),
      blocked: r.robots === 'found' && b.access === 'blocked',
    }
  })
}
