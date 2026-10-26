import type { LlmCredential, LlmModel, LlmSettings } from '../../infrastructure/api/types.gen'

export const STORED_ID = '0b9e6a4c-1d2f-4a3b-8c5d-6e7f8a9b0c1d'
export const OTHER_ID = '7f3e2d1c-0b9a-4876-9543-210fedcba987'

/** The key Vault delivers. */
export function vaultCredential(overrides: Partial<LlmCredential> = {}): LlmCredential {
  return {
    id: 'vault-anthropic',
    provider: 'anthropic',
    label: 'Vault key',
    source: 'vault',
    fingerprint: '3a7bd3e2360a',
    readable: true,
    lastTest: { ok: true, at: '2026-10-23T09:30:00Z', message: 'The key may use 9 models.' },
    ...overrides,
  }
}

/** A key an administrator added. */
export function storedCredential(overrides: Partial<LlmCredential> = {}): LlmCredential {
  return {
    id: STORED_ID,
    provider: 'openrouter',
    label: 'Course budget key',
    source: 'stored',
    fingerprint: '9f86d081884c',
    readable: true,
    createdBy: 'joris',
    createdAt: '2026-10-22T14:05:00Z',
    ...overrides,
  }
}

export function llmSettings(overrides: Partial<LlmSettings> = {}): LlmSettings {
  return {
    credentials: [vaultCredential(), storedCredential()],
    choice: 'default',
    active: { credentialId: 'vault-anthropic', model: 'claude-haiku-5-5' },
    matchingEnabled: true,
    budget: { monthlyTokens: 2_000_000, usedThisMonth: 48_211, analysesPerHour: 10, averageTokensPerScan: 29_137, scansMatched: 2 },
    events: [
      { at: '2026-10-22T14:05:00Z', actor: 'joris', action: 'credential_added', ok: true, message: 'Added openrouter key "Course budget key" (9f86d081884c).' },
      { at: '2026-10-23T09:30:00Z', actor: 'joris', action: 'credential_tested', ok: true, message: 'Tested anthropic key "Vault key": accepted.' },
    ],
    ...overrides,
  }
}

export function models(): LlmModel[] {
  return [
    { id: 'anthropic/claude-haiku-5-5', name: 'Anthropic: Claude Haiku 5.5' },
    { id: 'mistralai/mistral-small', name: 'Mistral: Small' },
    { id: 'openai/gpt-5-mini', name: 'openai/gpt-5-mini' },
  ]
}

// The response helpers, as testing/mountApp.ts has them. They are repeated here so the composable
// tests do not load the whole app's router.
type Handler = (request: Request) => Response | Promise<Response>

export function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

export function problem(status: number, title: string, detail?: string): Response {
  return new Response(JSON.stringify({ type: 'about:blank', title, status, ...(detail ? { detail } : {}) }), {
    status,
    headers: { 'Content-Type': 'application/problem+json' },
  })
}

export function routes(table: Record<string, Handler>): Handler {
  return (request) => {
    const handler = table[`${request.method} ${new URL(request.url).pathname}`]
    return handler ? handler(request) : problem(404, 'Not Found')
  }
}
