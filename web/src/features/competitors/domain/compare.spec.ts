import { describe, expect, it } from 'vitest'
import { bots, competitor, readiness } from '../../../testing/fixtures'
import {
  barLength,
  detailFigures,
  detailHeadline,
  familyBars,
  familyChartLabel,
  familyLine,
  figuresOf,
  headline,
  measuresOf,
  oxford,
  rankRows,
  RANKINGS,
  readinessScore,
  rowLabel,
  type Row,
  scoreText,
  shareText,
} from './compare'

const row = (name: string, share?: number, ready?: number): Row => ({ id: name.toLowerCase(), name, share, ready })

describe('oxford', () => {
  it('writes a list of names as a sentence does', () => {
    expect(oxford([])).toBe('')
    expect(oxford(['a'])).toBe('a')
    expect(oxford(['a', 'b'])).toBe('a and b')
    expect(oxford(['a', 'b', 'c'])).toBe('a, b, and c')
  })
})

describe('readinessScore', () => {
  it('counts the checks a site passes, the crawlers half when some are kept out', () => {
    expect(readinessScore(undefined)).toBeUndefined()
    expect(readinessScore(readiness())).toBe(5)
    expect(readinessScore(readiness({ bots: bots({ GPTBot: 'blocked' }) }))).toBe(4.5)
    expect(readinessScore(readiness({ bots: bots({ GPTBot: 'blocked' }), markdown: false, llmsFullTxt: false }))).toBe(2.5)
    expect(readinessScore(readiness({ robots: 'unreachable' }))).toBe(4)
  })
})

describe('scoreText and shareText', () => {
  it('writes a score and a share as the page does', () => {
    expect(scoreText(undefined, 5)).toBe('Not scanned')
    expect(scoreText(3, 5)).toBe('3 of 5')
    expect(scoreText(3.5, 5)).toBe('3½ of 5')
    expect(shareText(undefined)).toBe('No run yet')
    expect(shareText(0.156)).toBe('16%')
  })
})

describe('rankRows', () => {
  const rows = [row('Delta', undefined, 2), row('Alpha', 0.1, 5), row('Charlie', 0.3, undefined), row('Bravo', 0.1, 5)]
  it('puts those with a value first, the highest first, then by name', () => {
    expect(rankRows(rows, 'answers').map((r) => r.name)).toEqual(['Charlie', 'Alpha', 'Bravo', 'Delta'])
    expect(rankRows(rows, 'ready').map((r) => r.name)).toEqual(['Alpha', 'Bravo', 'Delta', 'Charlie'])
    expect(rankRows(rows, 'name').map((r) => r.name)).toEqual(['Alpha', 'Bravo', 'Charlie', 'Delta'])
    expect(rows[0]?.name).toBe('Delta')
    expect(RANKINGS.map((r) => r.value)).toEqual(['answers', 'ready', 'name'])
  })

  it('keeps the rows without a value in name order', () => {
    expect(rankRows([row('Zed'), row('Abe')], 'answers').map((r) => r.name)).toEqual(['Abe', 'Zed'])
  })
})

describe('rowLabel', () => {
  it('reads a row aloud', () => {
    expect(rowLabel(row('Twentebelt', 0.5, 3), 5)).toBe('Twentebelt. AI answers that name it: 50%. AI readiness checks passed: 3 of 5. Open its page.')
  })
})

describe('figuresOf', () => {
  const rows = [row('A', 0.5, 5), row('B', 0.4, 3), row('C', 0.3, 3), row('D', 0.2, 3), row('E', undefined, undefined)]
  const input = { count: 5, rows, tribeltShare: 0.1, tribeltReady: 3, outOf: 5 }

  it('counts the competitors, those named more than Tribelt, and those level with its readiness', () => {
    expect(figuresOf(input)).toEqual([
      { big: '5', text: 'competitors on the list', note: 'Archived companies are not counted.' },
      { big: '4', text: 'are named in more AI answers than Tribelt', note: 'A, B, and C lead.' },
      { big: '3', text: 'share Tribelt’s AI readiness score of 3 of 5', note: 'Among them B, C, and D.' },
    ])
  })

  it('speaks of one in the singular', () => {
    const figures = figuresOf({ ...input, count: 1, rows: [row('A', 0.5, 3)] })
    expect(figures.map((f) => f.text)).toEqual(['competitor on the list', 'is named in more AI answers than Tribelt', 'shares Tribelt’s AI readiness score of 3 of 5'])
    expect(figures[1]?.note).toBe('A leads.')
  })

  it('says when no one is ahead or level', () => {
    const figures = figuresOf({ ...input, rows: [row('A', 0.05, 4)] })
    expect(figures[1]).toMatchObject({ big: '0', note: 'Tribelt is named in as many AI answers as any competitor, or more.' })
    expect(figures[2]).toMatchObject({ big: '0', note: 'No other competitor passes as many checks.' })
  })

  it('leaves out what Tribelt has no value for', () => {
    expect(figuresOf({ ...input, tribeltShare: undefined, tribeltReady: undefined })).toHaveLength(1)
  })
})

describe('headline', () => {
  it('counts the competitors', () => {
    expect(headline(0)).toBe('The competitors that sell metal belts Tribelt’s buyers could buy instead.')
    expect(headline(1)).toBe('1 competitor sells metal belts that Tribelt’s buyers could buy instead.')
    expect(headline(24)).toBe('24 competitors sell metal belts that Tribelt’s buyers could buy instead.')
  })
})

describe('the family chart', () => {
  const tribelt = competitor('Tribelt', { beltNames: [{ name: 'Draadogenbanden', family: 'eyelink' }] })
  const bars = familyBars(
    [
      { family: 'eyelink', makers: 3 },
      { family: 'open_spiral', makers: 1 },
    ],
    tribelt,
  )

  it('has a bar per family with Tribelt’s own name for it', () => {
    expect(bars).toEqual([
      { family: 'eyelink', label: 'Eyelink', makers: 3, tribelt: 'Draadogenbanden', path: '/terms/families/eyelink' },
      { family: 'open_spiral', label: 'Open spiral', makers: 1, tribelt: null, path: '/terms/families/open-spiral' },
    ])
    expect(familyBars([{ family: 'eyelink', makers: 3 }], undefined)[0]?.tribelt).toBeNull()
  })

  it('says how Tribelt sells the family, and describes the chart', () => {
    expect(familyLine(bars[0] as (typeof bars)[number])).toBe('3 of the competitors sell it. Tribelt sells it as Draadogenbanden. How buyers search for it is on its page.')
    expect(familyLine(bars[1] as (typeof bars)[number])).toContain('Tribelt does not sell this family.')
    expect(familyChartLabel(bars)).toBe('Competitors that sell each belt family. Eyelink: 3. Open spiral: 1.')
  })
})

describe('detailHeadline', () => {
  it.each([
    ['not scanned', undefined, 0.2, 'Twentebelt, as its newest scan finds it.'],
    ['no Tribelt share', 0.3, undefined, 'Twentebelt is named in 30% of AI answers.'],
    ['neither named', 0, 0, 'No AI answer names Twentebelt or Tribelt.'],
    ['only it named', 0.3, 0, 'Twentebelt is named in 30% of AI answers, and no answer names Tribelt.'],
    ['twice', 0.4, 0.2, 'Twentebelt is named in twice as many AI answers as Tribelt.'],
    ['about three times', 0.5, 0.2, 'Twentebelt is named in about three times as many AI answers as Tribelt.'],
    ['exactly three times', 0.6, 0.2, 'Twentebelt is named in three times as many AI answers as Tribelt.'],
    ['more', 0.25, 0.2, 'Twentebelt is named in more AI answers than Tribelt.'],
    ['about as many', 0.2, 0.2, 'Twentebelt is named in about as many AI answers as Tribelt.'],
    ['fewer', 0.1, 0.2, 'Twentebelt is named in fewer AI answers than Tribelt.'],
  ])('reads %s', (_, share, tribelt, text) => {
    expect(detailHeadline('Twentebelt', share, tribelt)).toBe(text)
  })
})

describe('measuresOf', () => {
  const known = { name: 'Twentebelt', share: 0.4, tribeltShare: 0.2, ready: 3.5, tribeltReady: 5, outOf: 5, sitemap: 0.5, tribeltSitemap: 0.85, terms: 12, tribeltTerms: 1200, mapTerms: 2400 }
  const by = (input: Parameters<typeof measuresOf>[0]) => Object.fromEntries(measuresOf(input).map((m) => [m.id, m]))

  it('sets the competitor against Tribelt on four measures', () => {
    const m = by(known)
    expect(Object.keys(m)).toEqual(['answers', 'ready', 'sitemap', 'terms'])
    expect(m.answers).toMatchObject({ themText: '40%', usText: '20%', scale: 0.4, path: '/ai-search/answers' })
    expect(m.answers?.text).toBe('Twentebelt is named in 40% of the answers in the newest Prompt run, and Tribelt in 20%.')
    expect(m.ready).toMatchObject({ themText: '3½ of 5', usText: '5 of 5', scale: 5 })
    expect(m.ready?.text).toBe('Twentebelt’s site passes 3½ of 5 of the AI readiness checks, and tribelt.nl passes 5 of 5.')
    expect(m.sitemap).toMatchObject({ themText: '50%', usText: '85%', scale: 1 })
    expect(m.sitemap?.text).toBe('Twentebelt’s sitemap lists 50% of the pages its newest scan read, against 85% on tribelt.nl.')
    expect(m.terms).toMatchObject({ themText: '12', usText: '1,200', scale: 1200 })
    expect(m.terms?.text).toBe('tribelt.nl uses 1,200 of the 2,400 terms in the map. Twentebelt’s site uses 12.')
  })

  it('says what is not known', () => {
    const m = by({ ...known, share: undefined, tribeltShare: undefined, ready: undefined, tribeltReady: undefined, sitemap: undefined, tribeltSitemap: undefined, terms: undefined, tribeltTerms: undefined })
    expect(m.answers).toMatchObject({ themText: 'Not known', scale: 0.01, text: 'No Prompt run has counted the answers that name Twentebelt or Tribelt yet.' })
    expect(m.ready?.text).toBe('Twentebelt’s site has not been scanned yet, so its AI readiness is not known.')
    expect(m.sitemap?.text).toContain('is not known. Scan its site again')
    expect(m.terms).toMatchObject({ themText: 'Not known', scale: 1 })
  })

  it('says when both sites pass as many checks, or Tribelt has not been scanned', () => {
    expect(by({ ...known, ready: 5 }).ready?.text).toBe('Both sites pass 5 of 5 of the AI readiness checks.')
    expect(by({ ...known, tribeltReady: undefined }).ready?.text).toBe('Twentebelt’s site passes 3½ of 5 of the AI readiness checks, and tribelt.nl has not been scanned yet.')
  })
})

describe('barLength', () => {
  it('is the value against the scale, up to a full bar, and none when not known', () => {
    expect(barLength(undefined, 1)).toBeUndefined()
    expect(barLength(1, 0)).toBeUndefined()
    expect(barLength(0.5, 2)).toBe(0.25)
    expect(barLength(3, 2)).toBe(1)
  })
})

describe('detailFigures', () => {
  it('makes a figure of the answers, the readiness and the terms, with Tribelt beside', () => {
    const measures = measuresOf({ name: 'Twentebelt', share: 0.4, tribeltShare: 0.2, ready: 3, tribeltReady: 5, outOf: 5, sitemap: 0.5, tribeltSitemap: 0.85, terms: 12, tribeltTerms: 1200, mapTerms: 2400 })
    expect(detailFigures(measures)).toEqual([
      { big: '40%', text: 'of AI answers name it', note: 'Tribelt: 20%.' },
      { big: '3 of 5', text: 'AI readiness checks passed', note: 'Tribelt: 5 of 5.' },
      { big: '12', text: 'search terms from the map on its site', note: 'Tribelt: 1,200.' },
    ])
  })
})
