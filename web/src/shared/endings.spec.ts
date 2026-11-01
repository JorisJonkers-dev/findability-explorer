import { describe, expect, it } from 'vitest'
import { ENDINGS, endingLabel, endingTone } from './endings'

describe('how work ended', () => {
  it('says each ending in the same words wherever work is listed', () => {
    expect(ENDINGS.map(endingLabel)).toEqual(['Done', 'Done, with failures', 'Stopped by hand', 'Stopped by the budget', 'Failed'])
  })

  it('gives each ending the tone of its pill', () => {
    expect(ENDINGS.map(endingTone)).toEqual(['good', 'warn', 'warn', 'warn', 'bad'])
  })
})
