import { describe, expect, it } from 'vitest'
import { bots, readiness, readinessRow, summary } from '../../../testing/fixtures'
import { accessMark, checkSummary, readinessHeadline, readinessScore, scanRange, scanned } from './overview'

const ALL = Object.fromEntries(bots().map((b) => [b.bot, 'blocked' as const]))

describe('the readiness score', () => {
  it('counts five checks, half for crawlers when some are kept out', () => {
    expect(readinessScore(readiness())).toBe(5)
    expect(readinessScore(readiness({ bots: bots({ GPTBot: 'blocked' }), llmsFullTxt: false }))).toBe(3.5)
    expect(readinessScore(readiness({ bots: bots(ALL) }))).toBe(4)
    expect(readinessScore(readiness({ robots: 'unreachable', bots: [] }))).toBe(4)
    expect(readinessScore(readiness({ sitemap: { declared: false, files: 0, addresses: 0, languages: [] }, llmsTxt: { present: false, bytes: 0, links: 0 }, markdown: false }))).toBe(2)
  })
})

describe('the summary', () => {
  it('counts the scanned sites passing each check, with Tribelt beside it', () => {
    const rows = [
      readinessRow('Tribelt', { role: 'tribelt', readiness: readiness({ llmsTxt: { present: false, bytes: 0, links: 0 } }) }),
      readinessRow('Märtens'),
      readinessRow('Twentebelt', { readiness: readiness({ bots: bots({ GPTBot: 'blocked' }), advice: [] }) }),
      readinessRow('Never', { readiness: undefined, scan: undefined }),
    ]
    expect(scanned(rows)).toHaveLength(3)
    expect(checkSummary(rows).map((c) => [c.label, c.passing, c.sites, c.tribelt])).toEqual([
      ['Lets every AI crawler in', 2, 3, true],
      ['Has a sitemap', 3, 3, true],
      ['Sitemap in Dutch, German and English', 2, 3, true],
      ['Has llms.txt', 2, 3, false],
      ['Has llms-full.txt', 3, 3, true],
      ['Has Markdown pages', 3, 3, true],
      ['Organization structured data', 2, 3, true],
      ['Product structured data', 2, 3, true],
    ])
    expect(checkSummary([readinessRow('Märtens')])[0]?.tribelt).toBeNull()
  })
})

describe('the access marks', () => {
  it('differ in shape as well as colour', () => {
    const r = readiness({ bots: bots({ GPTBot: 'blocked', CCBot: 'partly' }) })
    expect(['Googlebot', 'GPTBot', 'CCBot', 'Nobody'].map((b) => accessMark(r, b).mark)).toEqual(['●', '✕', '◐', '?'])
    expect(accessMark(readiness({ robots: 'unreachable' }), 'GPTBot').text).toBe('robots.txt could not be read')
    expect(accessMark(undefined, 'GPTBot')).toEqual({ mark: '–', text: 'Not scanned', tone: 'unknown' })
  })
})

describe('the scans and the headline', () => {
  it('say when the scans were made', () => {
    const rows = [readinessRow('A', { scan: summary({ finishedAt: '2026-10-24T08:00:00Z' }) }), readinessRow('B', { scan: summary({ finishedAt: '2026-10-27T08:00:00Z' }) })]
    expect(scanRange(rows)).toBe('Last scans from 24 October 2026 to 27 October 2026')
    expect(scanRange(rows.slice(0, 1))).toBe('Scanned on 24 October 2026')
    expect(scanRange([readinessRow('C', { scan: undefined })])).toBeNull()
  })

  it('names the leader and where Tribelt stands', () => {
    const rows = [
      readinessRow('Tribelt', { role: 'tribelt', readiness: readiness({ llmsFullTxt: false, markdown: false }) }),
      readinessRow('Märtens'),
      readinessRow('Ashworth'),
      readinessRow('Mirror', { role: 'mirror' }),
    ]
    expect(readinessHeadline(rows)).toBe('Ashworth leads with 5 of five checks. Tribelt passes 3.')
    expect(readinessHeadline(rows.slice(1, 2))).toBe('Märtens leads with 5 of five checks.')
    expect(readinessHeadline(rows.slice(0, 1))).toBe('Who lets search engines and AI assistants in.')
  })
})
