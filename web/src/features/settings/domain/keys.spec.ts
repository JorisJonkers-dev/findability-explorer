import { describe, expect, it } from 'vitest'
import { storedCredential, vaultCredential } from '../fixtures'
import { KEY_PAGES, keyPage, keysOf, keyStatus, providerStatus } from './keys'

describe('the key pages', () => {
  it('has a page per provider, and none for the Admin key or an unknown one', () => {
    expect(keyPage('openrouter')).toBe(KEY_PAGES.openrouter)
    expect(keyPage('anthropic')?.title).toBe('Anthropic key')
    expect(keyPage('mistral')?.title).toBe('Mistral key')
    expect(keyPage('anthropic_admin')).toBeUndefined()
    expect(keyPage('openai')).toBeUndefined()
  })

  it('lists a provider’s keys, and says whether each works', () => {
    const credentials = [vaultCredential(), storedCredential(), storedCredential({ id: 'x', readable: false })]
    expect(keysOf(credentials, 'openrouter')).toHaveLength(2)
    expect(keyStatus(vaultCredential())).toBe('Works')
    expect(keyStatus(storedCredential())).toBe('Not tested')
    expect(keyStatus(storedCredential({ lastTest: { ok: false, at: '2026-10-23T09:30:00Z', message: 'Refused.' } }))).toBe('Refused')
    expect(keyStatus(storedCredential({ readable: false }))).toBe('Cannot be read')
    expect(providerStatus(credentials, 'anthropic')).toBe('Works')
    expect(providerStatus(credentials, 'mistral')).toBe('No key')
  })
})
