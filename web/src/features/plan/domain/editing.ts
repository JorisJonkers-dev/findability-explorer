import type { PlanImprovement, RecommendationEdit } from '../../../infrastructure/api/types.gen'
import { zProblem } from '../../../infrastructure/api/zod.gen'
import { longDate } from '../../../shared/dates'
import { describeError } from '../../../shared/describeError'

/** The form for one recommendation a run wrote: the steps one to a line. */
export type Draft = { title: string; where: string; why: string; steps: string; doneWhen: string; benefit: number; effort: number }

/** The recommendations a run wrote, which an administrator can edit, in the server's order. */
export function editable(items: readonly PlanImprovement[]): PlanImprovement[] {
  return items.filter((i) => i.generated)
}

/** The form filled from a recommendation. */
export function draftOf(it: PlanImprovement): Draft {
  return { title: it.title, where: it.where ?? '', why: it.why, steps: it.steps.join('\n'), doneWhen: it.doneWhen, benefit: it.benefit, effort: it.effort }
}

/** The form as the edit the server takes: blank lines and a blank page left out. */
export function toEdit(d: Draft): RecommendationEdit {
  const steps = d.steps
    .split('\n')
    .map((s) => s.trim())
    .filter((s) => s !== '')
  const where = d.where.trim()
  return { title: d.title.trim(), ...(where === '' ? {} : { where }), why: d.why.trim(), steps, doneWhen: d.doneWhen.trim(), benefit: d.benefit, effort: d.effort }
}

/** Whether a recommendation was edited, and what a run then does with it. */
export function editedLine(it: Pick<PlanImprovement, 'editedAt'>): string {
  return it.editedAt
    ? `Edited on ${longDate(it.editedAt)}. A new Recommendation run keeps it.`
    : 'Not edited yet. A new Recommendation run replaces it unless it is edited or done.'
}

/** A refused edit in words. */
export function editProblem(error: unknown): string {
  const problem = zProblem.safeParse(error)
  if (problem.success && problem.data.status === 403) return 'Only administrators can edit recommendations.'
  return describeError(error)
}
