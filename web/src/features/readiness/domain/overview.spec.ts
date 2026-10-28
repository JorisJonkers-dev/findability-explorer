import { describe, expect, it } from 'vitest'
import { bots, readiness, readinessRow, summary } from '../../../testing/fixtures'
import { accessMark, checkedAgain, checkSummary, readinessCheckLine, readinessHeadline, scanRange, scanned } from './overview'

describe('the summary', () => {
  it('counts the scanned sites passing each check, with Tribelt beside it', () => {
    const rows = [
      readinessRow('Tribelt', { role: 'tribelt', readiness: readiness({ llmsTxt: { present: false, bytes: 0, links: 0 } }) }),
      readinessRow('Märtens'),
      readinessRow('Twentebelt', { readiness: readiness({ bots: bots({ GPTBot: 'blocked' }), advice: [] }) }),
      readinessRow('Never', { readiness: undefined, scan: undefined }),
    ]
    expect(scanned(rows)).toHaveLength(3)
    expect(checkSummary(rows).map((c) => [c.label, c.counted, c.passing, c.sites, c.tribelt])).toEqual([
      ['Lets every AI crawler in', true, 2, 3, true],
      ['Has a sitemap', true, 3, 3, true],
      ['Has llms.txt', true, 2, 3, false],
      ['Has llms-full.txt', true, 3, 3, true],
      ['Has Markdown pages', true, 3, 3, true],
      ['Sitemap in Dutch, German and English', false, 2, 3, true],
      ['Organization structured data', false, 2, 3, true],
      ['Product structured data', false, 2, 3, true],
    ])
    expect(checkSummary([readinessRow('Märtens')])[0]?.tribelt).toBeNull()
  })
})

describe('the access marks', () => {
  it('differ in shape as well as colour', () => {
    const r = readiness({ bots: bots({ GPTBot: 'blocked', CCBot: 'partly' }) })
    expect(['Googlebot', 'GPTBot', 'CCBot', 'Nobody'].map((b) => accessMark(r, b).mark)).toEqual(['Allowed', 'Blocked', 'Partly', 'Unknown'])
    expect(accessMark(r, 'CCBot').text).toBe('CCBot may read the site, except some pages robots.txt closes.')
    expect(accessMark(r, 'GPTBot').text).toBe('GPTBot may not read any page.')
    expect(accessMark(r, 'Googlebot').text).toBe('Googlebot may read every page.')
    expect(accessMark(r, 'Nobody').text).toBe('Nobody was not checked.')
    expect(accessMark(readiness({ robots: 'unreachable' }), 'GPTBot')).toEqual({ mark: 'Unknown', text: 'robots.txt could not be read.', tone: 'unknown' })
    expect(accessMark(undefined, 'GPTBot')).toEqual({ mark: '–', text: 'Not scanned yet.', tone: 'unknown' })
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
    const costacurta = { competitorId: 'costacurta', name: 'Costacurta', role: 'competitor' as const, score: 3 }
    const tribelt = (score: number) => ({ competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt' as const, score })
    expect(readinessHeadline([costacurta, tribelt(3)])).toBe('Tribelt shares the lead with Costacurta, at 3 of five checks.')
    expect(readinessHeadline([costacurta, tribelt(4)])).toBe('Tribelt leads with 4 of five checks. Next is Costacurta with 3.')
    expect(readinessHeadline([costacurta, { ...costacurta, competitorId: 'audubon', name: 'Audubon' }, tribelt(2)])).toBe(
      'Costacurta and Audubon lead with 3 of five checks. Tribelt passes 2.',
    )
  })

})

describe('checking AI readiness again', () => {
  it('says how far a check has come, and how it ended', () => {
    expect(readinessCheckLine(undefined)).toBe('')
    expect(readinessCheckLine({ running: false, checked: 0, failed: 0, total: 0 })).toBe('')
    expect(readinessCheckLine({ running: true, checked: 0, failed: 0, total: 1 })).toBe('Checking 0 of 1 site again.')
    expect(readinessCheckLine({ running: true, checked: 12, failed: 0, total: 40 })).toBe('Checking 12 of 40 sites again.')
    expect(readinessCheckLine({ running: false, checked: 40, failed: 0, total: 40 })).toBe('Checked 40 sites again.')
    expect(readinessCheckLine({ running: false, checked: 40, failed: 1, total: 40 })).toBe("Checked 39 sites again. 1 site could not be read and keeps its previous result.")
    expect(readinessCheckLine({ running: false, checked: 2, failed: 2, total: 2 })).toBe("Checked 0 sites again. 2 sites could not be read and keep their previous result.")
  })

  it('names the newest day a site was checked again', () => {
    expect(checkedAgain([readinessRow('Märtens'), readinessRow('Never', { readiness: undefined })])).toBeNull()
    expect(
      checkedAgain([
        readinessRow('Märtens', { readiness: readiness({ checkedAt: '2026-10-26T09:00:00Z' }) }),
        readinessRow('Twentebelt', { readiness: readiness({ checkedAt: '2026-10-27T18:30:00Z' }) }),
      ]),
    ).toBe('AI readiness checked again on 27 October 2026')
  })
})
