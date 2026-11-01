import { describe, expect, it } from 'vitest'
import { promptRun, summary } from '../../../testing/fixtures'
import { endedOf, endedTone, hostOf, recentWork, runningWork, scanEnded, whenLabel } from './activity'

const HAND = 'Stopped by hand. The responses so far are kept, and a retry finishes the run.'
const BUDGET = "The month's budget for Prompt runs was reached; the answers so far are kept."

describe('endedOf', () => {
  it('reads a done run as Done, or Done with failed responses when some failed', () => {
    expect(endedOf(promptRun({ status: 'done', failed: 0 }))).toBe('Done')
    expect(endedOf(promptRun({ status: 'done', failed: 2 }))).toBe('Done, with failed responses')
  })

  it('tells a run stopped by hand from one stopped by the budget', () => {
    expect(endedOf(promptRun({ status: 'stopped', failure: HAND }))).toBe('Stopped by hand')
    expect(endedOf(promptRun({ status: 'stopped', failure: BUDGET }))).toBe('Stopped by the budget')
    expect(endedOf(promptRun({ status: 'stopped', failure: undefined }))).toBe('Stopped by the budget')
  })

  it('reads a failed run as Failed, and a running one as not ended', () => {
    expect(endedOf(promptRun({ status: 'failed', failure: 'No key.' }))).toBe('Failed')
    expect(endedOf(promptRun({ status: 'running' }))).toBeNull()
  })
})

describe('scanEnded', () => {
  it('reads a done scan as Done, a stopped one as Stopped by hand, a failed one as Failed', () => {
    expect(scanEnded(summary({ status: 'done' }))).toBe('Done')
    expect(scanEnded(summary({ status: 'failed', stoppedByHand: true }))).toBe('Stopped by hand')
    expect(scanEnded(summary({ status: 'failed', failure: 'Timed out.' }))).toBe('Failed')
    expect(scanEnded(summary({ status: 'queued' }))).toBeNull()
    expect(scanEnded(summary({ status: 'running' }))).toBeNull()
  })
})

describe('endedTone', () => {
  it('is good for done, bad for failed, and a warning otherwise', () => {
    expect(endedTone('Done')).toBe('good')
    expect(endedTone('Failed')).toBe('bad')
    expect(endedTone('Done, with failed responses')).toBe('warn')
    expect(endedTone('Stopped by hand')).toBe('warn')
    expect(endedTone('Stopped by the budget')).toBe('warn')
  })
})

describe('whenLabel and hostOf', () => {
  it('writes a moment in words, in UTC', () => {
    expect(whenLabel('2026-10-28T09:37:00Z')).toBe('28 October 2026, 09:37')
  })

  it('names a site by its host, without www', () => {
    expect(hostOf('https://www.tribelt.nl/en/')).toBe('tribelt.nl')
    expect(hostOf('not a url')).toBe('not a url')
  })
})

describe('recentWork', () => {
  it('lists the ended runs and scans, newest first, with how each ended, what it did, and its cost', () => {
    const lines = recentWork(
      [
        promptRun({ id: 'r1', status: 'done', failed: 0, answered: 200, tasks: 200, promptSetVersion: 2, finishedAt: '2026-10-28T09:37:00Z', costUsd: 1.12 }),
        promptRun({ id: 'r2', status: 'running', finishedAt: undefined }),
        promptRun({ id: 'r3', status: 'stopped', failure: HAND, answered: 40, tasks: 200, finishedAt: undefined, startedAt: '2026-10-26T06:00:00Z' }),
      ],
      [
        summary({ id: 's1', url: 'https://www.tribelt.nl/', status: 'done', pagesCrawled: 60, finishedAt: '2026-10-27T09:47:00Z' }),
        summary({ id: 's2', status: 'queued' }),
      ],
    )
    expect(lines.map((l) => l.id)).toEqual(['r1', 's1', 'r3'])
    expect(lines[0]).toEqual({
      id: 'r1',
      at: '2026-10-28T09:37:00Z',
      work: 'Prompt run',
      ended: 'Done',
      did: '200 of 200 responses, Prompt set version 2',
      costUsd: 1.12,
      to: '/manage/prompt-test/runs',
    })
    expect(lines[1]).toEqual({ id: 's1', at: '2026-10-27T09:47:00Z', work: 'Scan of tribelt.nl', ended: 'Done', did: '60 pages read', to: '/manage/scans/s1' })
    expect(lines[2]?.at).toBe('2026-10-26T06:00:00Z')
    expect(lines[2]?.ended).toBe('Stopped by hand')
  })

  it('keeps the last 20, and writes one page as a page', () => {
    const scans = Array.from({ length: 25 }, (_, i) =>
      summary({ id: `s${String(i)}`, status: 'done', pagesCrawled: 1, finishedAt: `2026-10-${String(i + 1).padStart(2, '0')}T08:00:00Z` }),
    )
    const lines = recentWork([], scans)
    expect(lines).toHaveLength(20)
    expect(lines[0]?.id).toBe('s24')
    expect(lines[0]?.did).toBe('1 page read')
    // A scan with no finish time sorts by when it was queued.
    expect(recentWork([], [summary({ id: 'x', status: 'failed', finishedAt: undefined, createdAt: '2026-10-01T00:00:00Z' })])[0]?.at).toBe('2026-10-01T00:00:00Z')
  })
})

describe('runningWork', () => {
  it('lists the running run, scans, and AI readiness check, each with its progress and its Stop', () => {
    const work = runningWork(
      [promptRun({ id: 'r1', status: 'running', answered: 90, failed: 10, tasks: 400, costUsd: 0.3, startedAt: '2026-11-01T08:29:00Z' }), promptRun()],
      [summary({ id: 's1', url: 'https://www.twentebelt.com/', status: 'running', pagesCrawled: 15, maxPages: 60 }), summary({ id: 's2', status: 'queued' })],
      { running: true, checked: 12, total: 48, failed: 0 },
      0.5,
    )
    expect(work).toEqual([
      {
        kind: 'promptRun',
        id: 'r1',
        label: 'Prompt run',
        done: 100,
        total: 400,
        unit: 'responses',
        costSoFarUsd: 0.3,
        costToGoUsd: 0.2,
        startedAt: '2026-11-01T08:29:00Z',
        to: '/manage/prompt-test/runs',
        stopLabel: 'Stop the Prompt run',
        keeps: 'Stopping keeps the responses so far. You can finish the run later with a retry.',
      },
      {
        kind: 'scan',
        id: 's1',
        label: 'Scan of twentebelt.com',
        done: 15,
        total: 60,
        unit: 'pages',
        startedAt: '2026-10-22T09:30:00Z',
        to: '/manage/scans/s1',
        stopLabel: 'Stop the scan',
        keeps: 'Stopping keeps the pages read so far.',
      },
      {
        kind: 'check',
        id: 'check',
        label: 'AI readiness check',
        done: 12,
        total: 48,
        unit: 'sites',
        to: '/ai-readiness',
        stopLabel: 'Stop the check',
        keeps: 'Stopping keeps what the check recorded for the sites it read. The others keep what their scan recorded.',
      },
    ])
  })

  it('has nothing when nothing runs, and leaves out the cost to go without an estimate', () => {
    expect(runningWork([promptRun()], [summary()], { running: false, checked: 0, total: 0, failed: 0 })).toEqual([])
    expect(runningWork([promptRun()], [summary()], undefined)).toEqual([])
    const [run] = runningWork([promptRun({ status: 'running', costUsd: 0.9 })], [], undefined, 0.5)
    expect(run?.costToGoUsd).toBe(0)
    expect(runningWork([promptRun({ status: 'running' })], [], undefined)[0]?.costToGoUsd).toBeUndefined()
  })
})
