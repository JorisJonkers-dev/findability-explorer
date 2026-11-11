import { flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { getLlmSettingsQueryKey } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { LlmSettings } from '../../../infrastructure/api/types.gen'
import { mountComposable } from '../../../testing/mountComposable'
import { json, llmSettings, models, OTHER_ID, problem, routes, STORED_ID, storedCredential, vaultCredential } from '../fixtures'
import {
  useActiveModel,
  useBudgetChange,
  useAddCredential,
  useCredentialTest,
  useLlmModels,
  useLlmSettings,
  useRemoveCredential,
  useReplaceCredential,
} from './useLlmSettings'

const SETTINGS = 'GET /api/v1/admin/llm'

describe('useLlmSettings', () => {
  it('loads the settings', async () => {
    const { state } = mountComposable(useLlmSettings, routes({ [SETTINGS]: () => json(llmSettings()) }))
    expect(state.isLoading.value).toBe(true)
    await flushPromises()
    expect(state.settings.value?.credentials).toHaveLength(2)
    expect(state.forbidden.value).toBe(false)
    expect(state.error.value).toBeNull()
  })

  it('tells a refusal for not being an administrator from another failure', async () => {
    const forbidden = mountComposable(useLlmSettings, routes({ [SETTINGS]: () => problem(403, 'Forbidden') }))
    await flushPromises()
    expect(forbidden.state.forbidden.value).toBe(true)
    expect(forbidden.state.error.value).toBe('Only administrators can change these settings.')
    const failing = mountComposable(useLlmSettings, routes({ [SETTINGS]: () => problem(500, 'Internal Server Error') }))
    await flushPromises()
    expect(failing.state.forbidden.value).toBe(false)
    expect(failing.state.error.value).toBe('Internal Server Error')
  })
})

describe('useLlmModels', () => {
  const MODELS = `GET /api/v1/admin/llm/credentials/${STORED_ID}/models`

  it('asks for nothing until a key is picked, then for its models', async () => {
    const id = ref('')
    let answer: (response: Response) => void = () => undefined
    const pending = new Promise<Response>((resolve) => {
      answer = resolve
    })
    const { state, sent } = mountComposable(() => useLlmModels(id), routes({ [MODELS]: () => pending }))
    await flushPromises()
    expect(sent).toEqual([])
    expect(state.models.value).toEqual([])
    expect(state.isLoading.value).toBe(false)
    id.value = STORED_ID
    await vi.waitFor(() => {
      expect(state.isLoading.value).toBe(true)
    })
    answer(json({ items: models() }))
    await flushPromises()
    expect(state.models.value).toHaveLength(3)
    expect(state.isLoading.value).toBe(false)
    expect(state.error.value).toBeNull()
  })

  it('says when the provider did not answer', async () => {
    const { state } = mountComposable(() => useLlmModels(ref(STORED_ID)), routes({ [MODELS]: () => problem(502, 'Bad Gateway') }))
    await flushPromises()
    expect(state.error.value).toBe('The provider did not reply. Try again in a minute.')
  })
})

describe('the changes', () => {
  it('adds a key, then asks for the settings again', async () => {
    const onAdded = vi.fn()
    const { state, sent } = mountComposable(useAddCredential, routes({ 'POST /api/v1/admin/llm/credentials': () => json(llmSettings().credentials[1], 201) }))
    state.add({ provider: 'openrouter', label: 'Course budget key', apiKey: 'sk-or-v1-secret' }, onAdded)
    await flushPromises()
    expect(onAdded).toHaveBeenCalledOnce()
    expect(sent).toEqual(['POST /api/v1/admin/llm/credentials'])
    expect(state.error.value).toBeNull()
    expect(state.isAdding.value).toBe(false)
  })

  it("puts the provider's reason for refusing a key", async () => {
    const onAdded = vi.fn()
    const { state } = mountComposable(useAddCredential, routes({ 'POST /api/v1/admin/llm/credentials': () => problem(422, 'Unprocessable', 'invalid key') }))
    state.add({ provider: 'mistral', label: 'x', apiKey: '12345678' }, onAdded)
    await flushPromises()
    expect(onAdded).not.toHaveBeenCalled()
    expect(state.error.value).toBe('The provider refused this key. invalid key')
  })

  it('tests a key and describes the outcome, or why it could not', async () => {
    let answer = (): Response => json({ ok: false, at: '2026-10-23T09:30:00Z', message: 'invalid x-api-key' })
    const { state } = mountComposable(useCredentialTest, routes({ [`POST /api/v1/admin/llm/credentials/${STORED_ID}/test`]: () => answer() }))
    expect(state.outcome.value).toBeNull()
    expect(state.error.value).toBeNull()
    state.test(STORED_ID)
    await flushPromises()
    expect(state.outcome.value).toEqual({ ok: false, text: 'Refused on 23 Oct 2026, 09:30 UTC: invalid x-api-key' })
    answer = () => problem(422, 'Unprocessable')
    state.test(STORED_ID)
    await flushPromises()
    expect(state.error.value).toBe('This key can no longer be read. Remove it and add it again.')
    expect(state.isTesting.value).toBe(false)
  })

  it('removes a key, or says why not', async () => {
    let status = 204
    const onRemoved = vi.fn()
    const { state } = mountComposable(useRemoveCredential, routes({ [`DELETE /api/v1/admin/llm/credentials/${STORED_ID}`]: () => (status === 204 ? new Response(null, { status }) : problem(status, 'Not Found')) }))
    state.remove(STORED_ID, onRemoved)
    await flushPromises()
    expect(onRemoved).toHaveBeenCalledOnce()
    expect(state.error.value).toBeNull()
    status = 404
    state.remove(STORED_ID, onRemoved)
    await flushPromises()
    expect(state.error.value).toBe('That key no longer exists. Reload the page.')
    expect(state.isRemoving.value).toBe(false)
  })

  it('chooses a model and turns matching off, keeping the settings each answers with', async () => {
    const chosen = llmSettings({ choice: 'chosen', active: { credentialId: STORED_ID, model: 'mistralai/mistral-small' } })
    const off = llmSettings({ choice: 'off', active: undefined, matchingEnabled: false })
    const { state, sent, queryClient } = mountComposable(useActiveModel, routes({
      'PUT /api/v1/admin/llm/active': () => json(chosen),
      'DELETE /api/v1/admin/llm/active': () => json(off),
    }))
    state.save({ credentialId: STORED_ID, model: 'mistralai/mistral-small' })
    await flushPromises()
    expect(state.notice.value).toBe('Product matching now uses mistralai/mistral-small.')
    expect(queryClient.getQueryData<LlmSettings>(getLlmSettingsQueryKey())?.choice).toBe('chosen')
    state.turnOff()
    await flushPromises()
    expect(state.notice.value).toBe('Product matching is off.')
    expect(queryClient.getQueryData<LlmSettings>(getLlmSettingsQueryKey())?.choice).toBe('off')
    expect(sent).toEqual(['PUT /api/v1/admin/llm/active', 'DELETE /api/v1/admin/llm/active'])
    expect(state.error.value).toBeNull()
    expect(state.isSaving.value).toBe(false)
  })

  it('says a chosen model waits behind Haiku while an Anthropic key works', async () => {
    const haiku = { credentialId: 'vault-anthropic', model: 'claude-haiku-5-5' }
    const chosen = llmSettings({ haiku, choice: 'chosen', active: { credentialId: STORED_ID, model: 'mistralai/mistral-small' } })
    const { state } = mountComposable(useActiveModel, routes({ 'PUT /api/v1/admin/llm/active': () => json(chosen) }))
    state.save({ credentialId: STORED_ID, model: 'mistralai/mistral-small' })
    await flushPromises()
    expect(state.notice.value).toBe('Saved. Scans use mistralai/mistral-small when no Anthropic key works.')
  })

  it('says why a model could not be chosen or matching not turned off', async () => {
    const { state } = mountComposable(useActiveModel, routes({
      'PUT /api/v1/admin/llm/active': () => problem(422, 'Unprocessable'),
      'DELETE /api/v1/admin/llm/active': () => problem(500, 'Internal Server Error'),
    }))
    state.save({ credentialId: STORED_ID, model: 'x' })
    await flushPromises()
    expect(state.error.value).toBe('This key can no longer be read. Choose another key.')
    state.turnOff()
    await flushPromises()
    expect(state.error.value).toBe('Internal Server Error')
    expect(state.notice.value).toBeNull()
  })
})

describe('useReplaceCredential', () => {
  const ADD = 'POST /api/v1/admin/llm/credentials'
  const REMOVE = `DELETE /api/v1/admin/llm/credentials/${STORED_ID}`

  it('adds the new key, then removes the stored one it replaces', async () => {
    const bodies: unknown[] = []
    const { state, sent } = mountComposable(
      useReplaceCredential,
      routes({
        [ADD]: async (r) => (bodies.push(await r.json()), json(storedCredential({ id: OTHER_ID }), 201)),
        [REMOVE]: () => new Response(null, { status: 204 }),
        [SETTINGS]: () => json(llmSettings()),
      }),
    )
    let replaced = 0
    state.replace(storedCredential(), 'New key', 'sk-or-new', () => replaced++)
    expect(state.isReplacing.value).toBe(true)
    await flushPromises()
    expect(bodies).toEqual([{ provider: 'openrouter', label: 'New key', apiKey: 'sk-or-new' }])
    expect(sent.filter((r) => !r.startsWith('GET'))).toEqual([ADD, REMOVE])
    expect(replaced).toBe(1)
    expect(state.error.value).toBeNull()
  })

  it('keeps the Vault key, which Vault delivers', async () => {
    const { state, sent } = mountComposable(useReplaceCredential, routes({ [ADD]: () => json(storedCredential({ id: OTHER_ID, provider: 'anthropic' }), 201) }))
    let replaced = 0
    state.replace(vaultCredential(), 'Course key', 'sk-ant-new', () => replaced++)
    await flushPromises()
    expect(sent).toEqual([ADD])
    expect(replaced).toBe(1)
  })

  it('says why the new key was refused, or why the old one stayed', async () => {
    const refused = mountComposable(useReplaceCredential, routes({ [ADD]: () => problem(422, 'Unprocessable', 'Invalid key.') }))
    refused.state.replace(storedCredential(), 'New key', 'sk-or-refused-key', () => undefined)
    await flushPromises()
    expect(refused.state.error.value).toBe('The provider refused this key. Invalid key.')
    expect(refused.sent).toEqual([ADD])
    const stuck = mountComposable(
      useReplaceCredential,
      routes({ [ADD]: () => json(storedCredential({ id: OTHER_ID }), 201), [REMOVE]: () => problem(404, 'Not Found'), [SETTINGS]: () => json(llmSettings()) }),
    )
    stuck.state.replace(storedCredential(), 'New key', 'sk-or-new', () => undefined)
    await flushPromises()
    expect(stuck.state.error.value).toBe('That key no longer exists. Reload the page.')
  })
})

describe('useBudgetChange', () => {
  it('sets the budget and keeps the settings it answers with', async () => {
    const bodies: unknown[] = []
    const { state, queryClient } = mountComposable(
      useBudgetChange,
      routes({ 'PATCH /api/v1/admin/llm/budget': async (r) => (bodies.push(await r.json()), json(llmSettings())) }),
    )
    let saved = 0
    state.save(20, () => saved++)
    await flushPromises()
    expect(bodies).toEqual([{ monthlyBudgetUsd: 20 }])
    expect(saved).toBe(1)
    expect(queryClient.getQueryData(getLlmSettingsQueryKey())).toBeDefined()
    expect(state.error.value).toBeNull()
  })

  it('says why the budget was refused', async () => {
    const { state } = mountComposable(useBudgetChange, routes({ 'PATCH /api/v1/admin/llm/budget': () => problem(422, 'Unprocessable', 'Too high.') }))
    state.save(20, () => undefined)
    await flushPromises()
    expect(state.error.value).toBe('The budget was not saved. Too high.')
  })
})
