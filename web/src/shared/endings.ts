import type { Work } from '../infrastructure/api/types.gen'

/** How a piece of work ended, as the API says it: a Prompt run, a scan, a review, or a Recommendation run. */
export type Ending = NonNullable<Work['ending']>

/** Every ending, in the API's order. */
export const ENDINGS: readonly Ending[] = ['done', 'doneWithFailures', 'stoppedByHand', 'stoppedByBudget', 'failed']

/** The one place endings become words, so Activity, the runs, and the Recommendation runs say the same. */
const WORDS: Record<Ending, string> = {
  done: 'Done',
  doneWithFailures: 'Done, with failures',
  stoppedByHand: 'Stopped by hand',
  stoppedByBudget: 'Stopped by the budget',
  failed: 'Failed',
}

/** An ending in words: "Stopped by hand". */
export function endingLabel(ending: Ending): string {
  return WORDS[ending]
}

/** The tone of an ending's pill. */
export type EndingTone = 'good' | 'warn' | 'bad'

/** The tone of an ending's pill: done is good, failed is bad, the rest are a warning. */
export function endingTone(ending: Ending): EndingTone {
  switch (ending) {
    case 'done':
      return 'good'
    case 'failed':
      return 'bad'
    case 'doneWithFailures':
    case 'stoppedByHand':
    case 'stoppedByBudget':
      return 'warn'
  }
}
