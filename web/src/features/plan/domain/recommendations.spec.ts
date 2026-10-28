import { describe, expect, it } from 'vitest'
import type { PlanImprovement } from '../../../infrastructure/api/types.gen'
import { details } from '../../../testing/fixtures'
import {
  alsoImproving,
  benefitText,
  compareLine,
  comparison,
  dotTip,
  effortNote,
  effortText,
  MEASURE_NAME,
  ordinal,
  placeNote,
  placeOf,
  rankOpen,
  scoreDots,
  SORTS,
} from './recommendations'

function imp(id: string, benefit: number, effort: number, extra: Partial<PlanImprovement> = {}): PlanImprovement {
  return { id, title: `Title ${id}`, moves: 'Moves.', measure: 'aiShareOfVoice', auto: false, done: false, ...details, benefit, effort, ...extra }
}

const items = [imp('a', 4, 1), imp('b', 3, 1), imp('c', 4, 3), imp('d', 5, 5, { done: true })]

describe('scores in words', () => {
  it('names each benefit and effort, and nothing for a score out of range', () => {
    expect([1, 3, 5].map(benefitText)).toEqual(['Very small benefit', 'Medium benefit', 'Very large benefit'])
    expect([1, 3, 5].map(effortText)).toEqual(['A few hours', 'A few days', 'Several weeks'])
    expect(benefitText(9)).toBe('')
    expect(effortText(0)).toBe('')
    expect(effortText(9)).toBe('')
  })

  it('fills as many squares as the score', () => {
    expect(scoreDots(2)).toEqual([true, true, false, false, false])
    expect(SORTS.map((s) => s.id)).toEqual(['value', 'benefit', 'effort'])
  })
})

describe('ranking what is open', () => {
  it('keeps the server order for best value, and drops what is done', () => {
    expect(rankOpen(items, 'value').map((r) => [r.item.id, r.rank])).toEqual([['a', 1], ['b', 2], ['c', 3]])
  })

  it('ranks by benefit, then least work', () => {
    expect(rankOpen(items, 'benefit').map((r) => r.item.id)).toEqual(['a', 'c', 'b'])
  })

  it('ranks by least work, then the larger benefit', () => {
    expect(rankOpen(items, 'effort').map((r) => r.item.id)).toEqual(['a', 'b', 'c'])
  })

  it('describes a dot for a screen reader', () => {
    expect(dotTip({ item: items[2] as PlanImprovement, rank: 3 })).toBe('3. Title c. Large benefit, a few days.')
  })
})

describe('comparing Tribelt with the leader and the target', () => {
  const watched = [{ target: 'aiShareOfVoice' as const, now: 0.24, goal: 0.4, max: 1, met: false }]
  const share = imp('a', 4, 1, { tribelt: 0.1, leader: { competitorId: 'ashworth', name: 'Ashworth', value: 0.2 } })

  it('draws every bar on the scale of the largest', () => {
    const rows = comparison(share, watched, 5)
    expect(rows.map((r) => [r.who, r.tone, r.length])).toEqual([['Tribelt', 'tribelt', 0.25], ['Ashworth', 'leader', 0.5], ['Target', 'target', 1]])
    expect(rows.map((r) => r.text)).toEqual(['10%', '20%', '40%'])
  })

  it('says so when there is no data, and draws no bar', () => {
    const rows = comparison(imp('a', 4, 1), [], 5)
    expect(rows).toEqual([{ who: 'Tribelt', text: 'No data yet', length: 0, tone: 'tribelt' }])
  })

  it('writes readiness checks against how many there are', () => {
    const rows = comparison(imp('r', 3, 1, { measure: 'readinessChecks', tribelt: 3, leader: { competitorId: 'm', name: 'Märtens', value: 5 } }), [], 5)
    expect(rows.map((r) => r.text)).toEqual(['3 of five', '5 of five'])
  })

  it('has no target for a measure nothing watches', () => {
    expect(comparison(imp('c', 1, 1, { measure: 'citationShare', tribelt: 0.1 }), watched, 5)).toHaveLength(1)
    expect(MEASURE_NAME.citationShare).toBe('Web answers that cite tribelt.nl')
  })

  it('sums it up in one line', () => {
    expect(compareLine(share, watched, 5)).toBe('AI answers that name Tribelt: 10%, Ashworth 20%, target 40%')
    expect(compareLine(imp('a', 4, 1), [], 5)).toBe('AI answers that name Tribelt: No data yet')
  })
})

describe('where a recommendation stands', () => {
  it('writes places as ordinals', () => {
    expect([1, 2, 3, 4, 11, 12, 13, 21, 22, 23, 101, 111].map(ordinal)).toEqual(['1st', '2nd', '3rd', '4th', '11th', '12th', '13th', '21st', '22nd', '23rd', '101st', '111th'])
  })

  it('finds the others on the same measure', () => {
    const other = imp('x', 1, 1, { measure: 'termsUsed' })
    expect(alsoImproving([...items, other], items[0] as PlanImprovement).map((i) => i.id)).toEqual(['b', 'c', 'd'])
    expect(alsoImproving([...items, other], other)).toEqual([])
  })

  it('places an open one among all, and nothing for one that is done', () => {
    expect(placeOf(items, 'b')).toEqual({ rank: 2, of: 4 })
    expect(placeOf(items, 'd')).toBeUndefined()
  })

  it('compares the effort with the others', () => {
    expect(effortNote([imp('a', 1, 1), imp('b', 1, 2), imp('c', 1, 3)], imp('a', 1, 1))).toBe('The least work of all three recommendations.')
    expect(effortNote([imp('a', 1, 5), imp('b', 1, 2), imp('c', 1, 3)], imp('a', 1, 5))).toBe('The most work of all three recommendations.')
    expect(effortNote([imp('a', 1, 3), imp('b', 1, 2), imp('c', 1, 4)], imp('a', 1, 3))).toBe('1 of the other two take less work.')
  })

  it('says what comes before', () => {
    expect(placeNote(1, 8)).toBe('It comes before the other seven.')
    expect(placeNote(3, 8)).toBe('2 of the other seven come before it.')
  })
})
