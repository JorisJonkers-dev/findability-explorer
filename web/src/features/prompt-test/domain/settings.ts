import type {
  CompetitorCandidate,
  CompetitorCandidateCounts,
  CompetitorCandidateStatus,
  LlmCredential,
  PromptModelChoice,
  PromptPrices,
  PromptQuestion,
} from '../../../infrastructure/api/types.gen'
import { zProblem } from '../../../infrastructure/api/zod.gen'
import { describeError } from '../../../shared/describeError'
import { familyName, scopeName } from './results'

const MOMENT = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', timeZone: 'UTC' })

/** A moment as the settings write it: "1 Nov 2026, 06:00 UTC". */
export function momentLabel(iso: string): string {
  return `${MOMENT.format(new Date(iso))} UTC`
}

/** The keys Prompt runs can use: the readable OpenRouter keys. */
export function openRouterKeys(credentials: readonly LlmCredential[]): LlmCredential[] {
  return credentials.filter((c) => c.provider === 'openrouter' && c.readable)
}

/** A budget as typed, "$12.50" or "12,5": dollars, or null when it is no amount above 0 and at most $1,000. */
export function parseBudget(text: string): number | null {
  const n = Number(text.replace(/[$\s]/g, '').replace(',', '.'))
  return Number.isFinite(n) && n > 0 && n <= 1000 ? Math.round(n * 100) / 100 : null
}

/** The prompt set grouped by belt family, for reading: each family's prompts by scope. */
export function promptGroups(prompts: readonly PromptQuestion[]): { family: string; prompts: { scope: string; text: string; id: string }[] }[] {
  const out: { family: string; prompts: { scope: string; text: string; id: string }[] }[] = []
  for (const p of prompts) {
    const family = familyName(p.family)
    let group = out.find((g) => g.family === family)
    if (!group) {
      group = { family, prompts: [] }
      out.push(group)
    }
    group.prompts.push({ scope: scopeName(p.scope), text: p.text, id: p.id })
  }
  return out
}

export const CANDIDATE_STATUSES: readonly CompetitorCandidateStatus[] = ['pending', 'added', 'dismissed']

const CANDIDATE_LABELS: Record<CompetitorCandidateStatus, string> = { pending: 'To review', added: 'Added', dismissed: 'Dismissed' }

/** A status tab: "To review (12)". */
export function candidateTab(status: CompetitorCandidateStatus, counts: CompetitorCandidateCounts | undefined): string {
  return counts ? `${CANDIDATE_LABELS[status]} (${String(counts[status])})` : CANDIDATE_LABELS[status]
}

/** The tab in the address, or the pending one. */
export function parseCandidateStatus(raw: unknown): CompetitorCandidateStatus {
  return CANDIDATE_STATUSES.find((s) => s === raw) ?? 'pending'
}

/** The website to add a candidate with: the first address the answers gave, or none. */
export function candidateWebsite(candidate: Pick<CompetitorCandidate, 'addresses'>): string {
  const first = candidate.addresses.find((a) => /^https?:\/\//.test(a))
  if (!first) return ''
  const url = new URL(first)
  return `${url.protocol}//${url.host}/`
}

/** A failed request's problem in words; the server's detail when it gives one. */
export function promptProblem(error: unknown): string {
  const problem = zProblem.safeParse(error)
  if (!problem.success) return describeError(error)
  if (problem.data.status === 403) return 'Only administrators can do this.'
  return problem.data.detail ?? describeError(error)
}

const PROVIDER_ID = /^[a-z0-9][a-z0-9-]{0,39}$/

/** The provider list with one more, or why it cannot take it. */
export function addProvider(list: readonly string[], raw: string): { list: string[]; problem: string | null } {
  const id = raw.trim().toLowerCase()
  if (!PROVIDER_ID.test(id)) return { list: [...list], problem: 'A provider is an OpenRouter id, such as openai or meta-llama.' }
  if (list.includes(id)) return { list: [...list], problem: `${id} is on already.` }
  if (list.length >= 20) return { list: [...list], problem: 'The Prompt test asks at most 20 providers.' }
  return { list: [...list, id], problem: null }
}

/** The model a provider asked in the latest run, and its fallback, in words. */
export function modelLine(models: readonly PromptModelChoice[], provider: string): string {
  const m = models.find((x) => x.provider === provider)
  if (!m) return 'Cheapest model chosen at the next run'
  return m.fallback ? `${m.model}, then ${m.fallback}` : m.model
}

/** Where the prices come from, in words. */
export function basisLine(prices: Pick<PromptPrices, 'basis'>): string {
  switch (prices.basis) {
    case 'run':
      return 'From what the latest run cost.'
    case 'catalogue':
      return 'An estimate from OpenRouter’s prices, until the first run.'
    case 'none':
      return 'No prices yet. They follow once the key works or a run has finished.'
  }
}
