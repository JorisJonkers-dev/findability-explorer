import { describe, expect, it } from 'vitest'
import { pct } from './summary'

describe('a share', () => {
  it('is written as a rounded percentage', () => {
    expect([pct(0.24), pct(0.245), pct(0), pct(1)]).toEqual(['24%', '25%', '0%', '100%'])
  })
})
