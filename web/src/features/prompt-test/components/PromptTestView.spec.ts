import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { PromptAnswer } from '../../../infrastructure/api/types.gen'
import { me, promptResults, promptRun, RUN_ID } from '../../../testing/fixtures'
import { json, mountApp, problem, routes, settle } from '../../../testing/mountApp'

function answer(overrides: Partial<PromptAnswer> = {}): PromptAnswer {
  return {
    id: '2b3c4d5e-6f7a-4b8c-9d0e-1f2a3b4c5d6e', promptId: 'eyelink-supplier-twente', family: 'eyelink', kind: 'supplier', scope: 'twente',
    language: 'nl', prompt: 'Welke bedrijven leveren draadogenbanden in Twente?', promptSetVersion: 1, provider: 'openai',
    model: 'openai/gpt-oss-20b', mode: 'web', text: 'Twentebelt en Tribelt.', citations: ['https://www.tribelt.nl/'], at: '2026-11-01T06:10:00Z', costUsd: 0.001,
    extraction: {
      failed: false, namesTribelt: true, citesTribelt: true, correctness: 'wrong', wrongClaims: ['Tribelt is in Hengelo.'], costUsd: 0.0001,
      companies: [{ name: 'Twentebelt', competitorId: 'twentebelt', position: 1 }, { name: 'Ashworth', position: 2 }],
      addresses: ['https://www.tribelt.nl/'],
    },
    ...overrides,
  }
}

function api(extra: Record<string, (r: Request) => Response> = {}) {
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: (r) => {
      const scope = new URL(r.url).searchParams.get('scope')
      const items =
        scope === 'twente'
          ? [answer(), answer({ id: '3c4d5e6f-7a8b-4c9d-8e1f-2a3b4c5d6e7f', mode: 'knowledge', failure: 'The model is rate limited.', extraction: undefined }),
             answer({ id: '4d5e6f7a-8b9c-4d0e-8f2a-3b4c5d6e7f8a', extraction: { failed: true, companies: [], addresses: [], namesTribelt: false, citesTribelt: false, wrongClaims: [], costUsd: 0 } }),
             answer({ id: '5e6f7a8b-9c0d-4e1f-8a3b-4c5d6e7f8a9b', extraction: { failed: false, companies: [], addresses: [], namesTribelt: false, citesTribelt: false, wrongClaims: [], costUsd: 0 } })]
          : []
      return json({ run: promptRun(), items })
    },
    ...extra,
  })
}

const section = (wrapper: VueWrapper, title: string) => wrapper.get(`section[aria-labelledby="${title}"]`)

describe('the Prompt test view', () => {
  it('shows the latest run’s shares, both modes side by side', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/prompt-test')
    expect(document.title).toBe('Prompt test · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Do AI assistants name Tribelt?')
    const run = section(wrapper, 'run-title')
    expect(run.get('h2').text()).toBe('The run of 1 Nov 2026')
    expect(run.text()).toContain('Done · 360 of 360 answers, 2 without an answer · $0.42 · prompt set 1, fact sheet 1')
    expect(run.findAll('dd').map((d) => d.text())).toEqual(['17% name Tribelt · 0% cite it', '33% name Tribelt · 17% cite it'])
    const rows = () => run.findAll('tbody tr').map((r) => r.get('th').text())
    expect(rows()).toEqual(['OpenAI openai/gpt-oss-20b', 'Google google/gemma:free'])
    await run.findAll('[aria-label="Break down"] button')[1]?.trigger('click')
    expect(rows()).toHaveLength(6)
    await run.findAll('[aria-label="Break down"] button')[2]?.trigger('click')
    expect(rows()[3]).toBe('Curve-going (Tri-Flex)')
  })

  it('follows the shares across runs, the pilot first, for a model and scope', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/prompt-test')
    const trend = section(wrapper, 'trend-title')
    expect(trend.findAll('tbody th').map((t) => t.text())).toEqual(['Pilot, Oct 2026by hand: 5 prompts in Gemini and ChatGPT', '1 Nov 2026Done'])
    const [provider, scope] = trend.findAll('select')
    await provider?.setValue('openai')
    await scope?.setValue('twente')
    expect(trend.findAll('tbody tr')).toHaveLength(1)
    expect(trend.get('tbody tr').text()).toContain('50%')
  })

  it('ranks the competitors named and lists the wrong claims', async () => {
    const { wrapper } = await mountApp(api(), '/ai-search/prompt-test')
    expect(section(wrapper, 'ranking-title').get('ol').text()).toBe('Twentebelt · 9 answers, first in 4')
    expect(section(wrapper, 'ranking-title').find('a[href="/settings/candidates"]').exists()).toBe(true)
    expect(section(wrapper, 'claims-title').text()).toContain('“Tribelt is in Hengelo.”')
  })

  it('reads every answer beside how it was read, filtered', async () => {
    const { wrapper, requests } = await mountApp(api(), '/ai-search/prompt-test')
    expect(requests()).toContain(`GET /api/v1/prompt-runs/${RUN_ID}/answers?provider=openai&scope=twente`)
    const answers = section(wrapper, 'answers-title')
    const items = answers.findAll('ul[aria-label="Answers"] > li')
    expect(items).toHaveLength(4)
    expect(items[0]?.text()).toContain('Twente (Dutch) · supplier · With web search')
    expect(items[0]?.get('[data-testid="extraction"]').text()).toContain('Names Tribelt · cites it')
    expect(items[0]?.get('[data-testid="extraction"]').text()).toContain('Wrong about Tribelt: “Tribelt is in Hengelo.”')
    expect(items[0]?.get('ol').text()).toContain('Ashworth (not on the list)')
    expect(items[1]?.text()).toContain('No answer: The model is rate limited.')
    expect(items[2]?.text()).toContain('The extractor could not read this answer.')
    expect(items[3]?.text()).toContain('Does not name Tribelt · does not cite it')
    expect(items[3]?.text()).toContain('Names no company.')
    const [, scope, family, mode] = answers.findAll('select')
    await family?.setValue('eyelink')
    await mode?.setValue('web')
    await scope?.setValue('english')
    await settle()
    expect(requests()).toContain(`GET /api/v1/prompt-runs/${RUN_ID}/answers?provider=openai&scope=english&family=eyelink&mode=web`)
    expect(answers.text()).toContain('No answers here yet.')
  })

  it('follows a running run, and says when there is none yet', async () => {
    const running = promptRun({ id: 'b1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d', status: 'running', finishedAt: undefined, answered: 100, failed: 0, cells: [] })
    const { wrapper } = await mountApp(api({ 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [running, promptRun()] })) }), '/ai-search/prompt-test')
    expect(wrapper.get('[data-testid="run-progress"]').text()).toContain('OpenAI: 180')
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuetext')).toBe('100 of 360 answers · $0.42 so far')
    wrapper.unmount()
    const empty = await mountApp(api({ 'GET /api/v1/me': () => json(me()), 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [], mentions: [], wrongClaims: [] })) }), '/ai-search/prompt-test')
    expect(empty.wrapper.get('h2').text()).toBe('No Prompt run yet')
    expect(empty.wrapper.text()).toContain('when an administrator starts one in Settings')
    expect(empty.wrapper.text()).toContain('No answer said anything about Tribelt')
    expect(empty.wrapper.find('section[aria-labelledby="answers-title"]').exists()).toBe(false)
  })

  it('says when the results or answers cannot load, or nothing ranks', async () => {
    const failing = await mountApp(api({ 'GET /api/v1/prompt-test': () => problem(500, 'Internal Server Error', 'Down.') }), '/ai-search/prompt-test')
    expect(failing.wrapper.get('[role="alert"]').text()).toBe('Down.')
    failing.wrapper.unmount()
    const quiet = await mountApp(
      api({
        'GET /api/v1/prompt-test': () => json(promptResults({ mentions: [] })),
        [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => problem(500, 'Internal Server Error', 'No answers today.'),
      }),
      '/ai-search/prompt-test',
    )
    expect(quiet.wrapper.text()).toContain('The answers name no competitor on the list.')
    expect(section(quiet.wrapper, 'answers-title').get('[role="alert"]').text()).toBe('No answers today.')
  })
})
