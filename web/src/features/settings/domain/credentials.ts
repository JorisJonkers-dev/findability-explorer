import type { ActiveLlm, LlmCredential, LlmSettings, LlmTest } from '../../../infrastructure/api/types.gen'
import { providerLabel } from './providers'
import { atLabel } from './time'

/** Where a key comes from, for people: "Vault" or "Stored". */
export function sourceLabel(credential: Pick<LlmCredential, 'source'>): string {
  return credential.source === 'vault' ? 'Vault' : 'Stored'
}

/** How a key is named in a list or a select: "OpenRouter · Course budget key". */
export function credentialName(credential: Pick<LlmCredential, 'provider' | 'label'>): string {
  return `${credential.label} (${providerLabel(credential.provider)})`
}

/** Who added a key, and when: "Added by joris on 23 Oct 2026, 09:30 UTC", or Vault's line. */
export function originLabel(credential: Pick<LlmCredential, 'source' | 'createdBy' | 'createdAt'>): string {
  if (credential.source === 'vault') return 'Delivered by Vault in ANTHROPIC_API_KEY'
  const by = credential.createdBy ? ` by ${credential.createdBy}` : ''
  const on = credential.createdAt ? ` on ${atLabel(credential.createdAt)}` : ''
  return `Added${by}${on}`
}

/** Only a key stored here can be removed here; Vault's is managed in Vault. */
export function canRemove(credential: Pick<LlmCredential, 'source'>): boolean {
  return credential.source === 'stored'
}

export type TestOutcome = { ok: boolean; text: string }

/** A test's outcome in words: "Accepted on 23 Oct 2026, 09:30 UTC: The key may use 312 models." */
export function testOutcome(test: LlmTest): TestOutcome {
  return { ok: test.ok, text: `${test.ok ? 'Accepted' : 'Refused'} on ${atLabel(test.at)}: ${test.message}` }
}

/** The keys a model can be chosen for: those the server can still read, an Admin key never. */
export function choosableCredentials(credentials: readonly LlmCredential[]): LlmCredential[] {
  return credentials.filter((c) => c.readable && c.provider !== 'anthropic_admin')
}

export type ChosenState =
  | { kind: 'off' }
  | { kind: 'none' }
  | { kind: 'active'; model: string; credential: string; isDefault: boolean; working: boolean }

export type ActiveState = ChosenState | { kind: 'haiku'; model: string; credential: string; otherwise: ChosenState }

type ActiveSettings = Pick<LlmSettings, 'choice' | 'active' | 'haiku' | 'credentials'>

/** What product matching uses now, from the settings: Haiku on an Anthropic key while one works, else the chosen model. */
export function activeState(settings: ActiveSettings): ActiveState {
  const otherwise = chosenState(settings)
  const haiku = settings.haiku
  if (!haiku) return otherwise
  return { kind: 'haiku', model: haiku.model, credential: keyName(settings.credentials, haiku), otherwise }
}

/** The model an admin chose, or the default, which matches products when no Anthropic key works. */
function chosenState(settings: ActiveSettings): ChosenState {
  if (settings.choice === 'off') return { kind: 'off' }
  const active: ActiveLlm | undefined = settings.active
  if (!active) return { kind: 'none' }
  return {
    kind: 'active',
    model: active.model,
    credential: keyName(settings.credentials, active),
    isDefault: settings.choice === 'default',
    working: settings.credentials.some((c) => c.id === active.credentialId && c.readable),
  }
}

function keyName(credentials: readonly LlmCredential[], model: ActiveLlm): string {
  const credential = credentials.find((c) => c.id === model.credentialId)
  return credential ? credentialName(credential) : 'a key that no longer exists'
}

/** The contract's limits on a new key. */
export const MAX_LABEL_LENGTH = 100
export const MIN_KEY_LENGTH = 8
export const MAX_KEY_LENGTH = 500

export type NewKeyCheck = { field: 'label' | 'apiKey'; message: string } | null

/** Checks a key before it is sent; the provider checks the rest. */
export function checkNewKey(label: string, apiKey: string): NewKeyCheck {
  if (label.trim() === '') return { field: 'label', message: 'Give the key a name, to tell it apart from others.' }
  if (apiKey.trim().length < MIN_KEY_LENGTH) return { field: 'apiKey', message: 'Paste the whole API key.' }
  if (apiKey.trim().length > MAX_KEY_LENGTH) return { field: 'apiKey', message: `An API key has at most ${String(MAX_KEY_LENGTH)} characters.` }
  return null
}
