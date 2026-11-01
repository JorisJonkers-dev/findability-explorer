import { describe, expect, it } from 'vitest'
import { definition } from '../../../testing/definition'
import { definitionLine, definitionProblem, outdatedLine, recheckLine, versionLine } from './definition'

const problemOf = (status: number, detail?: string) => ({ type: 'about:blank', title: 'Problem', status, ...(detail ? { detail } : {}) })

describe('the competitor definition in words', () => {
  it('says which version is in use and that saving starts no check', () => {
    expect(definitionLine(definition().current)).toBe('Version 3, saved 1 November 2026. Saving makes a new version and does not start a check.')
  })

  it('says what was checked against an older version, or nothing', () => {
    expect(outdatedLine(definition())).toBe('12 competitors and 40 candidates were checked against an older version of the definition.')
    expect(outdatedLine(definition({ outdatedCompetitors: 1, outdatedCandidates: 0 }))).toBe('1 competitor was checked against an older version of the definition.')
    expect(outdatedLine(definition({ outdatedCompetitors: 0, outdatedCandidates: 1 }))).toBe('1 candidate was checked against an older version of the definition.')
    expect(outdatedLine(definition({ outdatedCompetitors: 0, outdatedCandidates: 0 }))).toBeNull()
  })

  it('says what checking again should cost', () => {
    expect(recheckLine(definition())).toBe('52 checks, about $0.26 from the Anthropic key.')
    expect(recheckLine(definition({ outdatedCompetitors: 1, outdatedCandidates: 0, recheckCostUsd: 0.005 }))).toBe('1 check, about $0.005 from the Anthropic key.')
  })

  it('names who saved a version, and the one it restored', () => {
    expect(versionLine(definition().current)).toBe('Saved 1 November 2026 by Joris.')
    expect(versionLine({ ...definition().current, restoredFrom: 2 })).toBe('Saved 1 November 2026 by Joris. Restored from version 2.')
    expect(versionLine({ version: 2, text: 'x', savedAt: '2026-10-27T19:46:26Z', savedBy: 'import' })).toBe('In the code until 27 October 2026.')
  })

  it('says why a change was refused', () => {
    expect(definitionProblem(problemOf(403))).toBe('Only administrators can change the competitor definition.')
    expect(definitionProblem(problemOf(409, 'The competitor definition in use has this text already.'))).toBe('Nothing changed, so no new version was saved.')
    expect(definitionProblem(problemOf(422, 'A definition has 1 to 8000 characters.'))).toBe('A definition has 1 to 8000 characters.')
    expect(definitionProblem(new Error('offline'))).toBe('Something went wrong. Try again.')
  })
})
