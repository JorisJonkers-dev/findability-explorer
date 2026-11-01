import { describe, expect, it } from 'vitest'
import type { PlanImprovement, WatchedNumber } from '../../../infrastructure/api/types.gen'
import { details } from '../../../testing/fixtures'
import {
  cells,
  dataLine,
  doneLine,
  effortWhy,
  formatMeasure,
  goalBar,
  goalOf,
  goalSentence,
  inCell,
  KIND_LABEL,
  KIND_PLURAL,
  kindOf,
  kindsIn,
  lcFirst,
  listed,
  measureName,
  nowSentence,
  open,
  ordinal,
  placeLine,
  scoreLine,
  scrollCue,
  SORTS,
  whereText,
  whereLine,
  detailLine,
} from './recommendations'

const leader = { competitorId: 'example-belts', name: 'Example Belts', value: 0.3 }

function imp(id: string, benefit: number, effort: number, extra: Partial<PlanImprovement> = {}): PlanImprovement {
  return { id, title: `Title ${id}`, moves: 'Moves.', measure: 'aiShareOfVoice', auto: false, done: false, ...details, benefit, effort, ...extra }
}

// Best value first, as the server sends them.
const items = [imp('a', 4, 1), imp('b', 3, 1, { auto: true, kind: 'readiness' }), imp('c', 4, 3, { title: 'Another' }), imp('d', 5, 5, { done: true })]

const watched: WatchedNumber[] = [
  { target: 'aiShareOfVoice', now: 0.2, goal: 0.4, max: 1 },
  { target: 'readinessChecks', now: 3, goal: 6, max: 8 },
  { target: 'termsUsed', now: 20, goal: 50 },
  { target: 'citationShare', goal: 0.1, max: 1 },
]

describe('the kinds', () => {
  it('reads a kind as the server says it, and names each', () => {
    expect(kindOf(imp('x', 1, 1, { auto: true, kind: 'readiness' }))).toBe('readiness')
    expect(kindOf(imp('x', 1, 1, { kind: 'listing', generated: true }))).toBe('listing')
    expect(kindOf(imp('x', 1, 1))).toBe('general')
    expect(KIND_LABEL.term).toBe('Search term')
    expect(KIND_PLURAL.listing).toBe('Sites to get listed on')
  })

  it('lists only the kinds there are, in their order', () => {
    expect(kindsIn(items)).toEqual(['readiness', 'general'])
    expect(kindsIn([imp('x', 1, 1)])).toEqual(['general'])
  })
})

describe('the ranked list', () => {
  it('keeps the open ones in the order the server sends', () => {
    expect(open(items).map((i) => i.id)).toEqual(['a', 'b', 'c'])
  })

  it('sorts by each column, and the other way round', () => {
    const ids = (s: Parameters<typeof listed>[1], d: Parameters<typeof listed>[2] = 'desc') => listed(open(items), s, d).map((i) => i.id)
    expect(SORTS.map((s) => s.label)).toEqual(['Best value', 'Benefit', 'Effort', 'Kind', 'Name'])
    expect(ids('value')).toEqual(['a', 'b', 'c'])
    expect(ids('value', 'asc')).toEqual(['c', 'b', 'a'])
    // The larger benefit first; a tie keeps the best value.
    expect(ids('benefit')).toEqual(['a', 'c', 'b'])
    // The least work first.
    expect(ids('effort')).toEqual(['a', 'b', 'c'])
    expect(ids('kind')).toEqual(['b', 'a', 'c'])
    expect(ids('title')).toEqual(['c', 'a', 'b'])
  })

  it('groups the recommendations into bubbles of the same benefit and effort, the most benefit first, then the least work', () => {
    const many = [...open(items), imp('e', 4, 1)]
    expect(cells(many)).toEqual([
      { key: '4|1', benefit: 4, effort: 1, count: 2, label: '2 recommendations with benefit 4 of 5 and effort 1 of 5' },
      { key: '4|3', benefit: 4, effort: 3, count: 1, label: '1 recommendation with benefit 4 of 5 and effort 3 of 5' },
      { key: '3|1', benefit: 3, effort: 1, count: 1, label: '1 recommendation with benefit 3 of 5 and effort 1 of 5' },
    ])
    expect(inCell(many, '4|1').map((i) => i.id)).toEqual(['a', 'e'])
    expect(inCell(many, undefined)).toHaveLength(4)
  })

  it('says to scroll when the list holds more than it shows', () => {
    expect(scrollCue(5)).toBe('')
    expect(scrollCue(32)).toBe('Scroll inside the list to see all 32.')
  })

  it('writes the scores, where, and the data line', () => {
    expect(scoreLine(imp('x', 4, 1))).toBe('Benefit 4 of 5, effort 1 of 5')
    expect(whereText(imp('x', 1, 1))).toBe('The whole site')
    expect(whereText(imp('x', 1, 1, { generated: true, where: 'https://www.tribelt.nl/spiraalbanden' }))).toBe('www.tribelt.nl/spiraalbanden')
    expect(whereText(imp('x', 1, 1, { generated: true, whereNote: 'Where to put this is for the web agency to decide.' }))).toBe('For the web agency to place')
    expect(whereLine(imp('x', 1, 1))).toBe('This applies to the whole site.')
    expect(whereLine(imp('x', 1, 1, { generated: true, where: 'https://www.tribelt.nl/' }))).toBe('On https://www.tribelt.nl/.')
    expect(whereLine(imp('x', 1, 1, { generated: true, whereNote: 'Where to put this is for the web agency to decide.' }))).toBe(
      'Where to put this is for the web agency to decide.',
    )
    expect(dataLine(3, [imp('x', 1, 1)])).toBe('Written by hand, version 3 of the catalogue. No Recommendation run has written any yet.')
    expect(dataLine(3, [imp('x', 1, 1), imp('y', 1, 1, { generated: true })])).toBe(
      'Written by hand, version 3 of the catalogue, and by Recommendation runs: 1 of them.',
    )
    expect(detailLine(imp('x', 1, 1))).toBe('Written by hand, and applies in general.')
    expect(detailLine(imp('x', 1, 1, { generated: true, new: true }))).toBe('Written by the newest Recommendation run.')
    expect(detailLine(imp('x', 1, 1, { generated: true }))).toBe('Written by a Recommendation run.')
  })
})

describe('done', () => {
  it('says when, and who decided', () => {
    expect(doneLine(imp('x', 1, 1, { done: true, tickedBy: 'joris', tickedAt: '2026-11-02T09:00:00Z' }))).toBe('Marked done on 2 November 2026, by joris.')
    expect(doneLine(imp('x', 1, 1, { done: true, auto: true }))).toBe('Marked done, and the date was not recorded, by the scan of tribelt.nl.')
    expect(doneLine(imp('x', 1, 1, { done: true }))).toBe('Marked done, and the date was not recorded, by an administrator.')
  })
})

describe('the goal it improves', () => {
  it('names each measure, and keeps the capitals of AI inside a sentence', () => {
    expect(measureName('aiShareOfVoice')).toBe('AI responses that name Tribelt')
    expect(measureName('citationShare')).toBe('Web responses that cite tribelt.nl')
    expect(measureName('sitemapCoverage')).toBe('Pages the sitemaps list')
    expect(lcFirst(measureName('aiShareOfVoice'))).toBe('AI responses that name Tribelt')
    expect(lcFirst(measureName('readinessChecks'))).toBe('AI readiness checks passed')
    expect(lcFirst(measureName('termsUsed'))).toBe('search terms tribelt.nl uses')
    expect(lcFirst('')).toBe('')
  })

  it('finds the goal that watches a measure', () => {
    expect(goalOf('aiShareOfVoice', watched)).toBe(0.4)
    expect(goalOf('citationShare', watched)).toBe(0.1)
    expect(goalOf('termsUsed', watched)).toBe(50)
    expect(goalOf('readinessChecks', watched)).toBe(6)
    expect(goalOf('sitemapCoverage', watched)).toBeUndefined()
    expect(goalOf('termsUsed', [])).toBeUndefined()
  })

  it('writes a value as the measure counts it', () => {
    expect(formatMeasure('aiShareOfVoice', 0.15, 8)).toBe('15%')
    expect(formatMeasure('readinessChecks', 3, 8)).toBe('3 of 8')
    expect(formatMeasure('termsUsed', 1200, 8)).toBe('1,200')
  })

  it('says Tribelt is ahead of the best competitor when it leads', () => {
    expect(goalSentence({ measure: 'aiShareOfVoice', tribelt: 0.25, leader: { ...leader, value: 0.15 } }, 8)).toBe('Tribelt is ahead of the best competitor, Example Belts, at 15%.')
  })

  it('says the best competitor is ahead of Tribelt when it trails', () => {
    expect(goalSentence({ measure: 'aiShareOfVoice', tribelt: 0.2, leader }, 8)).toBe('Example Belts is ahead of Tribelt, with 30% against 20%.')
    expect(goalSentence({ measure: 'readinessChecks', tribelt: 3, leader: { ...leader, value: 7 } }, 8)).toBe('Example Belts is ahead of Tribelt, with 7 of 8 against 3 of 8.')
  })

  it('says when they are level, when Tribelt has no figure, and nothing without a best competitor', () => {
    expect(goalSentence({ measure: 'aiShareOfVoice', tribelt: 0.3, leader }, 8)).toBe('The best competitor, Example Belts, is level with Tribelt at 30%.')
    expect(goalSentence({ measure: 'aiShareOfVoice', leader }, 8)).toBe('The best competitor, Example Belts, is at 30%.')
    expect(goalSentence({ measure: 'aiShareOfVoice', tribelt: 0.3 }, 8)).toBe('')
  })

  it('says where Tribelt is now against the goal', () => {
    expect(nowSentence({ measure: 'aiShareOfVoice', tribelt: 0.2 }, 0.4, 8)).toBe('Tribelt now 20%, the goal 40%.')
    expect(nowSentence({ measure: 'sitemapCoverage', tribelt: 0.5 }, undefined, 8)).toBe('Tribelt now 50%. No goal watches this measure.')
    expect(nowSentence({ measure: 'termsUsed' }, 50, 8)).toBe('Tribelt has no figure on this yet, the goal 50.')
    expect(nowSentence({ measure: 'termsUsed' }, undefined, 8)).toBe('Tribelt has no figure on this yet.')
  })

  it('places Tribelt, the best competitor, and the goal on one bar', () => {
    expect(goalBar({ measure: 'aiShareOfVoice', tribelt: 0.2, leader }, 0.4, 8)).toEqual({ now: 0.2, leader: 0.3, goal: 0.4 })
    expect(goalBar({ measure: 'readinessChecks', tribelt: 4, leader: { ...leader, value: 8 } }, 6, 8)).toEqual({ now: 0.5, leader: 1, goal: 0.75 })
    // A count has no most, so the bar runs a little past the largest of the three.
    expect(goalBar({ measure: 'termsUsed', tribelt: 30, leader: { ...leader, value: 60 } }, 50, 8)).toEqual({ now: 0.5, leader: 1, goal: 50 / 60 })
    expect(goalBar({ measure: 'termsUsed' }, undefined, 8)).toEqual({ now: 0, leader: undefined, goal: undefined })
  })
})

describe('why it ranks here', () => {
  it('says why the effort is what it is, for each kind', () => {
    expect(effortWhy('term')).toBe('A search term on an existing page is a small change.')
    expect(effortWhy('page')).toBe('A new page is a larger piece of work.')
    expect(effortWhy('listing')).toBe('It needs a request to a site Tribelt does not run.')
    expect(effortWhy('fact')).toBe('It is a change to the fact sheet.')
    expect(effortWhy('readiness')).toBe('It is a change to the site or its files.')
    expect(effortWhy('general')).toBe('It is work for the web agency.')
  })

  it('places an open one among the others by best value', () => {
    expect(placeLine(items, 'a')).toBe('It ranks 1st of the 3 open recommendations by best value.')
    expect(placeLine(items, 'c')).toBe('It ranks 3rd of the 3 open recommendations by best value.')
    expect(placeLine(items, 'd')).toBe('')
  })

  it('writes a place in words', () => {
    expect([1, 2, 3, 4, 11, 12, 13, 21, 22, 23, 101].map(ordinal)).toEqual(['1st', '2nd', '3rd', '4th', '11th', '12th', '13th', '21st', '22nd', '23rd', '101st'])
  })
})
