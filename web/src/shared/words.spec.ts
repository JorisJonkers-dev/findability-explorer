import { describe, expect, it } from 'vitest'
import { BUYER_INTENT } from './tiers'
import { FAMILIES, familyLabel, LANGUAGES, languageLabel, scopeLabel, TIERS, tierLabel, tribeltProduct } from './vocabulary'
import { joinWords, oxford, usesOldWord, WORDS } from './words'

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

describe("the tool's words", () => {
  it('says response, search term, best competitor, typical competitor, buyer intent, and goal', () => {
    expect(WORDS).toEqual({
      response: 'response',
      responses: 'responses',
      searchTerm: 'search term',
      searchTerms: 'search terms',
      bestCompetitor: 'best competitor',
      typicalCompetitor: 'typical competitor',
      buyerIntent: 'buyer intent',
      goal: 'goal',
      goals: 'goals',
    })
  })

  it('knows the old words when it sees them', () => {
    for (const old of ['An answer', 'answers', 'a phrase', 'the leader', 'Leaders', 'intent tier', 'Target', 'targets']) {
      expect(usesOldWord(old), old).toBe(true)
    }
    for (const now of ['A response', 'search terms', 'the best competitor', 'buyer intent', 'goal', 'answered', 'targeted']) {
      expect(usesOldWord(now), now).toBe(false)
    }
  })

  it('never uses answer, phrase, leader, intent tier, or target in a shared label', () => {
    const labels = [
      ...Object.values(WORDS),
      ...TIERS.flatMap((t) => [tierLabel(t), BUYER_INTENT[t].label, BUYER_INTENT[t].help]),
      ...FAMILIES.flatMap((f) => [familyLabel(f), scopeLabel(f), tribeltProduct(f)]),
      ...LANGUAGES.map(languageLabel),
    ]
    expect(labels.filter(usesOldWord)).toEqual([])
  })
})
