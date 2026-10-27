import { describe, expect, it } from 'vitest'
import type { Standing } from '../../../infrastructure/api/types.gen'
import { barsOf, formatValue, headline, inWords, noteOf } from './standings'

const voice: Standing = {
  measure: 'aiShareOfVoice',
  tribelt: 0.24,
  median: 0.17,
  leader: { competitorId: 'ashworth', name: 'Ashworth', value: 0.57 },
  competitors: 38,
  trails: true,
  leads: false,
}

describe('the values', () => {
  it('write shares as percentages, scores with a half, counts with separators', () => {
    expect(formatValue('aiShareOfVoice', 0.236)).toBe('24%')
    expect(formatValue('citationShare', 0)).toBe('0%')
    expect(formatValue('readinessChecks', 3.5)).toBe('3½')
    expect(formatValue('readinessChecks', 5)).toBe('5')
    expect(formatValue('sitemapPages', 2140)).toBe('2,140')
    expect(formatValue('termsUsed', 18)).toBe('18')
  })

  it('count small numbers in words', () => {
    expect(inWords(0)).toBe('no')
    expect(inWords(4)).toBe('four')
    expect(inWords(12)).toBe('12')
  })
})

describe('the bars', () => {
  it('show Tribelt, the typical competitor and the leader by name, against the largest', () => {
    expect(barsOf(voice)).toEqual([
      { who: 'Tribelt', value: '24%', length: 0.24 / 0.57, tone: 'tribelt' },
      { who: 'Typical competitor', value: '17%', length: 0.17 / 0.57, tone: 'typical' },
      { who: 'Ashworth', value: '57%', length: 1, tone: 'leader' },
    ])
  })

  it('say when there is no data, and draw nothing when every value is zero', () => {
    const empty: Standing = { measure: 'sitemapPages', competitors: 0, trails: false, leads: false }
    expect(barsOf(empty).map((b) => [b.who, b.value, b.length])).toEqual([
      ['Tribelt', 'No data yet', 0],
      ['Typical competitor', 'No data yet', 0],
      ['The leader', 'No data yet', 0],
    ])
    const zeros: Standing = { ...voice, tribelt: 0, median: 0, leader: { competitorId: 'a', name: 'A', value: 0 } }
    expect(barsOf(zeros).map((b) => b.length)).toEqual([0, 0, 0])
  })
})

describe('the notes', () => {
  it('say who is ahead, or that Tribelt leads, or what is missing', () => {
    expect(noteOf(voice)).toBe('Ashworth is ahead.')
    expect(noteOf({ ...voice, trails: false, leads: true })).toBe('Tribelt leads here.')
    expect(noteOf({ ...voice, tribelt: undefined })).toBe('Tribelt has no data here yet.')
    expect(noteOf({ ...voice, leader: undefined })).toBe('No competitor has data here yet.')
  })
})

describe('the headline', () => {
  it('counts the measures where Tribelt trails, of those measured', () => {
    const measures = [voice, { ...voice, measure: 'termsUsed' as const }, { ...voice, measure: 'sitemapPages' as const, tribelt: undefined }]
    expect(headline(measures, 2)).toBe('Tribelt trails the leaders on two of two measures.')
    expect(headline(measures, 0)).toBe('Tribelt leads the field on every measure.')
    expect(headline([], 0)).toBe('Not measured yet. The first scans and Prompt run fill this page.')
  })
})
