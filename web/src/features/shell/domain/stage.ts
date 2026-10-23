/** The stage's design size, in its own pixels: a 16:9 slide, 4K wide. The views are drawn at it. */
export const STAGE_WIDTH = 3840
export const STAGE_HEIGHT = 2160

/**
 * When the views are drawn on the stage: a window at least 1024 by 576. Below that they stack
 * into a page that scrolls. styles.css declares the same query as the `stage:` variant.
 */
export const STAGE_QUERY = '(min-width: 1024px) and (min-height: 576px)'

export type StageFit = {
  /** How much the stage is scaled to fit the window. */
  scale: number
  /** The stage's size in its own pixels: the design size, grown along the window's longer side. */
  width: number
  height: number
}

/**
 * Fits the stage to a window of width by height so that it fills the window exactly. The scale is
 * the one that fits the 3840 by 2160 design whole, so every size the views are drawn at stays; the
 * stage then grows to the window's aspect: wider than 16:9, it is 2160 high and wider than 3840;
 * taller, it is 3840 wide and higher than 2160. The views lay themselves out fluidly inside it.
 */
export function fitStage(width: number, height: number): StageFit {
  const scale = Math.max(Math.min(width / STAGE_WIDTH, height / STAGE_HEIGHT), 0)
  if (scale === 0) return { scale, width: STAGE_WIDTH, height: STAGE_HEIGHT }
  return {
    scale,
    width: Math.max(STAGE_WIDTH, width / scale),
    height: Math.max(STAGE_HEIGHT, height / scale),
  }
}

/** The stage's size and transform for a fit, as a style binding. */
export function stageStyle(fit: StageFit): { width: string; height: string; transform: string } {
  return { width: `${String(fit.width)}px`, height: `${String(fit.height)}px`, transform: `scale(${String(fit.scale)})` }
}
