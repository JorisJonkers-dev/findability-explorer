import type { PlanImprovement, PlanTarget, PlanTargetValue, WatchedNumber } from '../../../infrastructure/api/types.gen'

/** The open recommendations no scan can check, so an administrator says when each is done. */
export function toTickOff<T extends Pick<PlanImprovement, 'auto' | 'done'>>(items: readonly T[]): T[] {
  return items.filter((i) => !i.auto && !i.done)
}

/** The recommendations an administrator marked as done, which Undo can open again. */
export function tickedOff<T extends Pick<PlanImprovement, 'auto' | 'done'>>(items: readonly T[]): T[] {
  return items.filter((i) => !i.auto && i.done)
}

/** Whether a goal is a share, typed as a percentage and stored as a fraction. */
function isShare(target: PlanTarget): boolean {
  switch (target) {
    case 'aiShareOfVoice':
    case 'twenteShareOfVoice':
    case 'citationShare':
      return true
    case 'readinessChecks':
    case 'termsUsed':
      return false
  }
}

/** The five goals in the order the form asks for them. */
const ORDER: readonly PlanTarget[] = ['aiShareOfVoice', 'twenteShareOfVoice', 'citationShare', 'readinessChecks', 'termsUsed']

/** A goal's label in the form: what it measures, and how to type it. */
export function goalLabel(target: PlanTarget, readinessChecks: number): string {
  switch (target) {
    case 'aiShareOfVoice':
      return 'AI responses that name Tribelt (percent)'
    case 'twenteShareOfVoice':
      return 'AI responses about the Twente region that name Tribelt (percent)'
    case 'citationShare':
      return 'Web responses that cite tribelt.nl (percent)'
    case 'readinessChecks':
      return `AI readiness checks passed (0 to ${String(readinessChecks)})`
    case 'termsUsed':
      return 'Search terms tribelt.nl uses'
  }
}

export type GoalField = { target: PlanTarget; label: string; value: string; max: number | undefined; step: number }

/** The goals as the form shows them: shares in percent, rounded to a tenth. */
export function goalFields(watched: readonly WatchedNumber[], readinessChecks: number): GoalField[] {
  return ORDER.flatMap((target) => {
    const w = watched.find((x) => x.target === target)
    if (!w) return []
    const share = isShare(target)
    const value = share ? String(Math.round(w.goal * 1000) / 10) : String(w.goal)
    const max = share ? 100 : target === 'readinessChecks' ? readinessChecks : w.max
    return [{ target, label: goalLabel(target, readinessChecks), value, max, step: share ? 0.1 : target === 'readinessChecks' ? 0.5 : 1 }]
  })
}

export type GoalCheck = { kind: 'ok'; values: PlanTargetValue[] } | { kind: 'invalid'; target: PlanTarget; message: string }

/** The typed goals, checked and turned back into what the server stores: shares as fractions. */
export function checkGoals(fields: readonly GoalField[]): GoalCheck {
  const values: PlanTargetValue[] = []
  for (const f of fields) {
    const n = Number(f.value.trim().replace(',', '.'))
    const top = f.max ?? Number.POSITIVE_INFINITY
    if (f.value.trim() === '' || !Number.isFinite(n) || n < 0 || n > top) {
      const range = f.max === undefined ? 'a number of 0 or more' : `a number from 0 to ${String(f.max)}`
      return { kind: 'invalid', target: f.target, message: `Type ${range} for ${f.label.replace(/ \(.*\)$/, '')}.` }
    }
    values.push({ target: f.target, value: isShare(f.target) ? Math.round(n * 10) / 1000 : n })
  }
  return { kind: 'ok', values }
}
