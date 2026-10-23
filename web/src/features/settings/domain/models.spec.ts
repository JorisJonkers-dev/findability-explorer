import { describe, expect, it } from 'vitest'
import { models } from '../fixtures'
import { filterModels, MAX_SHOWN_MODELS, modelLabel } from './models'

describe('the model list', () => {
  it('matches every typed word in the id or the name, ignoring case', () => {
    expect(filterModels(models(), '  ').total).toBe(3)
    expect(filterModels(models(), 'CLAUDE haiku').shown.map((m) => m.id)).toEqual(['anthropic/claude-haiku-5-5'])
    expect(filterModels(models(), 'mistral small').shown).toHaveLength(1)
    expect(filterModels(models(), 'llama').total).toBe(0)
  })

  it('shows at most so many, and says how many more match', () => {
    const many = Array.from({ length: 120 }, (_, i) => ({ id: `m-${String(i)}`, name: `Model ${String(i)}` }))
    const matches = filterModels(many, 'model')
    expect(matches.shown).toHaveLength(MAX_SHOWN_MODELS)
    expect(matches.hidden).toBe(70)
    expect(filterModels(many, '', 200).hidden).toBe(0)
  })

  it('names a model, with its id when the name differs', () => {
    expect(modelLabel({ id: 'a/b', name: 'B' })).toBe('B (a/b)')
    expect(modelLabel({ id: 'a/b', name: 'a/b' })).toBe('a/b')
  })
})
