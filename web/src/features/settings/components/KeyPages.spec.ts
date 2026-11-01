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

const widget = (wrapper: VueWrapper, title: string) => {
  const found = wrapper.findAll('section').find((s) => s.find('h2').exists() && s.get('h2').text() === title)
  if (!found) throw new Error(`no widget ${title}`)
  return found
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

describe('Keys and spending', () => {
  it('opens on this month\'s tokens, a card per key with its status and fingerprint, the budget, and the events', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage/keys')
    expect(document.title).toBe('Keys and spending · Findability Explorer')
    expect(wrapper.get('h1').text()).toBe('Keys and spending')
    expect(wrapper.get('nav[aria-label="Account"]').text()).toContain('Signed in as Ada')
    expect(wrapper.findAll('section > div > h2').map((h) => h.text())).toEqual(['Anthropic this month', 'OpenRouter key', 'Anthropic key', 'Budgets', 'Events', 'Models and instructions'])
    expect(widget(wrapper, 'Anthropic this month').text()).toContain('$0.48 of $15.00 spent this month (3%), $14.52 left')
    expect(widget(wrapper, 'OpenRouter key').text()).toContain('Not tested. It pays for the Prompt test')
    expect(widget(wrapper, 'OpenRouter key').text()).toContain('9f86d081884cfingerprint')
    expect(widget(wrapper, 'Anthropic key').text()).toContain('Works.')
    expect(widget(wrapper, 'Anthropic key').get('a').attributes('href')).toBe('/manage/keys/anthropic')
    expect(widget(wrapper, 'Events').text()).toContain('2events')
  })

  it('offers to add a missing key, and shows a Mistral key only while one is stored', async () => {
    const { wrapper } = await mountApp(server(llmSettings({ credentials: [storedCredential({ provider: 'mistral' })] })).handle, '/manage/keys')
    expect(widget(wrapper, 'OpenRouter key').text()).toContain('No key.')
    expect(widget(wrapper, 'OpenRouter key').get('a').text()).toBe('Add the key')
    expect(widget(wrapper, 'Mistral key').get('a').attributes('href')).toBe('/manage/keys/mistral')
  })

  it('tells someone who is not an administrator, and an account the server refuses', async () => {
    const reader = await mountApp(routes({}), '/manage/keys')
    expect(reader.wrapper.text()).toContain('Only administrators can start, stop, and change the work behind the results.')
    reader.wrapper.unmount()
    const { wrapper } = await mountApp(routes({ ...ADMIN, 'GET /api/v1/admin/llm': () => problem(403, 'Forbidden') }), '/manage/keys')
    expect(wrapper.get('[role="alert"]').text()).toContain('Only administrators can change these settings.')
    wrapper.unmount()
    const key = await mountApp(routes({ ...ADMIN, 'GET /api/v1/admin/llm': () => problem(403, 'Forbidden') }), '/manage/keys/openrouter')
    expect(key.wrapper.get('[role="alert"]').text()).toContain('Only administrators can change these settings.')
  })

  it('says when the settings cannot be read', async () => {
    const { wrapper } = await mountApp(routes({ ...ADMIN, 'GET /api/v1/admin/llm': () => problem(500, 'Internal Server Error') }), '/manage/keys')
    expect(wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
    wrapper.unmount()
    for (const path of ['/manage/keys/anthropic', '/manage/keys/budgets', '/manage/keys/events', '/manage/scans/settings']) {
      const page = await mountApp(routes({ ...ADMIN, 'GET /api/v1/admin/llm': () => problem(500, 'Internal Server Error') }), path)
      expect(page.wrapper.find('[role="alert"]').text()).toBe('Internal Server Error')
      page.wrapper.unmount()
    }
  })
})

describe('a key', () => {
  it('lists the key with its name, fingerprint, and whether it works, and what it pays for', async () => {
    const settings = llmSettings({ credentials: [vaultCredential(), storedCredential({ readable: false, fingerprint: '0123456789ab' })] })
    const { wrapper } = await mountApp(server(settings).handle, '/manage/keys/openrouter')
    expect(wrapper.get('h1').text()).toBe('OpenRouter key')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('The key that pays for the Prompt test.')
    expect(wrapper.get('nav[aria-label="Manage"] a[aria-current="page"]').text()).toBe('OpenRouter key')
    const stored = keyItem(wrapper, 'Course budget key (OpenRouter)')
    expect(stored.get('[data-part="status"]').text()).toBe('Cannot be read')
    expect(stored.text()).toContain('This key can no longer be read')
    expect(stored.text()).toContain('Fingerprint 0123456789ab')
    expect(stored.text()).toContain('Not tested yet.')
    expect(button(stored, 'Test the key').attributes('disabled')).toBeDefined()
    expect(widget(wrapper, 'What it pays for').text()).toContain('every question asked of every model')
  })

  it('shows the Vault key, this month\'s tokens, and the Admin key on the Anthropic page', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage/keys/anthropic')
    const vault = keyItem(wrapper, 'Vault key (Anthropic (Claude))')
    expect(vault.get('[data-part="status"]').text()).toBe('Works')
    expect(vault.text()).toContain('Accepted on 23 Oct 2026, 09:30 UTC: The key may use 9 models.')
    expect(vault.text()).toContain('Vault delivers this key; change or remove it in Vault.')
    expect(vault.findAll('button').map((b) => b.text())).toEqual(['Test the key', 'Usage and cost', 'Save the new key'])
    expect(widget(wrapper, 'Spent this month').text()).toContain('$0.48 of $15.00 spent this month (3%), $14.52 left')
    expect(widget(wrapper, 'Admin key for usage reports').text()).toContain('Add the Admin key')
  })

  it('replaces a key in a fold that stays open while the page redraws, and removes the old key', async () => {
    const { handle, bodies } = server()
    const { wrapper, requests, queryClient } = await mountApp(handle, '/manage/keys/openrouter')
    const fold = () => keyItem(wrapper, 'Course budget key (OpenRouter)').get('details')
    ;(fold().element as HTMLDetailsElement).open = true
    await queryClient.invalidateQueries()
    await settle()
    expect((fold().element as HTMLDetailsElement).open).toBe(true)
    const form = fold().get('form')
    expect(form.attributes('aria-label')).toBe('Replace Course budget key (OpenRouter)')
    await form.get('input[type="password"]').setValue('short')
    await form.trigger('submit')
    expect(form.get('[role="alert"]').text()).toBe('Paste the whole API key.')
    await form.get('input[type="password"]').setValue('sk-or-new-key-123')
    await form.trigger('submit')
    await settle()
    expect(bodies.add).toEqual({ provider: 'openrouter', label: 'Course budget key', apiKey: 'sk-or-new-key-123' })
    expect(requests()).toContain(`DELETE /api/v1/admin/llm/credentials/${STORED_ID}`)
  })

  it('opens a page without a key on the form to add it', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage/keys/mistral')
    expect(wrapper.get('h1').text()).toBe('Mistral key')
    expect(widget(wrapper, 'The key').text()).toContain('Add the Mistral key')
  })
})

describe('the budget and the events', () => {
  it('shows the month\'s spending and what a scan costs, and saves a new budget in dollars', async () => {
    const bodies: unknown[] = []
    const { wrapper } = await mountApp(
      server(llmSettings(), { 'PATCH /api/v1/admin/llm/budget': async (r) => (bodies.push(await r.json()), json(llmSettings())) }).handle,
      '/manage/keys/budgets',
    )
    expect(wrapper.get('h1').text()).toBe('Budgets')
    expect(wrapper.get('[data-testid="per-scan"]').text()).toBe(
      'A company scan costs $0.0044 on average (29,137 tokens), over 2 scans.What is left this month covers about 3,300 more scans.',
    )
    const form = wrapper.get('form[aria-label="Anthropic monthly budget"]')
    expect((form.get('input').element as HTMLInputElement).value).toBe('15.00')
    await form.get('input').setValue('none')
    await form.trigger('submit')
    expect(form.get('[role="alert"]').text()).toBe('Type a budget from $0.01 to $1,000.')
    await form.get('input').setValue('20')
    await form.trigger('submit')
    await settle()
    expect(bodies).toEqual([{ monthlyBudgetUsd: 20 }])
    expect(form.text()).toContain('The budget is saved.')
  })

  it('lists the events newest first, with who and whether the provider accepted', async () => {
    const settings = llmSettings({
      events: [...llmSettings().events, { at: '2026-10-21T08:00:00Z', actor: 'ada', action: 'credential_tested', ok: false, message: 'Tested mistral key "Old": refused.' }],
    })
    const { wrapper } = await mountApp(server(settings).handle, '/manage/keys/events')
    expect(wrapper.get('h1').text()).toBe('Events')
    const rows = wrapper.findAll('table[aria-label="Events"] tbody tr').map((r) => r.findAll('td').map((td) => td.text()))
    expect(rows[0]).toEqual(['23 Oct 2026, 09:30 UTC', 'Tested anthropic key "Vault key": accepted.', 'joris', 'Accepted'])
    expect(rows.at(-1)?.[3]).toBe('Refused')
    expect(wrapper.text()).toContain('The last 3 events')
  })

  it('says when nothing has changed yet', async () => {
    const { wrapper } = await mountApp(server(llmSettings({ events: [] })).handle, '/manage/keys/events')
    expect(wrapper.text()).toContain('Nothing has changed yet.')
  })
})

describe('adding a key', () => {
  const form = (wrapper: VueWrapper) => wrapper.get('form[aria-labelledby]')

  it('checks the name and key before sending them', async () => {
    const { wrapper, requests } = await mountApp(server().handle, '/manage/keys/mistral')
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
    const { wrapper } = await mountApp(handle, '/manage/keys/mistral')
    const key = form(wrapper).get('input[type="password"]')
    expect(key.attributes('autocomplete')).toBe('off')
    expect(form(wrapper).find('select').exists()).toBe(false)
    await form(wrapper).get('input[type="text"]').setValue(' Mistral key ')
    await key.setValue('fake-test-key')
    await form(wrapper).trigger('submit')
    await settle()
    expect(bodies.add).toEqual({ provider: 'mistral', label: 'Mistral key', apiKey: 'fake-test-key' })
    expect((key.element as HTMLInputElement).value).toBe('')
    expect(form(wrapper).text()).toContain('Added "Mistral key".')
  })

  it("shows the provider's reason for refusing it", async () => {
    const refuse = { 'POST /api/v1/admin/llm/credentials': () => problem(422, 'Unprocessable Entity', 'invalid api key') }
    const { wrapper } = await mountApp(server(llmSettings(), refuse).handle, '/manage/keys/mistral')
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
    const { wrapper } = await mountApp(server().handle, '/manage/keys/openrouter')
    const stored = () => keyItem(wrapper, 'Course budget key (OpenRouter)')
    await button(stored(), 'Test the key').trigger('click')
    await settle()
    expect(stored().get('[aria-live="polite"]').text()).toBe('Accepted on 23 Oct 2026, 11:00 UTC: The key may use 312 models.')
  })

  it('shows a refusal, and a test that could not run', async () => {
    let answer = (): Response => json({ ok: false, at: '2026-10-23T11:00:00Z', message: 'invalid x-api-key' })
    const { wrapper } = await mountApp(server(llmSettings(), { [`POST /api/v1/admin/llm/credentials/${STORED_ID}/test`]: () => answer() }).handle, '/manage/keys/openrouter')
    const stored = () => keyItem(wrapper, 'Course budget key (OpenRouter)')
    await button(stored(), 'Test the key').trigger('click')
    await settle()
    expect(stored().text()).toContain('Refused on 23 Oct 2026, 11:00 UTC: invalid x-api-key')
    answer = () => problem(502, 'Bad Gateway')
    await button(stored(), 'Test the key').trigger('click')
    await settle()
    expect(stored().get('[role="alert"]').text()).toBe('The provider did not answer. Try again in a minute.')
  })

  it('asks before removing a key, in place', async () => {
    const { wrapper, requests } = await mountApp(server().handle, '/manage/keys/openrouter')
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
    expect(document.activeElement?.id).toBe('keys-openrouter')
  })

  it('says why a key could not be removed', async () => {
    const { wrapper } = await mountApp(server(llmSettings(), { [`DELETE /api/v1/admin/llm/credentials/${STORED_ID}`]: () => problem(404, 'Not Found') }).handle, '/manage/keys/openrouter')
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
    const { wrapper, requests } = await mountApp(handle, '/manage/scans/settings')
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
    const { wrapper } = await mountApp(server().handle, '/manage/scans/settings')
    await chooser(wrapper).get('select').setValue(STORED_ID)
    await settle()
    await chooser(wrapper).get('input[type="search"]').setValue('llama')
    expect(chooser(wrapper).text()).toContain('No model matches.')
    expect(chooser(wrapper).find('fieldset').exists()).toBe(false)
  })

  it('lists hundreds of models a page at a time', async () => {
    const many = Array.from({ length: 312 }, (_, i) => ({ id: `vendor/model-${String(i)}`, name: `Model ${String(i)}` }))
    const { wrapper } = await mountApp(server(llmSettings(), { [`GET /api/v1/admin/llm/credentials/${STORED_ID}/models`]: () => json({ items: many }) }).handle, '/manage/scans/settings')
    await chooser(wrapper).get('select').setValue(STORED_ID)
    await settle()
    expect(chooser(wrapper).findAll('input[type="radio"]')).toHaveLength(50)
    expect(chooser(wrapper).text()).toContain('Showing 50 of 312 models: 262 more, refine the search.')
  })

  it('says when the provider did not answer', async () => {
    const { wrapper } = await mountApp(server(llmSettings(), { [`GET /api/v1/admin/llm/credentials/${STORED_ID}/models`]: () => problem(502, 'Bad Gateway') }).handle, '/manage/scans/settings')
    await chooser(wrapper).get('select').setValue(STORED_ID)
    await settle()
    expect(chooser(wrapper).get('[role="alert"]').text()).toBe('The provider did not answer. Try again in a minute.')
  })

  it('turns matching off', async () => {
    const { wrapper, requests } = await mountApp(server().handle, '/manage/scans/settings')
    await button(chooser(wrapper), 'Turn matching off').trigger('click')
    await settle()
    expect(requests()).toContain('DELETE /api/v1/admin/llm/active')
    expect(chooser(wrapper).text()).toContain('Product matching is off. Scans run without it until a model is chosen below.')
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
      '/manage/scans/settings',
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
      '/manage/keys/openrouter',
    )
    expect(requests().some((r) => r.endsWith('/usage'))).toBe(false)
    const stored = keyItem(wrapper, 'Course budget key (OpenRouter)')
    await button(stored, 'Usage and cost').trigger('click')
    await settle()
    expect(stored.get('[data-testid="key-usage"]').text()).toContain('$0.50 today, $1.00 this week, $2.00 this month, $3.00 in all.')
    expect(stored.get('[data-testid="key-usage"]').text()).toContain('No spending limit on this key.')
    await button(stored, 'Hide usage').trigger('click')
    expect(stored.find('[data-testid="key-usage"]').exists()).toBe(false)
    wrapper.unmount()
    const anthropic = await mountApp(
      server(llmSettings(), { 'GET /api/v1/admin/llm/credentials/vault-anthropic/usage': () => problem(502, 'Bad Gateway', 'Anthropic answered 500.') }).handle,
      '/manage/keys/anthropic',
    )
    const vault = keyItem(anthropic.wrapper, 'Vault key (Anthropic (Claude))')
    await button(vault, 'Usage and cost').trigger('click')
    await settle()
    expect(vault.get('[role="alert"]').text()).toBe('The provider did not answer: Anthropic answered 500. Try again in a minute.')
  })
})
