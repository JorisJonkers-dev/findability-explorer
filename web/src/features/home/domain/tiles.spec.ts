import { describe, expect, it } from 'vitest'
import type { OverviewAiSearch, OverviewCompetitors, OverviewTerms, Standing } from '../../../infrastructure/api/types.gen'
import { aiFigure, catchUpFigure, familyBars, gapBars, LANGUAGES_SHOWN, languageBars, originOf, termsFigure, voiceBars } from './tiles'

describe('the competitors tile', () => {
  it('counts the makers offering each family, of all of them', () => {
    const competitors: OverviewCompetitors = {
      makers: 2,
      families: [
        { family: 'eyelink', makers: 2 },
        { family: 'open_spiral', makers: 0 },
        { family: 'closed_spiral', makers: 0 },
        { family: 'tri_flex_turn_curve', makers: 1 },
        { family: 'other', makers: 0 },
      ],
    }
    expect(familyBars(competitors).map((b) => [b.label, b.value, b.first, b.zero])).toEqual([
      ['Eyelink', '2', 1, false],
      ['Open spiral', '0', 0, true],
      ['Closed spiral', '0', 0, true],
      ['Tri-Flex', '1', 0.5, false],
      ['Other belts', '0', 0, true],
    ])
  })

  it('shows every family at none until the figures load, or when the research found no makers', () => {
    expect(familyBars(undefined).map((b) => [b.label, b.value, b.first, b.zero])).toEqual([
      ['Eyelink', '0', 0, true],
      ['Open spiral', '0', 0, true],
      ['Closed spiral', '0', 0, true],
      ['Tri-Flex', '0', 0, true],
      ['Other belts', '0', 0, true],
    ])
    expect(familyBars({ makers: 0, families: [{ family: 'eyelink', makers: 0 }] }).map((b) => b.first)).toEqual([0])
  })
})

describe('the search terms tile', () => {
  const terms: OverviewTerms = {
    total: 4,
    tribelt: 2,
    languages: [
      { language: 'en', terms: 2, tribelt: 1 },
      { language: 'de', terms: 1, tribelt: 1 },
      { language: 'nl', terms: 1, tribelt: 0 },
    ],
  }

  it('charts the terms per language against the largest, the part Tribelt uses first', () => {
    expect(languageBars(terms).map((b) => [b.label, b.value, b.first, b.second])).toEqual([
      ['English', '2', 0.5, 0.5],
      ['German', '1', 0.5, 0],
      ['Dutch', '1', 0, 0.5],
    ])
    expect(languageBars(undefined)).toEqual([])
  })

  it('counts the terms and the languages, and says how many terms tribelt.nl uses and how that is known', () => {
    expect(termsFigure(terms)).toEqual({ big: '4', text: 'search terms in 3 languages. The newest scan of tribelt.nl found 2 of them.' })
    expect(termsFigure({ ...terms, from: 'research' }).text).toBe('search terms in 3 languages. Our research found 2 of them for Tribelt.')
    expect(termsFigure(undefined)).toEqual({ big: '…', text: 'search terms in the terminology map.' })
    expect(termsFigure({ total: 0, tribelt: 0, languages: [] }).big).toBe('…')
  })

  it('shows the eight largest languages only', () => {
    const languages = (['de', 'en', 'nl', 'fr', 'pl', 'it', 'es', 'sv', 'da'] as const).map((language) => ({ language, terms: 1, tribelt: 0 }))
    expect(languageBars({ total: 9, tribelt: 0, languages })).toHaveLength(LANGUAGES_SHOWN)
  })
})

describe('the AI search tile', () => {
  it("charts Tribelt's share of the newest run's answers against the four competitors named most", () => {
    const ai: OverviewAiSearch = {
      answers: 12,
      companies: [
        // The server's shares; the tile only writes them.
        { competitorId: 'a', name: 'Ashworth', role: 'competitor', answers: 6, share: 0.5 },
        { competitorId: 'b', name: 'B', role: 'competitor', answers: 5, share: 5 / 12 },
        { competitorId: 'tribelt', name: 'Tribelt', role: 'tribelt', answers: 3, share: 0.25 },
        { competitorId: 'c', name: 'C', role: 'competitor', answers: 2, share: 2 / 12 },
        { competitorId: 'd', name: 'D', role: 'competitor', answers: 1 },
      ],
    }
    expect(voiceBars(ai).map((b) => [b.label, b.value, b.first])).toEqual([
      ['Ashworth', '50%', 1],
      ['B', '42%', 5 / 6],
      ['Tribelt', '25%', 0.5],
      ['C', '17%', 2 / 6],
      ['D', '0%', 1 / 6],
    ])
    expect(aiFigure(ai)).toEqual({ big: '25%', text: 'of the AI answers in the newest Prompt run name Tribelt. Ashworth is named in 50%.' })
  })

  it('counts Tribelt as not named when no answer names it, and waits for the first run', () => {
    const none: OverviewAiSearch = { answers: 4, companies: [{ competitorId: 'tribelt', name: 'Tribelt', role: 'tribelt', answers: 0 }] }
    expect(voiceBars(none)).toEqual([{ label: 'Tribelt', value: '0%', first: 0, second: 0, zero: false }])
    expect(aiFigure(none)).toEqual({ big: '0%', text: 'of the AI answers in the newest Prompt run name Tribelt.' })
    expect(aiFigure({ answers: 0, companies: [] })).toEqual({ big: '…', text: 'of AI answers name Tribelt, once the first Prompt run is in.' })
    expect(voiceBars(undefined)).toEqual([])
  })
})

describe('the Catching up tile', () => {
  const measures: Standing[] = [
    { measure: 'aiShareOfVoice', tribelt: 0.24, leader: { competitorId: 'a', name: 'A', value: 0.48 }, competitors: 3, trails: true, leads: false },
    { measure: 'termsUsed', tribelt: 38, leader: { competitorId: 'b', name: 'B', value: 30 }, competitors: 3, trails: false, leads: true },
    { measure: 'sitemapPages', competitors: 0, trails: false, leads: false },
  ]

  it('draws Tribelt against the leader per measure, and counts the measures it trails on', () => {
    expect(gapBars(measures).map((b) => [b.label, b.value, b.first, b.second])).toEqual([
      ['AI answers', '24%', 0.5, 0.5],
      ['Search terms', '38', 1, 0],
      ['Sitemap', '–', 0, 0],
    ])
    expect(catchUpFigure(measures, 1)).toEqual({ big: '1', text: 'measure where Tribelt trails the leader.' })
    expect(catchUpFigure(measures, 3).text).toBe('measures where Tribelt trails the leaders.')
    expect(catchUpFigure([], 0).big).toBe('…')
  })
})

describe('the zoom', () => {
  it('grows from the centre of the tile', () => {
    expect(originOf({ left: 112, top: 800, width: 640, height: 1000 })).toBe('432px 1300px')
  })
})
