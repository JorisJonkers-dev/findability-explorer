import { describe, expect, it } from 'vitest'
import type { Plan } from '../../../infrastructure/api/types.gen'
import { competitor, promptAnswer, promptResults, promptRun } from '../../../testing/fixtures'
import { byModel, familyScopeRows, figuresOf, goalOf, share, whoNamed } from './sections'

const plan = (watched: Plan['watched']): Plan => ({ catalogueVersion: 1, improvements: [], watched, eyeOn: [], readinessChecks: 5 })

describe('a share', () => {
  it('is a whole percentage, or a dash with nothing to share', () => {
    expect(share(1, 4)).toBe('25%')
    expect(share(0, 0)).toBe('–')
  })
})

describe('the three figures', () => {
  it('measure the newest run against each goal', () => {
    const cells = promptRun().cells
    expect(figuresOf(cells, plan([{ target: 'aiShareOfVoice', goal: 0.4 }, { target: 'twenteShareOfVoice', goal: 0.75 }, { target: 'citationShare', goal: 0.5 }]))).toEqual([
      { value: '25%', label: 'AI responses that name Tribelt', goal: 'Goal 40%' },
      { value: '50%', label: 'AI responses about the Twente region name Tribelt', goal: 'Goal 75%' },
      { value: '17%', label: 'Web responses that cite tribelt.nl', goal: 'Goal 50%' },
    ])
  })

  it('read their goals from the plan', () => {
    expect(goalOf(plan([{ target: 'aiShareOfVoice', goal: 0.4 }]), 'aiShareOfVoice')).toBe(0.4)
    expect(goalOf(undefined, 'citationShare')).toBeUndefined()
  })

  it('leave out a goal that is not set, and show a dash without responses', () => {
    expect(figuresOf([], undefined).map((f) => `${f.value} ${f.goal}`)).toEqual(['– ', '– ', '– '])
  })
})

describe('by model', () => {
  it('has a row per model and one for all', () => {
    expect(byModel(promptRun())).toEqual([
      { provider: 'openai', label: 'OpenAI, gpt-oss-20b', knowledge: '33%', web: '67%', cited: '33%', total: false },
      { provider: 'google', label: 'Google, gemma:free', knowledge: '0%', web: '0%', cited: '0%', total: false },
      { provider: 'all', label: 'All models', knowledge: '17%', web: '33%', cited: '17%', total: true },
    ])
  })
})

describe('by belt family and scope', () => {
  it('has a row per family, scope, and buyer intent, each mode leading to its responses', () => {
    const rows = familyScopeRows([
      promptAnswer({ promptId: 'os-de', family: 'open_spiral', scope: 'german', mode: 'web' }, ['Tribelt:tribelt']),
      promptAnswer({ mode: 'knowledge' }, ['Tribelt:tribelt']),
      promptAnswer({ mode: 'web' }, ['Twentebelt:twentebelt']),
      promptAnswer({ mode: 'web', failure: 'x', extraction: undefined }),
    ])
    expect(rows.map((r) => [r.place, r.intent.name, r.knowledge.value, r.web.value, r.all.value])).toEqual([
      ['Eyelink, Twente', 'application', '100%', '0%', '50%'],
      ['Open spiral, German', 'application', '–', '100%', '100%'],
    ])
    expect(rows[0]?.web.to).toEqual({ path: '/ai-responses/all', query: { family: 'eyelink', scope: 'twente', mode: 'web' } })
    expect(rows[0]?.all.to).toEqual({ path: '/ai-responses/all', query: { family: 'eyelink', scope: 'twente' } })
  })
})

describe('who the responses name', () => {
  it('ranks the companies, marks Tribelt, group companies, and those that are no competitor', () => {
    const results = promptResults({
      mentions: [
        { competitorId: 'twentebelt', name: 'Twentebelt', role: 'competitor', answers: 9, first: 4, averagePosition: 1.6, share: 0.75 },
        { competitorId: 'forbo', name: 'Forbo', role: 'competitor', answers: 2, first: 0, averagePosition: 3, share: 0.1 },
        { competitorId: 'lesjofors', name: 'Lesjöfors', role: 'group', answers: 1, first: 0, averagePosition: 3, share: 0.05 },
        { competitorId: 'mirror', name: 'Mirror of tribelt.nl', role: 'mirror', answers: 1, first: 0, averagePosition: 3, share: 0.05 },
      ],
    })
    expect(whoNamed(results.mentions, [competitor('Forbo', { id: 'forbo', archived: true })])).toEqual([
      { id: 'twentebelt', name: 'Twentebelt', share: 0.75, value: '75%', tribelt: false, tag: '', to: '/competitors/twentebelt' },
      { id: 'forbo', name: 'Forbo', share: 0.1, value: '10%', tribelt: false, tag: 'Not a competitor', to: undefined },
      { id: 'lesjofors', name: 'Lesjöfors', share: 0.05, value: '5%', tribelt: false, tag: 'Group company', to: undefined },
      { id: 'mirror', name: 'Mirror of tribelt.nl', share: 0.05, value: '5%', tribelt: false, tag: 'Not a competitor', to: undefined },
      { id: 'tribelt', name: 'Tribelt', share: 0, value: '0%', tribelt: true, tag: '', to: undefined },
    ])
    expect(whoNamed(promptResults().mentions, [])[1]).toMatchObject({ id: 'tribelt', name: 'Tribelt', tribelt: true })
  })
})
