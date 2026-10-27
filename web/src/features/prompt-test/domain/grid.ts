import type { PromptAnswer, PromptFamily, PromptMode, PromptRun, PromptScope } from '../../../infrastructure/api/types.gen'
import { FAMILIES, providerName, SCOPES } from './results'

/** Tribelt's id among the companies. */
export const TRIBELT_ID = 'tribelt'

/** Where an answer puts Tribelt: first, second or third, later, not at all, or there is no answer to read. */
export type Place = 'first' | 'top3' | 'later' | 'none' | 'missing'

/** How each place looks in the grid and its legend: a fill, and a mark that tells it apart without colour. */
export const PLACES: Record<Place, { label: string; cell: string; mark: (position: number | undefined) => string }> = {
  first: { label: 'Tribelt named first', cell: 'border-orange bg-orange text-black', mark: () => '1' },
  top3: { label: '2nd or 3rd', cell: 'border-orange-soft bg-orange-soft text-black', mark: (p) => String(p ?? '') },
  later: { label: '4th or later', cell: 'border-orange-soft bg-orange-pale text-ink-2', mark: (p) => (p === undefined ? '+' : String(p)) },
  none: { label: 'Not named', cell: 'border-grey bg-grey text-muted', mark: () => '–' },
  missing: { label: 'No answer', cell: 'border-dashed border-silver bg-white text-orange-ink', mark: () => '✕' },
}

/** The legend's order. */
export const PLACE_ORDER: readonly Place[] = ['first', 'top3', 'later', 'none', 'missing']

/** Whether an answer counts: it was answered and read. */
export function counts(a: PromptAnswer | undefined): a is PromptAnswer & { extraction: NonNullable<PromptAnswer['extraction']> } {
  return a !== undefined && a.failure === undefined && a.extraction !== undefined && !a.extraction.failed
}

/** Tribelt's position in an answer, from 1, or undefined when it names Tribelt but not where. */
export function tribeltPosition(a: PromptAnswer): number | undefined {
  return a.extraction?.companies.find((c) => c.competitorId === TRIBELT_ID)?.position
}

/** Where an answer puts Tribelt. */
export function placeOf(a: PromptAnswer | undefined): Place {
  if (!counts(a)) return 'missing'
  const position = tribeltPosition(a)
  if (position === undefined) return a.extraction.namesTribelt ? 'later' : 'none'
  if (position === 1) return 'first'
  return position <= 3 ? 'top3' : 'later'
}

/** A grid column: one provider in one mode. */
export type Column = { provider: string; name: string; model: string; mode: PromptMode }

/** The columns of a run: each provider it asked, in its order, each from its own knowledge then with web search. */
export function columnsOf(run: Pick<PromptRun, 'models'>, mode: PromptMode | 'all' = 'all'): Column[] {
  const modes: PromptMode[] = mode === 'all' ? ['knowledge', 'web'] : [mode]
  return run.models.flatMap((m) => modes.map((md) => ({ provider: m.provider, name: providerName(m.provider), model: m.model, mode: md })))
}

/** A grid row: one question. */
export type Row = { promptId: string; family: PromptFamily; scope: PromptScope; question: string }

/** The questions a run's answers answer, by belt family then scope; only those in scope when one is picked. */
export function rowsOf(answers: readonly PromptAnswer[], scope: PromptScope | 'all' = 'all'): Row[] {
  const rows = new Map<string, Row>()
  for (const a of answers) {
    if (!rows.has(a.promptId) && (scope === 'all' || a.scope === scope)) {
      rows.set(a.promptId, { promptId: a.promptId, family: a.family, scope: a.scope, question: a.prompt })
    }
  }
  const order = (family: PromptFamily, s: PromptScope) => FAMILIES.findIndex((f) => f.value === family) * 10 + SCOPES.findIndex((x) => x.value === s)
  return [...rows.values()].sort((a, b) => order(a.family, a.scope) - order(b.family, b.scope))
}

/** The key of one answer in a run: its question, provider and mode. */
export function keyOf(a: Pick<PromptAnswer, 'promptId' | 'provider' | 'mode'>): string {
  return `${a.promptId}|${a.provider}|${a.mode}`
}

/** A run's answers by their key. */
export function byKey(answers: readonly PromptAnswer[]): Map<string, PromptAnswer> {
  return new Map(answers.map((a) => [keyOf(a), a]))
}

/** The scopes the grid can be narrowed to, as the current prompt set asks them. */
export const SCOPE_FILTERS: readonly { value: PromptScope | 'all'; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'twente', label: 'Twente' },
  { value: 'dutch', label: 'Dutch' },
  { value: 'german', label: 'German' },
  { value: 'english', label: 'English' },
]

export const MODE_FILTERS: readonly { value: PromptMode | 'all'; label: string }[] = [
  { value: 'all', label: 'Both' },
  { value: 'knowledge', label: 'Own knowledge' },
  { value: 'web', label: 'Web search' },
]

/** A filter value from the address, or 'all'. */
export function parseFilter<T extends string>(raw: unknown, options: readonly { value: T | 'all' }[]): T | 'all' {
  return options.find((o) => o.value === raw)?.value ?? 'all'
}

/** The run's headline numbers: the share of answers naming Tribelt, that share for Twente, and the web answers citing tribelt.nl. */
export function headlineNumbers(answers: readonly PromptAnswer[]): {
  named: number
  counted: number
  twente: number
  twenteCounted: number
  cited: number
  web: number
} {
  const counted = answers.filter(counts)
  const twente = counted.filter((a) => a.scope === 'twente')
  const web = counted.filter((a) => a.mode === 'web')
  return {
    named: counted.filter((a) => a.extraction.namesTribelt).length,
    counted: counted.length,
    twente: twente.filter((a) => a.extraction.namesTribelt).length,
    twenteCounted: twente.length,
    cited: web.filter((a) => a.extraction.citesTribelt).length,
    web: web.length,
  }
}

/** One company and the share of a run's answers that name it. */
export type Named = { id: string; share: number; tribelt: boolean }

/** Every company the answers name, as a share of the answers that count, most first; Tribelt is always in it. */
export function ranked(answers: readonly PromptAnswer[]): Named[] {
  const counted = answers.filter(counts)
  const tally = new Map<string, number>([[TRIBELT_ID, 0]])
  for (const a of counted) {
    for (const id of new Set(a.extraction.companies.flatMap((c) => (c.competitorId ? [c.competitorId] : [])))) tally.set(id, (tally.get(id) ?? 0) + 1)
  }
  return [...tally.entries()]
    .map(([id, n]) => ({ id, share: counted.length > 0 ? n / counted.length : 0, tribelt: id === TRIBELT_ID }))
    .sort((a, b) => b.share - a.share || a.id.localeCompare(b.id))
}

/** The companies the answers name most, with Tribelt always shown: it takes the last row when it ranks lower. */
export function mostNamed(answers: readonly PromptAnswer[], shown = 8): Named[] {
  const all = ranked(answers)
  const top = all.slice(0, shown)
  if (!top.some((n) => n.tribelt)) top[shown - 1] = all.find((n) => n.tribelt) as Named
  return top
}

/** The page's headline, from who the answers name most overall and in Twente. */
export function gridHeadline(answers: readonly PromptAnswer[], nameOf: (id: string) => string): string {
  const counted = answers.filter(counts)
  if (counted.length === 0) return 'Do AI assistants name Tribelt?'
  const leader = ranked(answers)[0] as Named
  const twente = ranked(counted.filter((a) => a.scope === 'twente'))[0] as Named
  if (leader.tribelt) return 'AI answers name Tribelt more than any competitor.'
  if (twente.tribelt && twente.share > 0) return `Twente buyers hear of Tribelt. Everyone else hears of ${nameOf(leader.id)}.`
  return `AI answers name ${nameOf(leader.id)} most.`
}

/** The start of an answer, cut at a word. */
export function excerpt(text: string, max = 280): string {
  const clean = text.replace(/\s+/g, ' ').trim()
  if (clean.length <= max) return clean
  const cut = clean.slice(0, max)
  return `${cut.slice(0, cut.lastIndexOf(' ') > 0 ? cut.lastIndexOf(' ') : max)}…`
}

/** An address's host without "www.": "tribelt.nl". */
export function hostOf(address: string): string {
  try {
    return new URL(address.includes('://') ? address : `https://${address}`).hostname.replace(/^www\./, '')
  } catch {
    return address
  }
}

/** Whether an address is on tribelt.nl. */
export function isTribeltSite(address: string): boolean {
  const host = hostOf(address)
  return host === 'tribelt.nl' || host.endsWith('.tribelt.nl')
}

const PLACE_CSV: Record<Place, string> = { first: 'first', top3: 'second or third', later: 'later', none: 'not named', missing: 'no answer' }

function csvCell(value: string | number): string {
  const text = String(value)
  return /[",\n\r]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text
}

/** A run's answers as CSV, one row per answer, with a header row. */
export function toCsv(answers: readonly PromptAnswer[]): string {
  const header = [
    'question id',
    'family',
    'scope',
    'language',
    'question',
    'provider',
    'model',
    'mode',
    "Tribelt's place",
    'companies named',
    'cites tribelt.nl',
    'sources',
    'input tokens',
    'output tokens',
    'cost in dollars',
    'failure',
    'answer',
  ]
  const rows = answers.map((a) => [
    a.promptId,
    a.family,
    a.scope,
    a.language,
    a.prompt,
    a.provider,
    a.model,
    a.mode,
    PLACE_CSV[placeOf(a)],
    (a.extraction?.companies ?? []).map((c) => c.name).join('; '),
    a.extraction?.citesTribelt ? 'yes' : 'no',
    (a.extraction?.addresses ?? a.citations).join('; '),
    a.inputTokens,
    a.outputTokens,
    a.costUsd,
    a.failure ?? '',
    a.text,
  ])
  return [header, ...rows].map((r) => r.map(csvCell).join(',')).join('\r\n')
}

/** A small cost in dollars, to the tenth of a cent or finer: "$0.0014", "$0.42". */
export function answerCost(usd: number): string {
  if (usd >= 0.1) return `$${usd.toFixed(2)}`
  return `$${usd.toPrecision(2).replace(/0+$/, '').replace(/\.$/, '')}`
}

/** The average cost of a given answer of the provider, or of any provider when it has none; 0 without answers. */
export function averageCost(answers: readonly PromptAnswer[], provider: string): number {
  const given = answers.filter((a) => a.failure === undefined)
  const mine = given.filter((a) => a.provider === provider)
  const list = mine.length > 0 ? mine : given
  return list.length === 0 ? 0 : list.reduce((sum, a) => sum + a.costUsd, 0) / list.length
}

/** How many of a run's answers are failed or missing, and what asking them again should cost. */
export function retryPlan(run: Pick<PromptRun, 'models' | 'tasks'>, answers: readonly PromptAnswer[]): { count: number; estimate: number } {
  const perProvider = run.models.length === 0 ? 0 : run.tasks / run.models.length
  return run.models.reduce(
    (plan, m) => {
      const missing = Math.max(0, perProvider - answers.filter((a) => a.provider === m.provider && a.failure === undefined).length)
      return { count: plan.count + missing, estimate: plan.estimate + missing * averageCost(answers, m.provider) }
    },
    { count: 0, estimate: 0 },
  )
}
