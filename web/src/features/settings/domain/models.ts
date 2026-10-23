import type { LlmModel } from '../../../infrastructure/api/types.gen'

/** The most models the chooser lists at once: OpenRouter alone offers hundreds. */
export const MAX_SHOWN_MODELS = 50

export type ModelMatches = {
  shown: LlmModel[]
  /** How many more match than are shown. */
  hidden: number
  /** How many match in all. */
  total: number
}

/** The models whose id or name contains every word typed, ignoring case, capped for the list. */
export function filterModels(models: readonly LlmModel[], query: string, max = MAX_SHOWN_MODELS): ModelMatches {
  const words = query.toLowerCase().split(/\s+/).filter((w) => w !== '')
  const matching = models.filter((m) => {
    const text = `${m.id} ${m.name}`.toLowerCase()
    return words.every((w) => text.includes(w))
  })
  return { shown: matching.slice(0, max), hidden: Math.max(matching.length - max, 0), total: matching.length }
}

/** A model as the list names it: its name, with the id when they differ. */
export function modelLabel(model: LlmModel): string {
  return model.name === model.id ? model.id : `${model.name} (${model.id})`
}
