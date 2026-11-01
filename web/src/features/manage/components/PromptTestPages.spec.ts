import type { DOMWrapper, VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { PromptTestSettings } from '../../../infrastructure/api/types.gen'
import { me, promptResults, promptRun, RUN_ID } from '../../../testing/fixtures'
import { manageCounts } from '../../../testing/manageFixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { llmSettings, STORED_ID } from '../../settings/fixtures'
import { promptHistory, promptSets, promptSettings } from '../fixtures'

const STOPPED = '00000000-0000-4000-8000-000000000009'
const HAND = 'Stopped by hand. The responses so far are kept, and a retry finishes the run.'

function server(extra: Record<string, Handler> = {}) {
  const bodies: unknown[] = []
  const sent: string[] = []
  const handle = routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    ...manageCounts(),
    'GET /api/v1/admin/prompt-test': () => json(promptSettings()),
    'GET /api/v1/admin/llm': () => json(llmSettings()),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'GET /api/v1/prompt-test/prompt-sets': () => json(promptSets()),
    'GET /api/v1/prompt-test/history': () => json(promptHistory()),
    'PATCH /api/v1/admin/prompt-test': async (r) => {
      const body = (await r.json()) as Partial<PromptTestSettings> & { factSheet?: string }
      bodies.push(body)
      const { factSheet, ...rest } = body
      return json(promptSettings({ ...rest, ...(factSheet ? { factSheet: { text: factSheet, version: 3 } } : {}) }))
    },
    'POST /api/v1/admin/prompt-runs': () => (sent.push('start'), json({ ...promptRun(), status: 'running', tasks: 8 }, 202)),
    [`POST /api/v1/admin/prompt-runs/${RUN_ID}/stop`]: () => (sent.push('stop'), new Response(null, { status: 204 })),
    [`GET /api/v1/admin/prompt-runs/${RUN_ID}/retry-plan`]: () =>
      json({ askTasks: 2, askCostUsd: 0.04, unknownTasks: 0, readTasks: 1, readCostUsd: 0.01, readingCostUsd: 0.0002, answerCosts: [] }),
    [`POST /api/v1/admin/prompt-runs/${RUN_ID}/retry`]: () => (sent.push('retry'), json({ run: { ...promptRun(), status: 'running' }, tasks: 2, estimatedCostUsd: 0.04 }, 202)),
    [`POST /api/v1/admin/prompt-runs/${RUN_ID}/reread`]: () => (sent.push('reread'), json({ run: promptRun(), tasks: 1, estimatedCostUsd: 0.01 }, 202)),
    ...extra,
  })
  return { handle, bodies, sent }
}

function widget(wrapper: VueWrapper, title: string): DOMWrapper<Element> {
  const heading = wrapper.findAll('section > div > h2').find((h) => h.text() === title)
  if (!heading) throw new Error(`no widget ${title}`)
  return wrapper.get(`section[aria-labelledby="${heading.attributes('id') ?? ''}"]`)
}

const button = (w: DOMWrapper<Element> | VueWrapper, text: string) => {
  const b = w.findAll('button').find((x) => x.text() === text)
  if (!b) throw new Error(`no button ${text}`)
  return b
}

describe('the Prompt test index', () => {
  it('shows the last run, the next, the cost, and the budget, then Start a Prompt run with its estimate, then a card per subpage', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage/prompt-test')
    expect(document.title).toBe('Prompt test · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Prompt test')
    expect(wrapper.get('header a[href="/ai-responses"]').text()).toBe('See the AI responses')
    expect(widget(wrapper, 'Last run').text()).toContain('1 November 2026')
    expect(widget(wrapper, 'Last run').text()).toContain('358 of 360 responses')
    expect(widget(wrapper, 'Next run').text()).toContain('Monthly, at 06:00')
    expect(widget(wrapper, 'Cost per run').text()).toContain('$0.252 models, 2 questions')
    expect(widget(wrapper, 'Budget left').get('[data-part="budget-left"]').text()).toBe('$7.50 left this month')
    const start = widget(wrapper, 'Start a Prompt run')
    expect(start.text()).toContain('Asks 2 questions of 2 models, each from its own knowledge and with web search, 8 responses in all.')
    expect(start.get('[data-part="estimate"]').text()).toBe('About $0.25. $7.50 of this month’s OpenRouter budget is left.')
    const titles = wrapper.findAll('section > div > h2').map((h) => h.text())
    expect(titles.indexOf('Start a Prompt run')).toBeLessThan(titles.indexOf('Questions'))
    expect(titles.slice(-5)).toEqual(['Questions', 'Models', 'Schedule', 'Fact sheet', 'Runs'])
    expect(widget(wrapper, 'Runs').text()).toContain('runs1need a retry1')
    expect(widget(wrapper, 'Questions').get('a').attributes('href')).toBe('/manage/prompt-test/questions')
  })

  it('starts a run on one click while it fits the budget', async () => {
    const { handle, sent } = server()
    const { wrapper } = await mountApp(handle, '/manage/prompt-test')
    await button(widget(wrapper, 'Start a Prompt run'), 'Start a Prompt run').trigger('click')
    await settle()
    expect(sent).toEqual(['start'])
  })

  it('warns before the click when a run costs more than the budget has left, and asks again', async () => {
    const { handle, sent } = server({ 'GET /api/v1/admin/prompt-test': () => json(promptSettings({ spentThisMonthUsd: 9.9 })) })
    const { wrapper } = await mountApp(handle, '/manage/prompt-test')
    const start = widget(wrapper, 'Start a Prompt run')
    expect(start.get('[data-part="over-budget"]').text()).toBe('This costs more than the $0.10 left of this month’s OpenRouter budget.')
    await button(start, 'Start a Prompt run').trigger('click')
    expect(sent).toEqual([])
    expect(start.get('[role="alert"]').text()).toContain('This costs about $0.25. Only $0.10 is left of this month’s OpenRouter budget.')
    await button(start, 'Cancel').trigger('click')
    await button(start, 'Start a Prompt run').trigger('click')
    await button(start, 'Start anyway').trigger('click')
    await settle()
    expect(sent).toEqual(['start'])
  })

  it('turns Start off while a run runs, and stops that run on the second click', async () => {
    const { handle, sent } = server({
      'GET /api/v1/prompt-test': () => json(promptResults({ runs: [promptRun({ status: 'running', finishedAt: undefined })] })),
    })
    const { wrapper } = await mountApp(handle, '/manage/prompt-test')
    const start = widget(wrapper, 'Start a Prompt run')
    expect(start.get('[data-part="disabled-note"]').text()).toBe('A Prompt run is already running. Stop it or wait for it to finish.')
    expect(button(start, 'Start a Prompt run').attributes('disabled')).toBeDefined()
    expect(widget(wrapper, 'Last run').text()).toContain('No run has finished.')
    const run = widget(wrapper, 'Prompt run')
    await button(run, 'Stop the Prompt run').trigger('click')
    expect(run.text()).toContain('Stopping keeps the responses so far. You can finish the run later with a retry.')
    await button(run.get('[data-part="confirm-stop"]'), 'Stop the Prompt run').trigger('click')
    await settle()
    expect(sent).toEqual(['stop'])
  })

  it('turns Start off without a working key, with the way to Keys and spending', async () => {
    const { wrapper } = await mountApp(server({ 'GET /api/v1/admin/prompt-test': () => json(promptSettings({ keyWorks: false })) }).handle, '/manage/prompt-test')
    const start = widget(wrapper, 'Start a Prompt run')
    expect(start.get('[data-part="disabled-note"]').text()).toBe('Add a working OpenRouter key first, in Keys and spending.')
    expect(start.get('a[href="/manage/keys"]').text()).toBe('Open Keys and spending')
  })

  it('says what the server refused', async () => {
    const { wrapper } = await mountApp(server({ 'GET /api/v1/admin/prompt-test': () => problem(500, 'Internal Server Error', 'Down.') }).handle, '/manage/prompt-test')
    expect(wrapper.get('[role="alert"]').text()).toBe('Down.')
  })
})

describe('Runs', () => {
  const runs = [
    promptRun({ id: STOPPED, status: 'stopped', failure: HAND, answered: 40, failed: 0, tasks: 360, startedAt: '2026-10-28T06:00:00Z', costUsd: 0.05 }),
    promptRun(),
  ]

  it('shows the figures, a panel per run with failed responses, and every run with how it ended', async () => {
    const { wrapper } = await mountApp(server({ 'GET /api/v1/prompt-test': () => json(promptResults({ runs })) }).handle, '/manage/prompt-test/runs')
    expect(wrapper.get('h1').text()).toBe('Runs')
    expect(wrapper.get('a[href="/manage/prompt-test"]').text()).toContain('Prompt test')
    expect(wrapper.get('nav[aria-label="Manage"] a[aria-current="page"]').text()).toBe('Runs')
    expect(widget(wrapper, 'Runs').text()).toContain('The last one was on 28 October 2026.')
    expect(widget(wrapper, 'Need a retry').text()).toContain('2')
    expect(widget(wrapper, 'Run status').get('[data-part="run-status"]').text()).toBe('Ready for a new run')
    const panel = wrapper.get('section[aria-label="Run of 1 November 2026"]')
    expect(panel.text()).toContain('2 of 360 responses failed.')
    expect(panel.findAll('[data-part="estimate"]').map((e) => e.text())).toEqual([
      'About $0.04. 2 responses. $7.50 of this month’s OpenRouter budget is left.',
      'About $0.01. 1 response. $7.50 of this month’s OpenRouter budget is left.',
    ])
    const rows = widget(wrapper, 'All runs').findAll('tbody tr').map((r) => r.findAll('th, td').map((c) => c.text()))
    expect(rows).toEqual([
      ['28 October 2026, 06:00', 'Version 1', '2', '40 of 360', 'Stopped by hand', '$0.05'],
      ['1 November 2026, 06:00', 'Version 1', '2', '358 of 360', 'Done, with failed responses', '$0.42'],
    ])
  })

  it('retries the failed responses and reads them again', async () => {
    const { handle, sent } = server()
    const { wrapper } = await mountApp(handle, '/manage/prompt-test/runs')
    const panel = wrapper.get('section[aria-label="Run of 1 November 2026"]')
    await button(panel, 'Retry the 2 failed responses').trigger('click')
    await settle()
    expect(wrapper.text()).toContain('Asking 2 answers again')
    await button(panel, 'Read them again with Claude Haiku').trigger('click')
    await settle()
    expect(sent).toEqual(['retry', 'reread'])
  })

  it('says when every run is complete, and turns the retries off while a run runs', async () => {
    const done = await mountApp(server({ 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [promptRun({ answered: 360, failed: 0 })] })) }).handle, '/manage/prompt-test/runs')
    expect(widget(done.wrapper, 'Finish the failed runs').text()).toContain('Every run is complete. Nothing needs a retry.')
    expect(widget(done.wrapper, 'Need a retry').text()).toContain('Every run is complete.')
    done.wrapper.unmount()
    const busy = await mountApp(
      server({ 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [promptRun({ id: STOPPED, status: 'running', finishedAt: undefined }), promptRun()] })) }).handle,
      '/manage/prompt-test/runs',
    )
    expect(widget(busy.wrapper, 'Run status').text()).toContain('A run is running')
    const notes = busy.wrapper.findAll('[data-part="disabled-note"]').map((n) => n.text())
    expect(notes).toEqual(['Wait for the running work to finish or stop it.', 'Wait for the running work to finish or stop it.'])
    expect(widget(busy.wrapper, 'All runs').text()).toContain('Running')
  })

  it('says when nothing has run', async () => {
    const { wrapper } = await mountApp(server({ 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [] })) }).handle, '/manage/prompt-test/runs')
    expect(widget(wrapper, 'Runs').text()).toContain('No run yet.')
    expect(widget(wrapper, 'All runs').text()).toContain('No run yet.')
  })
})

describe('Questions', () => {
  it('shows the published set grouped by belt family, read-only, and every version with its questions', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage/prompt-test/questions')
    expect(wrapper.get('h1').text()).toBe('Questions')
    const list = widget(wrapper, 'The questions')
    expect(list.text()).toContain('The published set, version 2, grouped by belt family.')
    expect(list.findAll('h3').map((h) => h.text())).toEqual(['Eyelink 1 question', 'Other metal belts 1 question'])
    expect(list.findAll('li')[0]?.text()).toBe('Dutch, TwenteWie levert draadogenbanden in Twente?Who supplies eyelink belts in Twente?')
    expect(list.findAll('textarea, input')).toHaveLength(0)
    expect(widget(wrapper, 'Version 2 is published').text()).toContain('Published on 27 October 2026, 2 questions.')
    const folds = widget(wrapper, 'Version history').findAll('details')
    expect(folds.map((f) => f.get('summary').text())).toEqual(['Version 2 PublishedPublished on 27 October 2026, 2 questions', 'Version 1 Published on 20 October 2026, 1 question'])
    expect(folds[1]?.text()).toContain('Dutch, Twente. Wie levert draadogenbanden in Twente?')
  })

  it('says what the server refused', async () => {
    const { wrapper } = await mountApp(server({ 'GET /api/v1/prompt-test/prompt-sets': () => problem(500, 'Internal Server Error', 'Down.') }).handle, '/manage/prompt-test/questions')
    expect(wrapper.get('[role="alert"]').text()).toBe('Down.')
  })
})

describe('Schedule', () => {
  it('shows how often, the next run, and what it costs, and changes how often', async () => {
    const { handle, bodies } = server()
    const { wrapper } = await mountApp(handle, '/manage/prompt-test/schedule')
    expect(wrapper.get('h1').text()).toBe('Schedule')
    const how = widget(wrapper, 'How often')
    expect(button(how, 'Monthly').attributes('aria-pressed')).toBe('true')
    expect(widget(wrapper, 'Next run').text()).toContain('1 December 2026')
    expect(widget(wrapper, 'What it costs').text()).toContain('a year, 12 runs$3.00')
    await button(how, 'Monthly').trigger('click')
    await button(how, 'Weekly').trigger('click')
    await settle()
    expect(bodies).toEqual([{ cadence: 'weekly' }])
    expect(wrapper.text()).toContain('A Prompt run will start weekly.')
  })

  it('offers Set to monthly and Change the budget when a weekly schedule costs more than the budget', async () => {
    const over = promptSettings({ cadence: 'weekly', prices: { ...promptSettings().prices, runUsd: 3, runsPerMonth: 4.33, monthUsd: 13, overBudget: true } })
    const { handle, bodies } = server({ 'GET /api/v1/admin/prompt-test': () => json(over) })
    const { wrapper } = await mountApp(handle, '/manage/prompt-test/schedule')
    const warn = widget(wrapper, 'This schedule costs more than the budget')
    expect(warn.get('[role="alert"]').text()).toContain('At this schedule the runs cost about $13.00 a month, which is more than the $10.00 monthly budget.')
    await button(warn, 'Change the budget').trigger('click')
    expect(document.activeElement).toBe(widget(wrapper, 'Monthly budget').get('input').element)
    await button(warn, 'Set to monthly').trigger('click')
    await settle()
    expect(bodies).toEqual([{ cadence: 'monthly' }])
  })

  it('saves the budget after checking it', async () => {
    const { handle, bodies } = server()
    const { wrapper } = await mountApp(handle, '/manage/prompt-test/schedule')
    const budget = widget(wrapper, 'Monthly budget')
    await budget.get('input').setValue('nothing')
    await budget.get('form').trigger('submit')
    expect(budget.get('[role="alert"]').text()).toBe('The budget is an amount in dollars above $0 and at most $1,000.')
    await budget.get('input').setValue('12,50')
    await budget.get('form').trigger('submit')
    await settle()
    expect(bodies).toEqual([{ monthlyBudgetUsd: 12.5 }])
    expect(wrapper.text()).toContain('The monthly budget is $12.50.')
  })
})

describe('Fact sheet', () => {
  it('shows the text and its version, and saves it as a new one', async () => {
    const { handle, bodies } = server()
    const { wrapper } = await mountApp(handle, '/manage/prompt-test/fact-sheet')
    expect(wrapper.get('h1').text()).toBe('Fact sheet')
    const text = widget(wrapper, 'The text')
    expect(text.get('label').text()).toContain('Fact sheet, version 2')
    expect(button(text, 'Save as a new version').attributes('disabled')).toBeDefined()
    expect(widget(wrapper, 'Versions').text()).toContain('Version 2 is in use, saved by joris on 26 October 2026, 09:00.')
    await text.get('textarea').setValue('Tribelt B.V. makes belts in Haaksbergen, Twente.')
    expect(text.get('[data-part="version-note"]').text()).toBe('Version 2 is in use. Saving makes version 3. You have unsaved edits.')
    await button(text, 'Undo my edits').trigger('click')
    expect((text.get('textarea').element as HTMLTextAreaElement).value).toBe('Tribelt makes metal conveyor belts in Haaksbergen.')
    await text.get('textarea').setValue('Tribelt B.V. makes belts in Haaksbergen, Twente.')
    await text.get('form').trigger('submit')
    await settle()
    expect(bodies).toEqual([{ factSheet: 'Tribelt B.V. makes belts in Haaksbergen, Twente.' }])
    expect(wrapper.text()).toContain('The fact sheet is saved as version 3.')
  })

  it('says what the server refused', async () => {
    const { wrapper } = await mountApp(
      server({ 'PATCH /api/v1/admin/prompt-test': () => problem(422, 'Unprocessable Entity', 'A fact sheet has 20 to 4000 characters.') }).handle,
      '/manage/prompt-test/fact-sheet',
    )
    const text = widget(wrapper, 'The text')
    await text.get('textarea').setValue('Tribelt makes belts, and the server refuses this.')
    await text.get('form').trigger('submit')
    await settle()
    expect(text.get('[role="alert"]').text()).toBe('A fact sheet has 20 to 4000 characters.')
  })
})

describe('Models', () => {
  it('shows the key, the providers, and what runs cost, and saves the list', async () => {
    const { handle, bodies } = server()
    const { wrapper } = await mountApp(handle, '/manage/prompt-test/models')
    expect(wrapper.get('h1').text()).toBe('Models')
    const list = widget(wrapper, 'The model list')
    expect(list.get('select').findAll('option').map((o) => o.text())).toEqual(['Choose a key', 'Course budget key (OpenRouter)'])
    expect(list.findAll('ul[aria-label="Providers"] li').map((l) => l.find('span').text())).toEqual([
      'OpenAIopenai/gpt-oss-20b',
      'Googlegoogle/gemma:free, then google/gemini-2.5-flash-lite',
    ])
    const rows = widget(wrapper, 'What runs cost').findAll('tbody tr').map((r) => r.findAll('th, td').map((c) => c.text()))
    expect(rows[0]).toEqual(['OpenAIopenai/gpt-oss-20b', '$0.0012', '$0.0061', '$0.15'])
    const input = list.get('input')
    await input.setValue('Open AI')
    await button(list, 'Add').trigger('click')
    expect(list.get('[role="alert"]').text()).toBe('A provider is an OpenRouter id, such as openai or meta-llama.')
    await input.setValue('x-ai')
    await input.trigger('keydown', { key: 'Enter' })
    await list.findAll('ul[aria-label="Providers"] button')[1]?.trigger('click')
    await list.get('form').trigger('submit')
    await settle()
    expect(bodies).toEqual([{ credentialId: STORED_ID, providers: ['openai', 'x-ai'] }])
    expect(wrapper.text()).toContain('The model list is saved.')
  })

  it('warns when the key does not work, and asks for a key when there is none', async () => {
    const { wrapper } = await mountApp(
      server({
        'GET /api/v1/admin/prompt-test': () => json(promptSettings({ keyWorks: false, credentialId: undefined })),
        'GET /api/v1/admin/llm': () => json(llmSettings({ credentials: [] })),
      }).handle,
      '/manage/prompt-test/models',
    )
    expect(wrapper.get('[role="alert"]').text()).toContain('The OpenRouter key does not work, so no Prompt run can start.')
    expect(wrapper.text()).toContain('Add an OpenRouter key in Keys and spending first.')
    await widget(wrapper, 'The model list').get('form').trigger('submit')
    await settle()
  })
})
