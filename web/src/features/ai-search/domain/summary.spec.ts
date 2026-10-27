import { describe, expect, it } from 'vitest'
import type { AiSearchSummary } from '../../../infrastructure/api/types.gen'
import { ACCESS, chartLabel, isTribelt, keepingOutNote, linesOf, nameList, pct, scaleTop, shortDate, trendLine, tribeltNote, tribeltShares, xOf, yOf } from './summary'

const followed: AiSearchSummary['followed'] = [
  { competitorId: 'tribelt', name: 'Tribelt', role: 'tribelt' },
  { competitorId: 'ashworth', name: 'Ashworth', role: 'competitor' },
  { competitorId: 'twentebelt', name: 'Twentebelt', role: 'competitor' },
]
const trend: AiSearchSummary['trend'] = [
  { runStartedAt: '2026-10-26T06:00:00Z', shares: [0.14, 0.62, 0.29] },
  { runStartedAt: '2026-11-02T06:00:00Z', shares: [0.24, 0.57, 0.24] },
]

describe('the chart', () => {
  it('scales to the next 20% above the highest share or the goal', () => {
    expect(scaleTop(trend, 0.4)).toBe(0.8)
    expect(scaleTop([], 0.4)).toBe(0.4)
    expect(scaleTop([{ runStartedAt: '', shares: [0.01] }], 0)).toBe(0.2)
    expect(scaleTop([{ runStartedAt: '', shares: [0.6] }], 0.4)).toBe(0.6)
  })

  it('places runs along the axis and shares up it', () => {
    expect(xOf(0, 2)).toBe(70)
    expect(xOf(1, 2)).toBe(1040)
    expect(xOf(0, 1)).toBe(555)
    expect(yOf(0, 0.8)).toBe(600)
    expect(yOf(0.8, 0.8)).toBe(100)
  })

  it("draws Tribelt's line last, and keeps the end labels apart", () => {
    const lines = linesOf({ followed, trend }, 0.8)
    expect(lines.map((l) => l.label)).toEqual(['Ashworth 57%', 'Twentebelt 24%', 'Tribelt 24%'])
    expect(lines[2]?.tribelt).toBe(true)
    expect(lines[0]?.points).toBe('70.0,212.5 1040.0,243.8')
    // Tribelt and Twentebelt end on the same share; one label moves down.
    expect(Math.abs((lines[1]?.labelY ?? 0) - (lines[2]?.labelY ?? 0))).toBe(36)
  })

  it('treats a missing share as none', () => {
    const lines = linesOf({ followed: [followed[0] as AiSearchSummary['followed'][number]], trend: [{ runStartedAt: '', shares: [] }] }, 0.2)
    expect(lines[0]?.label).toBe('Tribelt 0%')
    expect(linesOf({ followed: [followed[1] as AiSearchSummary['followed'][number]], trend: [] }, 0.2)[0]?.label).toBe('Ashworth 0%')
  })

  it('describes itself for a screen reader', () => {
    expect(chartLabel({ followed: followed.slice(0, 2), trend, goal: 0.4 })).toBe(
      'Share of answers naming each company over 2 Prompt runs. Tribelt went from 14% to 24%. Ashworth went from 62% to 57%. The goal is 40%.',
    )
    expect(chartLabel({ followed: followed.slice(0, 1), trend: [], goal: 0.4 })).toBe(
      'Share of answers naming each company over 0 Prompt runs. Tribelt went from 0% to 0%. The goal is 40%.',
    )
  })
})

describe("Tribelt's figure", () => {
  it('compares the newest run with the first one shown', () => {
    expect(pct(0.236)).toBe('24%')
    expect(shortDate('2026-10-26T06:00:00Z')).toBe('26 Oct')
    expect(tribeltShares({ followed, trend })).toEqual({ now: 0.24, first: 0.14, firstAt: '2026-10-26T06:00:00Z' })
    expect(trendLine({ followed, trend })).toBe('of the answers in the newest run, up from 14% on 26 Oct')
    expect(trendLine({ followed, trend: [...trend].reverse() })).toBe('of the answers in the newest run, down from 24% on 2 Nov')
    expect(trendLine({ followed, trend: [trend[0], { ...trend[0], runStartedAt: '2026-11-02T06:00:00Z' }] as AiSearchSummary['trend'] })).toBe(
      'of the answers in the newest run, the same as on 26 Oct',
    )
    expect(trendLine({ followed, trend: trend.slice(0, 1) })).toBe('of the answers in the first Prompt run')
  })

  it('says when there is no run yet, or no Tribelt to follow', () => {
    expect(tribeltShares({ followed, trend: [] })).toBeUndefined()
    expect(tribeltShares({ followed: followed.slice(1), trend })).toBeUndefined()
    expect(trendLine({ followed, trend: [] })).toBe('No Prompt run has finished yet.')
    expect(tribeltShares({ followed, trend: [{ runStartedAt: '', shares: [] }] })).toEqual({ now: 0, first: 0, firstAt: '' })
  })
})

describe('the crawlers', () => {
  it('say each access in a word of its own', () => {
    expect(Object.values(ACCESS).map((a) => a.text)).toEqual(['Allowed', 'Partly', 'Blocked', 'Not checked'])
  })

  it('say what tribelt.nl does with them', () => {
    const crawler = { bot: 'GPTBot', lab: 'OpenAI', gets: 'x', feeds: 'future' as const }
    expect(tribeltNote([{ ...crawler }])).toBe('tribelt.nl has not been scanned yet.')
    expect(tribeltNote([{ ...crawler, access: 'allowed' }])).toBe('tribelt.nl lets every one of them in.')
    expect(tribeltNote([{ ...crawler, access: 'blocked' }, { ...crawler, bot: 'CCBot', access: 'blocked' }])).toBe('tribelt.nl keeps GPTBot and CCBot out.')
  })

  it('name the competitors that keep them out', () => {
    const k = (name: string) => ({ competitorId: 'x', name, blocked: ['GPTBot'] })
    expect(keepingOutNote([])).toBe('No competitor keeps a crawler out.')
    expect(keepingOutNote([k('Ashworth')])).toBe('Ashworth shuts some of them out.')
    expect(keepingOutNote([k('A'), k('B'), k('C'), k('D'), k('E')])).toBe('A, B, C and 2 more shut some of them out.')
    expect(nameList([])).toBe('')
    expect(nameList(['A', 'B', 'C'])).toBe('A, B and C')
  })

  it("pick out Tribelt's row", () => {
    expect(isTribelt({ role: 'tribelt' })).toBe(true)
    expect(isTribelt({ role: 'group' })).toBe(false)
  })
})
