import type { LlmCredential, LlmProvider } from '../../../infrastructure/api/types.gen'

/** The providers with a key page of their own; the Admin key belongs on Anthropic's. */
export type KeyProvider = Exclude<LlmProvider, 'anthropic_admin'>

/** A key page: its name, what the key pays for, and the sentence under its title. */
export type KeyPage = { provider: KeyProvider; title: string; purpose: string; pays: string }

export const KEY_PAGES: Record<KeyProvider, KeyPage> = {
  openrouter: {
    provider: 'openrouter',
    title: 'OpenRouter key',
    purpose: 'The key that pays for the Prompt test.',
    pays: 'It pays for the Prompt test: every question asked of every model, from its own knowledge and with web search.',
  },
  anthropic: {
    provider: 'anthropic',
    title: 'Anthropic key',
    purpose: 'The key that pays for everything Claude does here.',
    pays: 'It pays for everything Claude does here: matching products in scans, reading responses, and the search term and candidate reviews.',
  },
  mistral: {
    provider: 'mistral',
    title: 'Mistral key',
    purpose: 'A key that can match products in scans.',
    pays: 'It pays for matching products in scans, when one of its models is chosen for product matching and no Anthropic key works.',
  },
}

/** The key page for a provider in an address, or undefined for one without a page. */
export function keyPage(provider: string): KeyPage | undefined {
  return provider === 'openrouter' || provider === 'anthropic' || provider === 'mistral' ? KEY_PAGES[provider] : undefined
}

/** A provider's keys, in the server's order: the Vault key first, then the stored ones oldest first. */
export function keysOf(credentials: readonly LlmCredential[], provider: LlmProvider): LlmCredential[] {
  return credentials.filter((c) => c.provider === provider)
}

/** Whether a key works, as its last test said: a key that cannot be read never does. */
export type KeyStatus = 'Works' | 'Refused' | 'Not tested' | 'Cannot be read'

export function keyStatus(credential: Pick<LlmCredential, 'readable' | 'lastTest'>): KeyStatus {
  if (!credential.readable) return 'Cannot be read'
  if (!credential.lastTest) return 'Not tested'
  return credential.lastTest.ok ? 'Works' : 'Refused'
}

/** A provider's status on its card: its first key's, or "No key" without one. */
export function providerStatus(credentials: readonly LlmCredential[], provider: LlmProvider): KeyStatus | 'No key' {
  const first = keysOf(credentials, provider)[0]
  return first ? keyStatus(first) : 'No key'
}
