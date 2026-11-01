import type { Tone } from './tones'

/** One run's value on a line; a partial run gets a dashed ring. */
export type LinesPoint = { x: string; y: number; partial?: boolean }

/** A line of LinesChart, in its tone; the strong one is drawn thicker. */
export type LinesSeries = { label: string; tone: Tone; strong?: boolean; points: readonly LinesPoint[] }

/** Where the measure changed: the lines stop and a labelled band starts. */
export type LinesBreak = { x: string; label: string }
