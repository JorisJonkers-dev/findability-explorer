/** How a column of a page's stage is shown: as designed, widened, or minimised to a strip. */
export type ColumnState = 'normal' | 'wide' | 'min'

/** How much more room a widened column takes than its designed share. */
export const WIDEN_FACTOR = 2.5

/** A minimised column's width on the stage, in the slide's pixels. */
export const STRIP_WIDTH = 150

/**
 * The stage grid's columns: each its designed share (base, in fr), a widened one WIDEN_FACTOR
 * times that, a minimised one a strip.
 */
export function columnTemplate(base: readonly number[], states: readonly ColumnState[]): string {
  return base
    .map((share, i) => {
      const state = states[i] ?? 'normal'
      if (state === 'min') return `${String(STRIP_WIDTH)}px`
      return `minmax(0,${String(state === 'wide' ? share * WIDEN_FACTOR : share)}fr)`
    })
    .join(' ')
}

/** Widens column i, or puts a widened one back; a minimised one opens widened. */
export function widened(states: readonly ColumnState[], i: number): ColumnState[] {
  return states.map((s, j) => (j === i ? (s === 'wide' ? 'normal' : 'wide') : s))
}

/** Minimises column i, unless it is the last one open: a page always shows one column. */
export function minimised(states: readonly ColumnState[], i: number): ColumnState[] {
  const open = states.filter((s) => s !== 'min').length
  if (states[i] === 'min' || open <= 1) return [...states]
  return states.map((s, j) => (j === i ? 'min' : s))
}

/** Restores column i to its designed width. */
export function restored(states: readonly ColumnState[], i: number): ColumnState[] {
  return states.map((s, j) => (j === i ? 'normal' : s))
}

/** Reads a stored layout back, or null when it is not one for count columns. */
export function parseLayout(raw: string | null, count: number): ColumnState[] | null {
  if (raw === null) return null
  try {
    const value: unknown = JSON.parse(raw)
    if (!Array.isArray(value) || value.length !== count) return null
    if (!value.every((s): s is ColumnState => s === 'normal' || s === 'wide' || s === 'min')) return null
    return value.some((s) => s !== 'min') ? value : null
  } catch {
    return null
  }
}
