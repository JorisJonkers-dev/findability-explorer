import { describe, expect, it } from 'vitest'
import { problem } from '../../../testing/mountApp'
import {
  addProvider,
  basisLine,
  candidateTab,
  candidateWebsite,
  modelLine,
  momentLabel,
  openRouterKeys,
  parseBudget,
  parseCandidateStatus,
  promptGroups,
  promptProblem,
} from './settings'

async function parsed(r: Response): Promise<unknown> {
  return JSON.parse(await r.text())
}

describe('the Prompt test’s settings', () => {
  it('reads what an administrator types', () => {
    expect([parseBudget('$12.50'), parseBudget('7,5'), parseBudget('0'), parseBudget('1001'), parseBudget('ten')]).toEqual([12.5, 7.5, null, null, null])
    expect(momentLabel('2026-11-01T06:00:00Z')).toBe('1 Nov 2026, 06:00 UTC')
  })

  it('offers only readable OpenRouter keys', () => {
    const key = { id: 'a', provider: 'openrouter', label: 'k', source: 'stored', fingerprint: '000000000000', readable: true } as const
    expect(openRouterKeys([key, { ...key, id: 'b', readable: false }, { ...key, id: 'c', provider: 'mistral' }]).map((k) => k.id)).toEqual(['a'])
  })

  it('groups the prompt set by belt family', () => {
    const groups = promptGroups([
      { id: 'eyelink-supplier-twente', family: 'eyelink', kind: 'supplier', scope: 'twente', language: 'nl', text: 'Welke?' },
      { id: 'other-supplier-english', family: 'other', kind: 'supplier', scope: 'english', language: 'en', text: 'Which?' },
      { id: 'eyelink-problem-german', family: 'eyelink', kind: 'problem', scope: 'german', language: 'de', text: 'Warum?' },
    ])
    expect(groups.map((g) => [g.family, g.prompts.length])).toEqual([['Eyelink', 2], ['Other metal belts', 1]])
    expect(groups[0]?.prompts[1]).toEqual({ scope: 'German, no region', text: 'Warum?', id: 'eyelink-problem-german' })
  })
})

describe('the competitor candidates', () => {
  it('names the tabs and reads the one in the address', () => {
    expect(candidateTab('pending', { pending: 12, added: 1, dismissed: 3 })).toBe('To review (12)')
    expect(candidateTab('dismissed', undefined)).toBe('Dismissed')
    expect([parseCandidateStatus('added'), parseCandidateStatus('nonsense'), parseCandidateStatus(undefined)]).toEqual(['added', 'pending', 'pending'])
  })

  it('suggests the website from the first address the answers gave', () => {
    expect(candidateWebsite({ addresses: ['ashworth', 'https://www.ashworth.com/belts/x?y=1'] })).toBe('https://www.ashworth.com/')
    expect(candidateWebsite({ addresses: [] })).toBe('')
  })

  it('reads a problem', async () => {
    expect(promptProblem(await parsed(problem(409, 'Conflict', 'A Prompt run is running already.')))).toBe('A Prompt run is running already.')
    expect(promptProblem(await parsed(problem(403, 'Forbidden')))).toBe('Only administrators can do this.')
    expect(promptProblem(await parsed(problem(500, 'Internal Server Error')))).toContain('Internal Server Error')
    expect(promptProblem(new Error('offline'))).toBe('Something went wrong. Try again.')
  })
})

describe('the providers and prices', () => {
  it('add a provider by its OpenRouter id, once, at most 20', () => {
    expect(addProvider(['openai'], ' Google ')).toEqual({ list: ['openai', 'google'], problem: null })
    expect(addProvider(['openai'], 'openai').problem).toBe('openai is on already.')
    expect(addProvider(['openai'], 'Open AI').problem).toBe('A provider is an OpenRouter id, such as openai or meta-llama.')
    expect(addProvider(Array.from({ length: 20 }, (_, i) => `p${String(i)}`), 'more').problem).toBe('The Prompt test asks at most 20 providers.')
  })

  it("say each provider's model from the latest run", () => {
    const models = [{ provider: 'google', model: 'google/gemma:free', free: true, fallback: 'google/gemini' }, { provider: 'openai', model: 'openai/gpt-oss-20b', free: false }]
    expect(modelLine(models, 'google')).toBe('google/gemma:free, then google/gemini')
    expect(modelLine(models, 'openai')).toBe('openai/gpt-oss-20b')
    expect(modelLine(models, 'deepseek')).toBe('Cheapest model chosen at the next run')
  })

  it('say where the prices come from', () => {
    expect(basisLine({ basis: 'run' })).toBe('From what the latest run cost.')
    expect(basisLine({ basis: 'catalogue' })).toContain('estimate')
    expect(basisLine({ basis: 'none' })).toContain('No prices yet.')
  })
})
