import { describe, expect, it } from 'vitest'
import { dollars, usageLines } from './usage'

describe('a key’s usage', () => {
  it('reads OpenRouter’s figures', () => {
    expect(
      usageLines({
        source: 'openrouter', message: 'x', models: [], totalUsd: 25.5, dailyUsd: 1.2, weeklyUsd: 5.6, monthlyUsd: 24.5,
        limitUsd: 100, limitRemainingUsd: 75.5, limitReset: 'monthly', freeRequests: { used: 10, limit: 1000, remaining: 990 },
      }),
    ).toEqual([
      '$1.20 today · $5.60 this week · $24.50 this month · $25.50 in all',
      'Limit $100.00 a month, $75.50 left.',
      'Free models: 10 of 1,000 requests today.',
    ])
    expect(usageLines({ source: 'openrouter', message: 'x', models: [], totalUsd: 0.0042 })).toEqual(['$0.0042 in all', 'No spending limit on this key.'])
    expect(usageLines({ source: 'openrouter', message: 'x', models: [], limitUsd: 5, limitReset: 'daily' })[1]).toBe('Limit $5.00 a day, $0.00 left.')
    expect(usageLines({ source: 'openrouter', message: 'x', models: [], limitUsd: 5, limitRemainingUsd: 1 })[1]).toBe('Limit $5.00, $1.00 left.')
  })

  it('reads Anthropic’s report, and nothing where there is none', () => {
    expect(
      usageLines({
        source: 'anthropic', message: 'x', since: '2026-10-01T00:00:00Z', organisationCostUsd: 2,
        models: [{ model: 'claude-haiku-5-5', inputTokens: 120000, outputTokens: 30000, cacheReadTokens: 0 }],
      }),
    ).toEqual(['Tokens since 1 Oct 2026:', 'claude-haiku-5-5: 120,000 in, 30,000 out, 0 from the cache', "The organisation's cost this month: $2.00."])
    expect(usageLines({ source: 'anthropic', message: 'x', models: [] })).toEqual(['No tokens.'])
    expect(usageLines({ source: 'none', message: 'Mistral reports nothing.', models: [] })).toEqual([])
    expect(dollars(10)).toBe('$10.00')
  })
})
