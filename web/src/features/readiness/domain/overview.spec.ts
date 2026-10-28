import { describe, expect, it } from 'vitest'
import { bots, readiness, readinessRow, summary } from '../../../testing/fixtures'
import { ACCESS_LEGEND, accessMark, checkedAgain, hasMark, readinessCheckLine, scanNote, scanRange } from './overview'

describe('the access marks', () => {
  it('differ in shape as well as colour', () => {
    const r = readiness({ bots: bots({ GPTBot: 'blocked', CCBot: 'partly' }) })
    expect(['Googlebot', 'GPTBot', 'CCBot', 'Nobody'].map((b) => accessMark(r, b).mark)).toEqual(['✓', '✕', '~', '–'])
    expect(accessMark(r, 'CCBot').text).toBe('CCBot may read the site, except some pages robots.txt closes.')
    expect(accessMark(r, 'GPTBot').text).toBe('GPTBot may not read any page.')
    expect(accessMark(r, 'Googlebot').text).toBe('Googlebot may read every page.')
    expect(accessMark(r, 'Nobody').text).toBe('Nobody was not checked.')
    expect(accessMark(readiness({ robots: 'unreachable' }), 'GPTBot')).toEqual({ mark: '–', text: 'robots.txt could not be read.', tone: 'unknown' })
    expect(accessMark(undefined, 'GPTBot')).toEqual({ mark: '–', text: 'Not scanned yet.', tone: 'unknown' })
  })

  it('has a legend line for each symbol the crawler columns show', () => {
    expect(ACCESS_LEGEND.map((l) => l.mark)).toEqual(['✓', '~', '✕', '–'])
  })

  it('ticks and crosses the files a site has', () => {
    expect(hasMark(true, 'llms.txt')).toEqual({ mark: '✓', text: 'Has llms.txt.', tone: 'ok' })
    expect(hasMark(false, 'Markdown pages')).toEqual({ mark: '✕', text: 'No Markdown pages.', tone: 'no' })
  })
})

describe('the scans', () => {
  it('say when the scans were made', () => {
    const rows = [readinessRow('A', { scan: summary({ finishedAt: '2026-10-24T08:00:00Z' }) }), readinessRow('B', { scan: summary({ finishedAt: '2026-10-27T08:00:00Z' }) })]
    expect(scanRange(rows)).toBe('Last scans from 24 October 2026 to 27 October 2026')
    expect(scanRange(rows.slice(0, 1))).toBe('Scanned on 24 October 2026')
    expect(scanRange([readinessRow('C', { scan: undefined })])).toBeNull()
  })

  it('joins the count of sites and the scan days as sentences', () => {
    const rows = [
      readinessRow('A', { scan: summary({ finishedAt: '2026-10-24T08:00:00Z' }), readiness: readiness({ checkedAt: '2026-10-27T18:30:00Z' }) }),
      readinessRow('B', { scan: summary({ finishedAt: '2026-10-27T08:00:00Z' }) }),
    ]
    expect(scanNote(rows)).toBe('Last scans from 24 October 2026 to 27 October 2026. AI readiness checked again on 27 October 2026.')
    expect(scanNote(rows.slice(1))).toBe('Scanned on 27 October 2026.')
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
