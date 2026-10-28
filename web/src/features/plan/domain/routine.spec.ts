import { describe, expect, it } from 'vitest'
import { eyeOnText, foundLine, metLine, MONTHS, ROUTINE, standingLine, statusLine, targetLabel, TARGETS, watchedRow, YEAR } from './routine'

describe('the routine and the year', () => {
  it('has four weekly steps, each linking to its page, and a year of twelve months in three phases', () => {
    expect(ROUTINE.map((s) => s.link.path)).toEqual(['/ai-search', '/catching-up', '/plan', '/competitors/tribelt'])
    expect(MONTHS).toHaveLength(12)
    expect(YEAR.reduce((n, p) => n + p.span, 0)).toBe(12)
  })
})

describe('the numbers to watch', () => {
  it('write now and the goal, and scale the bars', () => {
    expect(watchedRow({ target: 'aiShareOfVoice', now: 0.24, goal: 0.4, met: false })).toEqual({
      label: 'AI answers in the newest Prompt run that name Tribelt',
      now: '24%',
      goal: '40%',
      status: '24% to 40%',
      met: false,
      nowLength: 0.24,
      goalLength: 0.4,
    })
    expect(watchedRow({ target: 'readinessChecks', now: 3.5, goal: 5 })).toMatchObject({ now: '3½', goal: '5', nowLength: 0.7, goalLength: 1 })
    const terms = watchedRow({ target: 'termsUsed', now: 18, goal: 35 })
    expect([terms.now, terms.goal]).toEqual(['18', '35'])
    expect(terms.goalLength).toBeCloseTo(1 / 1.15)
    expect(watchedRow({ target: 'citations', goal: 20 })).toMatchObject({ now: 'No data yet', goal: '20', nowLength: 0 })
    expect(TARGETS.twenteShareOfVoice.format(0.75)).toBe('75%')
  })

  it('show a met target as met, with no target behind a bar the value fills, and ask to raise it', () => {
    const met = watchedRow({ target: 'termsUsed', now: 45, goal: 35, met: true })
    expect(met).toMatchObject({ met: true, status: 'Met: 45, target was 35', goalLength: undefined })
    // Past the target the scale reaches past the value, so the bar does not run off its end.
    expect(met.nowLength).toBeCloseTo(1 / 1.15)
    // Without a value there is nothing met, whatever the server says.
    expect(watchedRow({ target: 'citations', goal: 20, met: true }).met).toBe(false)
    const watched = [
      { target: 'termsUsed' as const, now: 45, goal: 35, met: true },
      { target: 'citations' as const, now: 3, goal: 50, met: false },
    ]
    expect(metLine(watched, true)).toBe('A target you have met is done. Raise it under Change the targets.')
    expect(metLine(watched, false)).toBe('A target you have met is done. An administrator can raise it.')
    expect(metLine(watched.slice(1), true)).toBe('')
  })

  it('name where the search terms come from before a scan of tribelt.nl', () => {
    expect(targetLabel('termsUsed', 'scan')).toBe("Search terms from the map found on tribelt.nl's newest scan")
    expect(targetLabel('termsUsed', 'research')).toBe('Search terms from the map found for Tribelt by our research')
    expect(watchedRow({ target: 'citations', goal: 50 }, 'research').label).toBe('Web answers in the newest Prompt run that cite tribelt.nl')
  })

  it('say why to keep an eye on a competitor', () => {
    expect(eyeOnText({ why: 'mostNamed' })).toBe('is named most in AI answers.')
    expect(eyeOnText({ why: 'bestPrepared' })).toBe('is best prepared for AI.')
    expect(eyeOnText({ why: 'twente' })).toBe('is named most when buyers ask for Twente.')
  })
})

describe('an improvement in words', () => {
  const leader = { competitorId: 'martens', name: 'Märtens', value: 5 }

  it('says where Tribelt stands on the measure it moves, and who leads', () => {
    expect(standingLine({ measure: 'aiShareOfVoice', tribelt: 0.24, leader: { ...leader, value: 0.57 } })).toBe(
      '24% of the AI answers in the newest Prompt run name Tribelt. Märtens leads with 57%.',
    )
    expect(standingLine({ measure: 'citationShare', tribelt: 0.05 })).toBe('5% of the web answers in the newest Prompt run cite tribelt.nl.')
    expect(standingLine({ measure: 'readinessChecks', tribelt: 2.5, leader })).toBe('tribelt.nl passes 2½ of five AI readiness checks. Märtens leads with 5.')
    expect(standingLine({ measure: 'sitemapCoverage', tribelt: 0.6, leader: { ...leader, value: 1 } })).toBe(
      'The sitemaps of tribelt.nl list 60% of the pages its newest scan read. Märtens leads with 100%.',
    )
    expect(standingLine({ measure: 'sitemapCoverage' })).toBe('Not measured on tribelt.nl yet. Scan it again.')
    expect(standingLine({ measure: 'termsUsed', tribelt: 45 })).toBe('The newest scan of tribelt.nl found 45 search terms from the map.')
    expect(standingLine({ measure: 'termsUsed', tribelt: 12 }, 'research')).toBe('Our research found 12 search terms from the map for Tribelt.')
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
