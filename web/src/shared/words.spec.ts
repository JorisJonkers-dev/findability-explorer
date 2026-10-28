import { describe, expect, it } from 'vitest'
import { joinWords, oxford } from './words'

describe('joinWords', () => {
  it('lists words as a sentence does, with and or or', () => {
    expect(joinWords([])).toBe('')
    expect(joinWords(['English'])).toBe('English')
    expect(joinWords(['English', 'German'])).toBe('English and German')
    expect(joinWords(['English', 'German', 'Dutch'])).toBe('English, German and Dutch')
    expect(joinWords(['llms.txt', 'llms-full.txt'], 'or')).toBe('llms.txt or llms-full.txt')
  })
})

describe('oxford', () => {
  it('lists words with the Oxford comma', () => {
    expect(oxford([])).toBe('')
    expect(oxford(['a'])).toBe('a')
    expect(oxford(['a', 'b'])).toBe('a and b')
    expect(oxford(['a', 'b', 'c'])).toBe('a, b, and c')
    expect(oxford(['a', 'b', 'c'], 'or')).toBe('a, b, or c')
  })
})
