import type { VueWrapper } from '@vue/test-utils'
import { originalStep } from '../../../testing/claudeStepFixtures'
import { describe, expect, it } from 'vitest'
import type { PromptTestSettingsChange } from '../../../infrastructure/api/types.gen'
import { me, promptResults, promptRun, RUN_ID } from '../../../testing/fixtures'
import { manageCounts } from '../../../testing/manageFixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { llmSettings, STORED_ID } from '../../settings/fixtures'
import { modelCatalogue, promptHistory, promptSets, promptSettings } from '../fixtures'

const STOPPED = '00000000-0000-4000-8000-000000000009'
const HAND = 'Stopped by hand. The responses so far are kept, and a retry finishes the run.'

function server(extra: Record<string, Handler> = {}) {
  const bodies: unknown[] = []
  const sent: string[] = []
  const handle = routes({
    'GET /api/v1/admin/claude-steps/reading': () => json(originalStep('reading', 'Reading responses', true)),
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    ...manageCounts(),
    'GET /api/v1/admin/prompt-test': () => json(promptSettings()),
    'GET /api/v1/admin/llm': () => json(llmSettings()),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'GET /api/v1/prompt-test/prompt-sets': () => json(promptSets()),
    'GET /api/v1/prompt-test/history': () => json(promptHistory()),
    'PATCH /api/v1/admin/prompt-test': async (r) => {
      const body = (await r.json()) as PromptTestSettingsChange
      bodies.push(body)
      const { factSheet, models, ...rest } = body
      return json(promptSettings({ ...rest, ...(models ? { modelList: models } : {}), ...(factSheet ? { factSheet: { text: factSheet, version: 3 } } : {}) }))
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

type Found = ReturnType<VueWrapper['get']>

function widget(wrapper: VueWrapper, title: string): Found {
  const heading = wrapper.findAll('section > div > h2').find((h) => h.text() === title)
  if (!heading) throw new Error(`no widget ${title}`)
  return wrapper.get(`section[aria-labelledby="${heading.attributes('id') ?? ''}"]`)
}

const button = (w: Found | VueWrapper, text: string) => {
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
    expect(titles.slice(-6)).toEqual(['Questions', 'Models', 'Schedule', 'Fact sheet', 'Runs', 'Reading responses'])
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
    expect(wrapper.findAll('a[href="/manage/prompt-test"]').map((a) => a.text())).toContain('Back to Prompt test')
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
      ['1 November 2026, 06:00', 'Version 1', '2', '358 of 360', 'Done, with failures', '$0.42'],
    ])
  })

  it('retries the failed responses and reads them again', async () => {
    const { handle, sent } = server()
    const { wrapper } = await mountApp(handle, '/manage/prompt-test/runs')
    const panel = wrapper.get('section[aria-label="Run of 1 November 2026"]')
    await button(panel, 'Retry the 2 failed responses').trigger('click')
    await settle()
    expect(wrapper.text()).toContain('Asking 2 responses again')
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
  const catalogueServer = (extra: Record<string, Handler> = {}) =>
    server({
      'GET /api/v1/admin/prompt-test': () => json(promptSettings({ modelList: ['openai/gpt-oss-20b', 'google/gemma:free'] })),
      'GET /api/v1/admin/prompt-test/catalogue': () => json(modelCatalogue()),
      ...extra,
    })
  const cells = (w: Found) => w.findAll('tbody tr').map((r) => r.findAll('td').map((c) => c.text()))

  it('shows the list with what each model costs a run, and every model on OpenRouter to add', async () => {
    const { wrapper } = await mountApp(catalogueServer().handle, '/manage/prompt-test/models')
    expect(wrapper.get('h1').text()).toBe('Models')
    const list = widget(wrapper, 'The model list')
    expect(list.text()).toContain('These models are asked in every run, 2 questions each. Each one is its own column on the AI responses page.')
    expect(list.get('select').findAll('option').map((o) => o.text())).toEqual(['Choose a key', 'Course budget key (OpenRouter)'])
    expect(cells(list)).toEqual([
      ['OpenAI', 'gpt-oss-20b', '$0.01', 'Paid', 'Remove gpt-oss-20b'],
      ['Google', 'Gemma 3 27B (free)', '$0.04', 'Free to ask', 'Remove Gemma 3 27B (free)'],
    ])
    const save = widget(wrapper, 'Save the model list')
    expect(save.text()).toContain('The list is saved. A run asks 2 models and costs about $0.06.')
    expect(button(save, 'Save the model list').attributes('disabled')).toBeDefined()
    const add = widget(wrapper, 'Add a model')
    expect(add.text()).toContain('Every text model on OpenRouter, 4 from 3 providers, as listed on 1 November 2026.')
    const rows = cells(add)
    expect(rows[0]).toEqual(['OpenAI', 'gpt-oss-20bRecommended', '5 August 2025', '$0.01', 'On the list'])
    expect(rows[2]).toEqual(['qwen', 'Qwen3 8B (free)', '', '$0.04Free, apart from web search', 'Add Qwen3 8B (free)'])
  })

  it('adds, removes, and saves the list, with what it changes', async () => {
    const { handle, bodies } = catalogueServer()
    const { wrapper } = await mountApp(handle, '/manage/prompt-test/models')
    const add = widget(wrapper, 'Add a model')
    await button(add, 'Add GPT-5 Nano').trigger('click')
    const list = widget(wrapper, 'The model list')
    expect(cells(list).map((r) => r[1])).toEqual(['gpt-oss-20b', 'Gemma 3 27B (free)', 'GPT-5 Nano'])
    expect(cells(add)[3]?.[4]).toBe('On the list')
    const save = widget(wrapper, 'Save the model list')
    expect(save.text()).toContain('With these changes, a run asks 3 models and costs about $0.08, $0.02 more than the saved list. Saving marks this date on the trend charts')
    await button(list, 'Remove gpt-oss-20b').trigger('click')
    expect(cells(list).map((r) => r[1])).toEqual(['Gemma 3 27B (free)', 'GPT-5 Nano'])
    await button(save, 'Undo the changes').trigger('click')
    expect(cells(list).map((r) => r[1])).toEqual(['gpt-oss-20b', 'Gemma 3 27B (free)'])
    await button(add, 'Add Qwen3 8B (free)').trigger('click')
    await button(save, 'Save the model list').trigger('click')
    await settle()
    expect(bodies).toEqual([{ credentialId: STORED_ID, models: ['openai/gpt-oss-20b', 'google/gemma:free', 'qwen/qwen3-8b:free'] }])
    expect(save.text()).toContain('The model list is saved. Charts will mark a break where it changed.')
  })

  it('narrows the catalogue to a provider and to the free models', async () => {
    const { wrapper } = await mountApp(catalogueServer().handle, '/manage/prompt-test/models')
    const add = widget(wrapper, 'Add a model')
    const provider = add.get('select')
    expect(provider.findAll('option').map((o) => o.text())).toEqual(['All 3 providers', 'Google (1)', 'OpenAI (2)', 'qwen (1)'])
    await provider.setValue('openai')
    expect(cells(add).map((r) => r[1])).toEqual(['gpt-oss-20bRecommended', 'GPT-5 Nano'])
    await provider.setValue('all')
    await add.get('input[type="checkbox"]').setValue(true)
    expect(cells(add).map((r) => r[1])).toEqual(['Gemma 3 27B (free)Recommended', 'Qwen3 8B (free)'])
    await provider.setValue('openai')
    expect(add.text()).toContain('No model matches these filters.')
  })

  it('starts from what runs ask before a list is saved, and saves a key chosen with it', async () => {
    const { handle, bodies } = server({
      'GET /api/v1/admin/prompt-test': () => json(promptSettings({ credentialId: undefined })),
      'GET /api/v1/admin/prompt-test/catalogue': () => json(modelCatalogue()),
    })
    const { wrapper } = await mountApp(handle, '/manage/prompt-test/models')
    const list = widget(wrapper, 'The model list')
    expect(cells(list).map((r) => r[1])).toEqual(['gpt-oss-20b', 'Gemma 3 27B (free)'])
    const save = button(widget(wrapper, 'Save the model list'), 'Save the model list')
    expect(save.attributes('disabled')).toBeDefined()
    await list.get('select').setValue(STORED_ID)
    expect(save.attributes('disabled')).toBeUndefined()
    await save.trigger('click')
    await settle()
    expect(bodies).toEqual([{ credentialId: STORED_ID, models: ['openai/gpt-oss-20b', 'google/gemma:free'] }])
  })

  it('cannot save an empty list, and names a model OpenRouter no longer lists', async () => {
    const { wrapper } = await mountApp(
      catalogueServer({ 'GET /api/v1/admin/prompt-test': () => json(promptSettings({ modelList: ['gone/model'] })) }).handle,
      '/manage/prompt-test/models',
    )
    const list = widget(wrapper, 'The model list')
    expect(cells(list)[0]).toEqual(['gone', 'gone/model', 'Not on OpenRouter now', 'Paid', 'Remove gone/model'])
    await button(list, 'Remove gone/model').trigger('click')
    expect(list.text()).toContain('The list is empty. Add a model below, or a run has nobody to ask.')
    expect(button(widget(wrapper, 'Save the model list'), 'Save the model list').attributes('disabled')).toBeDefined()
  })

  it('warns when the key does not work, and reads no catalogue without one', async () => {
    const { handle } = server({
      'GET /api/v1/admin/prompt-test': () => json(promptSettings({ keyWorks: false, credentialId: undefined })),
      'GET /api/v1/admin/llm': () => json(llmSettings({ credentials: [] })),
    })
    const { wrapper, requests } = await mountApp(handle, '/manage/prompt-test/models')
    expect(wrapper.get('[role="alert"]').text()).toContain('The OpenRouter key does not work, so no Prompt run can start.')
    expect(wrapper.text()).toContain('Add an OpenRouter key in Keys and spending first.')
    expect(widget(wrapper, 'Add a model').text()).toContain('OpenRouter’s models can be listed once the key works.')
    expect(requests().some((r) => r.includes('/catalogue'))).toBe(false)
  })

  it('says when the catalogue cannot be read', async () => {
    const { wrapper } = await mountApp(
      catalogueServer({ 'GET /api/v1/admin/prompt-test/catalogue': () => problem(422, 'No OpenRouter key works.') }).handle,
      '/manage/prompt-test/models',
    )
    expect(widget(wrapper, 'Add a model').get('[role="alert"]').text()).toBe('No OpenRouter key works.')
  })
})
