import type { LlmProvider } from '../../../infrastructure/api/types.gen'

const PROVIDER_LABELS: Record<LlmProvider, string> = {
  openrouter: 'OpenRouter',
  mistral: 'Mistral',
  anthropic: 'Anthropic (Claude)',
}

/** The providers in the order the add form offers them: OpenRouter first, as it reaches the most models. */
export const PROVIDER_OPTIONS: readonly { value: LlmProvider; label: string }[] = (
  ['openrouter', 'mistral', 'anthropic'] as const
).map((value) => ({ value, label: PROVIDER_LABELS[value] }))

/** A provider's name for people: "Anthropic (Claude)". */
export function providerLabel(provider: LlmProvider): string {
  return PROVIDER_LABELS[provider]
}
