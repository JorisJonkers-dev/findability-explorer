/** Words joined as a sentence lists them: "a", "a and b", "a, b and c"; or with "or". */
export function joinWords(words: readonly string[], word: 'and' | 'or' = 'and'): string {
  const last = words.slice(-1).join('')
  return words.length <= 1 ? last : `${words.slice(0, -1).join(', ')} ${word} ${last}`
}

/** Words joined as a sentence lists them, with the Oxford comma: "a", "a and b", "a, b, and c". */
export function oxford(words: readonly string[], word: 'and' | 'or' = 'and'): string {
  const last = words.slice(-1).join('')
  if (words.length <= 2) return words.length === 2 ? words.join(` ${word} `) : last
  return `${words.slice(0, -1).join(', ')}, ${word} ${last}`
}

/**
 * The tool's words for its own ideas, as the glossary has them: a response (never an answer), a
 * search term (never a phrase), the best competitor (never the leader), the typical competitor,
 * buyer intent (never an intent tier), and a goal (never a target).
 */
export const WORDS = {
  response: 'response',
  responses: 'responses',
  searchTerm: 'search term',
  searchTerms: 'search terms',
  bestCompetitor: 'best competitor',
  typicalCompetitor: 'typical competitor',
  buyerIntent: 'buyer intent',
  goal: 'goal',
  goals: 'goals',
} as const

const OLD_WORDS = /\b(answers?|phrases?|leaders?|intent tiers?|targets?)\b/i

/** Whether text uses a word the tool has replaced: answer, phrase, leader, intent tier, or target. */
export function usesOldWord(text: string): boolean {
  return OLD_WORDS.test(text)
}
