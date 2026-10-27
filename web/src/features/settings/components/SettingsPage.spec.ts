import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { LlmSettings } from '../../../infrastructure/api/types.gen'
import { me } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes, settle } from '../../../testing/mountApp'
import { llmSettings, models, OTHER_ID, STORED_ID, storedCredential, vaultCredential } from '../fixtures'

const ADMIN = { 'GET /api/v1/me': () => json(me({ name: 'Ada', isAdmin: true })) }

/** An administrator's server, with the settings as given; it records each request's body. */
function server(settings: LlmSettings = llmSettings(), extra: Record<string, Handler> = {}) {
  const bodies: Record<string, unknown> = {}
  const record = (key: string, answer: (body: unknown) => Response): Handler => async (request) => {
    const text = await request.text()
    bodies[key] = text === '' ? null : JSON.parse(text)
    return answer(bodies[key])
  }
  const handle = routes({
    ...ADMIN,
    'GET /api/v1/admin/llm': () => json(settings),
    'POST /api/v1/admin/llm/credentials': record('add', () => json(storedCredential({ id: OTHER_ID, provider: 'mistral', label: 'Mistral key' }), 201)),
    [`POST /api/v1/admin/llm/credentials/${STORED_ID}/test`]: () => json({ ok: true, at: '2026-10-23T11:00:00Z', message: 'The key may use 312 models.' }),
    [`DELETE /api/v1/admin/llm/credentials/${STORED_ID}`]: () => new Response(null, { status: 204 }),
    [`GET /api/v1/admin/llm/credentials/${STORED_ID}/models`]: () => json({ items: models() }),
    'PUT /api/v1/admin/llm/active': record('active', () => json(llmSettings({ choice: 'chosen', active: { credentialId: STORED_ID, model: 'mistralai/mistral-small' } }))),
    'DELETE /api/v1/admin/llm/active': () => json(llmSettings({ choice: 'off', active: undefined, matchingEnabled: false })),
    ...extra,
  })
  return { handle, bodies }
}

const keyItem = (wrapper: VueWrapper, name: string) => {
  const item = wrapper.findAll('li').find((li) => li.find('h3').exists() && li.get('h3').text() === name)
  if (!item) throw new Error(`no key named ${name}`)
  return item
}
const button = (wrapper: Pick<VueWrapper, 'findAll'>, text: string) => {
  const found = wrapper.findAll('button').find((b) => b.text() === text)
  if (!found) throw new Error(`no button "${text}"`)
  return found
}

describe('the settings page', () => {
  it('lists the keys, what is in use, the budget and the recent changes', async () => {
    const settings = llmSettings({
      credentials: [vaultCredential(), storedCredential({ readable: false, fingerprint: '0123456789ab' })],
      events: [
        ...llmSettings().events,
        { at: '2026-10-21T08:00:00Z', actor: 'ada', action: 'credential_tested', ok: false, message: 'Tested mistral key "Old": refused.' },
      ],
    })
    const { wrapper } = await mountApp(server(settings).handle, '/settings')
    expect(document.title).toBe('Settings · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Language models')
    expect(wrapper.get('header').text()).toContain('Signed in as Ada')

    const vault = keyItem(wrapper, 'Vault key (Anthropic (Claude))')
    expect(vault.text()).toContain('Vault')
    expect(vault.text()).toContain('Fingerprint 3a7bd3e2360a')
    expect(vault.text()).toContain('Accepted on 23 Oct 2026, 09:30 UTC: The key may use 9 models.')
    expect(vault.text()).toContain('Vault delivers this key; change or remove it in Vault.')
    expect(vault.findAll('button').map((b) => b.text())).toEqual(['Test', 'Usage and cost'])

    const stored = keyItem(wrapper, 'Course budget key (OpenRouter)')
    expect(stored.text()).toContain('This key can no longer be read')
    expect(stored.text()).toContain('Fingerprint 0123456789ab')
    expect(stored.text()).toContain('Not tested yet.')
    expect(button(stored, 'Test').attributes('disabled')).toBeDefined()

    expect(wrapper.text()).toContain('claude-haiku-5-5 with Vault key (Anthropic (Claude)), the default.')
    expect(wrapper.text()).toContain('48,211 of 10,000,000 tokens used this month (0%)')
    expect(wrapper.text()).not.toContain('per hour')
    expect(wrapper.get('[data-testid="per-scan"]').text()).toBe(
      'A company scan uses 29,137 tokens on average, over 2 scans.What is left this month covers about 341 more scans.',
    )
    const events = wrapper.get('section[aria-labelledby="events-title"]').findAll('li')
    expect(events.map((e) => e.text())[0]).toContain('Tested anthropic key "Vault key": accepted.')
    expect(events.at(-1)?.text()).toContain('Refused')
  })

  it('tells someone who is not an administrator, and shows no forms', async () => {
    const { wrapper, requests } = await mountApp(routes({ 'GET /api/v1/admin/llm': () => problem(403, 'Forbidden') }), '/settings')
    expect(wrapper.get('[role="alert"]').text()).toContain('Only administrators can change these settings.')
    // Only the account's sign-out form is left.
    expect(wrapper.findAll('form').map((f) => f.attributes('action'))).toEqual(['/auth/logout'])
    expect(wrapper.find('nav[aria-label="Account"] a[href="/settings"]').exists()).toBe(false)
    expect(requests()).toContain('GET /api/v1/admin/llm')
  })

  it('says when the settings cannot be read', async () => {
    const { wrapper } = await mountApp(routes({ ...ADMIN, 'GET /api/v1/admin/llm': () => problem(500, 'Internal Server Error') }), '/settings')
    expect(wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
  })

  it('says when there is no key and nothing in use', async () => {
    const { wrapper } = await mountApp(server(llmSettings({ credentials: [], active: undefined, events: [] })).handle, '/settings')
    expect(wrapper.text()).toContain('No keys yet. Add one below.')
    expect(wrapper.text()).toContain('No model is chosen, so scans run without product matching.')
    expect(wrapper.text()).toContain('Add a key first')
    expect(wrapper.text()).toContain('Nothing has changed yet.')
  })
})

describe('adding a key', () => {
  const form = (wrapper: VueWrapper) => wrapper.get('form[aria-labelledby="add-title"]')

  it('checks the name and key before sending them', async () => {
    const { wrapper, requests } = await mountApp(server().handle, '/settings')
    await form(wrapper).trigger('submit')
    expect(form(wrapper).get('[role="alert"]').text()).toBe('Give the key a name, to tell it apart from others.')
    await form(wrapper).get('input[type="text"]').setValue('Mistral key')
    await form(wrapper).get('input[type="password"]').setValue('short')
    await form(wrapper).trigger('submit')
    expect(form(wrapper).get('[role="alert"]').text()).toBe('Paste the whole API key.')
    expect(form(wrapper).get('input[type="password"]').attributes('aria-invalid')).toBe('true')
    expect(requests().filter((r) => r.startsWith('POST'))).toEqual([])
  })

  it('sends the key, then forgets it', async () => {
    const { handle, bodies } = server()
    const { wrapper } = await mountApp(handle, '/settings')
    const key = form(wrapper).get('input[type="password"]')
    expect(key.attributes('autocomplete')).toBe('off')
    expect(form(wrapper).findAll('option').map((o) => o.text())).toEqual(['OpenRouter', 'Mistral', 'Anthropic (Claude)', 'Anthropic Admin key'])
    await form(wrapper).get('select').setValue('mistral')
    await form(wrapper).get('input[type="text"]').setValue(' Mistral key ')
    await key.setValue('fake-test-key')
    await form(wrapper).trigger('submit')
    await settle()
    expect(bodies.add).toEqual({ provider: 'mistral', label: 'Mistral key', apiKey: 'fake-test-key' })
    expect((key.element as HTMLInputElement).value).toBe('')
    expect(form(wrapper).text()).toContain('Added "Mistral key". Choose a model for it above.')
  })

  it("shows the provider's reason for refusing it", async () => {
    const refuse = { 'POST /api/v1/admin/llm/credentials': () => problem(422, 'Unprocessable Entity', 'invalid api key') }
    const { wrapper } = await mountApp(server(llmSettings(), refuse).handle, '/settings')
    await form(wrapper).get('input[type="text"]').setValue('Bad key')
    await form(wrapper).get('input[type="password"]').setValue('sk-0123456789')
    await form(wrapper).trigger('submit')
    await settle()
    expect(form(wrapper).get('[role="alert"]').text()).toBe('The provider refused this key. invalid api key')
    // A refused key stays in the field, to be corrected.
    expect((form(wrapper).get('input[type="password"]').element as HTMLInputElement).value).toBe('sk-0123456789')
  })
})

describe('testing and removing a key', () => {
  it("tests a key and shows the provider's answer", async () => {
    const { wrapper } = await mountApp(server().handle, '/settings')
    const stored = () => keyItem(wrapper, 'Course budget key (OpenRouter)')
    await button(stored(), 'Test').trigger('click')
    await settle()
    expect(stored().get('[aria-live="polite"]').text()).toBe('Accepted on 23 Oct 2026, 11:00 UTC: The key may use 312 models.')
  })

  it('shows a refusal, and a test that could not run', async () => {
    let answer = (): Response => json({ ok: false, at: '2026-10-23T11:00:00Z', message: 'invalid x-api-key' })
    const { wrapper } = await mountApp(server(llmSettings(), { [`POST /api/v1/admin/llm/credentials/${STORED_ID}/test`]: () => answer() }).handle, '/settings')
    const stored = () => keyItem(wrapper, 'Course budget key (OpenRouter)')
    await button(stored(), 'Test').trigger('click')
    await settle()
    expect(stored().text()).toContain('Refused on 23 Oct 2026, 11:00 UTC: invalid x-api-key')
    answer = () => problem(502, 'Bad Gateway')
    await button(stored(), 'Test').trigger('click')
    await settle()
    expect(stored().get('[role="alert"]').text()).toBe('The provider did not answer. Try again in a minute.')
  })

  it('asks before removing a key, in place', async () => {
    const { wrapper, requests } = await mountApp(server().handle, '/settings')
    const stored = () => keyItem(wrapper, 'Course budget key (OpenRouter)')
    await button(stored(), 'Remove').trigger('click')
    await settle()
    expect(stored().get('[role="group"]').attributes('aria-label')).toBe('Remove Course budget key (OpenRouter)?')
    expect(document.activeElement?.textContent.trim()).toBe('Cancel')
    await button(stored(), 'Cancel').trigger('click')
    await settle()
    expect(stored().find('[role="group"]').exists()).toBe(false)
    expect(document.activeElement?.textContent.trim()).toBe('Remove')

    await button(stored(), 'Remove').trigger('click')
    await settle()
    await button(stored(), 'Remove key').trigger('click')
    await settle()
    expect(requests()).toContain(`DELETE /api/v1/admin/llm/credentials/${STORED_ID}`)
    expect(document.activeElement?.id).toBe('keys-title')
  })

  it('says why a key could not be removed', async () => {
    const { wrapper } = await mountApp(server(llmSettings(), { [`DELETE /api/v1/admin/llm/credentials/${STORED_ID}`]: () => problem(404, 'Not Found') }).handle, '/settings')
    const stored = () => keyItem(wrapper, 'Course budget key (OpenRouter)')
    await button(stored(), 'Remove').trigger('click')
    await settle()
    await button(stored(), 'Remove key').trigger('click')
    await settle()
    expect(stored().get('[role="alert"]').text()).toBe('That key no longer exists. Reload the page.')
  })
})

describe('choosing the model', () => {
  const chooser = (wrapper: VueWrapper) => wrapper.get('section[aria-labelledby="model-title"]')

  it("loads a key's models, finds one and makes it the one in use", async () => {
    const { handle, bodies } = server()
    const { wrapper, requests } = await mountApp(handle, '/settings')
    // The key in use is picked to begin with, and its model.
    expect((chooser(wrapper).get('select').element as HTMLSelectElement).value).toBe('vault-anthropic')
    await chooser(wrapper).get('select').setValue(STORED_ID)
    await settle()
    expect(requests()).toContain(`GET /api/v1/admin/llm/credentials/${STORED_ID}/models`)
    expect(chooser(wrapper).text()).toContain('3 models.')
    expect(button(chooser(wrapper), 'Save').attributes('disabled')).toBeDefined()
    await chooser(wrapper).get('input[type="search"]').setValue('small')
    expect(chooser(wrapper).findAll('input[type="radio"]')).toHaveLength(1)
    expect(chooser(wrapper).text()).toContain('1 model.')
    await chooser(wrapper).get('input[type="radio"]').setValue(true)
    expect(chooser(wrapper).text()).toContain('Chosen: mistralai/mistral-small')
    await chooser(wrapper).get('form').trigger('submit')
    await settle()
    expect(bodies.active).toEqual({ credentialId: STORED_ID, model: 'mistralai/mistral-small' })
    expect(chooser(wrapper).text()).toContain('Product matching now uses mistralai/mistral-small.')
    expect(chooser(wrapper).text()).toContain('mistralai/mistral-small with Course budget key (OpenRouter).')
  })

  it('says when no model matches the search', async () => {
    const { wrapper } = await mountApp(server().handle, '/settings')
    await chooser(wrapper).get('select').setValue(STORED_ID)
    await settle()
    await chooser(wrapper).get('input[type="search"]').setValue('llama')
    expect(chooser(wrapper).text()).toContain('No model matches.')
    expect(chooser(wrapper).find('fieldset').exists()).toBe(false)
  })

  it('lists hundreds of models a page at a time', async () => {
    const many = Array.from({ length: 312 }, (_, i) => ({ id: `vendor/model-${String(i)}`, name: `Model ${String(i)}` }))
    const { wrapper } = await mountApp(server(llmSettings(), { [`GET /api/v1/admin/llm/credentials/${STORED_ID}/models`]: () => json({ items: many }) }).handle, '/settings')
    await chooser(wrapper).get('select').setValue(STORED_ID)
    await settle()
    expect(chooser(wrapper).findAll('input[type="radio"]')).toHaveLength(50)
    expect(chooser(wrapper).text()).toContain('Showing 50 of 312 models: 262 more, refine the search.')
  })

  it('says when the provider did not answer', async () => {
    const { wrapper } = await mountApp(server(llmSettings(), { [`GET /api/v1/admin/llm/credentials/${STORED_ID}/models`]: () => problem(502, 'Bad Gateway') }).handle, '/settings')
    await chooser(wrapper).get('select').setValue(STORED_ID)
    await settle()
    expect(chooser(wrapper).get('[role="alert"]').text()).toBe('The provider did not answer. Try again in a minute.')
  })

  it('turns matching off', async () => {
    const { wrapper, requests } = await mountApp(server().handle, '/settings')
    await button(chooser(wrapper), 'Turn matching off').trigger('click')
    await settle()
    expect(requests()).toContain('DELETE /api/v1/admin/llm/active')
    expect(chooser(wrapper).text()).toContain('Product matching is off. Scans run without it.')
    expect(chooser(wrapper).text()).toContain('Product matching is off.')
  })

  it('warns when the key in use can no longer be read, and shows a failed save', async () => {
    const settings = llmSettings({
      choice: 'chosen',
      matchingEnabled: false,
      active: { credentialId: STORED_ID, model: 'm' },
      credentials: [vaultCredential(), storedCredential({ readable: false })],
    })
    const { wrapper } = await mountApp(
      server(settings, {
        'PUT /api/v1/admin/llm/active': () => problem(422, 'Unprocessable Entity'),
        'GET /api/v1/admin/llm/credentials/vault-anthropic/models': () => json({ items: models() }),
      }).handle,
      '/settings',
    )
    expect(chooser(wrapper).text()).toContain('Its key can no longer be read')
    // The unreadable key is not offered, so nothing is picked to begin with.
    expect((chooser(wrapper).get('select').element as HTMLSelectElement).value).toBe('')
    await chooser(wrapper).get('select').setValue('vault-anthropic')
    await settle()
    await chooser(wrapper).get('input[type="radio"]').setValue(true)
    await chooser(wrapper).get('form').trigger('submit')
    await settle()
    expect(chooser(wrapper).get('[role="alert"]').text()).toBe('This key can no longer be read. Choose another key.')
  })
})

describe('a key’s usage and cost', () => {
  it('asks the provider when someone looks, and shows what it reports', async () => {
    const { wrapper, requests } = await mountApp(
      server(llmSettings(), {
        [`GET /api/v1/admin/llm/credentials/${STORED_ID}/usage`]: () =>
          json({ source: 'openrouter', message: "OpenRouter's own figures for this key, in dollars.", models: [], totalUsd: 3, dailyUsd: 0.5, weeklyUsd: 1, monthlyUsd: 2 }),
        'GET /api/v1/admin/llm/credentials/vault-anthropic/usage': () => problem(502, 'Bad Gateway', 'Anthropic answered 500.'),
      }).handle,
      '/settings',
    )
    expect(requests().some((r) => r.endsWith('/usage'))).toBe(false)
    const stored = keyItem(wrapper, 'Course budget key (OpenRouter)')
    await button(stored, 'Usage and cost').trigger('click')
    await settle()
    expect(stored.get('[data-testid="key-usage"]').text()).toContain('$0.50 today, $1.00 this week, $2.00 this month, $3.00 in all.')
    expect(stored.get('[data-testid="key-usage"]').text()).toContain('No spending limit on this key.')
    await button(stored, 'Hide usage').trigger('click')
    expect(stored.find('[data-testid="key-usage"]').exists()).toBe(false)
    const vault = keyItem(wrapper, 'Vault key (Anthropic (Claude))')
    await button(vault, 'Usage and cost').trigger('click')
    await settle()
    expect(vault.get('[role="alert"]').text()).toBe('The provider did not answer: Anthropic answered 500. Try again in a minute.')
  })
})
