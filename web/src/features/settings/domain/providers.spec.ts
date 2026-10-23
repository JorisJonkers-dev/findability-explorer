import { describe, expect, it } from 'vitest'
import { PROVIDER_OPTIONS, providerLabel } from './providers'
import { atLabel } from './time'

describe('the providers', () => {
  it('offers OpenRouter, Mistral and Anthropic, in that order', () => {
    expect(PROVIDER_OPTIONS.map((p) => p.label)).toEqual(['OpenRouter', 'Mistral', 'Anthropic (Claude)'])
    expect(providerLabel('anthropic')).toBe('Anthropic (Claude)')
  })

  it('writes moments in UTC', () => {
    expect(atLabel('2026-10-23T09:30:00Z')).toBe('23 Oct 2026, 09:30 UTC')
  })
})
