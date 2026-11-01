import { describe, expect, it } from 'vitest'
import { bots, readiness } from '../../../testing/fixtures'
import { barLength, measuresOf, oxford, readinessScore, scoreText, shareText } from './compare'

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
    expect(readinessScore(readiness())).toBe(8)
    expect(readinessScore(readiness({ bots: bots({ GPTBot: 'blocked' }) }))).toBe(7.5)
    expect(readinessScore(readiness({ bots: bots({ GPTBot: 'blocked' }), markdown: false, llmsFullTxt: false }))).toBe(5.5)
    expect(readinessScore(readiness({ robots: 'unreachable' }))).toBe(7)
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

describe('measuresOf', () => {
  const known = { name: 'Twentebelt', share: 0.4, tribeltShare: 0.2, ready: 3.5, tribeltReady: 5, outOf: 5, sitemap: 0.5, tribeltSitemap: 0.85, terms: 12, tribeltTerms: 1200, mapTerms: 2400 }
  const by = (input: Parameters<typeof measuresOf>[0]) => Object.fromEntries(measuresOf(input).map((m) => [m.id, m]))

  it('sets the competitor against Tribelt on four measures', () => {
    const m = by(known)
    expect(Object.keys(m)).toEqual(['answers', 'ready', 'sitemap', 'terms'])
    expect(m.answers).toMatchObject({ themText: '40%', usText: '20%', scale: 0.4, path: '/ai-responses/all' })
    expect(m.answers?.text).toBe('Twentebelt is named in 40% of the answers in the newest Prompt run, and Tribelt in 20%.')
    expect(m.ready).toMatchObject({ themText: '3½ of 5', usText: '5 of 5', scale: 5 })
    expect(m.ready?.text).toBe('Twentebelt’s site passes 3½ of 5 of the AI readiness checks, and tribelt.nl passes 5 of 5.')
    expect(m.sitemap).toMatchObject({ themText: '50%', usText: '85%', scale: 1 })
    expect(m.sitemap?.text).toBe('Twentebelt’s sitemap lists 50% of the pages its newest scan read, against 85% on tribelt.nl.')
    expect(m.terms).toMatchObject({ themText: '12', usText: '1,200', scale: 1200 })
    expect(m.terms?.text).toBe('tribelt.nl uses 1,200 of the 2,400 terms on the search term list. Twentebelt’s site uses 12.')
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

