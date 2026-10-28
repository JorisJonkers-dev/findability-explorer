import { describe, expect, it } from 'vitest'
import type { AiSearchSummary, Plan, ReadinessRank } from '../../../infrastructure/api/types.gen'
import { bots, readiness } from '../../../testing/fixtures'
import { bestSites, checkGroups, CHECK_FIX, CHECK_WHAT, fixFor, LETS_IN, readinessFigures, readinessTitle, scoreWords, siteChecks, STATE_WORDS, tribeltStatus } from './checks'

const checks: AiSearchSummary['checks'] = [
  { check: 'crawlers', scored: true, passing: 2, tribelt: false },
  { check: 'sitemap', scored: true, passing: 5, tribelt: true },
  { check: 'llmsTxt', scored: true, passing: 0 },
  { check: 'product', scored: false, passing: 4, tribelt: true },
]

const rank = (name: string, role: ReadinessRank['role'], score: number): ReadinessRank => ({ competitorId: name.toLowerCase(), name, role, score })

describe('the checks in groups', () => {
  it('lists the scored checks first, then the ones checked beside them, with Tribelt’s standing', () => {
    const groups = checkGroups({ checks, sites: 5 }, 5)
    expect(groups.map((g) => [g.title, g.aria, g.checks.map((c) => c.kind)])).toEqual([
      ['The five checks in the score', 'The five checks in the score', ['crawlers', 'sitemap', 'llmsTxt']],
      ['Also checked', 'Also checked, not in the score', ['product']],
    ])
    expect(groups[0]?.checks[0]).toEqual({
      kind: 'crawlers',
      label: 'Lets every AI crawler in',
      passing: 2,
      sites: 5,
      share: 0.4,
      tribelt: false,
      status: 'Tribelt does not pass',
      aria: 'Lets every AI crawler in: 2 of 5 sites pass. Tribelt does not pass.',
    })
    expect(groups[0]?.checks.map((c) => c.status)).toEqual(['Tribelt does not pass', 'Tribelt passes', 'Not scanned yet'])
  })

  it('drops a group with no checks, and has none before the summary loads', () => {
    expect(checkGroups({ checks: checks.slice(0, 2), sites: 5 }, 5).map((g) => g.title)).toEqual(['The five checks in the score'])
    expect(checkGroups({ checks, sites: 0 }, 5)[0]?.checks[0]?.share).toBe(0)
    expect(checkGroups(undefined, 5)).toEqual([])
  })

  it('says what is said of tribelt.nl', () => {
    expect([tribeltStatus(null), tribeltStatus(true), tribeltStatus(false)]).toEqual(['Not scanned yet', 'Tribelt passes', 'Tribelt does not pass'])
  })

  it('explains every check, and links the fix of the ones a recommendation fixes', () => {
    expect(Object.keys(CHECK_WHAT)).toHaveLength(8)
    expect(Object.keys(CHECK_FIX)).toEqual(['sitemap', 'sitemapLanguages', 'llmsTxt', 'llmsFullTxt', 'markdown', 'product'])
  })
})

describe('the link to a fix', () => {
  const plan: Pick<Plan, 'improvements'> = {
    improvements: [{ id: 'other', title: 'Other' }, { id: 'llms-files', title: 'Add llms.txt and llms-full.txt' }] as Plan['improvements'],
  }

  it('numbers the recommendation that fixes a check', () => {
    expect(fixFor('llmsTxt', plan)).toEqual({ to: '/recommendations/llms-files', label: 'See recommendation 2, add llms.txt and llms-full.txt' })
  })

  it('has none for a check no recommendation fixes, a recommendation not in the plan, or no plan', () => {
    expect(fixFor('crawlers', plan)).toBeUndefined()
    expect(fixFor('markdown', plan)).toBeUndefined()
    expect(fixFor('llmsTxt', undefined)).toBeUndefined()
  })
})

describe('the headline', () => {
  it('says how many checks Tribelt passes, against the best competitor', () => {
    expect(readinessTitle([], 5)).toBe('How ready the sites are for AI assistants')
    expect(readinessTitle([rank('Tribelt', 'tribelt', 3.5)], 5)).toBe('Tribelt passes three and a half of the five readiness checks')
    expect(readinessTitle([rank('Tribelt', 'tribelt', 4), rank('A', 'competitor', 3)], 5)).toBe('Tribelt passes four of the five readiness checks, more than any competitor')
    expect(readinessTitle([rank('Tribelt', 'tribelt', 3), rank('A', 'competitor', 3)], 5)).toBe('Tribelt passes three of the five readiness checks, matching its best competitors')
    expect(readinessTitle([rank('Tribelt', 'tribelt', 0.5), rank('A', 'competitor', 3)], 5)).toBe('Tribelt passes half of one of the five readiness checks, fewer than its best competitors')
    expect(readinessTitle([rank('Tribelt', 'tribelt', 0), rank('A', 'competitor', 3)], 5)).toBe('Tribelt passes none of the five readiness checks, fewer than its best competitors')
  })

  it('writes a score in words', () => {
    expect([0, 0.5, 1, 2.5, 3].map(scoreWords)).toEqual(['none', 'half of one', 'one', 'two and a half', 'three'])
  })
})

describe('the sites with the best score', () => {
  it('lists the best sites, leaving the mirror out, and adds Tribelt when it is not among them', () => {
    const ranking = [rank('A', 'competitor', 5), rank('Mirror', 'mirror', 5), rank('B', 'competitor', 5), rank('Tribelt', 'tribelt', 3)]
    const best = bestSites(ranking)
    expect([best.top, best.sites.map((s) => s.name), best.tribeltAmong]).toEqual([5, ['A', 'B', 'Tribelt'], false])
    const among = bestSites([rank('A', 'competitor', 3), rank('Tribelt', 'tribelt', 3)])
    expect([among.sites.map((s) => s.name), among.tribeltAmong]).toEqual([['A', 'Tribelt'], true])
    expect(bestSites([rank('A', 'competitor', 3)]).tribeltAmong).toBe(false)
    expect(bestSites([])).toEqual({ top: 0, sites: [], tribeltAmong: false })
  })
})

describe('the figures', () => {
  it('count the sites, the checks Tribelt passes and the sites that share the best score', () => {
    const ranking = [rank('A', 'competitor', 5), rank('B', 'competitor', 5), rank('Tribelt', 'tribelt', 3.5)]
    expect(readinessFigures({ ranking, sites: 41, readinessChecks: 5 })).toEqual([
      { big: '41', text: 'sites checked', accent: false },
      { big: '3½ of 5', text: 'checks that tribelt.nl passes', accent: true },
      { big: '2', text: 'sites share the best score', accent: false },
    ])
  })

  it('say when Tribelt is among the best, or alone', () => {
    const tied = readinessFigures({ ranking: [rank('A', 'competitor', 4), rank('Tribelt', 'tribelt', 4)], sites: 1, readinessChecks: 5 })
    expect([tied[0]?.text, tied[2]?.text]).toEqual(['site checked', 'sites share the best score, Tribelt among them'])
    expect(readinessFigures({ ranking: [rank('Tribelt', 'tribelt', 4)], sites: 2, readinessChecks: 5 })[2]?.text).toBe('site has the best score, and it is Tribelt')
    const alone = readinessFigures({ ranking: [rank('A', 'competitor', 4), rank('Tribelt', 'tribelt', 1)], sites: 2, readinessChecks: 5 })
    expect([alone[1]?.big, alone[2]?.text]).toEqual(['1 of 5', 'site has the best score'])
    expect(readinessFigures({ ranking: [], sites: 0, readinessChecks: 5 })[1]?.big).toBe('–')
  })
})

describe("a site's five checks", () => {
  const states = (r: Parameters<typeof siteChecks>[0]) => siteChecks(r).map((c) => c.state)

  it('read the way the server scores them', () => {
    expect(siteChecks({ readiness: readiness() }).map((c) => c.label)).toEqual(['Lets every AI crawler in', 'Has a sitemap', 'Has llms.txt', 'Has llms-full.txt', 'Has Markdown pages'])
    expect(states({ readiness: readiness() })).toEqual(['yes', 'yes', 'yes', 'yes', 'yes'])
    expect(states({ readiness: readiness({ sitemap: { declared: false, files: 0, addresses: 0, languages: [] }, llmsTxt: { present: false, bytes: 0, links: 0 }, llmsFullTxt: false, markdown: false }) })).toEqual(['yes', 'no', 'no', 'no', 'no'])
  })

  it('call the crawlers partly let in, kept out, or not known', () => {
    expect(states({ readiness: readiness({ bots: bots({ GPTBot: 'blocked' }) }) })[0]).toBe('partly')
    expect(states({ readiness: readiness({ bots: readiness().bots.map((b) => ({ ...b, access: 'blocked' as const })) }) })[0]).toBe('no')
    expect(states({ readiness: readiness({ robots: 'unreachable' }) })[0]).toBe('unknown')
    expect(states({ readiness: undefined })).toEqual(['unknown', 'unknown', 'unknown', 'unknown', 'unknown'])
  })

  it('have words for every state and every crawler access', () => {
    expect(STATE_WORDS).toEqual({ yes: 'Yes', partly: 'Partly', no: 'No', unknown: 'Not known' })
    expect(Object.keys(LETS_IN)).toEqual(['allowed', 'partly', 'blocked', 'unknown'])
  })
})
