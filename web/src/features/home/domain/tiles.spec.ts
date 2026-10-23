import { describe, expect, it } from 'vitest'
import { competitor, term } from '../../../testing/fixtures'
import { crawlerBars, languageCount, termsText, familyBars, LANGUAGES_SHOWN, languageBars, makerCount, originOf, tribeltTermCount, usedByTribelt } from './tiles'

describe('the competitors tile', () => {
  const makers = [
    competitor('Twentebelt', { beltNames: [{ name: 'TwenteFlex', family: 'tri_flex_turn_curve' }, { name: 'Eyelink belts', family: 'eyelink' }] }),
    competitor('Märtens', { beltNames: [{ name: 'Drahtösengliederband', family: 'eyelink' }] }),
    competitor('Tribelt B.V.', { groupCompany: true }),
    competitor('Example Belts', { source: 'added' }),
  ]

  it('counts the makers the research found, not the group or companies added since', () => {
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
  })

  it('counts the languages, and says how many terms Tribelt uses', () => {
    expect(languageCount(terms)).toBe(4)
    expect(termsText(terms)).toBe("search terms in 4 languages. Tribelt's own website uses 2 of them.")
  })

  it('shows the eight largest languages only', () => {
    const many = ['de', 'en', 'nl', 'fr', 'pl', 'it', 'es', 'sv', 'da'].map((language) => term({ language: language as 'de' }))
    expect(languageBars(many)).toHaveLength(LANGUAGES_SHOWN)
  })
})

describe('the AI search tile', () => {
  it('charts the pages each crawler fetched against the most, marking the zeros', () => {
    expect(crawlerBars([{ name: 'ClaudeBot', pages: 248 }, { name: 'Googlebot', pages: 0 }]).map((b) => [b.label, b.first, b.zero])).toEqual([
      ['ClaudeBot', 1, false],
      ['Googlebot', 0, true],
    ])
    expect(crawlerBars([{ name: 'Googlebot', pages: 0 }])[0]?.first).toBe(0)
  })
})

describe('the zoom', () => {
  it('grows from the centre of the tile', () => {
    expect(originOf({ left: 112, top: 800, width: 640, height: 1000 })).toBe('432px 1300px')
  })
})
