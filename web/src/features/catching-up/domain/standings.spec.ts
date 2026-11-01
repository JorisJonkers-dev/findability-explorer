import { describe, expect, it } from 'vitest'
import { formatValue, inWords } from './standings'

describe('the values', () => {
  it('write shares as percentages, scores with a half, counts with separators', () => {
    expect(formatValue('aiShareOfVoice', 0.236)).toBe('24%')
    expect(formatValue('citationShare', 0)).toBe('0%')
    expect(formatValue('readinessChecks', 3.5)).toBe('3½')
    expect(formatValue('readinessChecks', 5)).toBe('5')
    expect(formatValue('sitemapCoverage', 0.85)).toBe('85%')
    expect(formatValue('termsUsed', 18)).toBe('18')
  })

  it('count small numbers in words', () => {
    expect(inWords(0)).toBe('no')
    expect(inWords(4)).toBe('four')
    expect(inWords(12)).toBe('12')
  })
})
