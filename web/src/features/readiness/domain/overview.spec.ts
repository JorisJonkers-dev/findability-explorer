import { describe, expect, it } from 'vitest'
import { bots, readiness, readinessRow, summary } from '../../../testing/fixtures'
import { accessMark, checkSummary, readinessHeadline, scanRange, scanned } from './overview'

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

  it('names the leader of the backend ranking and where Tribelt stands', () => {
    const ranking = [
      { competitorId: 'martens', name: 'Märtens', role: 'competitor' as const, score: 5 },
      { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt' as const, score: 3.5 },
    ]
    expect(readinessHeadline(ranking)).toBe('Märtens leads with 5 of five checks. Tribelt passes 3½.')
    expect(readinessHeadline(ranking.slice(0, 1))).toBe('Märtens leads with 5 of five checks.')
    expect(readinessHeadline(ranking.slice(1))).toBe('Who lets search engines and AI assistants in.')
  })

})
