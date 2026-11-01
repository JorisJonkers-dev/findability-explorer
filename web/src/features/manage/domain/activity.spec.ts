import { describe, expect, it } from 'vitest'
import type { Work } from '../../../infrastructure/api/types.gen'
import { promptRun, summary } from '../../../testing/fixtures'
import {
  costOf,
  dayLabel,
  didLabel,
  endedOf,
  lastEnded,
  missingKeys,
  needsFinishing,
  queuedScans,
  recentWork,
  runAsWork,
  runningWork,
  spendingRows,
  timeLabel,
  waitLine,
  whenLabel,
} from './activity'

const HAND = 'Stopped by hand. The responses so far are kept, and a retry finishes the run.'
const BUDGET = "The month's budget for Prompt runs was reached; the responses so far are kept."

function work(overrides: Partial<Work> = {}): Work {
  return { id: 'w1', kind: 'promptRun', startedAt: '2026-10-28T09:00:00Z', done: 0, total: 0, failed: 0, openRouterCostUsd: 0, anthropicCostUsd: 0, ...overrides }
}

describe('endedOf', () => {
  it('reads a done run as done, or done with failures when some responses failed', () => {
    expect(endedOf(promptRun({ status: 'done', failed: 0 }))).toBe('done')
    expect(endedOf(promptRun({ status: 'done', failed: 2 }))).toBe('doneWithFailures')
  })

  it('tells a run stopped by hand from one stopped by the budget', () => {
    expect(endedOf(promptRun({ status: 'stopped', failure: HAND }))).toBe('stoppedByHand')
    expect(endedOf(promptRun({ status: 'stopped', failure: BUDGET }))).toBe('stoppedByBudget')
    expect(endedOf(promptRun({ status: 'stopped', failure: undefined }))).toBe('stoppedByBudget')
  })

  it('reads a failed run as failed, and a running one as not ended', () => {
    expect(endedOf(promptRun({ status: 'failed', failure: 'No key.' }))).toBe('failed')
    expect(endedOf(promptRun({ status: 'running' }))).toBeNull()
  })
})

describe('moments', () => {
  it('writes a moment, a day, and a time in words, in UTC', () => {
    expect(whenLabel('2026-10-28T09:37:00Z')).toBe('28 October 2026, 09:37')
    expect(dayLabel('2026-10-27T06:03:00Z')).toBe('27 October 2026')
    expect(timeLabel('2026-10-27T06:03:00Z')).toBe('06:03')
  })
})

describe('what a piece of work did and cost', () => {
  it('counts its parts, of the total when known, with those that failed', () => {
    expect(didLabel(work({ done: 200, total: 200 }))).toBe('200 of 200 responses')
    expect(didLabel(work({ kind: 'termReview', done: 1240, total: 0 }))).toBe('1,240 search terms')
    expect(didLabel(work({ kind: 'readinessCheck', done: 65, total: 65, failed: 2 }))).toBe('65 of 65 sites, 2 failed')
  })

  it('adds what both keys paid', () => {
    expect(costOf(work({ openRouterCostUsd: 0.1, anthropicCostUsd: 0.2 }))).toBe(0.3)
  })
})

describe('recentWork', () => {
  it('lists the ended work with how each ended, what it did, its cost, and where to open it', () => {
    const lines = recentWork([
      work({ id: 'r1', ending: 'done', done: 200, total: 200, finishedAt: '2026-10-28T09:37:00Z', openRouterCostUsd: 1.12 }),
      work({ id: 's1', kind: 'scan', ending: 'stoppedByHand', done: 12, total: 60, anthropicCostUsd: 0.02 }),
      work({ id: 'x', kind: 'rescan' }),
    ])
    expect(lines).toEqual([
      { id: 'r1', at: '2026-10-28T09:37:00Z', work: 'Prompt run', ended: 'done', did: '200 of 200 responses', costUsd: 1.12, to: '/manage/prompt-test/runs' },
      { id: 's1', at: '2026-10-28T09:00:00Z', work: 'Scan', ended: 'stoppedByHand', did: '12 of 60 pages', costUsd: 0.02, to: '/manage/scans/s1' },
    ])
  })

  it('links every kind of work to where it is managed', () => {
    const kinds = ['rescan', 'readinessCheck', 'termReview', 'candidateReview', 'competitorCheck', 'recommendationRun'] as const
    expect(recentWork(kinds.map((kind) => work({ kind, ending: 'done' }))).map((l) => [l.work, l.to])).toEqual([
      ['Rescan of competitors', '/manage/scans'],
      ['AI readiness check', '/ai-readiness'],
      ['Term review', '/manage/terms'],
      ['Review of the competitor candidates', '/manage/competitors/candidates'],
      ['Check of the competitor list', '/manage/competitors'],
      ['Recommendation run', '/manage/recommendations'],
    ])
  })
})

describe('runningWork', () => {
  it('gives each piece its progress, cost so far, and Stop, and a Prompt run its cost to go', () => {
    const [run, review, rec] = runningWork(
      [
        work({ id: 'r1', done: 100, total: 400, openRouterCostUsd: 0.1 }),
        work({ id: 't1', kind: 'termReview', done: 12, total: 40, anthropicCostUsd: 0.01 }),
        work({ id: 'rec', kind: 'recommendationRun' }),
      ],
      0.25,
    )
    expect(run).toEqual({
      kind: 'promptRun',
      id: 'r1',
      label: 'Prompt run',
      done: 100,
      total: 400,
      unit: 'responses',
      costSoFarUsd: 0.1,
      costToGoUsd: 0.15,
      startedAt: '2026-10-28T09:00:00Z',
      to: '/manage/prompt-test/runs',
      stop: { label: 'Stop the Prompt run', keeps: 'Stopping keeps the responses so far. You can finish the run later with a retry.' },
    })
    expect(review?.costToGoUsd).toBeUndefined()
    expect(review?.stop?.label).toBe('Stop the term review')
    expect(rec?.stop).toBeNull()
    expect(runningWork([work({ openRouterCostUsd: 0.9 })], 0.5)[0]?.costToGoUsd).toBe(0)
    expect(runningWork([work()])[0]?.costToGoUsd).toBeUndefined()
  })
})

describe('runAsWork', () => {
  it('turns a Prompt run into a piece of work', () => {
    expect(runAsWork(promptRun({ id: 'r', status: 'stopped', failure: HAND, answered: 40, failed: 2, tasks: 360, costUsd: 0.05, finishedAt: '2026-10-28T09:00:00Z' }))).toEqual({
      id: 'r',
      kind: 'promptRun',
      startedAt: '2026-11-01T06:00:00Z',
      finishedAt: '2026-10-28T09:00:00Z',
      ending: 'stoppedByHand',
      done: 42,
      total: 360,
      failed: 2,
      openRouterCostUsd: 0.05,
      anthropicCostUsd: 0,
    })
    const running = runAsWork(promptRun({ status: 'running', finishedAt: undefined }))
    expect(running.ending).toBeUndefined()
    expect(running.finishedAt).toBeUndefined()
  })
})

describe('spendingRows', () => {
  it('names each kind and key, and adds them up', () => {
    expect(
      spendingRows([
        { kind: 'promptRuns', key: 'openRouter', costUsd: 1.88 },
        { kind: 'scans', key: 'anthropic', costUsd: 1.53 },
        { kind: 'readingResponses', key: 'anthropic', costUsd: 0.2 },
        { kind: 'termReview', key: 'anthropic', costUsd: 0.52 },
        { kind: 'candidateChecks', key: 'anthropic', costUsd: 0.22 },
        { kind: 'recommendationRuns', key: 'anthropic', costUsd: 0 },
      ]),
    ).toEqual({
      rows: [
        { kind: 'Prompt runs', paidFrom: 'OpenRouter', usd: 1.88 },
        { kind: 'Scans', paidFrom: 'Anthropic', usd: 1.53 },
        { kind: 'Reading responses', paidFrom: 'Anthropic', usd: 0.2 },
        { kind: 'Term review', paidFrom: 'Anthropic', usd: 0.52 },
        { kind: 'Candidate checks', paidFrom: 'Anthropic', usd: 0.22 },
        { kind: 'Recommendation runs', paidFrom: 'Anthropic', usd: 0 },
      ],
      totalUsd: 4.35,
    })
  })
})

describe('the runs of the Prompt test', () => {
  it('knows which ended runs a retry would finish', () => {
    expect(needsFinishing(promptRun({ status: 'done', answered: 358, tasks: 360 }))).toBe(true)
    expect(needsFinishing(promptRun({ status: 'stopped', answered: 40, tasks: 360 }))).toBe(true)
    expect(needsFinishing(promptRun({ status: 'done', answered: 360, tasks: 360 }))).toBe(false)
    expect(needsFinishing(promptRun({ status: 'running', answered: 0, tasks: 360 }))).toBe(false)
  })

  it('finds the newest run that has ended', () => {
    expect(lastEnded([promptRun({ id: 'a', status: 'running' }), promptRun({ id: 'b' })])?.id).toBe('b')
    expect(lastEnded([promptRun({ status: 'running' })])).toBeUndefined()
  })
})

describe('what waits', () => {
  it('counts the scans in the queue', () => {
    expect(queuedScans([summary({ status: 'queued' }), summary({ status: 'running' }), summary({ status: 'queued' })])).toBe(2)
  })

  it('names the keys that are missing or cannot be read', () => {
    expect(missingKeys([])).toEqual(['openrouter', 'anthropic'])
    expect(missingKeys([{ provider: 'openrouter', readable: true }, { provider: 'anthropic', readable: false }])).toEqual(['anthropic'])
    expect(missingKeys([{ provider: 'openrouter', readable: true }, { provider: 'anthropic', readable: true }])).toEqual([])
  })

  it('says how many wait', () => {
    expect(waitLine(1, 'search term', 'search terms')).toBe('1 search term waits')
    expect(waitLine(6072, 'search term', 'search terms')).toBe('6,072 search terms wait')
  })
})
