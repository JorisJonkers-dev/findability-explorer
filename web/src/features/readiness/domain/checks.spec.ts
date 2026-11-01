import { describe, expect, it } from 'vitest'
import type { AiReadiness, Plan } from '../../../infrastructure/api/types.gen'
import { bots, readiness, readinessRow } from '../../../testing/fixtures'
import {
  CHECK_NAMES,
  CHECK_WHY,
  checkFix,
  checkStates,
  COMPARE_WORDS,
  compareChecks,
  compareIntro,
  everySite,
  platformOf,
  readinessDataLine,
  SCORED_CHECKS,
  siteName,
  siteScore,
  STATE_WORDS,
} from './checks'

const without = (r: AiReadiness, check: string): AiReadiness => ({ ...r, advice: r.advice.map((a) => (a.check === check ? { ...a, ok: false } : a)) })

describe('the eight checks', () => {
  it('names and explains each, in the order the pages list them', () => {
    expect(SCORED_CHECKS).toEqual(['crawlers', 'sitemap', 'sitemapLanguages', 'languages', 'organization', 'llmsTxt', 'llmsFullTxt', 'markdown'])
    expect(SCORED_CHECKS.map((c) => CHECK_NAMES[c])).toEqual([
      'AI crawlers let in',
      'A sitemap',
      'A sitemap in every language',
      'Pages in Dutch, German, and English',
      'Structured data about the company',
      'llms.txt',
      'llms-full.txt',
      'Markdown pages',
    ])
    expect(CHECK_WHY.crawlers).toBe('robots.txt lets search and AI crawlers read the site, so assistants can find and learn its pages.')
    expect(STATE_WORDS).toEqual({ yes: 'Passed.', partly: 'Partly passed, as some crawlers are kept out.', no: 'Not passed.', unknown: 'Not known, as robots.txt could not be read.' })
  })

  it('reads how a site stands on each, the way the server scores it', () => {
    expect(Object.values(checkStates(readiness()))).toEqual(Array(8).fill('yes'))
    const gappy = without(
      without(
        without(
          readiness({
            bots: bots({ GPTBot: 'blocked' }),
            sitemap: { declared: true, files: 0, addresses: 0, languages: [] },
            llmsTxt: { present: false, bytes: 0, links: 0 },
            llmsFullTxt: false,
            markdown: false,
          }),
          'organization',
        ),
        'languages',
      ),
      'sitemapLanguages',
    )
    expect(checkStates(gappy)).toEqual({ crawlers: 'partly', sitemap: 'no', sitemapLanguages: 'no', languages: 'no', organization: 'no', llmsTxt: 'no', llmsFullTxt: 'no', markdown: 'no' })
    expect(checkStates(readiness({ bots: bots(Object.fromEntries(bots().map((b) => [b.bot, 'blocked']))) })).crawlers).toBe('no')
    expect(checkStates(readiness({ robots: 'unreachable', bots: [] })).crawlers).toBe('unknown')
    expect(checkStates(readiness({ bots: [] })).crawlers).toBe('unknown')
  })

  it('scores a site, the crawlers counting half when some are kept out', () => {
    expect(siteScore(readiness())).toBe(8)
    expect(siteScore(readiness({ bots: bots({ GPTBot: 'blocked' }), markdown: false }))).toBe(6.5)
  })

  it('links a failed check to the recommendation that fixes it, and the crawlers to the site page', () => {
    const plan: Pick<Plan, 'improvements'> = {
      improvements: [
        { id: 'llms-files', title: 'Add llms.txt and llms-full.txt' },
        { id: 'structured-data', title: 'Add structured data to the belt pages' },
      ] as Plan['improvements'],
    }
    expect(checkFix('llmsFullTxt', plan, 'tribelt')).toEqual({ to: '/recommendations/llms-files', label: 'Recommended fix, add llms.txt and llms-full.txt' })
    expect(checkFix('organization', plan, 'tribelt')?.label).toBe('Recommended fix, add structured data to the belt pages')
    expect(checkFix('crawlers', plan, 'tribelt')).toEqual({ to: '/ai-readiness/tribelt', label: 'See which crawlers tribelt.nl turns away' })
    expect(checkFix('markdown', plan, 'tribelt')).toBeUndefined()
    expect(checkFix('markdown', undefined, 'tribelt')).toBeUndefined()
  })
})

const tribelt = readinessRow('Tribelt B.V.', { competitorId: 'tribelt', role: 'tribelt', readiness: readiness({ llmsFullTxt: false, markdown: false }) })
const competitors = [
  readinessRow('ESFO'),
  readinessRow('Twentebelt', { readiness: readiness({ llmsTxt: { present: false, bytes: 0, links: 0 }, markdown: false, llmsFullTxt: false }) }),
  readinessRow('Unibelt', { readiness: undefined }),
]
const group = readinessRow('Lesjöfors', { role: 'group', readiness: readiness({ stack: {} }) })
const mirror = readinessRow('Mirror of tribelt.nl', { competitorId: 'tribelt-mirror', role: 'mirror' })

describe('tribelt.nl against the competitors', () => {
  it('counts, per check, the competitors with a recorded readiness that pass it', () => {
    const c = compareChecks([group, ...competitors, tribelt, mirror])
    expect(c?.competitors).toBe(2)
    expect(c?.rows.find((r) => r.check === 'llmsTxt')).toEqual({ check: 'llmsTxt', name: 'llms.txt', tribelt: 'yes', passing: 1, share: 0.5 })
    expect(c?.rows.find((r) => r.check === 'markdown')).toMatchObject({ tribelt: 'no', passing: 1 })
    expect(c?.rows.map((r) => r.check)).toEqual(SCORED_CHECKS)
  })

  it('says where tribelt.nl is ahead and where it is behind', () => {
    const c = compareChecks([...competitors, tribelt])
    expect(c && compareIntro(c)).toBe(
      'For each check, how many of the 2 competitors pass it, and whether tribelt.nl does. tribelt.nl passes no checks that most competitors fail. It fails 2 that some competitors pass.',
    )
    const noLlms = { llmsTxt: { present: false, bytes: 0, links: 0 }, llmsFullTxt: false }
    const ahead = compareChecks([
      readinessRow('W', { readiness: readiness({ ...noLlms, markdown: false }) }),
      readinessRow('X', { readiness: readiness({ llmsFullTxt: false, markdown: false }) }),
      readinessRow('Y', { readiness: readiness(noLlms) }),
      readinessRow('Z', { readiness: readiness({ ...noLlms, markdown: false }) }),
      tribelt,
    ])
    expect(ahead && compareIntro(ahead)).toBe(
      'For each check, how many of the 4 competitors pass it, and whether tribelt.nl does. tribelt.nl passes 1 check that most competitors fail. It fails 1 that some competitors pass.',
    )
    const row = { check: 'sitemap', name: 'A sitemap', tribelt: 'yes', passing: 0, share: 0 } as const
    expect(compareIntro({ competitors: 3, rows: [row, row] })).toContain('tribelt.nl passes 2 checks that most competitors fail.')
    expect(compareIntro({ competitors: 3, rows: [] })).toBe(
      'For each check, how many of the 3 competitors pass it, and whether tribelt.nl does. tribelt.nl passes no checks that most competitors fail. It fails none that some competitors pass.',
    )
  })

  it('names each standing in the comparison, and does not count a check it cannot tell as failed', () => {
    expect(COMPARE_WORDS).toEqual({ yes: 'Passes', partly: 'Partly', no: 'Fails', unknown: 'Not known' })
    const row = (tribelt: 'unknown' | 'partly') => ({ check: 'crawlers', name: 'AI crawlers let in', tribelt, passing: 2, share: 1 }) as const
    expect(compareIntro({ competitors: 2, rows: [row('unknown')] })).toContain('It fails none that some competitors pass.')
    expect(compareIntro({ competitors: 2, rows: [row('partly')] })).toContain('It fails 1 that some competitors pass.')
  })

  it('has nothing to compare before tribelt.nl is scanned', () => {
    expect(compareChecks(competitors)).toBeUndefined()
    const none = compareChecks([tribelt])
    expect(none?.competitors).toBe(0)
    expect(none?.rows[0]?.share).toBe(0)
  })
})

describe('every site', () => {
  const rows = [group, ...competitors, tribelt, mirror]
  const names = (r: ReturnType<typeof everySite>) => r.map((x) => x.name)

  it('keeps the competitors with a recorded readiness, Tribelt first, highest score first', () => {
    expect(names(everySite(rows, 'competitors', { key: 'score', direction: 'desc' }, ''))).toEqual(['Tribelt B.V.', 'ESFO', 'Twentebelt'])
  })

  it('shows every site with a recorded readiness when asked, and finds one by name', () => {
    expect(names(everySite(rows, 'all', { key: 'name', direction: 'asc' }, ''))).toEqual(['Tribelt B.V.', 'ESFO', 'Lesjöfors', 'Mirror of tribelt.nl', 'Twentebelt'])
    expect(names(everySite(rows, 'all', null, 'twente'))).toEqual(['Twentebelt'])
    expect(names(everySite(rows, 'all', null, 'tribelt'))).toEqual(['Tribelt B.V.', 'Mirror of tribelt.nl'])
  })

  it('sorts by a check and by the platform', () => {
    expect(names(everySite(rows, 'competitors', { key: 'markdown', direction: 'asc' }, ''))).toEqual(['Tribelt B.V.', 'Twentebelt', 'ESFO'])
    expect(names(everySite(rows, 'all', { key: 'platform', direction: 'asc' }, ''))).toEqual(['Tribelt B.V.', 'Lesjöfors', 'ESFO', 'Twentebelt', 'Mirror of tribelt.nl'])
    const partly = readinessRow('Partly', { readiness: readiness({ bots: bots({ GPTBot: 'blocked' }) }) })
    expect(names(everySite([...rows, partly], 'competitors', { key: 'crawlers', direction: 'asc' }, ''))).toEqual(['Tribelt B.V.', 'Partly', 'ESFO', 'Twentebelt'])
  })

  it('names Tribelt as Tribelt, and says the platform or that none was found', () => {
    expect(siteName(tribelt)).toBe('Tribelt')
    expect(siteName(group)).toBe('Lesjöfors')
    expect(platformOf(readiness())).toBe('WordPress')
    expect(platformOf(readiness({ stack: {} }))).toBe('Not detected')
  })

  it('says where the figures come from', () => {
    expect(readinessDataLine([...competitors, tribelt])).toBe("From each site's newest scan. 3 sites.")
    expect(readinessDataLine([tribelt, readinessRow('X', { readiness: readiness({ checkedAt: '2026-10-28T08:00:00Z' }) })])).toBe(
      "From each site's newest scan, checked again on 28 October 2026. 2 sites.",
    )
    expect(readinessDataLine([tribelt])).toBe("From each site's newest scan. 1 site.")
    expect(readinessDataLine([])).toBe('No site has been scanned yet.')
  })
})
