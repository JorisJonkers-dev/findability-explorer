import { describe, expect, it } from 'vitest'
import type { AiReadiness } from '../../../infrastructure/api/types.gen'
import { bots, readiness, readinessRow } from '../../../testing/fixtures'
import {
  adviceSummary,
  bandOf,
  cell,
  changeLabel,
  COLUMNS,
  GAPS,
  hasGap,
  markedLines,
  policySummary,
  readinessFacts,
  sortValue,
  tableRows,
} from './readiness'

/** A site with gaps: GPTBot blocked, no llms.txt, no Product data, Dutch missing everywhere. */
function gappy(): AiReadiness {
  return readiness({
    bots: bots({ GPTBot: 'blocked', ClaudeBot: 'partly' }),
    sitemap: { declared: false, files: 1, addresses: 40, languages: ['de', 'en'] },
    llmsTxt: { present: false, bytes: 0, links: 0 },
    llmsFullTxt: false,
    markdown: false,
    structuredData: ['Organization'],
    languages: ['de', 'en'],
    stack: { botProtection: 'Cloudflare' },
    advice: [
      { check: 'sitemapDeclared', ok: false },
      { check: 'sitemapLanguages', ok: false },
      { check: 'llmsTxt', ok: false },
      { check: 'markdown', ok: false },
      { check: 'organization', ok: true },
      { check: 'product', ok: false },
      { check: 'languages', ok: false },
    ],
    gaps: 7,
  })
}

describe('the AI readiness table', () => {
  it('has a column for the company, each crawler, each signal and the gaps', () => {
    expect(COLUMNS.map((c) => c.short)).toEqual([
      'Company', 'Googlebot', 'Google-Extended', 'Bingbot', 'OAI-SearchBot', 'ChatGPT-User', 'GPTBot', 'Claude-SearchBot', 'ClaudeBot',
      'PerplexityBot', 'CCBot', 'Sitemaps', 'Sitemap languages', 'llms.txt', 'llms-full.txt', 'Markdown', 'Organization', 'Product',
      'Languages', 'CMS', 'SEO plugin', 'Bot protection', 'Gaps',
    ])
    expect(COLUMNS.find((c) => c.key === 'bot:GPTBot')?.label).toBe('GPTBot (OpenAI training)')
  })

  it('reads each cell, marking whether it follows the recommendation', () => {
    const good = readinessRow('Mirror', { role: 'mirror' })
    const bad = readinessRow('Twentebelt', { readiness: gappy() })
    expect(cell(good, 'name')).toEqual({ text: 'Mirror', ok: null })
    expect(cell(good, 'bot:GPTBot')).toEqual({ text: 'Allowed', ok: true })
    expect(cell(bad, 'bot:GPTBot')).toEqual({ text: 'Blocked', ok: false })
    expect(cell(bad, 'bot:ClaudeBot')).toEqual({ text: 'Partly', ok: true })
    expect(cell(readinessRow('A', { readiness: readiness({ bots: bots({}, false) }) }), 'bot:CCBot')).toEqual({ text: 'Allowed, not mentioned', ok: true })
    expect(cell(good, 'bot:NoSuchBot')).toEqual({ text: 'Not checked', ok: null })
    expect(cell(good, 'sitemaps')).toEqual({ text: 'Yes, 2 files, 124 addresses', ok: true })
    expect(cell(bad, 'sitemaps')).toEqual({ text: 'No', ok: false })
    expect(cell(good, 'sitemapLanguages')).toEqual({ text: 'NL DE EN', ok: true })
    expect(cell(readinessRow('A', { readiness: readiness({ sitemap: { declared: true, files: 0, addresses: 0, languages: [] } }) }), 'sitemapLanguages').text).toBe('None')
    expect(cell(good, 'llmsTxt')).toEqual({ text: 'Yes, 124 links', ok: true })
    expect(cell(bad, 'llmsTxt')).toEqual({ text: 'No', ok: false })
    expect(cell(good, 'llmsFullTxt')).toEqual({ text: 'Yes', ok: null })
    expect(cell(bad, 'markdown')).toEqual({ text: 'No', ok: false })
    expect(cell(bad, 'organization')).toEqual({ text: 'Yes', ok: true })
    expect(cell(bad, 'product')).toEqual({ text: 'No', ok: false })
    expect(cell(bad, 'languages')).toEqual({ text: 'DE EN', ok: false })
    expect(cell(good, 'cms')).toEqual({ text: 'WordPress', ok: null })
    expect(cell(good, 'botProtection')).toEqual({ text: 'None found', ok: null })
    expect(cell(good, 'gaps')).toEqual({ text: '0', ok: true })
    expect(cell(bad, 'gaps')).toEqual({ text: '7', ok: false })
  })

  it('says what is unknown: a company not checked, robots.txt unreachable, a home page not read', () => {
    const unchecked = readinessRow('New', { readiness: undefined })
    expect(cell(unchecked, 'bot:GPTBot')).toEqual({ text: 'Not checked', ok: null })
    const walled = readinessRow('Walled', { readiness: readiness({ robots: 'unreachable', bots: [], homeRead: false, advice: [] }) })
    expect(cell(walled, 'bot:GPTBot')).toEqual({ text: 'Unknown', ok: null })
    expect(cell(walled, 'organization')).toEqual({ text: 'Unknown', ok: null })
    expect(cell(walled, 'languages')).toEqual({ text: 'Unknown', ok: null })
    expect(cell(walled, 'llmsTxt')).toEqual({ text: 'Yes, 124 links', ok: false })
  })

  it('sorts by any column, a company not checked lowest', () => {
    const good = readinessRow('Good')
    const bad = readinessRow('Bad', { readiness: gappy() })
    const unchecked = readinessRow('Unchecked', { readiness: undefined })
    expect(sortValue(good, 'name')).toBe('Good')
    expect([sortValue(good, 'bot:GPTBot'), sortValue(bad, 'bot:ClaudeBot'), sortValue(bad, 'bot:GPTBot'), sortValue(good, 'bot:None')]).toEqual([2, 1, 0, -1])
    expect([sortValue(good, 'sitemaps'), sortValue(bad, 'sitemaps')]).toEqual([124, -0.5])
    expect([sortValue(good, 'llmsTxt'), sortValue(bad, 'llmsTxt')]).toEqual([124, -0.5])
    expect([sortValue(good, 'sitemapLanguages'), sortValue(bad, 'languages')]).toEqual([3, 2])
    expect([sortValue(good, 'cms'), sortValue(bad, 'cms')]).toEqual(['WordPress', '￿'])
    expect([sortValue(good, 'gaps'), sortValue(unchecked, 'gaps'), sortValue(unchecked, 'markdown')]).toEqual([0, Number.MAX_SAFE_INTEGER, -1])
    expect([sortValue(good, 'markdown'), sortValue(bad, 'product'), sortValue(good, 'llmsFullTxt')]).toEqual([true, false, true])
  })

  it('pins Tribelt and the mirror on top, then bands the group and the competitors, each sorted', () => {
    const rows = [
      readinessRow('Twentebelt', { readiness: gappy() }),
      readinessRow('ESFO'),
      readinessRow('Lesjöfors', { role: 'group' }),
      readinessRow('Mirror of tribelt.nl', { role: 'mirror' }),
      readinessRow('Tribelt B.V.', { role: 'tribelt', readiness: gappy() }),
    ]
    const names = (gap: Parameters<typeof tableRows>[2], sort: Parameters<typeof tableRows>[1] = null) =>
      tableRows(rows, sort, gap).map((b) => `${b.band}: ${b.rows.map((r) => r.name).join(', ')}`)
    expect(names(null)).toEqual(['baseline: Tribelt B.V., Mirror of tribelt.nl', 'group: Lesjöfors', 'competitor: Twentebelt, ESFO'])
    expect(names(null, { key: 'name', direction: 'asc' })[2]).toBe('competitor: ESFO, Twentebelt')
    expect(names('bot:GPTBot')).toEqual(['baseline: Tribelt B.V., Mirror of tribelt.nl', 'competitor: Twentebelt'])
    expect(bandOf({ role: 'tribelt' })).toBe('baseline')
  })

  it('filters by every kind of gap', () => {
    const bad = readinessRow('Bad', { readiness: gappy() })
    expect(GAPS.map((g) => g.value).filter((g) => hasGap(bad, g))).toEqual(['any', 'bot:GPTBot', 'sitemapDeclared', 'sitemapLanguages', 'llmsTxt', 'markdown', 'product', 'languages'])
    expect(hasGap(readinessRow('Good'), 'any')).toBe(false)
    expect(hasGap(readinessRow('Unchecked', { readiness: undefined }), 'any')).toBe(false)
  })
})

describe('the AI readiness on a company card', () => {
  it('sums up the crawler policy in a line', () => {
    expect(policySummary(undefined)).toBe('Not checked yet. The next scan will check it.')
    expect(policySummary(readiness({ robots: 'unreachable', bots: [] }))).toBe('Its robots.txt could not be read, so its crawler policy is unknown.')
    expect(policySummary(readiness())).toBe('Lets every search and AI crawler in.')
    expect(policySummary(readiness({ bots: bots({ CCBot: 'partly' }) }))).toBe('Lets every search and AI crawler in, but closes some pages to 1 crawler.')
    expect(policySummary(readiness({ bots: bots({ GPTBot: 'blocked' }) }))).toBe('Blocks GPTBot.')
    expect(policySummary(readiness({ bots: bots({ GPTBot: 'blocked', ClaudeBot: 'blocked' }) }))).toBe('Blocks GPTBot and ClaudeBot.')
    expect(policySummary(readiness({ bots: bots({ GPTBot: 'blocked', ClaudeBot: 'blocked', CCBot: 'blocked', Bingbot: 'partly' }) }))).toBe(
      'Blocks GPTBot, ClaudeBot and CCBot, and closes some pages to 1 more crawler.',
    )
    expect(policySummary(readiness({ bots: bots({ GPTBot: 'blocked', Bingbot: 'partly', CCBot: 'partly' }) }))).toBe('Blocks GPTBot, and closes some pages to 2 more crawlers.')
    expect(policySummary(readiness({ bots: bots({ Bingbot: 'partly', CCBot: 'partly' }) }))).toBe('Lets every search and AI crawler in, but closes some pages to 2 crawlers.')
  })

  it('counts the recommendations followed, and lists the other signals', () => {
    expect(adviceSummary(gappy())).toBe('Follows 10 of 17 recommendations.')
    expect(readinessFacts(gappy())).toEqual([
      { label: 'Sitemaps', value: 'No', ok: false },
      { label: 'Sitemap languages', value: 'DE EN', ok: false },
      { label: 'llms.txt', value: 'No', ok: false },
      { label: 'llms-full.txt', value: 'No', ok: null },
      { label: 'Markdown versions', value: 'No', ok: false },
      { label: 'Structured data', value: 'Organization', ok: false },
      { label: 'Language versions', value: 'DE EN', ok: false },
      { label: 'Built with', value: 'Cloudflare', ok: null },
    ])
    const facts = readinessFacts(readiness())
    expect(facts[5]).toEqual({ label: 'Structured data', value: 'Organization, Product', ok: true })
    expect(facts[7]).toEqual({ label: 'Built with', value: 'WordPress, Yoast SEO', ok: null })
    expect(readinessFacts(readiness({ structuredData: [] }))[5]?.value).toBe('None')
    expect(readinessFacts(readiness({ homeRead: false }))[5]).toEqual({ label: 'Structured data', value: 'Unknown', ok: null })
    expect(readinessFacts(readiness({ stack: {} }))[7]?.value).toBe('Nothing it shows')
  })
})

describe('changes and raw files', () => {
  it('reads a readiness change', () => {
    expect(changeLabel({ field: 'bot:GPTBot', from: 'allowed', to: 'blocked' })).toBe('GPTBot: allowed → blocked')
    expect(changeLabel({ field: 'llmsTxt', from: 'no', to: 'yes' })).toBe('llms.txt: no → yes')
    expect(changeLabel({ field: 'somethingNew', from: 'a', to: 'b' })).toBe('somethingNew: a → b')
  })

  it('marks the lines of a raw file that name a search or AI crawler', () => {
    expect(markedLines('')).toEqual([])
    expect(markedLines('User-agent: GPTBot\nDisallow: /\nuser-agent: meta-externalagent')).toEqual([
      { text: 'User-agent: GPTBot', ai: true },
      { text: 'Disallow: /', ai: false },
      { text: 'user-agent: meta-externalagent', ai: true },
    ])
  })
})
