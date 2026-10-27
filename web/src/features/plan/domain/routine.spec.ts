import { describe, expect, it } from 'vitest'
import { eyeOnText, foundLine, MONTHS, ROUTINE, standingLine, statusLine, TARGETS, watchedRow, YEAR } from './routine'

describe('the routine and the year', () => {
  it('has four weekly steps, each linking to its page, and a year of twelve months in three phases', () => {
    expect(ROUTINE.map((s) => s.link.path)).toEqual(['/ai-search', '/catching-up', '/plan', '/competitors/tribelt'])
    expect(MONTHS).toHaveLength(12)
    expect(YEAR.reduce((n, p) => n + p.span, 0)).toBe(12)
  })
})

describe('the numbers to watch', () => {
  it('write now and the goal, and scale the bars', () => {
    expect(watchedRow({ target: 'aiShareOfVoice', now: 0.24, goal: 0.4 })).toEqual({ label: 'AI answers that name Tribelt', now: '24%', goal: '40%', nowLength: 0.24, goalLength: 0.4 })
    expect(watchedRow({ target: 'readinessChecks', now: 3.5, goal: 5 })).toMatchObject({ now: '3½', goal: '5', nowLength: 0.7, goalLength: 1 })
    const terms = watchedRow({ target: 'termsUsed', now: 18, goal: 35 })
    expect([terms.now, terms.goal]).toEqual(['18', '35'])
    expect(terms.goalLength).toBeCloseTo(1 / 1.15)
    expect(watchedRow({ target: 'citations', goal: 20 })).toMatchObject({ now: 'No data yet', goal: '20', nowLength: 0 })
    expect(TARGETS.twenteShareOfVoice.format(0.75)).toBe('75%')
  })

  it('say why to keep an eye on a competitor', () => {
    expect(eyeOnText({ why: 'mostNamed' })).toBe('is named most in AI answers.')
    expect(eyeOnText({ why: 'bestPrepared' })).toBe('has the best prepared website.')
    expect(eyeOnText({ why: 'twente' })).toBe('is named most when buyers ask for Twente.')
  })
})

describe('an improvement in words', () => {
  const leader = { competitorId: 'martens', name: 'Märtens', value: 5 }

  it('says where Tribelt stands on the measure it moves, and who leads', () => {
    expect(standingLine({ measure: 'aiShareOfVoice', tribelt: 0.24, leader: { ...leader, value: 0.57 } })).toBe(
      'AI answers name Tribelt 24% of the time. Märtens leads with 57%.',
    )
    expect(standingLine({ measure: 'citationShare', tribelt: 0.05 })).toBe('Web answers cite tribelt.nl 5% of the time.')
    expect(standingLine({ measure: 'readinessChecks', tribelt: 2.5, leader })).toBe('tribelt.nl passes 2½ of five AI readiness checks. Märtens leads with 5.')
    expect(standingLine({ measure: 'sitemapPages', tribelt: 2140 })).toBe('The sitemaps of tribelt.nl list 2,140 pages.')
    expect(standingLine({ measure: 'termsUsed', tribelt: 12 })).toBe('tribelt.nl uses 12 search terms from the map.')
    // Without data, it says what is missing.
    expect(standingLine({ measure: 'termsUsed', leader })).toBe('No scan of tribelt.nl yet. Märtens leads with 5.')
    expect(standingLine({ measure: 'citationShare' })).toBe('No Prompt run has finished yet.')
  })

  it('gives each improvement one status, what the scan found when it checks it', () => {
    const parts = [{ name: 'llms.txt', present: true }]
    expect(statusLine({ measure: 'readinessChecks', tribelt: 3, parts })).toBe('The latest scan found llms.txt.')
    expect(statusLine({ measure: 'readinessChecks', tribelt: 3, parts: [] })).toBe('tribelt.nl passes 3 of five AI readiness checks.')
  })

  it('says what the latest scan of tribelt.nl found, and what it did not', () => {
    const part = (name: string, present: boolean) => ({ name, present })
    expect(foundLine([])).toBe('')
    expect(foundLine([part('llms.txt', true), part('llms-full.txt', true)])).toBe('The latest scan found llms.txt and llms-full.txt.')
    expect(foundLine([part('llms.txt', false), part('llms-full.txt', false)])).toBe('The latest scan did not find llms.txt or llms-full.txt.')
    expect(foundLine([part('a Dutch sitemap', true), part('a German sitemap', false), part('an English sitemap', true)])).toBe(
      'The latest scan found a Dutch sitemap and an English sitemap, but not a German sitemap.',
    )
    expect(foundLine([part('Product markup', false), part('FAQ markup', true)])).toBe('The latest scan found FAQ markup, but not Product markup.')
  })
})
