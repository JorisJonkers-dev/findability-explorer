import { describe, expect, it } from 'vitest'
import type { AiReadinessRow } from '../../../infrastructure/api/types.gen'
import { promptRun, summary } from '../../../testing/fixtures'
import { overviewFixture, planFixture } from '../../../testing/numbersFixtures'
import { latestRows, overviewDataLine } from './latest'

const NOW = new Date('2026-11-01T12:00:00Z')
const row = (competitorId: string, role: AiReadinessRow['role'], finishedAt?: string): AiReadinessRow => ({
  competitorId,
  name: competitorId,
  role,
  ...(finishedAt ? { scan: summary({ finishedAt }) } : {}),
})
const readiness = [
  row('tribelt', 'tribelt', '2026-10-26T09:00:00Z'),
  row('ashworth', 'competitor', '2026-10-20T09:00:00Z'),
  row('martens', 'competitor', '2026-09-01T09:00:00Z'),
  row('quiet', 'competitor'),
  row('lesjofors', 'group', '2026-10-30T09:00:00Z'),
]

describe('Latest runs and scans', () => {
  it('says when each process last ran and what comes next, with its Results and Manage pages', () => {
    const rows = latestRows({ runs: [promptRun({ status: 'running', finishedAt: undefined, startedAt: '2026-11-01T06:00:00Z' }), promptRun({ startedAt: '2026-10-27T06:00:00Z' })], nextRunAt: '2026-11-08T05:00:00Z', readiness, overview: overviewFixture(), plan: planFixture(), now: NOW })
    expect(rows.map((r) => [r.process, r.latest, r.next, r.to, r.manage])).toEqual([
      ['Prompt test', 'Last complete run on 27 October 2026', 'The next run is scheduled for 8 November 2026.', '/ai-responses', '/manage/prompt-test'],
      ['Website scans', 'tribelt.nl scanned on 26 October 2026', '1 of 3 competitors were scanned in the last 30 days.', '/ai-readiness', '/manage/scans'],
      ['Search term list', 'The list holds 387 search terms.', 'Each scan suggests new search terms for review.', '/terms', '/manage/terms'],
      ['Recommendations', '3 recommendations are open and 1 is done.', 'A scan of tribelt.nl ticks off the ones it can check, and the rest are ticked off by hand.', '/recommendations', '/manage/recommendations'],
    ])
  })

  it('says so before the first run and scan, and leaves the next run to administrators', () => {
    const plan = planFixture()
    plan.improvements = plan.improvements.map((i) => ({ ...i, done: false }))
    const rows = latestRows({ runs: [], readiness: [], overview: overviewFixture(), plan, now: NOW })
    expect(rows.map((r) => [r.latest, r.next])).toEqual([
      ['No Prompt run has finished yet.', ''],
      ['tribelt.nl has not been scanned yet.', '0 of 0 competitors were scanned in the last 30 days.'],
      ['The list holds 387 search terms.', 'Each scan suggests new search terms for review.'],
      ['4 recommendations are open and none is done.', 'A scan of tribelt.nl ticks off the ones it can check, and the rest are ticked off by hand.'],
    ])
  })
})

describe('the recommendations row', () => {
  it('counts several done', () => {
    const plan = planFixture()
    plan.improvements = plan.improvements.map((i) => ({ ...i, done: true }))
    expect(latestRows({ runs: [], readiness: [], overview: overviewFixture(), plan, now: NOW })[3]?.latest).toBe('0 recommendations are open and 4 are done.')
  })
})

describe('the Overview data line', () => {
  it('names the newest Prompt run and the newest scan', () => {
    expect(overviewDataLine('2026-10-27T06:00:00Z', readiness)).toBe('From the Prompt run of 27 October 2026 and the scans up to 30 October 2026.')
    expect(overviewDataLine(undefined, readiness)).toBe('From the scans up to 30 October 2026.')
    expect(overviewDataLine('2026-10-27T06:00:00Z', [])).toBe('From the Prompt run of 27 October 2026.')
    expect(overviewDataLine(undefined, [])).toBeUndefined()
  })
})
