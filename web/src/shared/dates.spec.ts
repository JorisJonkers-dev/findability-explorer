import { describe, expect, it } from 'vitest'
import { clock, dayMonth, localDate, longDate } from './dates'

describe('the dates a chart and its notes show', () => {
  it('writes a moment in UTC, as the runs are stamped', () => {
    expect(longDate('2026-10-27T06:03:12Z')).toBe('27 October 2026')
    expect(dayMonth('2026-10-27T23:59:00Z')).toBe('27 October')
    expect(clock('2026-10-27T06:03:12Z')).toBe('06:03')
    expect(clock('2026-10-27T00:00:00Z')).toBe('00:00')
  })
  it('writes a day as it was in the Netherlands', () => {
    expect(localDate('2026-10-31T23:30:00Z')).toBe('1 November 2026')
    expect(localDate('2026-10-31T22:30:00Z')).toBe('31 October 2026')
  })
})
