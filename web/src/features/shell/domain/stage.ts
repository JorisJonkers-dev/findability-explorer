/** The stage's own size, in its own pixels: a 16:9 slide, 4K wide. */
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
  /** Where its top-left corner lands, in window pixels: it is centred, with white bands around. */
  left: number
  top: number
}

/** Fits the stage inside a window of width by height, whole and centred. */
export function fitStage(width: number, height: number): StageFit {
  const scale = Math.max(Math.min(width / STAGE_WIDTH, height / STAGE_HEIGHT), 0)
  return {
    scale,
    left: (width - STAGE_WIDTH * scale) / 2,
    top: (height - STAGE_HEIGHT * scale) / 2,
  }
}

/** The stage's transform for a fit. */
export function stageTransform(fit: StageFit): string {
  return `translate(${String(fit.left)}px, ${String(fit.top)}px) scale(${String(fit.scale)})`
}
