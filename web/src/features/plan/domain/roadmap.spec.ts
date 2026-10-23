import { describe, expect, it } from 'vitest'
import { GOALS, MONTHS, MOVES, PHASES } from './content'
import { gridColumn, phaseLabel, toggleOpen } from './roadmap'

describe('the roadmap', () => {
  it('lays the three phases over twelve months', () => {
    expect(PHASES.map(gridColumn)).toEqual(['1 / span 3', '4 / span 3', '7 / span 6'])
    expect(PHASES.reduce((months, p) => months + p.span, 0)).toBe(MONTHS.length)
    expect(MONTHS.at(-1)).toBe('M12')
  })

  it('names a phase for a screen reader', () => {
    expect(phaseLabel(PHASES[1] as (typeof PHASES)[number])).toBe('Build and target, months 3–6, implementing')
  })

  it('opens one move at a time, and closes the open one', () => {
    expect(toggleOpen(0, 0)).toBeNull()
    expect(toggleOpen(0, 2)).toBe(2)
    expect(toggleOpen(null, 1)).toBe(1)
  })

  it('has five moves and five goals, two of them with a figure today', () => {
    expect(MOVES.map((m) => m.short)).toEqual(['Measure qualified leads', 'Restructure paid search', 'A page per belt type', 'Get indexed for AI search', 'Reviews after delivery'])
    expect(GOALS.filter((g) => g.bar).map((g) => g.id)).toEqual(['G2', 'G4'])
  })
})
