/** Words joined as a sentence lists them: "a", "a and b", "a, b and c"; or with "or". */
export function joinWords(words: readonly string[], word: 'and' | 'or' = 'and'): string {
  const last = words.slice(-1).join('')
  return words.length <= 1 ? last : `${words.slice(0, -1).join(', ')} ${word} ${last}`
}
