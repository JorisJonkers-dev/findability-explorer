import { describe, expect, it } from 'vitest'
import { fitStage, STAGE_HEIGHT, STAGE_WIDTH, stageTransform } from './stage'

describe('fitting the stage to the window', () => {
  it('fills a 16:9 window exactly', () => {
    expect(fitStage(1920, 1080)).toEqual({ scale: 0.5, left: 0, top: 0 })
  })

  it('letterboxes a window that is too tall, and pillarboxes one that is too wide', () => {
    expect(fitStage(1920, 1200)).toEqual({ scale: 0.5, left: 0, top: 60 })
    expect(fitStage(2400, 1080)).toEqual({ scale: 0.5, left: 240, top: 0 })
  })

  it('never scales below nothing', () => {
    expect(fitStage(0, 0).scale).toBe(0)
    expect(STAGE_WIDTH / STAGE_HEIGHT).toBeCloseTo(16 / 9)
  })

  it('writes the fit as a transform', () => {
    expect(stageTransform({ scale: 0.5, left: 0, top: 60 })).toBe('translate(0px, 60px) scale(0.5)')
  })
})
