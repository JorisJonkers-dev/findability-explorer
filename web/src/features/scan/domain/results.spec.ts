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
  scoreLabel,
  siteHost,
  siteName,
  spectrum,
  wordsLabel,
} from './results'

const keyword = (phrase: string, language: string, score: number): Keyword => ({ phrase, language, score, pages: 1 })
const known = (term: string, tier: KnownTerm['tier'], occurrences: number, usedByTribelt = false): KnownTerm => ({
  term,
  language: 'en',
  family: 'tri_flex_turn_curve',
  tier,
  occurrences,
  usedByTribelt,
})

describe('siteHost and siteName', () => {
  it('shows the host without www., and anything else as it is', () => {
    expect(siteHost('https://www.twentebelt.com/conveyor-belts/')).toBe('twentebelt.com')
    expect(siteHost('not an address')).toBe('not an address')
  })

  it("names the site after the company whose website it is, or else its host", () => {
    const sites = [
      { name: 'Cambridge', website: 'https://www.cambridge-es.com (blocked by Cloudflare 403)' },
      { name: 'Twentebelt', website: 'https://www.twentebelt.com' },
    ]
    expect(siteName('https://twentebelt.com/conveyor-belts/', sites)).toBe('Twentebelt')
    expect(siteName('https://www.cambridge-es.com/', sites)).toBe('Cambridge')
    expect(siteName('https://www.example.com/', sites)).toBe('example.com')
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
    const shown = phrases(keywords, { language: '', singleWords: false })
    expect(shown.map((p) => [p.phrase, p.share])).toEqual([
      ['metal conveyor belts', 1],
      ['conveyor belts', 540 / 548.2],
      ['förderbänder aus metall', 100 / 548.2],
    ])
  })

  it('shows single words on request, and one language', () => {
    expect(phrases(keywords, { language: 'en', singleWords: true }).map((p) => p.phrase)).toEqual([
      'belts',
      'metal conveyor belts',
      'conveyor belts',
    ])
    expect(phrases(keywords, { language: 'nl', singleWords: true })).toEqual([])
    expect(phrases([keyword('zero score', 'en', 0)], { language: '', singleWords: false })[0]?.share).toBe(0)
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
