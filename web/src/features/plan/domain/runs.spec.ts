import { describe, expect, it } from 'vitest'
import type { RecommendationRun } from '../../../infrastructure/api/types.gen'
import { recommendationRuns } from '../../../testing/recommendationRuns'
import { estimateLine, readsLine, replacesLine, runProblem, runRow } from './runs'

const problemOf = (status: number, detail?: string) => ({ type: 'about:blank', title: 'Problem', status, ...(detail ? { detail } : {}) })

describe('the Recommendation run in words', () => {
  it('says what a run reads', () => {
    expect(readsLine(recommendationRuns())).toBe(
      'Reads the newest Prompt run (1 November 2026), the tribelt.nl scan (31 October 2026), and the search term list (420 terms). It writes new recommendations in place of the open ones that an earlier run wrote.',
    )
    expect(readsLine(recommendationRuns({ promptRunAt: undefined, scanAt: undefined, terms: 1 }))).toBe(
      'Reads the newest Prompt run (none yet), the tribelt.nl scan (none yet), and the search term list (1 term). It writes new recommendations in place of the open ones that an earlier run wrote.',
    )
  })

  it('says how many open recommendations it replaces', () => {
    expect(replacesLine(12)).toBe('This replaces 12 open recommendations. Those you marked as done or edited stay.')
    expect(replacesLine(1)).toBe('This replaces 1 open recommendation. Those you marked as done or edited stay.')
    expect(replacesLine(0)).toBe('There are no open recommendations to replace. Those you marked as done or edited stay.')
  })

  it('says what it should cost', () => {
    expect(estimateLine(recommendationRuns())).toBe('40 gaps to write, about $0.07 from the Anthropic key.')
    expect(estimateLine(recommendationRuns({ gaps: 0, estimatedCostUsd: 0 }))).toBe('The rules find no gaps to write now.')
  })

  it('writes a past run as a row', () => {
    const [done, stopped] = recommendationRuns().runs as [RecommendationRun, RecommendationRun]
    expect(runRow(done)).toEqual({ when: '1 November 2026, 08:00', ending: 'Done', model: 'claude-haiku-5-5', added: '38', replaced: '10', kept: '2', cost: '$0.06' })
    expect(runRow(stopped).ending).toBe('Stopped by hand')
    expect(runRow({ ...done, ending: undefined, finishedAt: undefined }).ending).toBe('Running')
    expect(runRow({ ...done, ending: 'doneWithFailures' }).ending).toBe('Done, with failures')
  })

  it('says why a run could not start', () => {
    expect(runProblem(problemOf(403))).toBe('Only administrators can start a Recommendation run.')
    expect(runProblem(problemOf(409, 'A Recommendation run is running already.'))).toBe('A Recommendation run is running already.')
    expect(runProblem(new Error('offline'))).toBe('Something went wrong. Try again.')
  })
})
