// The server matches a query against a term by folding both (internal/terminology/domain Fold):
// lower case, Unicode NFD without combining marks, ß → ss, ø → o, ł → l, and every run of
// whitespace, '-', '/' or '_' as one space, trimmed. This module folds the same way, and keeps
// track of where each folded character came from so a match can be marked in the original text.

const LETTERS: Readonly<Record<string, string>> = { ß: 'ss', ø: 'o', ł: 'l' }
const SEPARATOR = /^[\s\-/_]$/u
const COMBINING_MARK = /\p{Mn}/gu

/** One code point folded: lower case, accents dropped, and the letters NFD keeps spelled out. */
function foldCodePoint(char: string): string {
  return Array.from(char.toLowerCase().normalize('NFD').replace(COMBINING_MARK, ''))
    .map((c) => LETTERS[c] ?? c)
    .join('')
}

type Folded = {
  /** The folded text. */
  text: string
  /** For each UTF-16 unit of the folded text, the original's [start, end) it came from. */
  spans: [number, number][]
}

function foldMapped(original: string): Folded {
  let text = ''
  const spans: [number, number][] = []
  let offset = 0
  for (const char of original) {
    const start = offset
    offset += char.length
    const folded = foldCodePoint(char)
    if (SEPARATOR.test(folded)) {
      const last = spans.at(-1)
      // A leading separator is trimmed; one after another widens the space already there.
      if (last === undefined) continue
      if (text.endsWith(' ')) {
        last[1] = offset
        continue
      }
      text += ' '
      spans.push([start, offset])
      continue
    }
    for (const unit of folded) {
      text += unit
      spans.push([start, offset])
    }
  }
  if (text.endsWith(' ')) {
    text = text.slice(0, -1)
    spans.pop()
  }
  return { text, spans }
}

/** Folds text the way the server does before it compares a query with a term. */
export function fold(text: string): string {
  return foldMapped(text).text
}

export type Segment = { text: string; match: boolean }

/**
 * Splits text into runs that do and do not match query, by the server's folding rules, so the
 * matches can be marked. Every occurrence is marked; a query that folds to nothing marks nothing.
 */
export function highlight(text: string, query: string): Segment[] {
  const needle = fold(query)
  if (needle === '') return [{ text, match: false }]
  const { text: haystack, spans } = foldMapped(text)
  const segments: Segment[] = []
  let cursor = 0
  let from = 0
  for (let at = haystack.indexOf(needle, from); at >= 0; at = haystack.indexOf(needle, from)) {
    from = at + needle.length
    const first = spans[at] as [number, number]
    const last = spans[from - 1] as [number, number]
    // A match that begins inside a letter an earlier match already covered (the second s of ß)
    // starts where that one ended.
    const start = Math.max(first[0], cursor)
    if (last[1] <= start) continue
    if (start > cursor) segments.push({ text: text.slice(cursor, start), match: false })
    segments.push({ text: text.slice(start, last[1]), match: true })
    cursor = last[1]
  }
  if (cursor < text.length) segments.push({ text: text.slice(cursor), match: false })
  return segments
}
