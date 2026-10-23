import type { Phase } from './content'

/** A phase's place on the 12-month grid, as a grid-column value: "4 / span 3". */
export function gridColumn(phase: Pick<Phase, 'start' | 'span'>): string {
  return `${String(phase.start)} / span ${String(phase.span)}`
}

/** A phase's button in words, for a screen reader: "Measure and fix, months 0–3, preparing". */
export function phaseLabel(phase: Pick<Phase, 'name' | 'when' | 'mode'>): string {
  return `${phase.name}, ${phase.when.toLowerCase()}, ${phase.mode.toLowerCase()}`
}

/** Which move to open next: clicking the open one closes it, any other opens instead. */
export function toggleOpen(open: number | null, index: number): number | null {
  return open === index ? null : index
}
