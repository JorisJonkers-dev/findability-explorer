import type { AiSearchSummary, Plan, PromptCell, PromptTestResults } from '../../../infrastructure/api/types.gen'
import { formatValue, inWords } from '../../catching-up/domain/standings'
import { latestRun, providerName, tally } from '../../prompt-test/domain/results'
import { companyPath } from '../../shell/domain/sections'
import { oxford } from '../../../shared/words'
import { pct } from './summary'

/** One company in the list of who the answers name. */
export type VoiceRow = {
  id: string
  name: string
  role: AiSearchSummary['followed'][number]['role']
  share: number
  rank: number
  /** Where its name leads, or undefined when it is not a competitor with a page. */
  to: string | undefined
  /** A line under the name, or empty. */
  note: string
  tribelt: boolean
}

/** What is said under a company that is not a competitor. */
const ROLE_NOTES = {
  group: 'A group company of Tribelt, not a competitor',
  mirror: 'The mirror of tribelt.nl, not a competitor',
} as const

/** The followed companies with their share in the newest run, the most named first; an equal share keeps the followed order. */
export function voiceRows(summary: Pick<AiSearchSummary, 'followed' | 'trend' | 'goal'>): VoiceRow[] {
  const last = summary.trend.at(-1)
  if (!last) return []
  return summary.followed
    .map((c, i) => ({ c, i, share: last.shares[i] ?? 0 }))
    .sort((a, b) => b.share - a.share || a.i - b.i)
    .map(({ c, share }, index): VoiceRow => ({
      id: c.competitorId,
      name: c.name,
      role: c.role,
      share,
      rank: index + 1,
      to: c.role === 'competitor' ? companyPath(c.competitorId) : undefined,
      note: c.role === 'tribelt' ? `Target ${pct(summary.goal)}` : c.role === 'competitor' ? '' : ROLE_NOTES[c.role],
      tribelt: c.role === 'tribelt',
    }))
}

const ORDINALS = ['first', 'second', 'third', 'fourth', 'fifth', 'sixth', 'seventh', 'eighth', 'ninth', 'tenth']

/** The page's headline: where Tribelt ranks among the companies the answers name. */
export function voiceHeadline(rows: readonly VoiceRow[], listed: number): string {
  const tribelt = rows.find((r) => r.tribelt)
  if (!tribelt) return 'AI assistants have not been asked about Tribelt yet.'
  if (tribelt.rank === 1) return 'Tribelt is the company AI answers name most.'
  const place = ORDINALS[tribelt.rank - 1]
  // The list holds only the companies named most, so a last place on a full list means "below them all".
  if (place === undefined || (rows.length >= listed && tribelt.rank === rows.length)) {
    return `Tribelt is not among the ${inWords(rows.length - 1)} companies AI answers name most.`
  }
  return `Tribelt ranks ${place} among the companies AI answers name.`
}

/** Per provider, how many of its read answers name Tribelt. */
export function providerTallies(cells: readonly PromptCell[]): { provider: string; answers: number; named: number }[] {
  const providers = [...new Set(cells.map((c) => c.provider))]
  return providers.map((provider) => ({ provider, ...tally(cells, { provider }) }))
}

/** The sentence under the headline: what the newest run asked, and which provider names Tribelt most. */
export function voiceLede(cells: readonly PromptCell[]): string {
  const all = tally(cells)
  if (all.answers === 0) return 'No Prompt run has finished yet. The first one starts on the schedule.'
  const byProvider = providerTallies(cells)
  const providers = byProvider.length === 1 ? 'one provider' : `${inWords(byProvider.length)} providers`
  const asked = `In the newest Prompt run, AI assistants from ${providers} gave ${String(all.answers)} answers to buyers' questions.`
  const best = [...byProvider].sort((a, b) => b.named - a.named)[0]
  if (!best || best.named === 0) return `${asked} No provider named Tribelt in any answer.`
  const tied = byProvider.filter((p) => p.named === best.named)
  if (tied.length > 1) return `${asked} ${oxford(tied.map((p) => providerName(p.provider)))} name Tribelt most often, in ${String(best.named)} answers each.`
  return `${asked} ${providerName(best.provider)} names Tribelt most often, in ${String(best.named)} of its ${String(best.answers)} answers.`
}

/** A key figure on the page. */
export type Figure = { to: string; big: string; text: string; note: string }

/** The goal of a number to watch, if the plan has it. */
function goalOf(plan: Plan | undefined, target: Plan['watched'][number]['target']): number | undefined {
  return plan?.watched.find((w) => w.target === target)?.goal
}

/** Tribelt's AI readiness score: the ranking's, or the checks it passes when the ranking leaves it out. */
export function readinessScore(summary: Pick<AiSearchSummary, 'ranking' | 'checks'>): number | undefined {
  const ranked = summary.ranking.find((r) => r.role === 'tribelt')
  if (ranked) return ranked.score
  const scored = summary.checks.filter((c) => c.scored)
  if (scored.length === 0 || scored.every((c) => c.tribelt === undefined)) return undefined
  return scored.filter((c) => c.tribelt === true).length
}

/** The three key figures: answers naming Tribelt, web answers citing tribelt.nl, and the readiness checks passed. */
export function figures(summary: AiSearchSummary, cells: readonly PromptCell[], plan: Plan | undefined): Figure[] {
  const all = tally(cells)
  const web = tally(cells, { mode: 'web' })
  const named = goalOf(plan, 'aiShareOfVoice') ?? summary.goal
  const cited = goalOf(plan, 'citations')
  const score = readinessScore(summary)
  return [
    {
      to: '/numbers/aiShareOfVoice',
      big: all.answers === 0 ? '–' : pct(all.named / all.answers),
      text: 'of AI answers name Tribelt',
      note: `Target ${pct(named)}.`,
    },
    {
      to: '/numbers/citationShare',
      big: web.answers === 0 ? '–' : pct(web.cited / web.answers),
      text: 'of web answers cite tribelt.nl',
      note: cited === undefined ? '' : `Target ${String(cited)} web answers in a run.`,
    },
    {
      to: '/ai-search/readiness',
      big: score === undefined ? '–' : `${formatValue('readinessChecks', score)} of ${String(summary.readinessChecks)}`,
      text: 'AI readiness checks passed',
      note: 'See how AI readiness is checked.',
    },
  ]
}

/** Which answers the donut shows. */
export type Scope = 'all' | 'twente'

/** What the donut shows for one scope. */
export type Slice = { label: string; named: number; total: number; namedText: string; restText: string; goal: string; centre: string; sub: string; aria: string }

const answers = (n: number): string => (n === 1 ? '1 answer' : `${String(n)} answers`)

/** The donut for a scope of the newest run's answers. */
export function slice(scope: Scope, cells: readonly PromptCell[], summary: Pick<AiSearchSummary, 'goal'>, plan: Plan | undefined): Slice {
  const t = scope === 'twente' ? tally(cells, { scope: 'twente' }) : tally(cells)
  const label = scope === 'twente' ? 'Answers about Twente' : 'All answers'
  const goal = scope === 'twente' ? goalOf(plan, 'twenteShareOfVoice') : (goalOf(plan, 'aiShareOfVoice') ?? summary.goal)
  const rest = t.answers - t.named
  const centre = t.answers === 0 ? '–' : pct(t.named / t.answers)
  return {
    label,
    named: t.named,
    total: t.answers,
    namedText: `${answers(t.named)} ${t.named === 1 ? 'names' : 'name'} Tribelt`,
    restText: `${answers(rest)} ${rest === 1 ? 'does' : 'do'} not`,
    goal: goal === undefined ? '' : `The target for the year is ${pct(goal)} of ${scope === 'twente' ? 'the answers about Twente' : 'all answers'}.`,
    centre,
    sub: `${String(t.named)} of ${String(t.answers)}`,
    aria: `${label}: ${String(t.named)} of ${String(t.answers)} answers, ${centre}, name Tribelt.`,
  }
}

/** The sentence on the card that leads to AI readiness by site. */
export function readinessCard(summary: Pick<AiSearchSummary, 'checks' | 'readinessChecks'>): string {
  const scored = summary.checks.filter((c) => c.scored && c.tribelt !== undefined)
  if (scored.length === 0) return 'See how each site is set up for AI.'
  const failing = scored.filter((c) => !c.tribelt).length
  const of = inWords(summary.readinessChecks)
  if (failing === 0) return `Tribelt passes all ${of} checks. See how the other sites compare.`
  return `Tribelt fails ${inWords(failing)} of the ${of} checks. See which ones.`
}

/** The newest run's cells, or none before a run. */
export function newestCells(results: Pick<PromptTestResults, 'runs'> | undefined): readonly PromptCell[] {
  return latestRun(results)?.cells ?? []
}
