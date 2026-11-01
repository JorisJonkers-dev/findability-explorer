import type { RouteLocationRaw } from 'vue-router'
import type { Competitor, Plan, PlanTarget, PromptAnswer, PromptCell, PromptMention, PromptMode, PromptRun } from '../../../infrastructure/api/types.gen'
import { counts, TRIBELT_ID } from '../../prompt-test/domain/grid'
import { tally } from '../../prompt-test/domain/results'
import { gridOf, modelLabel, namesTribelt } from '../../prompt-test/domain/responses'
import { companyPath, familyPath } from '../../shell/domain/sections'

/** A share as the page writes it: "25%", or "–" when there is nothing to share. */
export function share(part: number, whole: number): string {
  return whole === 0 ? '–' : `${String(Math.round((part / whole) * 100))}%`
}

/** One of the three figures at the top of AI responses. */
export type Figure = { value: string; label: string; goal: string }

/** A goal from the plan, or undefined when it is not set. */
export function goalOf(plan: Plan | undefined, target: PlanTarget): number | undefined {
  return plan?.watched.find((w) => w.target === target)?.goal
}

function goalLine(plan: Plan | undefined, target: PlanTarget): string {
  const goal = goalOf(plan, target)
  return goal === undefined ? '' : `Goal ${share(goal, 1)}`
}

/** Responses that name Tribelt, those about Twente, and web responses that cite tribelt.nl, each against its goal. */
export function figuresOf(cells: readonly PromptCell[], plan: Plan | undefined): Figure[] {
  const all = tally(cells)
  const twente = tally(cells, { scope: 'twente' })
  const web = tally(cells, { mode: 'web' })
  return [
    { value: share(all.named, all.answers), label: 'AI responses that name Tribelt', goal: goalLine(plan, 'aiShareOfVoice') },
    { value: share(twente.named, twente.answers), label: 'AI responses about the Twente region name Tribelt', goal: goalLine(plan, 'twenteShareOfVoice') },
    { value: share(web.cited, web.answers), label: 'Web responses that cite tribelt.nl', goal: goalLine(plan, 'citationShare') },
  ]
}

/** A row of By model. */
export type ModelRow = { provider: string; label: string; knowledge: string; web: string; cited: string; total: boolean }

function modelRow(provider: string, label: string, cells: readonly PromptCell[], total: boolean): ModelRow {
  const knowledge = tally(cells, { mode: 'knowledge' })
  const web = tally(cells, { mode: 'web' })
  return { provider, label, knowledge: share(knowledge.named, knowledge.answers), web: share(web.named, web.answers), cited: share(web.cited, web.answers), total }
}

/** How often each model names Tribelt in each mode and cites tribelt.nl, and the same for all models. */
export function byModel(run: Pick<PromptRun, 'models' | 'cells'>): ModelRow[] {
  return [
    ...run.models.map((m) => modelRow(m.provider, modelLabel(m.provider, m.model), run.cells.filter((c) => c.provider === m.provider), false)),
    modelRow('all', 'All models', run.cells, true),
  ]
}

/** A figure of By belt family and scope, and where its responses are. */
export type Cell = { value: string; to: RouteLocationRaw }

/** A row of By belt family and scope: one family, scope, and buyer intent. */
export type FamilyScopeRow = { key: string; place: string; family: { name: string; to: string }; scope: string; intent: { name: string; help: string }; knowledge: Cell; web: Cell; all: Cell }

/** The share of the read responses that name Tribelt, per question, from its own knowledge, with web search, and both. */
export function familyScopeRows(answers: readonly PromptAnswer[]): FamilyScopeRow[] {
  const read = answers.filter(counts)
  return gridOf({ models: [] }, answers).rows.map((row) => {
    const mine = read.filter((a) => a.promptId === row.promptId)
    const cell = (mode: PromptMode | 'all'): Cell => {
      const xs = mode === 'all' ? mine : mine.filter((a) => a.mode === mode)
      const query: Record<string, string> = { family: row.family, scope: row.scope, ...(mode === 'all' ? {} : { mode }) }
      return { value: share(xs.filter(namesTribelt).length, xs.length), to: { path: '/ai-responses/all', query } }
    }
    return { key: row.promptId, place: row.place, family: { name: row.familyName, to: familyPath(row.family) }, scope: row.scopeName, intent: row.intent, knowledge: cell('knowledge'), web: cell('web'), all: cell('all') }
  })
}

/** A company in Who the responses name. */
export type NamedRow = { id: string; name: string; share: number; value: string; tribelt: boolean; tag: '' | 'Group company' | 'Not a competitor'; to: string | undefined }

/** The companies the newest run's responses name, most first, with Tribelt always among them. */
export function whoNamed(mentions: readonly PromptMention[], competitors: readonly Pick<Competitor, 'id' | 'archived'>[]): NamedRow[] {
  const archived = new Set(competitors.filter((c) => c.archived).map((c) => c.id))
  const rows = mentions.map((m): NamedRow => {
    const tribelt = m.role === 'tribelt'
    const tag = m.role === 'group' ? 'Group company' : m.role === 'mirror' || archived.has(m.competitorId) ? 'Not a competitor' : ''
    return {
      id: m.competitorId,
      name: tribelt ? 'Tribelt' : m.name,
      share: m.share,
      value: share(m.share, 1),
      tribelt,
      tag,
      to: m.role === 'competitor' && tag === '' ? companyPath(m.competitorId) : undefined,
    }
  })
  if (!rows.some((r) => r.tribelt)) rows.push({ id: TRIBELT_ID, name: 'Tribelt', share: 0, value: '0%', tribelt: true, tag: '', to: undefined })
  return rows
    .map((r, i) => ({ r, i }))
    .sort((a, b) => b.r.share - a.r.share || a.i - b.i)
    .map(({ r }) => r)
}
