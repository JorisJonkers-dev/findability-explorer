import { describe, expect, it } from 'vitest'
import { fitStage, STAGE_HEIGHT, STAGE_WIDTH, stageStyle } from './stage'

describe('fitting the stage to the window', () => {
  it('fills a 16:9 window at the design size', () => {
    expect(fitStage(1920, 1080)).toEqual({ scale: 0.5, width: 3840, height: 2160 })
  })

  it('grows wider for a window wider than 16:9, keeping the design scale', () => {
    // A 2560 by 1440 screen less the browser's own bars.
    const fit = fitStage(2560, 1300)
    expect(fit.scale).toBeCloseTo(1300 / 2160)
    expect(fit.height).toBe(STAGE_HEIGHT)
    expect(fit.width * fit.scale).toBeCloseTo(2560)
    expect(fitStage(2400, 1080)).toEqual({ scale: 0.5, width: 4800, height: 2160 })
  })

  it('grows higher for a window taller than 16:9', () => {
    expect(fitStage(1920, 1200)).toEqual({ scale: 0.5, width: 3840, height: 2400 })
  })

  it('keeps the design size for a window of nothing', () => {
    expect(fitStage(0, 0)).toEqual({ scale: 0, width: STAGE_WIDTH, height: STAGE_HEIGHT })
    expect(STAGE_WIDTH / STAGE_HEIGHT).toBeCloseTo(16 / 9)
  })

  it('writes the fit as a style', () => {
    expect(stageStyle({ scale: 0.5, width: 3840, height: 2400 })).toEqual({ width: '3840px', height: '2400px', transform: 'scale(0.5)' })
  })
})
