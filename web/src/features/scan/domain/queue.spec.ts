import { describe, expect, it } from 'vitest'
import { competitor, summary } from '../../../testing/fixtures'
import { lastScanned, newestScans, pagesRead, rescanGroups, rescanRequest, scanEstimate, scanDay, scanQueue, scanStamp, stoppedLine } from './queue'

const NOW = new Date('2026-10-28T12:00:00Z')

describe('the scan queue', () => {
  it('splits the scans in progress into running and waiting, oldest first', () => {
    const scans = [
      summary({ id: 'c', status: 'queued', createdAt: '2026-10-28T10:02:00Z' }),
      summary({ id: 'b', status: 'running', createdAt: '2026-10-28T10:01:00Z' }),
      summary({ id: 'a', status: 'queued', createdAt: '2026-10-28T10:00:00Z' }),
      summary({ id: 'd', status: 'done' }),
    ]
    const q = scanQueue(scans)
    expect(q.running.map((s) => s.id)).toEqual(['b'])
    expect(q.waiting.map((s) => s.id)).toEqual(['a', 'c'])
  })
})

describe('the scans of the companies', () => {
  const companies = [
    competitor('Old', { lastScan: summary({ id: 'o', createdAt: '2026-09-01T08:00:00Z', number: 2 }), scans: 2 }),
    competitor('New', { lastScan: summary({ id: 'n', createdAt: '2026-10-27T08:00:00Z' }), scans: 1 }),
    competitor('Never', { scans: 0 }),
    competitor('Gone', { archived: true, lastScan: summary({ id: 'g', createdAt: '2026-10-28T08:00:00Z' }), scans: 1 }),
    competitor('Mirror', { role: 'mirror', lastScan: summary({ id: 'm', createdAt: '2026-08-01T08:00:00Z' }), scans: 1 }),
  ]

  it('lists each company’s newest scan, newest first, the archived ones too', () => {
    expect(newestScans(companies).map((r) => [r.company.name, r.scan.id])).toEqual([
      ['Gone', 'g'],
      ['New', 'n'],
      ['Old', 'o'],
      ['Mirror', 'm'],
    ])
    expect(lastScanned(companies)?.company.name).toBe('Gone')
    expect(lastScanned([])).toBeUndefined()
  })

  it('groups the competitors to rescan: all, over 30 days old, and never scanned', () => {
    const g = rescanGroups(companies, NOW)
    expect(g.all.map((c) => c.name)).toEqual(['Old', 'New', 'Never'])
    expect(g.stale.map((c) => c.name)).toEqual(['Old', 'Never'])
    expect(g.never.map((c) => c.name)).toEqual(['Never'])
  })

  it('names the companies of a group, so the scans queued are the ones counted', () => {
    const g = rescanGroups(companies, NOW)
    expect(rescanRequest(g.all, 40)).toEqual({ competitorIds: ['old', 'new', 'never'], maxPages: 40 })
    expect(rescanRequest(g.stale, 40)).toEqual({ competitorIds: ['old', 'never'], maxPages: 40 })
  })

  it('leaves out a competitor without a website', () => {
    expect(rescanGroups([competitor('Nowhere', { websiteUrl: undefined })], NOW).all).toEqual([])
  })
})

describe('a scan in words', () => {
  it('dates a scan to the minute, in UTC', () => {
    expect(scanStamp('2026-10-28T05:48:10Z')).toBe('28 October 2026, 05:48')
    expect(scanDay('2026-10-28T23:48:10Z')).toBe('28 October 2026')
  })

  it('says how many pages it read', () => {
    expect(pagesRead(summary({ pagesCrawled: 22, maxPages: 60 }))).toBe('22 of 60')
  })

  it('says how a scan stopped by hand ended', () => {
    expect(stoppedLine(summary({ status: 'failed', stoppedByHand: true, pagesCrawled: 4 }))).toBe('Stopped by hand. It keeps the 4 pages it read.')
    expect(stoppedLine(summary({ status: 'failed', stoppedByHand: true, pagesCrawled: 1 }))).toBe('Stopped by hand. It keeps the 1 page it read.')
    expect(stoppedLine(summary({ status: 'failed', stoppedByHand: true, pagesCrawled: 0 }))).toBe('Taken off the queue by hand before it read a page.')
    expect(stoppedLine(summary({ status: 'failed' }))).toBeNull()
  })
})

describe('what scans cost', () => {
  const budget = { monthlyTokens: 10_000_000, usedThisMonth: 48_211, averageTokensPerScan: 29_137 }

  it('estimates the tokens from the average scan, and says what is left', () => {
    expect(scanEstimate(budget, 1)).toBe("About 29,137 tokens. 9,951,789 of this month's tokens are left.")
    expect(scanEstimate(budget, 2)).toBe("About 58,274 tokens for 2 scans. 9,951,789 of this month's tokens are left.")
    expect(scanEstimate({ ...budget, usedThisMonth: 20_000_000 }, 1)).toBe("About 29,137 tokens. 0 of this month's tokens are left.")
  })

  it('says so when no scan has matched products yet, and nothing before the budget is known', () => {
    expect(scanEstimate({ ...budget, averageTokensPerScan: 0 }, 1)).toBe("No scan has matched products yet, so there is no estimate. 9,951,789 of this month's tokens are left.")
    expect(scanEstimate(undefined, 1)).toBe('')
  })
})
