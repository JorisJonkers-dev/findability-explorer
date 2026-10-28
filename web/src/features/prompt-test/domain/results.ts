import type {
  PromptCell,
  PromptCorrectness,
  PromptFamily,
  PromptMode,
  PromptRun,
  PromptRunStatus,
  PromptScope,
  PromptTestResults,
} from '../../../infrastructure/api/types.gen'

const PERCENT = new Intl.NumberFormat('en-GB', { style: 'percent', maximumFractionDigits: 0 })
const NUMBER = new Intl.NumberFormat('en-GB')
const DAY = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC' })

/** The providers' names for people, by their OpenRouter id; an unknown one shows its id. */
const PROVIDERS: Record<string, string> = {
  openai: 'OpenAI',
  google: 'Google',
  anthropic: 'Anthropic',
  mistralai: 'Mistral',
  'meta-llama': 'Meta',
  'x-ai': 'xAI',
  deepseek: 'DeepSeek',
}

export function providerName(id: string): string {
  return PROVIDERS[id] ?? id
}

export const SCOPES: readonly { value: PromptScope; label: string }[] = [
  { value: 'twente', label: 'Twente (Dutch)' },
  { value: 'netherlands', label: 'Netherlands (Dutch)' },
  { value: 'germany', label: 'Germany (German)' },
  { value: 'dutch', label: 'Dutch, no region' },
  { value: 'german', label: 'German, no region' },
  { value: 'english', label: 'English, no region' },
]

export const FAMILIES: readonly { value: PromptFamily; label: string }[] = [
  { value: 'eyelink', label: 'Eyelink' },
  { value: 'open_spiral', label: 'Open spiral' },
  { value: 'closed_spiral', label: 'Closed spiral' },
  { value: 'tri_flex_turn_curve', label: 'Curve-going (Tri-Flex)' },
  { value: 'other', label: 'Other metal belts' },
]

export const MODES: readonly { value: PromptMode; label: string }[] = [
  { value: 'knowledge', label: 'From its own knowledge' },
  { value: 'web', label: 'With web search' },
]

export function scopeName(scope: PromptScope): string {
  return SCOPES.find((s) => s.value === scope)?.label ?? scope
}

export function familyName(family: PromptFamily): string {
  return FAMILIES.find((f) => f.value === family)?.label ?? family
}

export function modeName(mode: PromptMode): string {
  return MODES.find((m) => m.value === mode)?.label ?? mode
}

/** Counted answers, and how many name and cite Tribelt. */
export type Tally = { answers: number; named: number; cited: number }

const EMPTY: Tally = { answers: 0, named: 0, cited: 0 }

function add(a: Tally, b: Tally): Tally {
  return { answers: a.answers + b.answers, named: a.named + b.named, cited: a.cited + b.cited }
}

/** A share in words: "17%", or "–" when there is nothing to share. */
export function shareLabel(part: number, whole: number): string {
  return whole === 0 ? '–' : PERCENT.format(part / whole)
}

/** A share from 0 to 1, or 0 when there is nothing to share. */
export function fraction(part: number, whole: number): number {
  return whole === 0 ? 0 : part / whole
}

/** The cells that match a filter, summed. */
export function tally(cells: readonly PromptCell[], filter: Partial<Pick<PromptCell, 'provider' | 'scope' | 'family' | 'mode'>> = {}): Tally {
  return cells
    .filter((c) => (Object.keys(filter) as (keyof typeof filter)[]).every((k) => filter[k] === undefined || c[k] === filter[k]))
    .reduce<Tally>((t, c) => add(t, c), EMPTY)
}

export function runDate(run: Pick<PromptRun, 'startedAt'>): string {
  return DAY.format(new Date(run.startedAt))
}

const STATUS: Record<PromptRunStatus, string> = {
  running: 'Running',
  done: 'Done',
  stopped: 'Stopped at the budget',
  failed: 'Failed',
}

export function statusName(status: PromptRunStatus): string {
  return STATUS[status]
}

/** The line under the headline: which run this is and how it ended. */
export function runLine(run: Pick<PromptRun, 'startedAt' | 'status'>): string {
  const how = run.status === 'done' ? 'Finished' : statusName(run.status)
  return `Prompt run of ${runDate(run)}. ${how}.`
}

/** How far a run got: "640 of 1,260 answers", with failures when there were any. */
export function progressLabel(run: Pick<PromptRun, 'answered' | 'failed' | 'tasks'>): string {
  const failed = run.failed > 0 ? `, ${NUMBER.format(run.failed)} without an answer` : ''
  return `${NUMBER.format(run.answered + run.failed)} of ${NUMBER.format(run.tasks)} answers${failed}`
}

/** A run's cost in dollars: "$2.75". */
export function costLabel(usd: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(usd)
}

/** The run the results open on: the newest that has answers to show, else the newest. */
export function latestRun(results: Pick<PromptTestResults, 'runs'> | undefined): PromptRun | undefined {
  return results?.runs.find((r) => r.cells.length > 0) ?? results?.runs[0]
}

/** How often the latest run's answers name a company, for its card; null before any run named it. */
export function mentionLine(results: PromptTestResults | undefined, companyId: string): string | null {
  const run = latestRun(results)
  if (!results || !run) return null
  const answers = tally(run.cells).answers
  const m = results.mentions.find((x) => x.competitorId === companyId)
  if (!m) return `Named in none of the ${NUMBER.format(answers)} answers of the Prompt run of ${runDate(run)}.`
  const first = m.first > 0 ? `, first in ${NUMBER.format(m.first)}` : ''
  return `Named in ${NUMBER.format(m.answers)} of ${NUMBER.format(answers)} answers of the Prompt run of ${runDate(run)}${first}.`
}

/** How what an answer says of Tribelt holds against the fact sheet, in words. */
export const CORRECTNESS: Record<PromptCorrectness, string> = {
  correct: 'Correct about Tribelt',
  partly: 'Partly correct about Tribelt',
  wrong: 'Wrong about Tribelt',
}
