import type { PlanPart } from '../../../infrastructure/api/types.gen'
import { joinWords } from '../../../shared/words'

/** What the latest scan of tribelt.nl found of an improvement it checks, and what it did not. */
export function foundLine(parts: readonly PlanPart[]): string {
  const found = parts.filter((p) => p.present).map((p) => p.name)
  const missing = parts.filter((p) => !p.present).map((p) => p.name)
  if (found.length === 0 && missing.length === 0) return ''
  if (found.length === 0) return `The latest scan did not find ${joinWords(missing, 'or')}.`
  if (missing.length === 0) return `The latest scan found ${joinWords(found)}.`
  return `The latest scan found ${joinWords(found)}, but not ${joinWords(missing, 'or')}.`
}
