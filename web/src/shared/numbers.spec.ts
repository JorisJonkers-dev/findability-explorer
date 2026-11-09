import { describe, expect, it } from 'vitest'
import { dollars, plural } from './numbers'

describe('dollars', () => {
  it('writes cents by default, more decimals where a place needs them, or two significant digits for tiny costs', () => {
    expect(dollars(2.75)).toBe('$2.75')
    expect(dollars(1234.5)).toBe('$1,234.50')
    expect(dollars(0.0042, { decimals: 4 })).toBe('$0.0042')
    expect(dollars(1.2, { decimals: 4 })).toBe('$1.20')
    expect(dollars(0.00143, { significant: 2 })).toBe('$0.0014')
    expect(dollars(0, { significant: 2 })).toBe('$0')
  })
})

describe('plural', () => {
  it('counts a thing in the singular or the plural, with thousands grouped', () => {
    expect(plural(1, 'question')).toBe('1 question')
    expect(plural(1200, 'question')).toBe('1,200 questions')
    expect(plural(0, 'company', 'companies')).toBe('0 companies')
  })
})
