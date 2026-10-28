import type { Overview, Plan } from '../../../infrastructure/api/types.gen'
import type { Section, SectionId } from '../../shell/domain/sections'
import { section } from '../../shell/domain/sections'

/** One of the section cards under the numbers: a figure from that section and the way in. */
export type SectionCard = { id: SectionId; number: string; label: string; big: string; text: string; cta: string; path: string }

function card(id: SectionId, big: string, text: string): SectionCard {
  const s: Section = section(id)
  return { id, number: s.number, label: s.label.toUpperCase(), big, text, cta: `Open ${s.label}`, path: s.path }
}

/** A title as the middle of a sentence: "Submit tribelt.nl" becomes "submit tribelt.nl". */
export function lowerFirst(text: string): string {
  return text.charAt(0).toLowerCase() + text.slice(1)
}

/** The four section cards, with their figures from the overview and the plan. */
export function sectionCards(overview: Overview, plan: Plan): SectionCard[] {
  const named = overview.aiSearch.companies.find((c) => c.role === 'tribelt')?.share
  const first = plan.improvements[0]
  return [
    named === undefined
      ? card('ai-search', '–', 'No Prompt run has finished yet. Open it to see how AI answers name Tribelt.')
      : card('ai-search', `${String(Math.round(named * 100))}%`, 'of AI answers name Tribelt. Open it to see who they name instead.'),
    card('competitors', String(overview.competitors.makers), 'competitors on the list. Open one to see how AI answers name it.'),
    card('terms', String(overview.terms.tribelt), `of the ${String(overview.terms.total)} terms buyers search with appear on tribelt.nl.`),
    card(
      'recommendations',
      String(plan.improvements.length),
      first ? `changes to tribelt.nl, ranked. The first is to ${lowerFirst(first.title)}.` : 'changes to tribelt.nl, ranked.',
    ),
  ]
}
