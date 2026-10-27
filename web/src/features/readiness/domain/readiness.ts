import type { AiReadiness, AiReadinessRow, CrawlerAccess, ReadinessChange, ReadinessCheck } from '../../../infrastructure/api/types.gen'
import type { Sort, SortDirection, SortValue } from '../../../shared/sort'
import { sortRows } from '../../../shared/sort'
import { joinWords } from '../../../shared/words'

/** The search and AI crawlers a scan records, in the server's order, with who runs them and why. */
export const BOTS: readonly { token: string; owner: string }[] = [
  { token: 'Googlebot', owner: 'Google Search' },
  { token: 'Google-Extended', owner: 'Gemini' },
  { token: 'Bingbot', owner: 'Bing and Copilot' },
  { token: 'OAI-SearchBot', owner: 'ChatGPT search' },
  { token: 'ChatGPT-User', owner: 'ChatGPT, for a user' },
  { token: 'GPTBot', owner: 'OpenAI training' },
  { token: 'Claude-SearchBot', owner: 'Claude search' },
  { token: 'ClaudeBot', owner: 'Anthropic training' },
  { token: 'PerplexityBot', owner: 'Perplexity search' },
  { token: 'CCBot', owner: 'Common Crawl' },
]

/** A column of the AI readiness table. */
export type ReadinessColumn =
  | 'name'
  | `bot:${string}`
  | 'sitemaps'
  | 'sitemapLanguages'
  | 'llmsTxt'
  | 'llmsFullTxt'
  | 'markdown'
  | 'organization'
  | 'product'
  | 'languages'
  | 'cms'
  | 'seoPlugin'
  | 'botProtection'
  | 'gaps'

/** Each column: its head, what it is named by for a screen reader, and the direction it sorts first. */
export const COLUMNS: readonly { key: ReadinessColumn; label: string; short: string; first: SortDirection }[] = [
  { key: 'name', label: 'Company', short: 'Company', first: 'asc' },
  ...BOTS.map((b) => ({ key: `bot:${b.token}` as const, label: `${b.token} (${b.owner})`, short: b.token, first: 'desc' as const })),
  { key: 'sitemaps', label: 'Sitemaps in robots.txt', short: 'Sitemaps', first: 'desc' },
  { key: 'sitemapLanguages', label: 'Sitemap languages', short: 'Sitemap languages', first: 'desc' },
  { key: 'llmsTxt', label: 'llms.txt', short: 'llms.txt', first: 'desc' },
  { key: 'llmsFullTxt', label: 'llms-full.txt', short: 'llms-full.txt', first: 'desc' },
  { key: 'markdown', label: 'Markdown versions', short: 'Markdown', first: 'desc' },
  { key: 'organization', label: 'Organization structured data', short: 'Organization', first: 'desc' },
  { key: 'product', label: 'Product structured data', short: 'Product', first: 'desc' },
  { key: 'languages', label: 'Language versions', short: 'Languages', first: 'desc' },
  { key: 'cms', label: 'CMS', short: 'CMS', first: 'asc' },
  { key: 'seoPlugin', label: 'SEO plugin', short: 'SEO plugin', first: 'asc' },
  { key: 'botProtection', label: 'Bot protection', short: 'Bot protection', first: 'asc' },
  { key: 'gaps', label: 'Recommendations not followed', short: 'Gaps', first: 'asc' },
]

const ACCESS: Record<CrawlerAccess, string> = { allowed: 'Allowed', partly: 'Partly', blocked: 'Blocked' }

/** How far a crawler is let in, ranked so that allowed sorts highest. */
const ACCESS_RANK: Record<CrawlerAccess, number> = { allowed: 2, partly: 1, blocked: 0 }

/**
 * A table cell: what it reads, and whether it follows the report's recommendation (true), does
 * not (false), or says nothing of it (null: a fact, or not checked).
 */
export type Cell = { text: string; ok: boolean | null }

const NOT_CHECKED: Cell = { text: 'Not checked', ok: null }

function yesNo(value: boolean, ok: boolean | null = value): Cell {
  return { text: value ? 'Yes' : 'No', ok }
}

/** The languages Dutch, German and English, as the table writes them: "NL DE EN". */
function languageCodes(languages: readonly string[]): string {
  return languages.length === 0 ? 'None' : languages.map((l) => l.toUpperCase()).join(' ')
}

function adviceOk(r: AiReadiness, check: ReadinessCheck): boolean {
  return r.advice.find((a) => a.check === check)?.ok ?? false
}

/** A crawler's cell: its access, and "(by *)" when robots.txt does not name it. */
function botCell(r: AiReadiness, token: string): Cell {
  if (r.robots === 'unreachable') return { text: 'Unknown', ok: null }
  const rule = r.bots.find((b) => b.bot === token)
  if (!rule) return NOT_CHECKED
  return { text: rule.named ? ACCESS[rule.access] : `${ACCESS[rule.access]}, not mentioned`, ok: rule.followsAdvice }
}

function isBotColumn(column: ReadinessColumn): column is `bot:${string}` {
  return column.startsWith('bot:')
}

/** What a company's row shows in a column. */
export function cell(row: AiReadinessRow, column: ReadinessColumn): Cell {
  const r = row.readiness
  if (column === 'name') return { text: row.name, ok: null }
  if (!r) return NOT_CHECKED
  if (isBotColumn(column)) return botCell(r, column.slice(4))
  switch (column) {
    case 'sitemaps':
      return { text: r.sitemap.declared ? `Yes, ${String(r.sitemap.files)} files, ${String(r.sitemap.addresses)} addresses` : 'No', ok: adviceOk(r, 'sitemapDeclared') }
    case 'sitemapLanguages':
      return { text: languageCodes(r.sitemap.languages), ok: adviceOk(r, 'sitemapLanguages') }
    case 'llmsTxt':
      return { text: r.llmsTxt.present ? `Yes, ${String(r.llmsTxt.links)} links` : 'No', ok: adviceOk(r, 'llmsTxt') }
    case 'llmsFullTxt':
      return yesNo(r.llmsFullTxt, null)
    case 'languages':
    case 'organization':
    case 'product':
      return homeCell(r, column)
    case 'cms':
    case 'seoPlugin':
    case 'botProtection':
      return { text: r.stack[column] ?? 'None found', ok: null }
    case 'gaps':
      return { text: String(r.gaps), ok: r.gaps === 0 }
    case 'markdown':
      return yesNo(r.markdown, adviceOk(r, 'markdown'))
  }
}

/** The cells the home page tells, which a site whose home page could not be read leaves unknown. */
function homeCell(r: AiReadiness, column: 'languages' | 'organization' | 'product'): Cell {
  if (!r.homeRead) return { text: 'Unknown', ok: null }
  if (column === 'languages') return { text: languageCodes(r.languages), ok: adviceOk(r, 'languages') }
  return yesNo(adviceOk(r, column))
}

/** What a row sorts by in a column; rows not checked sort lowest. */
export function sortValue(row: AiReadinessRow, column: ReadinessColumn): SortValue {
  const r = row.readiness
  if (column === 'name') return row.name
  if (!r) return column === 'gaps' ? Number.MAX_SAFE_INTEGER : -1
  if (isBotColumn(column)) {
    const rule = r.bots.find((b) => b.bot === column.slice(4))
    return rule ? ACCESS_RANK[rule.access] : -1
  }
  switch (column) {
    case 'sitemaps':
      return r.sitemap.declared ? r.sitemap.addresses : -0.5
    case 'sitemapLanguages':
      return r.sitemap.languages.length
    case 'llmsTxt':
      return r.llmsTxt.present ? r.llmsTxt.links : -0.5
    case 'languages':
      return r.languages.length
    case 'cms':
    case 'seoPlugin':
    case 'botProtection':
      return r.stack[column] ?? '\uffff' // after every name
    case 'gaps':
      return r.gaps
    case 'llmsFullTxt':
    case 'markdown':
    case 'organization':
    case 'product':
      return cell(row, column).text === 'Yes'
  }
}

/** The bands of the table, top to bottom: Tribelt and the mirror pinned, the group, the competitors. */
export type Band = 'baseline' | 'group' | 'competitor'

export function bandOf(row: Pick<AiReadinessRow, 'role'>): Band {
  if (row.role === 'tribelt' || row.role === 'mirror') return 'baseline'
  return row.role === 'group' ? 'group' : 'competitor'
}

export const BAND_LABELS: Record<Band, string> = {
  baseline: 'Tribelt today, and the target',
  group: "Tribelt's group",
  competitor: 'Competitors',
}

/** A gap to filter the table by: a crawler blocked, or a recommendation not followed. */
export type Gap = 'any' | `bot:${string}` | ReadinessCheck

export const GAPS: readonly { value: Gap; label: string }[] = [
  { value: 'any', label: 'Any gap' },
  ...BOTS.map((b) => ({ value: `bot:${b.token}` as const, label: `Blocks ${b.token}` })),
  { value: 'sitemapDeclared', label: 'No sitemap in robots.txt' },
  { value: 'sitemapLanguages', label: 'Sitemaps miss a language' },
  { value: 'llmsTxt', label: 'No llms.txt' },
  { value: 'markdown', label: 'No Markdown versions' },
  { value: 'organization', label: 'No Organization data' },
  { value: 'product', label: 'No Product data' },
  { value: 'languages', label: 'Misses a language version' },
]

/** Whether a row has the gap; a row not checked has none. */
export function hasGap(row: AiReadinessRow, gap: Gap): boolean {
  const r = row.readiness
  if (!r) return false
  if (gap === 'any') return r.gaps > 0
  if (gap.startsWith('bot:')) return r.bots.some((b) => b.bot === gap.slice(4) && b.access === 'blocked')
  return r.advice.some((a) => a.check === gap && !a.ok)
}

/**
 * The rows to show, band by band: Tribelt and the mirror stay on top whatever the sort or filter,
 * then the group, then the competitors, each band sorted and, with a gap picked, only its rows
 * with that gap.
 */
export function tableRows(rows: readonly AiReadinessRow[], sort: Sort<ReadinessColumn>, gap: Gap | null): { band: Band; rows: AiReadinessRow[] }[] {
  const bands: Band[] = ['baseline', 'group', 'competitor']
  return bands
    .map((band) => {
      const inBand = rows.filter((r) => bandOf(r) === band && (band === 'baseline' || gap === null || hasGap(r, gap)))
      // Tribelt, the baseline, above the mirror, the target.
      const pinned = inBand.filter((r) => r.role === 'tribelt').concat(inBand.filter((r) => r.role === 'mirror'))
      return { band, rows: band === 'baseline' ? pinned : sortRows(inBand, sort, sortValue) }
    })
    .filter((b) => b.rows.length > 0)
}

/** "Blocks GPTBot and ClaudeBot" and the like: what a site's robots.txt does to the crawlers, in a line. */
export function policySummary(r: AiReadiness | undefined): string {
  if (!r) return 'Not checked yet: the next scan records it.'
  if (r.robots === 'unreachable') return 'Its robots.txt could not be read, so its crawler policy is unknown.'
  const blocked = r.bots.filter((b) => b.access === 'blocked').map((b) => b.bot)
  const partly = r.bots.filter((b) => b.access === 'partly').length
  if (blocked.length > 0) return `Blocks ${joinWords(blocked)}${partly > 0 ? `, and keeps some pages from ${String(partly)} more` : ''}.`
  return partly > 0 ? `Lets every search and AI crawler in, keeping some pages from ${String(partly)}.` : 'Lets every search and AI crawler in.'
}

/** How many of the report's recommendations a site follows: "Follows 14 of 17 recommendations." */
export function adviceSummary(r: AiReadiness): string {
  const total = r.advice.length + r.bots.length
  return `Follows ${String(total - r.gaps)} of ${String(total)} recommendations.`
}

/** The card's lines about a site's readiness, besides its crawler policy. */
export function readinessFacts(r: AiReadiness): { label: string; value: string; ok: boolean | null }[] {
  const row: AiReadinessRow = { competitorId: 'x', name: '', role: 'competitor', readiness: r }
  const stack = [r.stack.cms, r.stack.seoPlugin, r.stack.botProtection].filter((s) => s !== undefined)
  const fact = (label: string, column: ReadinessColumn) => ({ label, ...cellAsValue(cell(row, column)) })
  return [
    fact('Sitemaps', 'sitemaps'),
    fact('Sitemap languages', 'sitemapLanguages'),
    fact('llms.txt', 'llmsTxt'),
    fact('llms-full.txt', 'llmsFullTxt'),
    fact('Markdown versions', 'markdown'),
    { label: 'Structured data', value: r.homeRead ? (r.structuredData.length > 0 ? r.structuredData.join(', ') : 'None') : 'Unknown', ok: r.homeRead ? adviceOk(r, 'organization') && adviceOk(r, 'product') : null },
    fact('Language versions', 'languages'),
    { label: 'Built with', value: stack.length > 0 ? stack.join(', ') : 'Nothing it shows', ok: null },
  ]
}

function cellAsValue(c: Cell): { value: string; ok: boolean | null } {
  return { value: c.text, ok: c.ok }
}

/** The labels of a readiness change's field: a crawler's token, or what the field is. */
const FIELD_LABELS: Record<string, string> = {
  robots: 'robots.txt',
  sitemapDeclared: 'Sitemaps in robots.txt',
  sitemapLanguages: 'Sitemap languages',
  llmsTxt: 'llms.txt',
  llmsFullTxt: 'llms-full.txt',
  markdown: 'Markdown versions',
  organization: 'Organization structured data',
  product: 'Product structured data',
  languages: 'Language versions',
  cms: 'CMS',
  seoPlugin: 'SEO plugin',
  botProtection: 'Bot protection',
}

/** A readiness change in words: "GPTBot: allowed → blocked". */
export function changeLabel(change: ReadinessChange): string {
  const field = change.field.startsWith('bot:') ? change.field.slice(4) : (FIELD_LABELS[change.field] ?? change.field)
  return `${field}: ${change.from} → ${change.to}`
}

/** The tokens a robots.txt line is about an AI or search crawler by, folded to lower case. */
const AI_TOKENS = [...BOTS.map((b) => b.token.toLowerCase()), 'claude-user', 'perplexity-user', 'applebot-extended', 'meta-externalagent', 'bytespider', 'amazonbot', 'cohere-ai', 'mistralai-user']

/** A raw file's lines, each marked when it names a search or AI crawler. */
export function markedLines(text: string): { text: string; ai: boolean }[] {
  if (text === '') return []
  return text.split('\n').map((line) => ({ text: line, ai: AI_TOKENS.some((t) => line.toLowerCase().includes(t)) }))
}
