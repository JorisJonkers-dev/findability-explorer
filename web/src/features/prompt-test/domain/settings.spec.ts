import { describe, expect, it } from 'vitest'
import { problem } from '../../../testing/mountApp'
import { candidateTab, candidateWebsite, momentLabel, openRouterKeys, parseBudget, parseCandidateStatus, parseProviders, promptGroups, promptProblem } from './settings'

async function parsed(r: Response): Promise<unknown> {
  return JSON.parse(await r.text())
}

describe('the Prompt test’s settings', () => {
  it('reads what an administrator types', () => {
    expect(parseProviders(' OpenAI, google  meta-llama,,openai ')).toEqual(['openai', 'google', 'meta-llama'])
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
