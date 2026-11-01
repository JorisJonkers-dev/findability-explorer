import { describe, expect, it } from 'vitest'
import type { PromptCatalogueModel } from '../../../infrastructure/api/types.gen'
import { promptSettings } from '../fixtures'
import {
  addModel,
  catalogueIntro,
  catalogueRows,
  filterCatalogue,
  listCost,
  listRows,
  MAX_MODELS,
  modelName,
  providerOptions,
  removeModel,
  saveLine,
  savedList,
  sameList,
} from './models'

function entry(id: string, extra: Partial<PromptCatalogueModel> = {}): PromptCatalogueModel {
  const provider = id.split('/')[0] ?? id
  return { id, name: id, provider, free: false, recommended: false, basis: 'catalogue', knowledgeUsd: 0.001, webUsd: 0.021, ...extra }
}

const catalogue: PromptCatalogueModel[] = [
  entry('openai/gpt-oss-20b', { name: 'OpenAI: gpt-oss-20b', recommended: true, basis: 'run', knowledgeUsd: 0.0012, webUsd: 0.0061, listedAt: '2025-08-05T17:17:09Z' }),
  entry('google/gemma:free', { name: 'Google: Gemma 3 (free)', recommended: true, free: true, knowledgeUsd: 0, webUsd: 0.02 }),
  entry('qwen/qwen3:free', { name: 'Qwen3 8B (free)', free: true, knowledgeUsd: 0, webUsd: 0.02 }),
  entry('openai/gpt-5-nano', { name: 'OpenAI: GPT-5 Nano', knowledgeUsd: 0.0004, webUsd: 0.0104 }),
]

describe('the model list', () => {
  it('starts from the saved list, else what the providers ask, else what the last run asked', () => {
    expect(savedList(promptSettings({ modelList: ['qwen/qwen3:free'] }))).toEqual(['qwen/qwen3:free'])
    expect(savedList(promptSettings({ modelList: [] }))).toEqual(['openai/gpt-oss-20b', 'google/gemma:free'])
    expect(savedList(promptSettings({ modelList: [], prices: { ...promptSettings().prices, providers: [] } }))).toEqual(['openai/gpt-oss-20b', 'google/gemma:free'])
    expect(savedList(promptSettings({ modelList: [], prices: { ...promptSettings().prices, providers: [] }, models: [] }))).toEqual([])
  })

  it('adds a model once, up to twenty, and removes one', () => {
    expect(addModel(['a/b'], 'c/d')).toEqual(['a/b', 'c/d'])
    expect(addModel(['a/b'], 'a/b')).toEqual(['a/b'])
    const full = Array.from({ length: MAX_MODELS }, (_, i) => `p/m${String(i)}`)
    expect(addModel(full, 'c/d')).toEqual(full)
    expect(removeModel(['a/b', 'c/d'], 'a/b')).toEqual(['c/d'])
    expect(sameList(['a/b', 'c/d'], ['c/d', 'a/b'])).toBe(true)
    expect(sameList(['a/b'], ['a/b', 'c/d'])).toBe(false)
  })

  it('names a model without its provider, as OpenRouter writes it', () => {
    expect(modelName(catalogue[0])).toBe('gpt-oss-20b')
    expect(modelName(catalogue[2])).toBe('Qwen3 8B (free)')
    expect(modelName(undefined, 'gone/model')).toBe('gone/model')
  })

  it('gives each listed model its provider, name, cost per run, and whether it is free', () => {
    expect(listRows(['google/gemma:free', 'gone/model'], catalogue, 2)).toEqual([
      { id: 'google/gemma:free', provider: 'Google', name: 'Gemma 3 (free)', runUsd: 0.04, free: true, known: true },
      { id: 'gone/model', provider: 'gone', name: 'gone/model', runUsd: 0, free: false, known: false },
    ])
  })

  it('costs a list: its models per run, and reading every response', () => {
    // 2 questions: gpt-oss 2 × (0.0012 + 0.0061), gemma 2 × 0.02, reading 0.0002 × 2 questions × 2 modes × 2 models.
    expect(listCost(['openai/gpt-oss-20b', 'google/gemma:free'], catalogue, 2, 0.0002)).toBeCloseTo(0.0146 + 0.04 + 0.0016, 10)
    expect(listCost(['gone/model'], catalogue, 2, 0)).toBe(0)
  })

  it('says what saving changes, and that the charts mark the date', () => {
    expect(saveLine(['openai/gpt-oss-20b'], ['openai/gpt-oss-20b'], catalogue, 2, 0)).toBe('The list is saved. A run asks 1 model and costs about $0.01.')
    expect(saveLine(['openai/gpt-oss-20b'], ['openai/gpt-oss-20b', 'google/gemma:free'], catalogue, 2, 0)).toBe(
      'With these changes, a run asks 2 models and costs about $0.05, $0.04 more than the saved list. Saving marks this date on the trend charts, so results from before and after are not compared.',
    )
    expect(saveLine(['openai/gpt-oss-20b', 'google/gemma:free'], ['openai/gpt-oss-20b'], catalogue, 2, 0)).toContain('$0.04 less than the saved list.')
    expect(saveLine(['openai/gpt-oss-20b'], [], catalogue, 2, 0)).toBe('The list is empty. Add a model below, or a run has nobody to ask.')
  })
})

describe('Add a model', () => {
  it('offers every provider with its count, and narrows to one and to the free ones', () => {
    expect(providerOptions(catalogue)).toEqual([
      { value: 'all', label: 'All 3 providers' },
      { value: 'google', label: 'Google (1)' },
      { value: 'openai', label: 'OpenAI (2)' },
      { value: 'qwen', label: 'qwen (1)' },
    ])
    expect(filterCatalogue(catalogue, 'openai', false).map((e) => e.id)).toEqual(['openai/gpt-oss-20b', 'openai/gpt-5-nano'])
    expect(filterCatalogue(catalogue, 'all', true).map((e) => e.id)).toEqual(['google/gemma:free', 'qwen/qwen3:free'])
    expect(filterCatalogue(catalogue, 'all', false)).toHaveLength(4)
  })

  it('gives each row its provider, name, listing date, cost per run, and whether it is on the list', () => {
    const rows = catalogueRows(catalogue, ['google/gemma:free'], 2)
    expect(rows[0]).toEqual({
      id: 'openai/gpt-oss-20b',
      provider: 'OpenAI',
      name: 'gpt-oss-20b',
      recommended: true,
      listed: '5 August 2025',
      listedAt: '2025-08-05T17:17:09Z',
      runUsd: 0.0146,
      free: false,
      onList: false,
      search: 'openai/gpt-oss-20b OpenAI: gpt-oss-20b OpenAI',
    })
    expect(rows[1]).toMatchObject({ listed: '', listedAt: '', onList: true, free: true })
  })

  it('says what the catalogue holds and when it was read', () => {
    expect(catalogueIntro(catalogue, '2026-11-01T06:00:00Z', 20)).toBe(
      'Every text model on OpenRouter, 4 from 3 providers, as listed on 1 November 2026. The list is read again daily, so new models appear by themselves. First come the recommended models, the cheapest of the five providers whose assistants buyers use most. The rest follow, cheapest first. Costs are for one run of 20 questions. Each question is asked twice, once from the model’s own knowledge and once with web search. Models already asked show what their runs cost.',
    )
  })
})
