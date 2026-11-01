import type { CompetitorCandidateList, TermCandidateList } from '../infrastructure/api/types.gen'
import { promptResults, promptRun, summary } from './fixtures'
import { type Handler, json } from './mountApp'

/** What the Manage menu counts: terms and candidates waiting, Prompt runs and scans running. */
export type ManageCounts = { terms: number; candidates: number; runs: number; scans: number }

/** The API answers the Manage menu's counts come from, as a table for routes(). */
export function manageCounts(counts: Partial<ManageCounts> = {}): Record<string, Handler> {
  const { terms = 0, candidates = 0, runs = 0, scans = 0 } = counts
  return {
    'GET /api/v1/admin/term-candidates': () =>
      json({ items: [], counts: { pending: terms, accepted: 0, rejected: 0 }, review: { on: true, running: false, waiting: 0, monthTokens: 0, monthCostUsd: 0 } } satisfies TermCandidateList),
    'GET /api/v1/admin/competitor-candidates': () =>
      json({
        items: [],
        counts: { pending: candidates, added: 0, dismissed: 0, unchecked: 0 },
        review: { running: false },
      } satisfies CompetitorCandidateList),
    'GET /api/v1/prompt-test': () =>
      json(promptResults({ runs: [...Array.from({ length: runs }, (_, i) => promptRun({ id: `00000000-0000-4000-8000-${String(i).padStart(12, '0')}`, status: 'running' })), promptRun()] })),
    'GET /api/v1/analyses': () =>
      json({ items: [...Array.from({ length: scans }, (_, i) => summary({ id: `00000000-0000-4000-9000-${String(i).padStart(12, '0')}`, status: i % 2 ? 'queued' : 'running' })), summary()] }),
  }
}
