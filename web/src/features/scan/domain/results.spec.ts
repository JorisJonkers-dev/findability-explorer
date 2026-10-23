import { describe, expect, it } from 'vitest'
import type { Keyword, KnownTerm } from '../../../infrastructure/api/types.gen'
import {
  equivalentRows,
  isSingleWord,
  joinWords,
  keywordLanguages,
  missingTerms,
  pageLanguages,
  pagePath,
  pagesReadTitle,
  phrases,
  readSummary,
  scannedAt,
  PHRASE_KINDS,
  scoreLabel,
  siteHost,
  spectrum,
  wordsLabel,
  matchedBy,
  newTermCount,
  resultLine,
} from './results'

const keyword = (phrase: string, language: string, score: number, inMap = false): Keyword => ({
  phrase,
  language,
  score,
  pages: 1,
  tier: 'broad',
  inMap,
})
const known = (term: string, tier: KnownTerm['tier'], occurrences: number, usedByTribelt = false): KnownTerm => ({
  term,
  language: 'en',
  family: 'tri_flex_turn_curve',
  tier,
  occurrences,
  usedByTribelt,
})

describe('siteHost', () => {
  it('shows the host without www., and anything else as it is', () => {
    expect(siteHost('https://www.twentebelt.com/conveyor-belts/')).toBe('twentebelt.com')
    expect(siteHost('not an address')).toBe('not an address')
  })
})

describe('scannedAt', () => {
  it('writes the day in full and the time on a 24-hour clock', () => {
    expect(scannedAt('2026-10-22T13:10:44Z')).toMatch(/^22 October 2026 at \d\d:\d\d$/)
  })
})

describe('joinWords and pageLanguages', () => {
  it('joins a list as a sentence does', () => {
    expect(joinWords([])).toBe('')
    expect(joinWords(['English'])).toBe('English')
    expect(joinWords(['English', 'German'])).toBe('English and German')
    expect(joinWords(['English', 'German', 'Dutch'])).toBe('English, German and Dutch')
  })

  it('names the languages the most used first, skipping pages without one', () => {
    const pages = [{ language: 'de' }, { language: 'en' }, { language: 'en' }, { language: '' }, { language: 'nl' }]
    expect(pageLanguages(pages)).toEqual(['English', 'German', 'Dutch'])
  })
})

describe('readSummary', () => {
  it('says how many pages were read, and in what', () => {
    expect(readSummary({ maxPages: 10 }, Array(10).fill({ language: 'en' }))).toBe('Read 10 of 10 pages, all in English.')
    expect(readSummary({ maxPages: 15 }, [{ language: 'en' }, { language: 'de' }])).toBe(
      'Read 2 of 15 pages, in German and English.',
    )
    expect(readSummary({ maxPages: 1 }, [])).toBe('Read 0 of 1 page.')
  })
})

describe('spectrum', () => {
  it('counts the known terms by tier, every tier in the legend', () => {
    const view = spectrum([known('a', 'spec', 1), known('b', 'spec', 1), known('c', 'broad', 1)])
    expect(view.total).toBe(3)
    expect(view.bands.map((b) => [b.tier, b.label, b.count])).toEqual([
      ['spec', 'Specification', 2],
      ['product', 'Product', 0],
      ['application', 'Application', 0],
      ['problem', 'Problem', 0],
      ['broad', 'Broad', 1],
    ])
    expect(view.description).toBe('3 terms: 2 specification, 1 broad')
  })

  it('says so when there is nothing to show', () => {
    expect(spectrum([]).description).toBe('0 terms')
    expect(spectrum([known('a', 'problem', 1)]).description).toBe('1 term: 1 problem')
  })
})

describe('missingTerms', () => {
  it("keeps the terms Tribelt's website lacks, most frequent first", () => {
    const terms = [known('pitch', 'spec', 15), known('spiral belt', 'product', 50), known('eyelink', 'product', 9, true)]
    expect(missingTerms(terms).map((t) => t.term)).toEqual(['spiral belt', 'pitch'])
  })
})

describe('the phrases', () => {
  const keywords = [
    keyword('conveyor belts', 'en', 540),
    keyword('metal conveyor belts', 'en', 548.2),
    keyword('belts', 'en', 900),
    keyword('förderbänder aus metall', 'de', 100),
  ]

  it('lists each language once, by code', () => {
    expect(keywordLanguages(keywords)).toEqual(['de', 'en'])
  })

  it('knows a single word', () => {
    expect(isSingleWord(' belts ')).toBe(true)
    expect(isSingleWord('wire mesh')).toBe(false)
  })

  it('hides single words, sorts by score, and measures each bar against the top one', () => {
    const shown = phrases(keywords, { language: '', singleWords: false, kind: 'all' })
    expect(shown.map((p) => [p.phrase, p.share])).toEqual([
      ['metal conveyor belts', 1],
      ['conveyor belts', 540 / 548.2],
      ['förderbänder aus metall', 100 / 548.2],
    ])
  })

  it('shows single words on request, and one language', () => {
    expect(phrases(keywords, { language: 'en', singleWords: true, kind: 'all' }).map((p) => p.phrase)).toEqual([
      'belts',
      'metal conveyor belts',
      'conveyor belts',
    ])
    expect(phrases(keywords, { language: 'nl', singleWords: true, kind: 'all' })).toEqual([])
    expect(phrases([keyword('zero score', 'en', 0)], { language: '', singleWords: false, kind: 'all' })[0]?.share).toBe(0)
  })

  it('shows only the new terms, or only the map\'s own, on request', () => {
    const mixed = [keyword('spiral belts', 'en', 10), keyword('metal conveyor belts', 'en', 20, true)]
    expect(phrases(mixed, { language: '', singleWords: false, kind: 'new' }).map((p) => p.phrase)).toEqual(['spiral belts'])
    expect(phrases(mixed, { language: '', singleWords: false, kind: 'inMap' }).map((p) => p.phrase)).toEqual([
      'metal conveyor belts',
    ])
    expect(PHRASE_KINDS.map((k) => k.label)).toEqual(['All', 'New terms', 'In the map'])
  })

  it('shows a score in whole points', () => {
    expect(scoreLabel(548.2)).toBe('548')
    expect(scoreLabel(1234.5)).toBe('1,235')
  })
})

describe('equivalentRows', () => {
  it("pairs each product with Tribelt's, listing the site's words once each", () => {
    expect(
      equivalentRows([
        {
          product: 'TwenteFlex spiral belts',
          family: 'tri_flex_turn_curve',
          reason: 'A curve-going spiral belt.',
          searchTerms: [
            { term: 'spiral belt', language: 'en' },
            { term: 'spiral conveyor belt', language: 'en' },
            { term: 'spiral belt', language: 'en' },
          ],
        },
      ]),
    ).toEqual([
      {
        theirs: 'TwenteFlex spiral belts',
        ours: 'Tri-Flex',
        words: 'spiral belt, spiral conveyor belt',
        reason: 'A curve-going spiral belt.',
      },
    ])
  })
})

describe('the pages read', () => {
  it("shows a page by its path, without a trailing slash", () => {
    expect(pagePath('https://www.twentebelt.com/conveyor-belts/')).toBe('/conveyor-belts')
    expect(pagePath('https://www.twentebelt.com/')).toBe('/')
    expect(pagePath('https://www.twentebelt.com/search?q=belt')).toBe('/search?q=belt')
    expect(pagePath('nonsense')).toBe('nonsense')
  })

  it('counts words and pages', () => {
    expect(wordsLabel(2620)).toBe('2,620 words')
    expect(wordsLabel(1)).toBe('1 word')
    expect(pagesReadTitle(10)).toBe('The 10 pages read')
    expect(pagesReadTitle(1)).toBe('The page read')
  })
})

describe('the result line', () => {
  it('counts the pages read and the new terms', () => {
    expect(newTermCount([{ inMap: true }, { inMap: false }, { inMap: false }])).toBe(2)
    expect(resultLine(15, 4)).toBe('15 pages read · 4 new terms')
    expect(resultLine(1, 1)).toBe('1 page read · 1 new term')
  })

  it('names the model that matched the products, when there was one', () => {
    expect(matchedBy('openrouter/anthropic/claude-haiku')).toBe('Matched by openrouter/anthropic/claude-haiku')
    expect(matchedBy(undefined)).toBeNull()
    expect(matchedBy(' ')).toBeNull()
  })
})
