/** A fraction from 0 to 1 as a CSS length for a bar: 0.25 is "25%". Out-of-range values are clamped. */
export function percent(fraction: number): string {
  const clamped = Number.isFinite(fraction) ? Math.min(Math.max(fraction, 0), 1) : 0
  return `${String(Math.round(clamped * 10_000) / 100)}%`
}
