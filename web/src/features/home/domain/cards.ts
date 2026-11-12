import type { Overview, Plan } from '../../../infrastructure/api/types.gen'
import { formatValue } from '../../catching-up/domain/standings'
import { section, type SectionId } from '../../shell/domain/sections'

/** A numbered tile of the Overview: one part of the app, its key figure, and the way in. */
export type SectionTile = { id: SectionId; number: string; label: string; big: string; text: string; path: string }

function tile(id: SectionId, big: string, text: string): SectionTile {
  const s = section(id)
  return { id, number: s.number, label: s.label, big, text, path: s.path }
}

/** The five tiles, one per part of the app, with their figures from the overview and the plan. */
export function sectionTiles(overview: Overview, plan: Plan): SectionTile[] {
  const named = overview.aiSearch.companies.find((c) => c.role === 'tribelt')?.share
  const score = plan.watched.find((w) => w.target === 'readinessChecks')?.now
  const open = plan.improvements.filter((i) => !i.done).length
  return [
    named === undefined ? tile('ai-responses', '–', 'No Prompt run has finished yet.') : tile('ai-responses', formatValue('aiShareOfVoice', named), 'of AI responses name Tribelt.'),
    score === undefined
      ? tile('ai-readiness', '–', 'tribelt.nl has not been checked for AI readiness yet.')
      : tile('ai-readiness', `${formatValue('readinessChecks', score)} of ${String(plan.readinessChecks)}`, 'AI readiness checks passed by tribelt.nl.'),
    tile('terms', formatValue('termsUsed', overview.terms.tribelt), `of the ${formatValue('termsUsed', overview.terms.total)} search terms appear on tribelt.nl.`),
    tile('competitors', formatValue('termsUsed', overview.competitors.makers), 'competitors make metal belts buyers could choose instead.'),
    tile('recommendations', String(open), 'changes to make on tribelt.nl, ranked by value.'),
  ]
}
