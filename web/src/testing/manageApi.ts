import type { DOMWrapper, VueWrapper } from '@vue/test-utils'
import type { Competitor } from '../infrastructure/api/types.gen'
import { llmSettings } from '../features/settings/fixtures'
import { claudeStep, stepVersion } from './claudeStepFixtures'
import { me, readinessTable } from './fixtures'
import { manageCounts } from './manageFixtures'
import { type Handler, json, routes } from './mountApp'

/**
 * An administrator's server for a Manage page: the menu's counts, the companies, an empty term
 * map, the language-model settings, and an empty AI readiness table, each replaceable by extra.
 */
export function manageApi(extra: Record<string, Handler> = {}, companies: Competitor[] = []): Handler {
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    ...manageCounts(),
    'GET /api/v1/competitors': () => json({ items: companies }),
    'GET /api/v1/terms': () => json({ items: [], total: 0, facets: { languages: [], tiers: [], families: [] } }),
    'GET /api/v1/admin/llm': () => json(llmSettings()),
    'GET /api/v1/ai-readiness': () => json(readinessTable([])),
    // Matching products on the instructions it came with, on Claude Haiku 5.5.
    'GET /api/v1/admin/claude-steps/matching': () =>
      json(claudeStep({ current: stepVersion({ number: 1, model: 'claude-haiku-5-5', modelName: 'Claude Haiku 5.5', createdBy: 'the tool' }), jobCostUsd: 0.0044 })),
    ...extra,
  })
}

/** A 204, as a stop or a removal answers. */
export const noContent = (): Response => new Response(null, { status: 204 })

/** The button with exactly this text, or an error naming it. */
export function buttonNamed(wrapper: Pick<VueWrapper, 'findAll'>, text: string): DOMWrapper<HTMLButtonElement> {
  const found = wrapper.findAll('button').find((b) => b.text() === text)
  if (!found) throw new Error(`no button "${text}"`)
  return found
}
