import { describe, expect, it } from 'vitest'
import { percent } from './percent'

describe('percent', () => {
  it('writes a fraction as a bar length, clamped to the bar', () => {
    expect(percent(0.25)).toBe('25%')
    expect(percent(2.69 / 78.89)).toBe('3.41%')
    expect(percent(1.5)).toBe('100%')
    expect(percent(-1)).toBe('0%')
    expect(percent(Number.NaN)).toBe('0%')
  })
})
