import { describe, expect, it } from 'vitest'
import type { Standing } from '../../../infrastructure/api/types.gen'
import { competitor, promptResults, term } from '../../../testing/fixtures'
import { aiFigure, catchUpFigure, gapBars, languageCount, termsText, familyBars, languageBars, makerCount, originOf, tribeltTermCount, usedByTribelt, voiceBars } from './tiles'

describe('the competitors tile', () => {
  const makers = [
    competitor('Twentebelt', { beltNames: [{ name: 'TwenteFlex', family: 'tri_flex_turn_curve' }, { name: 'Eyelink belts', family: 'eyelink' }] }),
    competitor('Märtens', { beltNames: [{ name: 'Drahtösengliederband', family: 'eyelink' }] }),
    competitor('Tribelt B.V.', { groupCompany: true }),
    competitor('Example Belts', { source: 'added' }),
    competitor('Mirror of tribelt.nl', { role: 'mirror' }),
  ]

  it('counts the makers the research found, not the group, the mirror or companies added since', () => {
    expect(makerCount(makers)).toBe(2)
  })

  it('counts the makers offering each family, of all of them', () => {
    expect(familyBars(makers).map((b) => [b.label, b.value, b.first, b.zero])).toEqual([
      ['Eyelink', '2', 1, false],
      ['Open spiral', '0', 0, true],
      ['Closed spiral', '0', 0, true],
      ['Tri-Flex', '1', 0.5, false],
      ['Other belts', '0', 0, true],
    ])
    expect(familyBars([]).every((b) => b.first === 0)).toBe(true)
  })
})

describe('the search terms tile', () => {
  const terms = [
    term({ language: 'en', usedBy: ['Tribelt (Tri-Flex)', 'Twentebelt'] }),
    term({ language: 'en-US', usedBy: ['Ashworth'] }),
    term({ language: 'de', usedBy: ['Tribelt'] }),
    term({ language: 'nl', usedBy: ['Märtens'] }),
  ]

  it("knows Tribelt's own terms, also those it credits to a product line", () => {
    expect(usedByTribelt(terms[0] as (typeof terms)[0])).toBe(true)
    expect(usedByTribelt({ usedBy: ['Tribeltish'] })).toBe(false)
    expect(tribeltTermCount(terms)).toBe(2)
  })

  it('charts the terms per language, American English as English, against the largest', () => {
    expect(languageBars(terms).map((b) => [b.label, b.value, b.first, b.second])).toEqual([
      ['English', '2', 0.5, 0.5],
      ['German', '1', 0.5, 0],
      ['Dutch', '1', 0, 0.5],
    ])
    expect(languageBars([])).toEqual([])
    // The server's counts cover terms that were not loaded, so a language past the limit still shows.
    const counted = languageBars(terms, [
      { value: 'de', count: 120 },
      { value: 'en', count: 70 },
      { value: 'en-US', count: 10 },
      { value: 'nl', count: 187 },
      { value: 'fr', count: 0 },
    ])
    expect(counted.map((b) => [b.label, b.value])).toEqual([
      ['Dutch', '187'],
      ['German', '120'],
      ['English', '80'],
    ])
    expect(languageCount(terms, [{ value: 'de', count: 1 }, { value: 'nl', count: 2 }, { value: 'fr', count: 0 }])).toBe(2)
    expect(termsText(terms, [{ value: 'nl', count: 2 }])).toMatch(/^search terms in 1 languages\./)
  })

  it('counts the languages, and says how many terms Tribelt uses', () => {
    expect(languageCount(terms)).toBe(3)
    expect(termsText(terms)).toBe("search terms in 3 languages. Tribelt's own website uses 2 of them.")
  })

  it('shows every language with a term, none cut off', () => {
    const many = ['de', 'en', 'nl', 'fr', 'pl', 'it', 'es', 'sv', 'da'].map((language) => term({ language: language as 'de' }))
    expect(languageBars(many)).toHaveLength(9)
  })
})

describe('the AI search tile', () => {
  it("charts Tribelt's share of the newest run's answers against the four competitors named most", () => {
    const results = promptResults({
      mentions: [
        { competitorId: 'a', name: 'Ashworth', role: 'competitor', answers: 6, first: 1, averagePosition: 1 },
        { competitorId: 'b', name: 'B', role: 'competitor', answers: 5, first: 0, averagePosition: 2 },
        { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', answers: 3, first: 1, averagePosition: 2 },
        { competitorId: 'c', name: 'C', role: 'competitor', answers: 2, first: 0, averagePosition: 2 },
        { competitorId: 'd', name: 'D', role: 'competitor', answers: 1, first: 0, averagePosition: 2 },
        { competitorId: 'e', name: 'E', role: 'competitor', answers: 1, first: 0, averagePosition: 2 },
      ],
    })
    expect(voiceBars(results).map((b) => [b.label, b.value, b.first])).toEqual([
      ['Ashworth', '50%', 1],
      ['B', '42%', 5 / 6],
      ['Tribelt', '25%', 0.5],
      ['C', '17%', 2 / 6],
      ['D', '8%', 1 / 6],
    ])
    expect(aiFigure(results)).toEqual({ big: '25%', text: 'of AI answers name Tribelt. Ashworth is named in 50%.' })
  })

  it('counts Tribelt as not named when no answer names it, and waits for the first run', () => {
    const none = promptResults({ mentions: [] })
    expect(voiceBars(none)).toEqual([{ label: 'Tribelt', value: '0%', first: 0, second: 0, zero: false }])
    expect(aiFigure(none)).toEqual({ big: '0%', text: 'of AI answers name Tribelt.' })
    expect(aiFigure(promptResults({ runs: [] }))).toEqual({ big: '…', text: 'of AI answers name Tribelt, once the first Prompt run is in.' })
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
