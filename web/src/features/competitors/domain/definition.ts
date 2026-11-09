import type { CompetitorDefinition, CompetitorDefinitionVersion } from '../../../infrastructure/api/types.gen'
import { zProblem } from '../../../infrastructure/api/zod.gen'
import { longDate } from '../../../shared/dates'
import { describeError } from '../../../shared/describeError'
import { dollars } from '../../settings/domain/usage'
import { plural } from '../../../shared/numbers'


/** "Version 3, saved 1 November 2026. Saving makes a new version and does not start a check." */
export function definitionLine(d: CompetitorDefinitionVersion): string {
  return `Version ${String(d.version)}, saved ${longDate(d.savedAt)}. Saving makes a new version and does not start a check.`
}

/** What was checked against an older version, or null when every check used the one in use. */
export function outdatedLine(d: CompetitorDefinition): string | null {
  const parts = [
    d.outdatedCompetitors > 0 ? plural(d.outdatedCompetitors, 'competitor') : '',
    d.outdatedCandidates > 0 ? plural(d.outdatedCandidates, 'candidate') : '',
  ].filter((p) => p !== '')
  if (parts.length === 0) return null
  const verb = d.outdatedCompetitors + d.outdatedCandidates === 1 ? 'was' : 'were'
  return `${parts.join(' and ')} ${verb} checked against an older version of the definition.`
}

/** "52 checks, about $0.26 from the Anthropic key." */
export function recheckLine(d: CompetitorDefinition): string {
  return `${plural(d.outdatedCompetitors + d.outdatedCandidates, 'check')}, about ${dollars(d.recheckCostUsd)} from the Anthropic key.`
}

/** Who saved a version and when, and the version it restored. */
export function versionLine(v: CompetitorDefinitionVersion): string {
  if (v.savedBy === 'import') return `In the code until ${longDate(v.savedAt)}.`
  const restored = v.restoredFrom === undefined ? '' : ` Restored from version ${String(v.restoredFrom)}.`
  return `Saved ${longDate(v.savedAt)} by ${v.savedBy}.${restored}`
}

/** A refused change in words. */
export function definitionProblem(error: unknown): string {
  const problem = zProblem.safeParse(error)
  if (problem.success && problem.data.status === 403) return 'Only administrators can change the competitor definition.'
  if (problem.success && problem.data.status === 409) return 'Nothing changed, so no new version was saved.'
  return describeError(error)
}
