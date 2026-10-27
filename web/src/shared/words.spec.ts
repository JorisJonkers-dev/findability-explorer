import { describe, expect, it } from 'vitest'
import { joinWords } from './words'

describe('joinWords', () => {
  it('lists words as a sentence does, with and or or', () => {
    expect(joinWords([])).toBe('')
    expect(joinWords(['English'])).toBe('English')
    expect(joinWords(['English', 'German'])).toBe('English and German')
    expect(joinWords(['English', 'German', 'Dutch'])).toBe('English, German and Dutch')
    expect(joinWords(['llms.txt', 'llms-full.txt'], 'or')).toBe('llms.txt or llms-full.txt')
  })
})
