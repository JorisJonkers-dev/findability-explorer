import { describe, expect, it } from 'vitest'
import { llmSettings, storedCredential, vaultCredential } from '../fixtures'
import { activeState, canRemove, checkNewKey, choosableCredentials, credentialName, originLabel, sourceLabel, testOutcome } from './credentials'

describe('a credential', () => {
  it('says where it comes from and who added it', () => {
    expect(sourceLabel(vaultCredential())).toBe('Vault')
    expect(sourceLabel(storedCredential())).toBe('Stored')
    expect(originLabel(vaultCredential())).toBe('Delivered by Vault in ANTHROPIC_API_KEY')
    expect(originLabel(storedCredential())).toBe('Added by joris on 22 Oct 2026, 14:05 UTC')
    expect(originLabel({ source: 'stored' })).toBe('Added')
  })

  it('is named by provider and label, and only a stored one can be removed', () => {
    expect(credentialName(storedCredential())).toBe('OpenRouter · Course budget key')
    expect(canRemove(storedCredential())).toBe(true)
    expect(canRemove(vaultCredential())).toBe(false)
  })

  it('describes a test either way', () => {
    expect(testOutcome({ ok: true, at: '2026-10-23T09:30:00Z', message: 'The key may use 312 models.' })).toEqual({
      ok: true,
      text: 'Accepted on 23 Oct 2026, 09:30 UTC: The key may use 312 models.',
    })
    expect(testOutcome({ ok: false, at: '2026-10-23T09:30:00Z', message: 'invalid x-api-key' }).text).toBe(
      'Refused on 23 Oct 2026, 09:30 UTC: invalid x-api-key',
    )
  })

  it('offers only readable keys for a model', () => {
    const unreadable = storedCredential({ readable: false })
    expect(choosableCredentials([vaultCredential(), unreadable]).map((c) => c.id)).toEqual(['vault-anthropic'])
    expect(choosableCredentials([storedCredential({ provider: 'anthropic_admin' })])).toEqual([])
  })
})

describe('the active model', () => {
  it('is off, missing, or a model on a key', () => {
    expect(activeState(llmSettings({ choice: 'off', active: undefined }))).toEqual({ kind: 'off' })
    expect(activeState(llmSettings({ active: undefined }))).toEqual({ kind: 'none' })
    expect(activeState(llmSettings())).toEqual({
      kind: 'active',
      model: 'claude-haiku-5-5',
      credential: 'Anthropic (Claude) · Vault key',
      isDefault: true,
      working: true,
    })
  })

  it('says when its key is gone or the choice was made by hand', () => {
    const state = activeState(llmSettings({ choice: 'chosen', matchingEnabled: false, active: { credentialId: 'vault-anthropic', model: 'm' }, credentials: [] }))
    expect(state).toEqual({ kind: 'active', model: 'm', credential: 'a key that no longer exists', isDefault: false, working: false })
  })
})

describe('a new key', () => {
  it('needs a name and the whole key', () => {
    expect(checkNewKey(' ', 'sk-or-v1-0000')).toEqual({ field: 'label', message: 'Give the key a name, to tell it apart from others.' })
    expect(checkNewKey('Course', 'short')).toEqual({ field: 'apiKey', message: 'Paste the whole API key.' })
    expect(checkNewKey('Course', 'x'.repeat(501))?.message).toBe('An API key has at most 500 characters.')
    expect(checkNewKey('Course', 'sk-or-v1-0000')).toBeNull()
  })
})
