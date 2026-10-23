import { type Phase, PHASES, type PhaseTone } from '../../plan/domain/content'

/** The poster's own words, as the approved design has them. */
export const EYEBROW = 'Digital strategy · Tribelt B.V. · 2026'
export const LEAD =
  'Buyers of metal conveyor belts search with all of them. This strategy makes sure the right new buyers in the Netherlands and Germany find Tribelt, on Google and in Gemini.'
export const COURSE_LINE = 'A student project for Tribelt B.V. in Digital Strategy (201900119)'
/** Still open in the report itself: the group's names and the supervisor fill these in. */
export const TEAM_LINE = '[GROUP MEMBERS] · Supervisor: [SUPERVISOR] · October 2026'

export const TILE_TEXT = {
  buyers: { big: '43×', text: 'Organic visitors contact Tribelt 43 times more often than paid ones.' },
  aiSearch: { big: '0', text: 'Googlebot visits to our mirror in more than three weeks. AI crawlers read every page.' },
  competitors: 'makers sell the same belts in Europe, each under its own names.',
  plan: { big: '12', text: 'months in three phases, with a decision at the end of each.' },
} as const

/** The phases on the plan tile's twelve-month bar, by their short names. */
const SHORT_PHASE_NAMES = ['Measure', 'Build', 'Expand and learn'] as const

export const PLAN_BARS: readonly { label: string; start: number; span: number; tone: PhaseTone }[] = SHORT_PHASE_NAMES.map((label, index) => {
  const phase = PHASES[index] as Phase
  return { label, start: phase.start, span: phase.span, tone: phase.tone }
})
