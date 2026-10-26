import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { PromptTestSettings } from '../../../infrastructure/api/types.gen'
import { me, promptResults } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { llmSettings, STORED_ID } from '../../settings/fixtures'

function settings(overrides: Partial<PromptTestSettings> = {}): PromptTestSettings {
  return {
    credentialId: STORED_ID,
    providers: ['openai', 'google'],
    monthlyBudgetUsd: 10,
    spentThisMonthUsd: 2.5,
    cadence: 'monthly',
    nextDueAt: '2026-11-01T06:00:00Z',
    factSheet: { text: 'Tribelt makes metal conveyor belts in Haaksbergen.', version: 2, updatedBy: 'joris', updatedAt: '2026-10-26T09:00:00Z' },
    promptSet: {
      version: 1,
      prompts: [
        { id: 'eyelink-supplier-twente', family: 'eyelink', kind: 'supplier', scope: 'twente', language: 'nl', text: 'Welke bedrijven leveren draadogenbanden in Twente?' },
        { id: 'other-supplier-english', family: 'other', kind: 'supplier', scope: 'english', language: 'en', text: 'Which companies supply rod belts?' },
      ],
    },
    extractorModel: 'google/gemini-2.5-flash-lite',
    ...overrides,
  }
}

function server(extra: Record<string, Handler> = {}) {
  const bodies: unknown[] = []
  const handle = routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/admin/prompt-test': () => json(settings()),
    'GET /api/v1/admin/llm': () => json(llmSettings()),
    'GET /api/v1/prompt-test': () => json(promptResults()),
    'PATCH /api/v1/admin/prompt-test': async (r) => {
      const body = (await r.json()) as { factSheet?: string; cadence?: 'monthly' | 'weekly' }
      bodies.push(body)
      return json(settings({ ...(body.factSheet ? { factSheet: { text: body.factSheet, version: 3 } } : {}), cadence: body.cadence ?? 'monthly' }))
    },
    'POST /api/v1/admin/prompt-runs': () => json({ ...promptResults().runs[0], status: 'running', tasks: 1260 }, 202),
    ...extra,
  })
  return { handle, bodies }
}

const form = (wrapper: VueWrapper, title: string) => wrapper.get(`form[aria-labelledby="${title}"]`)

describe('the Prompt test settings', () => {
  it('shows the spend, the runs, the choices, the fact sheet and the prompt set', async () => {
    const { wrapper } = await mountApp(server().handle, '/settings/prompt-test')
    expect(document.title).toBe('Prompt test settings · Findability Explorer')
    expect(wrapper.text()).toContain('$2.50 of $10.00 spent this month. The next scheduled run starts 1 Nov 2026, 06:00 UTC.')
    expect(wrapper.get('table').text()).toContain('40,000 / 90,000')
    expect(wrapper.get('table').text()).toContain('$0.42')
    const choice = form(wrapper, 'choice-title')
    expect(choice.get('select').findAll('option').map((o) => o.text())).toEqual(['Choose a key', 'OpenRouter · Course budget key'])
    expect((choice.get('input').element as HTMLInputElement).value).toBe('openai, google')
    expect(form(wrapper, 'facts-title').text()).toContain('Version 2, by joris on 26 Oct 2026, 09:00 UTC.')
    expect(wrapper.findAll('details summary').map((s) => s.text())).toEqual(['Eyelink (1)', 'Other metal belts (1)'])
    expect(wrapper.text()).toContain('Every answer is read by google/gemini-2.5-flash-lite.')
  })

  it('saves the key, providers, budget and schedule, after checking them', async () => {
    const { handle, bodies } = server()
    const { wrapper } = await mountApp(handle, '/settings/prompt-test')
    const choice = form(wrapper, 'choice-title')
    const [providers, budget] = choice.findAll('input')
    await budget?.setValue('nothing')
    await choice.trigger('submit')
    expect(choice.get('[role="alert"]').text()).toBe('The budget is an amount in dollars above $0 and at most $1,000.')
    await budget?.setValue('12,50')
    await providers?.setValue(' , ')
    await choice.trigger('submit')
    expect(choice.get('[role="alert"]').text()).toBe('Name at least one provider, such as openai.')
    await providers?.setValue('OpenAI, x-ai')
    await choice.findAll('select')[1]?.setValue('weekly')
    await choice.trigger('submit')
    await settle()
    expect(bodies).toEqual([{ credentialId: STORED_ID, providers: ['openai', 'x-ai'], monthlyBudgetUsd: 12.5, cadence: 'weekly' }])
    expect(wrapper.text()).toContain('The settings are saved.')
  })

  it('saves the fact sheet as a new version, and says what the server refused', async () => {
    const { handle, bodies } = server()
    const { wrapper } = await mountApp(handle, '/settings/prompt-test')
    const facts = form(wrapper, 'facts-title')
    await facts.get('textarea').setValue('Tribelt B.V. makes belts in Haaksbergen, Twente.')
    await facts.trigger('submit')
    await settle()
    expect(bodies).toEqual([{ factSheet: 'Tribelt B.V. makes belts in Haaksbergen, Twente.' }])
    expect(wrapper.text()).toContain('The fact sheet is saved as a new version.')
    wrapper.unmount()
    const refused = await mountApp(server({ 'PATCH /api/v1/admin/prompt-test': () => problem(422, 'Unprocessable Entity', 'A fact sheet has 20 to 4000 characters.') }).handle, '/settings/prompt-test')
    await form(refused.wrapper, 'facts-title').trigger('submit')
    await settle()
    expect(refused.wrapper.text()).toContain('A fact sheet has 20 to 4000 characters.')
  })

  it('starts a run now, and says why it cannot', async () => {
    const { wrapper } = await mountApp(server().handle, '/settings/prompt-test')
    const run = wrapper.findAll('button').find((b) => b.text() === 'Run now')
    await run?.trigger('click')
    await settle()
    expect(wrapper.text()).toContain('A Prompt run started: 1260 answers to ask.')
    wrapper.unmount()
    const busy = await mountApp(server({ 'POST /api/v1/admin/prompt-runs': () => problem(409, 'Conflict', 'A Prompt run is running already.') }).handle, '/settings/prompt-test')
    await busy.wrapper.findAll('button').find((b) => b.text() === 'Run now')?.trigger('click')
    await settle()
    expect(busy.wrapper.get('[role="alert"]').text()).toBe('A Prompt run is running already.')
  })

  it('asks for a key first, and tells a non-administrator', async () => {
    const keyless = await mountApp(
      server({ 'GET /api/v1/admin/llm': () => json(llmSettings({ credentials: [] })), 'GET /api/v1/admin/prompt-test': () => json(settings({ credentialId: undefined, factSheet: { text: 'Facts about Tribelt, written once.', version: 1 } })) }).handle,
      '/settings/prompt-test',
    )
    expect(keyless.wrapper.text()).toContain('Add an OpenRouter key in the language-model settings first.')
    expect(keyless.wrapper.text()).toContain('Version 1. Answers about Tribelt')
    keyless.wrapper.unmount()
    const viewer = await mountApp(server({ 'GET /api/v1/me': () => json(me()), 'GET /api/v1/admin/prompt-test': () => problem(403, 'Forbidden') }).handle, '/settings/prompt-test')
    expect(viewer.wrapper.get('[role="alert"]').text()).toBe('Only administrators can change these settings.')
    viewer.wrapper.unmount()
    const broken = await mountApp(server({ 'GET /api/v1/admin/prompt-test': () => problem(500, 'Internal Server Error', 'Down.') }).handle, '/settings/prompt-test')
    expect(broken.wrapper.get('[role="alert"]').text()).toBe('Down.')
  })
})
