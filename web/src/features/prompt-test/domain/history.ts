import type { PromptRunFigures, PromptSetVersionInfo } from '../../../infrastructure/api/types.gen'
import { clock, longDate } from '../../../shared/dates'
import { joinWords } from '../../../shared/words'
import type { Tone } from '../../../shared/ui/charts/tones'

/** One run's figure on a line: when the run started, its share, and whether it was partial. */
export type Point = { x: string; y: number; partial: boolean }

/** A line over time. */
export type Series = { label: string; points: Point[] }

/** A line a chart draws: one measure of the runs, in its tone. */
export type ChartSeries = Series & { tone: Tone; strong?: boolean }

/** A share the history gives per run, with the line it is drawn as. */
export type Measure = { label: string; tone: Tone; strong?: boolean; pick: (r: PromptRunFigures) => number | undefined }

/** A place where the questions or the model list changed: the chart shades from here, and a note says why. */
export type Break = { x: string; label: string; note: string }

/** The Prompt set filter's choice: one version, or every version. */
export type VersionChoice = number | 'all'

/** The lines of AI responses' Over time: all responses, and each language the questions were asked in. */
export const LANGUAGE_MEASURES: readonly Measure[] = [
  { label: 'All', tone: 'orange', strong: true, pick: (r) => r.all },
  { label: 'Dutch', tone: 'blue', pick: (r) => r.dutch },
  { label: 'German', tone: 'aqua', pick: (r) => r.german },
  { label: 'English', tone: 'violet', pick: (r) => r.english },
]

const at = (iso: string): number => Date.parse(iso)
const int = (n: number): string => n.toLocaleString('en-GB')
const questions = (n: number): string => `${int(n)} ${n === 1 ? 'question' : 'questions'}`

/** One point per run that has a value for the measure, in the runs' order. */
export function seriesOf(runs: readonly PromptRunFigures[], pick: (r: PromptRunFigures) => number | undefined): Point[] {
  return runs.flatMap((r) => {
    const y = pick(r)
    return y === undefined ? [] : [{ x: r.startedAt, y, partial: r.partial }]
  })
}

/** A line per measure, ready for the chart. */
export function chartSeries(runs: readonly PromptRunFigures[], measures: readonly Measure[]): ChartSeries[] {
  return measures.map((m) => ({ label: m.label, tone: m.tone, ...(m.strong ? { strong: true } : {}), points: seriesOf(runs, m.pick) }))
}

/** The runs of one Prompt set version, or every run. */
export function forVersion(runs: readonly PromptRunFigures[], v: VersionChoice): PromptRunFigures[] {
  return v === 'all' ? [...runs] : runs.filter((r) => r.promptSetVersion === v)
}

/**
 * A break where each later version was published, but only where runs sit on both sides of it; and
 * one at each run that asked other models than the run before it, in the order of the chart.
 */
export function breaksOf(versions: readonly PromptSetVersionInfo[], runs: readonly PromptRunFigures[]): Break[] {
  const questionsChanged = versions
    .slice(1)
    .filter((s) => runs.some((r) => at(r.startedAt) < at(s.publishedAt)) && runs.some((r) => at(r.startedAt) >= at(s.publishedAt)))
    .map((s) => ({
      x: s.publishedAt,
      label: `Prompt set version ${String(s.version)}`,
      note: `The questions changed here (prompt set version ${String(s.version)}), so the figures before and after are not the same measure.`,
    }))
  return [...questionsChanged, ...modelBreaks(runs)].sort((a, b) => at(a.x) - at(b.x))
}

/** A break at each run whose models differ from the run before it; a run that does not say has none. */
function modelBreaks(runs: readonly PromptRunFigures[]): Break[] {
  const out: Break[] = []
  let before: readonly string[] | undefined
  for (const r of runs) {
    const now = r.models
    if (before && now) {
      const was = before
      const added = now.filter((m) => !was.includes(m))
      const removed = was.filter((m) => !now.includes(m))
      const what = [added.length ? `added ${joinWords(added)}` : '', removed.length ? `removed ${joinWords(removed)}` : ''].filter(Boolean).join(', and ')
      if (what) out.push({ x: r.startedAt, label: 'Model list', note: `The model list changed here (${what}), so the figures before and after are not the same measure.` })
    }
    before = now
  }
  return out
}

// A refused key is said plainly; any other reason is quoted as the run gave it.
const why = (failure: string): string => (/\b401\b/.test(failure) ? 'the OpenRouter key was refused' : failure.replace(/\.$/, ''))

/** Why a partial run's figure rests on few responses. */
export function partialNote(r: PromptRunFigures): string {
  const reason = r.failure ? ` (the rest failed because ${why(r.failure)})` : ''
  return `${longDate(r.startedAt)}, ${clock(r.startedAt)}: a partial run of version ${String(r.promptSetVersion)}. Only ${int(r.answered)} of its ${int(r.tasks)} requests got a response${reason}, and ${int(r.read)} of those could be read, so its figure rests on few responses.`
}

/** What the chosen version asks, or how the versions differ. */
export function versionNote(versions: readonly PromptSetVersionInfo[], v: VersionChoice, runs: readonly PromptRunFigures[]): string {
  if (v === 'all') {
    const last = versions.at(-1)
    const parts = versions.map((s, i) => `${i ? 'version' : 'Version'} ${String(s.version)} ${s === last ? 'asks' : 'asked'} ${questions(s.questions)}`)
    if (parts.length <= 1) return parts.length ? `${parts.join('')}.` : ''
    return `The questions changed with each version, so the line breaks between versions. ${parts.slice(0, -1).join(', ')}, and ${parts.slice(-1).join('')}.`
  }
  const set = versions.find((s) => s.version === v)
  if (!set) return ''
  const lone = runs.length === 1 ? ' It has one run so far, so there is no line yet.' : ''
  return `Version ${String(set.version)}, published ${longDate(set.publishedAt)}: ${questions(set.questions)}. ${set.note}${lone}`
}

/** The filter's buttons: every version, then each version, the newest first and called current. */
export function versionOptions(versions: readonly PromptSetVersionInfo[]): { value: VersionChoice; label: string }[] {
  const newest = versions.at(-1)
  const each = [...versions].reverse().map((s) => ({ value: s.version, label: `Version ${String(s.version)}${s === newest ? ', current' : ''}` }))
  return [{ value: 'all', label: 'All versions' }, ...each]
}

/** The choice kept in the address (`?version=1`); anything else is every version. */
export function parseVersion(raw: unknown, versions: readonly PromptSetVersionInfo[]): VersionChoice {
  const value: unknown = Array.isArray(raw) ? raw[0] : raw
  if (typeof value !== 'string' || !/^\d+$/.test(value)) return 'all'
  const n = Number(value)
  return versions.some((s) => s.version === n) ? n : 'all'
}

const shown = (r: PromptRunFigures, measures: readonly Measure[]): boolean => measures.some((m) => m.pick(r) !== undefined)

/** The notes under a chart: each break it shows, then each partial run that has a point on it. */
export function chartNotes(breaks: readonly Break[], runs: readonly PromptRunFigures[], measures: readonly Measure[]): string[] {
  const partial = runs.filter((r) => r.partial && shown(r, measures)).map(partialNote)
  return [...breaks.map((b) => `${longDate(b.x)}: ${b.note}`), ...partial]
}

/** The chart's figures as a table: a row per run with a point, oldest first, a value per measure. */
export function tableRows(runs: readonly PromptRunFigures[], measures: readonly Measure[]): { x: string; when: string; partial: boolean; values: (number | undefined)[] }[] {
  return runs
    .filter((r) => shown(r, measures))
    .map((r) => ({ x: r.startedAt, when: `${longDate(r.startedAt)}, ${clock(r.startedAt)}`, partial: r.partial, values: measures.map((m) => m.pick(r)) }))
}
