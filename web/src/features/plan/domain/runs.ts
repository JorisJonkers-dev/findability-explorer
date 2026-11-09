import type { RecommendationRun, RecommendationRuns } from '../../../infrastructure/api/types.gen'
import { zProblem } from '../../../infrastructure/api/zod.gen'
import { clock, longDate } from '../../../shared/dates'
import { describeError } from '../../../shared/describeError'
import { endingLabel } from '../../../shared/endings'
import { dollars } from '../../settings/domain/usage'
import { plural } from '../../../shared/numbers'

const NUMBER = new Intl.NumberFormat('en-GB')
const when = (iso: string | undefined) => (iso ? longDate(iso) : 'none yet')

/** What a run reads, and that it writes in place of the open recommendations. */
export function readsLine(r: RecommendationRuns): string {
  return `Reads the newest Prompt run (${when(r.promptRunAt)}), the tribelt.nl scan (${when(r.scanAt)}), and the search term list (${plural(r.terms, 'term')}). It writes new recommendations in place of the open ones that an earlier run wrote.`
}

/** How many open recommendations a run replaces, and what stays. */
export function replacesLine(n: number): string {
  const lead = n === 0 ? 'There are no open recommendations to replace.' : `This replaces ${plural(n, 'open recommendation')}.`
  return `${lead} Those you marked as done or edited stay.`
}

/** How many gaps a run writes, and what it should cost. */
export function estimateLine(r: Pick<RecommendationRuns, 'gaps' | 'estimatedCostUsd'>): string {
  if (r.gaps === 0) return 'The rules find no gaps to write now.'
  return `${plural(r.gaps, 'gap')} to write, about ${dollars(r.estimatedCostUsd)} from the Anthropic key.`
}

/** A past run as a row of the table. */
export function runRow(run: RecommendationRun) {
  return {
    when: `${longDate(run.startedAt)}, ${clock(run.startedAt)}`,
    ending: run.ending ? endingLabel(run.ending) : 'Running',
    model: run.model,
    added: NUMBER.format(run.added),
    replaced: NUMBER.format(run.replaced),
    kept: NUMBER.format(run.kept),
    cost: dollars(run.costUsd),
  }
}

/** A refused start or stop in words. */
export function runProblem(error: unknown): string {
  const problem = zProblem.safeParse(error)
  if (problem.success && problem.data.status === 403) return 'Only administrators can start a Recommendation run.'
  return describeError(error)
}
