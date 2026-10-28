/** The colour roles a chart mark can take. */
export type Tone = 'orange' | 'blue-grey' | 'ink' | 'silver'

/** A tone as the app's Tailwind colour token. */
export const TONE_VAR: Record<Tone, string> = {
  orange: 'var(--color-orange)',
  'blue-grey': 'var(--color-blue-grey)',
  ink: 'var(--color-ink-2)',
  silver: 'var(--color-silver)',
}

export const TONE_BG: Record<Tone, string> = {
  orange: 'bg-orange',
  'blue-grey': 'bg-blue-grey',
  ink: 'bg-ink-2',
  silver: 'bg-silver',
}
