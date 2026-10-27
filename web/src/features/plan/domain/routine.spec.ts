import { describe, expect, it } from 'vitest'
import { eyeOnText, MONTHS, ROUTINE, TARGETS, watchedRow, YEAR } from './routine'

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
