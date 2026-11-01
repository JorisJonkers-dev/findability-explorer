import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Plan } from '../../../infrastructure/api/types.gen'
import { competitor, me, promptAnswer, promptResults, promptRun, promptSets, RUN_ID } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes } from '../../../testing/mountApp'
import { words } from '../../../testing/words'

const plan: Plan = {
  catalogueVersion: 1,
  improvements: [],
  watched: [
    { target: 'aiShareOfVoice', goal: 0.4 },
    { target: 'twenteShareOfVoice', goal: 0.75 },
    { target: 'citationShare', goal: 0.5 },
  ],
  eyeOn: [],
  readinessChecks: 5,
}

const answers = [
  promptAnswer({}, ['Tribelt:tribelt']),
  promptAnswer({ mode: 'web' }, ['Twentebelt:twentebelt']),
  promptAnswer({ promptId: 'os-de', family: 'open_spiral', scope: 'german', kind: 'problem', language: 'de', prompt: 'Wer liefert Spiralbänder?' }, ['Twentebelt:twentebelt']),
]

function api(extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/prompt-test': () =>
      json(
        promptResults({
          mentions: [
            { competitorId: 'twentebelt', name: 'Twentebelt', role: 'competitor', answers: 9, first: 4, averagePosition: 1.6, share: 0.75 },
            { competitorId: 'tribelt', name: 'Tribelt B.V.', role: 'tribelt', answers: 3, first: 1, averagePosition: 2, share: 0.25 },
            { competitorId: 'lesjofors', name: 'Lesjöfors', role: 'group', answers: 1, first: 0, averagePosition: 3, share: 0.1 },
            { competitorId: 'forbo', name: 'Forbo', role: 'competitor', answers: 1, first: 0, averagePosition: 3, share: 0.05 },
          ],
        }),
      ),
    [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => json({ run: promptRun(), items: answers }),
    'GET /api/v1/prompt-test/prompt-sets': () => json(promptSets()),
    'GET /api/v1/plan': () => json(plan),
    'GET /api/v1/competitors': () => json({ items: [competitor('Twentebelt'), competitor('Forbo', { archived: true })] }),
    ...extra,
  })
}


const section = (wrapper: VueWrapper, title: string) => {
  const heading = wrapper.findAll('h2').find((h) => h.text() === title)
  if (!heading) throw new Error(`no section ${title}`)
  return wrapper.get(`section[aria-labelledby="${String(heading.attributes('id'))}"]`)
}

describe('AI responses', () => {
  it('has the frame: title, purpose, data line, and Help', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    expect(wrapper.get('h1').text()).toBe('AI responses')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe(
      'How often AI assistants name and cite Tribelt when buyers ask which metal conveyor belt to use and who makes it.',
    )
    expect(wrapper.get('[data-part="data-line"]').text()).toBe(
      'From the Prompt run of 1 November 2026: 90 questions to 2 models, each from the model’s own knowledge and with web search, 358 responses in all. Prompt set version 1.',
    )
    expect(wrapper.get('a[href="/help/prompt-test"]').text()).toBe('How this is measured, in Help')
    expect(wrapper.text()).not.toMatch(/\banswers?\b/i)
  })

  it('shows the three figures, each against its goal, and leads to every response', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    expect(wrapper.get('[aria-label="Key figures"]').findAll('li').map(words)).toEqual([
      '25% AI responses that name Tribelt Goal 40%',
      '50% AI responses about the Twente region name Tribelt Goal 75%',
      '17% Web responses that cite tribelt.nl Goal 50%',
    ])
    expect(wrapper.get('a[href="/ai-responses/all"]').text()).toBe('See every response')
  })

  it('leaves a place for Over time after See every response', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    expect(wrapper.find('[data-slot="over-time"]').exists()).toBe(true)
  })

  it('ranks who the responses name, with Tribelt highlighted and the others marked', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    const who = section(wrapper, 'Who the responses name')
    expect(who.text()).toContain('The share of the 12 responses that did not fail name each company. Tribelt is highlighted.')
    const rows = who.findAll('li')
    expect(rows.map(words)).toEqual(['Twentebelt 75%', 'Tribelt 25%', 'Lesjöfors Group company 10%', 'Forbo Not a competitor 5%'])
    expect(rows[1]?.attributes('data-tribelt')).toBe('true')
    expect(rows[0]?.get('a').attributes('href')).toBe('/competitors/twentebelt')
    expect(rows[3]?.find('a').exists()).toBe(false)
  })

  it('shows By model beside the questions asked, each in its own language with its translation, per belt family', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    const models = section(wrapper, 'By model')
    expect(models.findAll('tbody tr').map((r) => r.findAll('th, td').map(words))).toEqual([
      ['OpenAI, gpt-oss-20b', '33%', '67%', '33%'],
      ['Google, gemma:free', '0%', '0%', '0%'],
      ['All models', '17%', '33%', '17%'],
    ])
    const questions = section(wrapper, 'The questions asked')
    expect(questions.findAll('details summary').map(words)).toEqual(['Eyelink belts 1 question'])
    expect(questions.get('[lang="nl"]').text()).toBe('We bouwen een doorloopoven voor een industriële bakkerij. Welke fabrikanten in de regio Twente?')
    expect(questions.text()).toContain('We are building a tunnel oven for an industrial bakery. Which makers in the Twente region can supply the belt?')
    expect(questions.text()).toContain('Twente (Dutch)')
  })

  it('shows By belt family and scope with the buyer intent explained, each figure leading to its responses', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    const rows = section(wrapper, 'By belt family and scope').findAll('tbody tr')
    expect(words(rows[0]?.get('th') ?? { element: document.body })).toContain('Eyelink, Twente, application')
    expect(rows[0]?.get('[title]').attributes('title')).toBe('Buyer intent: application. The buyer describes what the belt is for, such as a bakery oven.')
    expect(rows[0]?.findAll('td').map(words)).toEqual(['100%', '0%', '50%'])
    expect(words(rows[1]?.get('th') ?? { element: document.body })).toContain('Open spiral, German, problem')
    expect(rows[0]?.findAll('td a')[1]?.attributes('href')).toBe('/ai-responses/all?family=eyelink&scope=twente&mode=web')
  })

  it('links administrators to the Prompt test', async () => {
    const reader = await mountApp(api(), '/ai-responses')
    expect(reader.wrapper.find('a[href="/manage/prompt-test"]').exists()).toBe(false)
    const admin = await mountApp(api({ 'GET /api/v1/me': () => json(me({ isAdmin: true })) }), '/ai-responses')
    expect(admin.wrapper.get('a[href="/manage/prompt-test"]').text()).toBe('Manage the Prompt test')
  })

  it('says so before the first run', async () => {
    const { wrapper } = await mountApp(api({ 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [], mentions: [] })) }), '/ai-responses')
    expect(wrapper.text()).toContain('No Prompt run has finished yet. The first one starts on the schedule.')
    expect(wrapper.find('[aria-label="Key figures"]').exists()).toBe(false)
  })

  it('reports a failure of the results or of the responses', async () => {
    const down = await mountApp(api({ 'GET /api/v1/prompt-test': () => problem(500, 'Internal Server Error', 'Down.') }), '/ai-responses')
    expect(down.wrapper.get('[role="alert"]').text()).toContain('Down.')
    const gone = await mountApp(api({ [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => problem(500, 'Internal Server Error', 'Gone.') }), '/ai-responses')
    expect(gone.wrapper.get('[role="alert"]').text()).toContain('Gone.')
  })
})
