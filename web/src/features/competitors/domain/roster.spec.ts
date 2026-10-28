import { describe, expect, it } from 'vitest'
import { bots, competitor, promptResults, readiness, summary } from '../../../testing/fixtures'
import { chipsOf, keeps, lastScanOf, parseRosterFilter, placeOf, roleLabel, rosterOrder, shareOf } from './roster'

const ALL = Object.fromEntries(bots().map((b) => [b.bot, 'blocked' as const]))

describe('the readiness chips', () => {
  it('mark the five checks by shape', () => {
    expect(chipsOf(readiness()).map((c) => `${c.label}${c.mark}`)).toEqual(['crawlers✓', 'sitemap✓', 'llms.txt✓', 'llms-full✓', 'Markdown✓'])
    const some = chipsOf(readiness({ bots: bots({ GPTBot: 'blocked' }), markdown: false, llmsFullTxt: false }))
    expect(some.map((c) => c.mark)).toEqual(['Some', '✓', '✓', '✕', '✕'])
    expect(some[0]?.tip).toBe('Keeps some AI crawlers out')
    expect(chipsOf(readiness({ bots: bots(ALL) }))[0]?.tip).toBe('Keeps every AI crawler out')
    expect(chipsOf(readiness({ robots: 'unreachable', bots: [] }))[0]?.tip).toBe('robots.txt could not be read')
    const none = chipsOf(readiness({ sitemap: { declared: false, files: 0, addresses: 0, languages: [] }, llmsTxt: { present: false, bytes: 0, links: 0 } }))
    expect(none.slice(1, 3).map((c) => c.tip)).toEqual(['No sitemap', 'No llms.txt'])
    expect(chipsOf(undefined)).toEqual([])
  })
})

describe("a company's share of the answers", () => {
  it('comes from the newest run with answers', () => {
    const results = promptResults()
    expect(shareOf(results, 'twentebelt')).toBe(0.75)
    expect(shareOf(results, 'esfo')).toBe(0)
    expect(shareOf(promptResults({ runs: [] }), 'twentebelt')).toBeUndefined()
    expect(shareOf(undefined, 'twentebelt')).toBeUndefined()
  })
})

describe('the list', () => {
  it('filters by role and by never scanned', () => {
    const us = competitor('Tribelt', { role: 'tribelt', scans: 3 })
    const group = competitor('De Spiraal', { role: 'group', scans: 1 })
    const never = competitor('Unibelt', { scans: 0 })
    const scanned = competitor('ESFO', { scans: 2 })
    const kept = (f: string) => [us, group, never, scanned].filter((c) => keeps(c, parseRosterFilter(f))).map((c) => c.name)
    expect(kept('all')).toEqual(['Tribelt', 'De Spiraal', 'Unibelt', 'ESFO'])
    expect(kept('competitor')).toEqual(['Unibelt', 'ESFO'])
    expect(kept('ours')).toEqual(['Tribelt', 'De Spiraal'])
    expect(kept('never')).toEqual(['Unibelt'])
    expect(parseRosterFilter('nonsense')).toBe('all')
  })

  it('lets only an administrator show what Haiku is unsure of, or archived', () => {
    expect(parseRosterFilter('unsure')).toBe('all')
    expect(parseRosterFilter('unsure', true)).toBe('unsure')
    expect(parseRosterFilter('archived', true)).toBe('archived')
    const c = competitor('Intralox')
    expect(keeps(c, 'unsure', true)).toBe(true)
    expect(keeps(c, 'unsure')).toBe(false)
    expect(keeps(c, 'archived', true)).toBe(false)
  })

  it('puts Tribelt first, then the most named competitors, then the group', () => {
    const list = [competitor('D'), competitor('B'), competitor('De Spiraal', { role: 'group' }), competitor('A'), competitor('C'), competitor('Tribelt', { role: 'tribelt' })]
    const share = (c: { name: string }) => ({ A: 0.1, C: 0.5 })[c.name as 'A' | 'C']
    expect(rosterOrder(list, share).map((c) => c.name)).toEqual(['Tribelt', 'C', 'A', 'B', 'D', 'De Spiraal'])
  })

  it('writes place, role and the last scan', () => {
    expect(placeOf({ country: 'DE' })).toBe('Germany')
    expect(placeOf({ country: 'unknown' })).toBe('unknown')
    expect(['tribelt', 'mirror', 'group', 'competitor'].map((role) => roleLabel({ role } as never))).toEqual(['Us', 'Mirror', 'Group company', 'Competitor'])
    expect(lastScanOf({ scans: 0 })).toEqual({ text: 'Never', attention: true })
    expect(lastScanOf({ scans: 1, lastScan: summary({ status: 'running' }) }).text).toBe('Scanning')
    expect(lastScanOf({ scans: 1, lastScan: summary({ blocked: true }) })).toEqual({ text: 'Blocked', attention: true })
    expect(lastScanOf({ scans: 1, lastScan: summary() })).toEqual({ text: '22 Oct', attention: false })
    expect(lastScanOf({ scans: 1, lastScan: summary({ status: 'failed', finishedAt: undefined }) })).toEqual({ text: '22 Oct', attention: true })
  })
})
