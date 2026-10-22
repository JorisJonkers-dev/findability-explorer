import { describe, expect, it } from 'vitest'
import { fold, highlight } from './fold'

describe('fold', () => {
  it('folds case and accents', () => {
    expect(fold('Drahtösengliederband')).toBe('drahtosengliederband')
    expect(fold('MÄRTENS')).toBe('martens')
    expect(fold('Ça va')).toBe('ca va')
  })

  it('spells out the letters NFD keeps whole', () => {
    expect(fold('Maßband')).toBe('massband')
    expect(fold('GROẞ')).toBe('gross')
    expect(fold('Ørsted Łódź')).toBe('orsted lodz')
  })

  it('reads every run of separators as one space, trimmed', () => {
    expect(fold('  Tri-Flex / turn_curve\tbelt  ')).toBe('tri flex turn curve belt')
    expect(fold(' - / ')).toBe('')
    expect(fold('')).toBe('')
  })
})

/** The marked parts of a highlight, for short assertions. */
function marked(text: string, query: string): string {
  return highlight(text, query)
    .map((s) => (s.match ? `[${s.text}]` : s.text))
    .join('')
}

describe('highlight', () => {
  it('marks nothing for an empty query', () => {
    expect(highlight('Eyelink belt', '  ')).toEqual([{ text: 'Eyelink belt', match: false }])
  })

  it('marks every occurrence, keeping the original spelling', () => {
    expect(highlight('Spiral belt, spiral', 'SPIRAL')).toEqual([
      { text: 'Spiral', match: true },
      { text: ' belt, ', match: false },
      { text: 'spiral', match: true },
    ])
  })

  it('agrees with the server on accents and ß', () => {
    expect(marked('Drahtösengliederband', 'osen')).toBe('Draht[ösen]gliederband')
    expect(marked('Maßband', 'massb')).toBe('[Maßb]and')
    expect(marked('Strasse', 'straße')).toBe('[Strasse]')
  })

  it('marks a whole letter that folds to two', () => {
    expect(marked('Maß', 's')).toBe('Ma[ß]')
  })

  it('matches across any run of separators', () => {
    expect(marked('Tri-Flex  belt', 'tri flex')).toBe('[Tri-Flex]  belt')
    expect(marked('Tri-Flex  belt', 'flex belt')).toBe('Tri-[Flex  belt]')
    expect(marked(' -Tri', 'tri')).toBe(' -[Tri]')
  })

  it('marks nothing when the query does not occur', () => {
    expect(highlight('Eyelink', 'spiral')).toEqual([{ text: 'Eyelink', match: false }])
  })
})
