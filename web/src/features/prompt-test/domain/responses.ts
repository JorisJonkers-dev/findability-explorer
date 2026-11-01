import type { LocationQuery, RouteLocationRaw } from 'vue-router'
import type {
  IntentTier,
  PromptAnswer,
  PromptFamily,
  PromptKind,
  PromptMode,
  PromptNamedCompany,
  PromptRun,
  PromptScope,
  PromptSetQuestion,
} from '../../../infrastructure/api/types.gen'
import { clock, longDate } from '../../../shared/dates'
import { BUYER_INTENT } from '../../../shared/tiers'
import { isTribeltSite, TRIBELT_ID } from './grid'
import { providerName } from './results'

const NUMBER = new Intl.NumberFormat('en-GB')

const plural = (n: number, one: string, many = `${one}s`): string => `${NUMBER.format(n)} ${n === 1 ? one : many}`

/** The belt families in their order, short in a row label and in full elsewhere. */
const FAMILIES: Record<PromptFamily, { short: string; long: string }> = {
  eyelink: { short: 'Eyelink', long: 'Eyelink belts' },
  open_spiral: { short: 'Open spiral', long: 'Open spiral belts' },
  closed_spiral: { short: 'Closed spiral', long: 'Closed spiral belts' },
  tri_flex_turn_curve: { short: 'Tri-Flex', long: 'Tri-Flex curve and spiral belts' },
  other: { short: 'Other', long: 'Other metal belts' },
}
const FAMILY_ORDER = Object.keys(FAMILIES) as PromptFamily[]

/** The scopes in their order: a language, with or without a region. */
const SCOPES: Record<PromptScope, { short: string; long: string }> = {
  twente: { short: 'Twente', long: 'Twente, in Dutch' },
  netherlands: { short: 'Netherlands', long: 'The Netherlands, in Dutch' },
  germany: { short: 'Germany', long: 'Germany, in German' },
  dutch: { short: 'Dutch', long: 'Dutch, no region' },
  german: { short: 'German', long: 'German, no region' },
  english: { short: 'English', long: 'English, no region' },
}
const SCOPE_ORDER: PromptScope[] = Object.keys(SCOPES)

/** A scope's names; a scope a later Prompt set version added goes by its id until it has words here. */
function scopeText(scope: PromptScope): { short: string; long: string } {
  return SCOPES[scope] ?? { short: scope, long: scope }
}

/** Each kind of question as the buyer intent it stands for: a supplier question names the type of belt. */
const INTENTS: Record<PromptKind, IntentTier> = { supplier: 'product', application: 'application', problem: 'problem' }
const INTENT_ORDER = Object.keys(INTENTS) as PromptKind[]

/** A question's buyer intent, and what it means for the hover text. */
export function intentOf(kind: PromptKind): { name: string; help: string } {
  const intent = BUYER_INTENT[INTENTS[kind]]
  const name = intent.label.toLowerCase()
  return { name, help: `Buyer intent: ${name}. ${intent.help}` }
}

/** The search modes, in full and as a column heading. */
export const MODES: Record<PromptMode, { long: string; short: string }> = {
  knowledge: { long: 'from its own knowledge', short: 'Own knowledge' },
  web: { long: 'with web search', short: 'Web search' },
}

/** A model as people read it: "OpenAI, gpt-oss-20b". */
export function modelLabel(provider: string, model: string): string {
  return `${providerName(provider)}, ${model.slice(model.lastIndexOf('/') + 1)}`
}

/** A response's title: its model and mode, "OpenAI, gpt-oss-20b, with web search". */
export function responseTitle(a: Pick<PromptAnswer, 'provider' | 'model' | 'mode'>): string {
  return `${modelLabel(a.provider, a.model)}, ${MODES[a.mode].long}`
}

/** Where one response is, with its run, so the page can find it. */
export function responsePath(a: Pick<PromptAnswer, 'id'>, runId: string): RouteLocationRaw {
  return { path: `/ai-responses/${a.id}`, query: { run: runId } }
}

/** Whether a company a response names is Tribelt: by its id, or by its name when the reading filed it under another. */
export function isTribeltCompany(c: PromptNamedCompany): boolean {
  return c.competitorId === TRIBELT_ID || /\btribelt\b/i.test(c.name)
}

/** Tribelt's place among the companies a response names, from 1, or undefined. */
export function tribeltPlace(a: PromptAnswer): number | undefined {
  return a.extraction?.companies.find(isTribeltCompany)?.position
}

/** What a response does with Tribelt; failed when no response came back, unread when no model read it. */
export type ResponseState = 'first' | 'named' | 'not' | 'failed' | 'unread'

export function stateOf(a: PromptAnswer): ResponseState {
  if (a.failure !== undefined) return 'failed'
  if (!a.extraction || a.extraction.failed) return 'unread'
  const place = tribeltPlace(a)
  if (place === 1) return 'first'
  return place !== undefined || a.extraction.namesTribelt ? 'named' : 'not'
}

/** Whether a response names Tribelt, wherever. */
export function namesTribelt(a: PromptAnswer): boolean {
  const state = stateOf(a)
  return state === 'first' || state === 'named'
}

/** Tribelt's place in words: "Tribelt is named, number 2". */
export function placeSentence(a: PromptAnswer): string {
  const state = stateOf(a)
  switch (state) {
    case 'first':
      return 'Tribelt is named first'
    case 'named': {
      const place = tribeltPlace(a)
      return place === undefined ? 'Tribelt is named' : `Tribelt is named, number ${String(place)}`
    }
    case 'not':
      return 'Tribelt is not named'
    case 'failed':
      return 'The response failed'
    case 'unread':
      return 'No model has read this response yet'
  }
}

/** The key of the grid's squares, in its order. */
export const KEY: readonly { state: ResponseState; label: string }[] = [
  { state: 'first', label: 'Tribelt is named first.' },
  { state: 'named', label: 'Tribelt is named, but not first.' },
  { state: 'not', label: 'Tribelt is not named.' },
  { state: 'failed', label: 'The response failed.' },
  { state: 'unread', label: 'No model has read the response yet.' },
]

/** A run in the run picker: "27 October 2026, 08:36, prompt set version 2. 200 responses". */
export function runLabel(run: Pick<PromptRun, 'startedAt' | 'promptSetVersion' | 'status' | 'answered' | 'tasks'>): string {
  const of = `${NUMBER.format(run.answered)} of ${NUMBER.format(run.tasks)} responses`
  let how: string
  switch (run.status) {
    case 'running':
      how = `Running, ${of}`
      break
    case 'stopped':
      how = `Stopped after ${of}`
      break
    case 'failed':
      how = run.answered === 0 ? 'Failed, no responses' : `Failed after ${of}`
      break
    case 'done':
      how = plural(run.answered, 'response')
      break
  }
  return `${longDate(run.startedAt)}, ${clock(run.startedAt)}, prompt set version ${String(run.promptSetVersion)}. ${how}`
}

/** Where the figures of a page about one run come from. */
export function runDataLine(run: Pick<PromptRun, 'startedAt' | 'promptSetVersion' | 'models' | 'tasks' | 'answered'>): string {
  const models = run.models.length
  const questions = models === 0 ? 0 : Math.round(run.tasks / (models * 2))
  return (
    `From the Prompt run of ${longDate(run.startedAt)}: ${plural(questions, 'question')} to ${plural(models, 'model')}, ` +
    `each from the model’s own knowledge and with web search, ${plural(run.answered, 'response')} in all. Prompt set version ${String(run.promptSetVersion)}.`
  )
}

/** What Every response shows, as its address keeps it. */
export type Filters = {
  mode: PromptMode | 'all'
  scope: PromptScope
  family: PromptFamily | 'all'
  model: string
  names: 'all' | 'yes' | 'no'
}

function pick<T extends string>(raw: unknown, allowed: readonly T[]): T | 'all' {
  return allowed.find((v) => v === raw) ?? 'all'
}

/** The filters in an address; anything it does not know is all. */
export function filtersFrom(query: LocationQuery | Record<string, unknown>): Filters {
  return {
    mode: pick(query.mode, ['knowledge', 'web'] as const),
    scope: pick(query.scope, SCOPE_ORDER),
    family: pick(query.family, FAMILY_ORDER),
    model: typeof query.model === 'string' && query.model !== '' ? query.model : 'all',
    names: pick(query.names, ['yes', 'no'] as const),
  }
}

/** The responses the filters keep. */
export function shown(answers: readonly PromptAnswer[], f: Filters): PromptAnswer[] {
  return answers.filter((a) => {
    const names = f.names === 'all' || (f.names === 'yes' ? namesTribelt(a) : stateOf(a) === 'not')
    return (
      (f.mode === 'all' || a.mode === f.mode) &&
      (f.scope === 'all' || a.scope === f.scope) &&
      (f.family === 'all' || a.family === f.family) &&
      (f.model === 'all' || a.provider === f.model) &&
      names
    )
  })
}

/** The scopes a run asked, for the scope filter. */
export function scopeOptions(answers: readonly PromptAnswer[]): { value: PromptScope; label: string }[] {
  const asked = new Set(answers.map((a) => a.scope))
  return [{ value: 'all' as const, label: 'All scopes' }, ...[...SCOPE_ORDER.filter((s) => asked.has(s)), ...[...asked].filter((s) => !SCOPE_ORDER.includes(s)).sort()].map((s) => ({ value: s, label: scopeText(s).long }))]
}

/** Every belt family, for the family filter. */
export const FAMILY_OPTIONS: readonly { value: PromptFamily | 'all'; label: string }[] = [
  { value: 'all', label: 'All belt families' },
  ...FAMILY_ORDER.map((f) => ({ value: f, label: FAMILIES[f].long })),
]

/** The key of one response in a run: its question, model, and mode. */
export function keyOf(a: Pick<PromptAnswer, 'promptId' | 'provider' | 'mode'>): string {
  return `${a.promptId}|${a.provider}|${a.mode}`
}

export type GridGroup = { provider: string; label: string; modes: PromptMode[] }
export type GridColumn = { provider: string; mode: PromptMode }
export type GridRow = {
  promptId: string
  question: string
  family: PromptFamily
  scope: PromptScope
  /** The belt family and scope: "Eyelink, Twente". */
  place: string
  intent: { name: string; help: string }
  cells: (PromptAnswer | undefined)[]
}

/** The grid of Every response: a column group per model with its modes under it, and a row per question. */
export function gridOf(run: Pick<PromptRun, 'models'>, list: readonly PromptAnswer[]): { groups: GridGroup[]; columns: GridColumn[]; rows: GridRow[] } {
  const providers = [...new Set([...run.models.map((m) => m.provider), ...list.map((a) => a.provider)])]
  const groups = providers.flatMap((provider): GridGroup[] => {
    const mine = list.filter((a) => a.provider === provider)
    const first = mine[0]
    if (!first) return []
    const model = run.models.find((m) => m.provider === provider)?.model ?? first.model
    const modes = (['knowledge', 'web'] as const).filter((mode) => mine.some((a) => a.mode === mode))
    return [{ provider, label: modelLabel(provider, model), modes }]
  })
  const columns = groups.flatMap((g) => g.modes.map((mode) => ({ provider: g.provider, mode })))
  const found = new Map(list.map((a) => [keyOf(a), a]))
  const questions = new Map<string, PromptAnswer>()
  for (const a of list) if (!questions.has(a.promptId)) questions.set(a.promptId, a)
  const order = (a: PromptAnswer) => FAMILY_ORDER.indexOf(a.family) * 100 + SCOPE_ORDER.indexOf(a.scope) * 10 + INTENT_ORDER.indexOf(a.kind)
  const rows = [...questions.values()]
    .sort((a, b) => order(a) - order(b))
    .map((q) => ({
      promptId: q.promptId,
      question: q.prompt,
      family: q.family,
      scope: q.scope,
      place: `${FAMILIES[q.family].short}, ${scopeText(q.scope).short}`,
      intent: intentOf(q.kind),
      cells: columns.map((c) => found.get(keyOf({ promptId: q.promptId, ...c }))),
    }))
  return { groups, columns, rows }
}

/** A company a response names, with Tribelt marked; a Tribelt filed under another id links nowhere. */
export type NamedCompany = { position: number; name: string; competitorId: string | undefined; tribelt: boolean }

export function companiesNamed(a: PromptAnswer): NamedCompany[] {
  return [...(a.extraction?.companies ?? [])]
    .sort((x, y) => x.position - y.position)
    .map((c) => {
      const tribelt = isTribeltCompany(c)
      return { position: c.position, name: c.name, competitorId: tribelt ? undefined : c.competitorId, tribelt }
    })
}

/** The sources a response cites, as its reading found them or else as web search gave them, with tribelt.nl marked. */
export function sourcesOf(a: PromptAnswer): { address: string; tribelt: boolean }[] {
  return (a.extraction?.addresses ?? a.citations).map((address) => ({ address, tribelt: isTribeltSite(address) }))
}

/** The claims about Tribelt the fact sheet contradicts. */
export function wrongClaimsOf(a: PromptAnswer): string[] {
  return a.extraction?.wrongClaims ?? []
}

/** The facts on the fact sheet a response naming Tribelt leaves out. */
export function leftOutOf(a: PromptAnswer): string[] {
  return a.extraction?.leftOut ?? []
}

/** Who checked a response. The API does not yet say which model read it. */
export function checkedBy(a: PromptAnswer): string {
  if (!a.extraction || a.extraction.failed) return 'No model has checked this response yet.'
  return 'A model read this response to find the companies it names and to check it against the fact sheet.'
}

/** One question a run asked, in its own language, with its English translation when one is known. */
export type Question = { id: string; text: string; language: string; where: string; english: string | undefined }

/**
 * The questions a run asked, per belt family: the published set's, with their translations, or
 * else the ones its responses answer, once each.
 */
export function questionsOf(set: readonly PromptSetQuestion[] | undefined, answers: readonly PromptAnswer[]): { family: PromptFamily; label: string; questions: Question[] }[] {
  const seen = new Map<string, { family: PromptFamily; question: Question }>()
  if (set) {
    for (const q of set) seen.set(q.id, { family: q.family, question: { id: q.id, text: q.text, language: q.language, where: q.scopeName, english: q.english } })
  } else {
    for (const a of answers) {
      if (!seen.has(a.promptId)) {
        seen.set(a.promptId, { family: a.family, question: { id: a.promptId, text: a.prompt, language: a.language, where: scopeText(a.scope).long, english: undefined } })
      }
    }
  }
  return FAMILY_ORDER.flatMap((family) => {
    const mine = [...seen.values()].filter((q) => q.family === family).map((q) => q.question)
    return mine.length === 0 ? [] : [{ family, label: FAMILIES[family].long, questions: mine }]
  })
}

function csvCell(value: string | number): string {
  const text = String(value)
  return /[",\n\r]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text
}

/** The responses as CSV, one row per response, with a header row in the page's words. */
export function responsesCsv(answers: readonly PromptAnswer[]): string {
  const header = [
    'question id',
    'belt family',
    'scope',
    'buyer intent',
    'language',
    'question',
    'model',
    'search mode',
    "Tribelt's place",
    'companies named',
    'cites tribelt.nl',
    'sources',
    'input tokens',
    'output tokens',
    'cost in dollars',
    'failure',
    'response',
  ]
  const rows = answers.map((a) => [
    a.promptId,
    FAMILIES[a.family].long,
    scopeText(a.scope).long,
    intentOf(a.kind).name,
    a.language,
    a.prompt,
    a.model,
    MODES[a.mode].long,
    placeSentence(a),
    (a.extraction?.companies ?? []).map((c) => c.name).join('; '),
    a.extraction?.citesTribelt ? 'yes' : 'no',
    sourcesOf(a)
      .map((s) => s.address)
      .join('; '),
    a.inputTokens,
    a.outputTokens,
    a.costUsd,
    a.failure ?? '',
    a.text,
  ])
  return [header, ...rows].map((r) => r.map(csvCell).join(',')).join('\r\n')
}

/** The CSV's file name, after the run's day: "responses-2026-10-27.csv". */
export function responsesFile(run: Pick<PromptRun, 'startedAt'>): string {
  return `responses-${run.startedAt.slice(0, 10)}.csv`
}
