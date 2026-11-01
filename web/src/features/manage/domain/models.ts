import type { PromptCatalogueModel, PromptTestSettings } from '../../../infrastructure/api/types.gen'
import { longDate } from '../../../shared/dates'
import { costLabel, providerName } from '../../prompt-test/domain/results'

/** The most models the Model list holds, as the server allows. */
export const MAX_MODELS = 20

/**
 * The list as saved: the Model list, or before one is saved what runs ask now (the cheapest model
 * of each provider, as the prices give them), or else what the latest run asked.
 */
export function savedList(s: PromptTestSettings): string[] {
  if (s.modelList.length > 0) return [...s.modelList]
  if (s.prices.providers.length > 0) return s.prices.providers.map((p) => p.model)
  return s.models.map((m) => m.model)
}

/** The list with one more model, unless it is on it already or the list is full. */
export function addModel(list: readonly string[], id: string): string[] {
  return list.includes(id) || list.length >= MAX_MODELS ? [...list] : [...list, id]
}

/** The list without one model. */
export function removeModel(list: readonly string[], id: string): string[] {
  return list.filter((m) => m !== id)
}

/** Whether two lists hold the same models, in any order. */
export function sameList(a: readonly string[], b: readonly string[]): boolean {
  return a.length === b.length && a.every((m) => b.includes(m))
}

/** A model's name without the provider OpenRouter puts before it: "gpt-oss-20b" of "OpenAI: gpt-oss-20b". */
export function modelName(e: PromptCatalogueModel | undefined, id = ''): string {
  if (!e) return id
  const cut = e.name.indexOf(': ')
  return cut >= 0 ? e.name.slice(cut + 2) : e.name
}

const providerOf = (id: string): string => id.split('/')[0] ?? id

/** What one run of the model costs: every question from its own knowledge and with web search. */
const perRun = (e: PromptCatalogueModel, questions: number): number => questions * (e.knowledgeUsd + e.webUsd)

/** One row of the model list. known is false for a model OpenRouter no longer lists. */
export type ListRow = { id: string; provider: string; name: string; runUsd: number; free: boolean; known: boolean }

export function listRows(list: readonly string[], catalogue: readonly PromptCatalogueModel[], questions: number): ListRow[] {
  return list.map((id) => {
    const e = catalogue.find((c) => c.id === id)
    return { id, provider: providerName(e?.provider ?? providerOf(id)), name: modelName(e, id), runUsd: e ? perRun(e, questions) : 0, free: e?.free ?? false, known: e !== undefined }
  })
}

/** What a run of the list costs: each model's responses, and reading every one of them. */
export function listCost(list: readonly string[], catalogue: readonly PromptCatalogueModel[], questions: number, readingUsd: number): number {
  return list.reduce((sum, id) => {
    const e = catalogue.find((c) => c.id === id)
    return e ? sum + perRun(e, questions) + questions * 2 * readingUsd : sum
  }, 0)
}

const models = (n: number): string => `${String(n)} ${n === 1 ? 'model' : 'models'}`

/** What Save the model list says: the list as saved, or what the changes cost against it. */
export function saveLine(saved: readonly string[], draft: readonly string[], catalogue: readonly PromptCatalogueModel[], questions: number, readingUsd: number): string {
  if (draft.length === 0) return 'The list is empty. Add a model below, or a run has nobody to ask.'
  const now = listCost(draft, catalogue, questions, readingUsd)
  if (sameList(saved, draft)) return `The list is saved. A run asks ${models(draft.length)} and costs about ${costLabel(now)}.`
  const before = listCost(saved, catalogue, questions, readingUsd)
  return `With these changes, a run asks ${models(draft.length)} and costs about ${costLabel(now)}, ${costLabel(Math.abs(now - before))} ${now >= before ? 'more' : 'less'} than the saved list. Saving marks this date on the trend charts, so results from before and after are not compared.`
}

/** The provider filter: every provider, then each by name with how many models it has. */
export function providerOptions(catalogue: readonly PromptCatalogueModel[]): { value: string; label: string }[] {
  const counts = new Map<string, number>()
  for (const e of catalogue) counts.set(e.provider, (counts.get(e.provider) ?? 0) + 1)
  const each = [...counts.entries()]
    .sort(([a], [b]) => providerName(a).localeCompare(providerName(b)))
    .map(([p, n]) => ({ value: p, label: `${providerName(p)} (${String(n)})` }))
  return [{ value: 'all', label: `All ${String(counts.size)} providers` }, ...each]
}

/** The models of one provider, or all, and only the free ones when asked. */
export function filterCatalogue(catalogue: readonly PromptCatalogueModel[], provider: string, freeOnly: boolean): PromptCatalogueModel[] {
  return catalogue.filter((e) => (provider === 'all' || e.provider === provider) && (!freeOnly || e.free))
}

/** One row of Add a model. */
export type CatalogueRow = {
  id: string
  provider: string
  name: string
  recommended: boolean
  listed: string
  listedAt: string
  runUsd: number
  free: boolean
  onList: boolean
  search: string
}

export function catalogueRows(catalogue: readonly PromptCatalogueModel[], list: readonly string[], questions: number): CatalogueRow[] {
  return catalogue.map((e) => ({
    id: e.id,
    provider: providerName(e.provider),
    name: modelName(e),
    recommended: e.recommended,
    listed: e.listedAt ? longDate(e.listedAt) : '',
    listedAt: e.listedAt ?? '',
    runUsd: perRun(e, questions),
    free: e.free,
    onList: list.includes(e.id),
    search: `${e.id} ${e.name} ${providerName(e.provider)}`,
  }))
}

/** What Add a model lists, and when OpenRouter's catalogue was read. */
export function catalogueIntro(catalogue: readonly PromptCatalogueModel[], readAt: string, questions: number): string {
  const providers = new Set(catalogue.map((e) => e.provider)).size
  return (
    `Every text model on OpenRouter, ${String(catalogue.length)} from ${String(providers)} providers, as listed on ${longDate(readAt)}. ` +
    'The list is read again daily, so new models appear by themselves. ' +
    'First come the recommended models, the cheapest of the five providers whose assistants buyers use most. The rest follow, cheapest first. ' +
    `Costs are for one run of ${String(questions)} questions. Each question is asked twice, once from the model’s own knowledge and once with web search. ` +
    'Models already asked show what their runs cost.'
  )
}
