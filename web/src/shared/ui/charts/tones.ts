/** The colour roles a chart mark can take. */
export type Tone = 'orange' | 'blue-grey' | 'ink' | 'silver' | 'blue' | 'aqua' | 'violet'

/** A tone as the app's Tailwind colour token. */
export const TONE_VAR: Record<Tone, string> = {
  orange: 'var(--color-orange)',
  'blue-grey': 'var(--color-blue-grey)',
  ink: 'var(--color-ink-2)',
  silver: 'var(--color-silver)',
  blue: 'var(--color-line-blue)',
  aqua: 'var(--color-line-aqua)',
  violet: 'var(--color-line-violet)',
}

export const TONE_BG: Record<Tone, string> = {
  orange: 'bg-orange',
  'blue-grey': 'bg-blue-grey',
  ink: 'bg-ink-2',
  silver: 'bg-silver',
  blue: 'bg-line-blue',
  aqua: 'bg-line-aqua',
  violet: 'bg-line-violet',
}
