import { describe, expect, it } from 'vitest'
import { PLAN_BARS } from './content'

describe("the plan tile's bar", () => {
  it('names the three phases short, in their months', () => {
    expect(PLAN_BARS.map((b) => [b.label, b.start, b.span, b.tone])).toEqual([
      ['Measure', 1, 3, 'orange'],
      ['Build', 4, 3, 'black'],
      ['Expand and learn', 7, 6, 'blue-grey'],
    ])
  })
})
