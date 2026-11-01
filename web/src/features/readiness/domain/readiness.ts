import type { AiReadiness, ReadinessChange } from '../../../infrastructure/api/types.gen'
import { oxford } from '../../../shared/words'

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

/** "Blocks GPTBot and ClaudeBot" and the like: what a site's robots.txt does to the crawlers, in a line. */
export function policySummary(r: AiReadiness | undefined): string {
  if (!r) return 'Not checked yet. The next scan will check it.'
  if (r.robots === 'unreachable') return 'Its robots.txt could not be read, so its crawler policy is unknown.'
  const blocked = r.bots.filter((b) => b.access === 'blocked').map((b) => b.bot)
  const partly = r.bots.filter((b) => b.access === 'partly').length
  const more = (n: number) => (n === 1 ? '1 more crawler' : `${String(n)} more crawlers`)
  const some = (n: number) => (n === 1 ? '1 crawler' : `${String(n)} crawlers`)
  if (blocked.length > 0) return `Blocks ${oxford(blocked)}${partly > 0 ? `, and closes some pages to ${more(partly)}` : ''}.`
  return partly > 0 ? `Lets every search and AI crawler in, but closes some pages to ${some(partly)}.` : 'Lets every search and AI crawler in.'
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
