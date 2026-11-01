import { describe, expect, it } from 'vitest'
import { foundLine } from './routine'

describe('an improvement in words', () => {
  it('says what the latest scan of tribelt.nl found, and what it did not', () => {
    const part = (name: string, present: boolean) => ({ name, present })
    expect(foundLine([])).toBe('')
    expect(foundLine([part('llms.txt', true), part('llms-full.txt', true)])).toBe('The latest scan found llms.txt and llms-full.txt.')
    expect(foundLine([part('llms.txt', false), part('llms-full.txt', false)])).toBe('The latest scan did not find llms.txt or llms-full.txt.')
    expect(foundLine([part('a Dutch sitemap', true), part('a German sitemap', false), part('an English sitemap', true)])).toBe(
      'The latest scan found a Dutch sitemap and an English sitemap, but not a German sitemap.',
    )
    expect(foundLine([part('Product structured data', false), part('FAQ structured data', true)])).toBe('The latest scan found FAQ structured data, but not Product structured data.')
  })
})
