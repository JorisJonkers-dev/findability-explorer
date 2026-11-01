import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { Plan } from '../../../infrastructure/api/types.gen'
import { competitor, me, promptAnswer, promptResults, promptRun, promptSets, RUN_ID } from '../../../testing/fixtures'
import { promptHistory } from '../../../testing/historyFixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
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
    'GET /api/v1/prompt-test/history': () => json(promptHistory()),
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

  it('puts Over time after See every response, before who the responses name', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    const text = wrapper.text()
    expect(text.indexOf('See every response')).toBeLessThan(text.indexOf('Over time'))
    expect(text.indexOf('Over time')).toBeLessThan(text.indexOf('Who the responses name'))
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

describe('Over time on the AI responses page', () => {
  const BREAK_NOTE = '27 October 2026: The questions changed here (prompt set version 2), so the figures before and after are not the same measure.'
  const PARTIAL_NOTE =
    '27 October 2026, 06:03: a partial run of version 1. Only 62 of its 1,260 requests got a response (the rest failed because the OpenRouter key was refused), and 47 of those could be read, so its figure rests on few responses.'

  it('shows every version by default: a line per language, the goal, the break, the partial run, and the figures as a table', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses')
    const section = wrapper.get('section[aria-labelledby="over-time-title"]')
    expect(section.get('h2').text()).toBe('Over time')
    expect(section.text()).toContain('How often AI responses name Tribelt in each Prompt run. The lines show each language the questions were asked in. Twente counts as Dutch.')
    const buttons = section.findAll('[role="group"][aria-label="Prompt set"] button')
    expect(buttons.map((b) => [b.text(), b.attributes('aria-pressed')])).toEqual([
      ['All versions', 'true'],
      ['Version 2, current', 'false'],
      ['Version 1', 'false'],
    ])
    expect(section.text()).toContain('The questions changed with each version, so the line breaks between versions. Version 1 asked 90 questions, and version 2 asks 20 questions.')
    expect(section.get('[data-part="legend"]').text()).toBe('AllDutchGermanEnglishGoalA dashed ring marks a partial run.')
    expect(section.findAll('[data-part="end"] text').map((t) => t.text())).toEqual(['All 25%', 'Dutch 41%', 'English 10%', 'Goal 40%'])
    expect(section.get('[data-part="band"]').text()).toBe('Prompt set version 2')
    expect(section.findAll('[data-part="notes"] li').map((l) => l.text())).toEqual([BREAK_NOTE, PARTIAL_NOTE])
    const table = section.get('details')
    expect(table.get('summary').text()).toBe('Show the figures as a table')
    expect(table.findAll('th[scope="col"]').map((t) => t.text())).toEqual(['Date', 'All', 'Dutch', 'German', 'English'])
    expect(table.findAll('tbody tr').map((r) => r.findAll('th, td').map((c) => c.text()))).toEqual([
      ['27 October 2026, 06:03 (partial run)', '40%', '48%', '', ''],
      ['27 October 2026, 08:36', '25%', '41%', '', '10%'],
    ])
  })

  it('keeps the choice of version in the address, and says what that version asks', async () => {
    const { wrapper, router } = await mountApp(api(), '/ai-responses')
    const section = wrapper.get('section[aria-labelledby="over-time-title"]')
    await section.findAll('[role="group"][aria-label="Prompt set"] button')[2]?.trigger('click')
    await settle()
    expect(router.currentRoute.value.fullPath).toBe('/ai-responses?version=1')
    expect(section.get('[aria-pressed="true"]').text()).toBe('Version 1')
    expect(section.text()).toContain(
      'Version 1, published 26 October 2026: 90 questions. A supplier, an application, and a problem question for each belt family, in six scopes. It has one run so far, so there is no line yet.',
    )
    expect(section.findAll('[data-part="notes"] li').map((l) => l.text())).toEqual([PARTIAL_NOTE])
    expect(section.find('[data-part="band"]').exists()).toBe(false)
    expect(section.findAll('polyline')).toHaveLength(0)
  })

  it('opens at the version the address names', async () => {
    const { wrapper } = await mountApp(api(), '/ai-responses?version=2')
    const section = wrapper.get('section[aria-labelledby="over-time-title"]')
    expect(section.get('[aria-pressed="true"]').text()).toBe('Version 2, current')
    expect(section.text()).toContain('Version 2, published 27 October 2026: 20 questions.')
    expect(section.find('[data-part="notes"]').exists()).toBe(false)
  })

  it('says so before a run has a figure, and when the history cannot be loaded', async () => {
    const empty = await mountApp(api({ 'GET /api/v1/prompt-test/history': () => json(promptHistory({ runs: [] })) }), '/ai-responses')
    expect(empty.wrapper.get('section[aria-labelledby="over-time-title"]').text()).toContain('No complete runs yet.')
    empty.wrapper.unmount()
    const down = await mountApp(api({ 'GET /api/v1/prompt-test/history': () => problem(500, 'Internal Server Error', 'History down.') }), '/ai-responses')
    expect(down.wrapper.get('section[aria-labelledby="over-time-title"] [role="alert"]').text()).toContain('History down.')
  })
})
