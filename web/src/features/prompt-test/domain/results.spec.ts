import { describe, expect, it } from 'vitest'
import { promptResults, promptRun } from '../../../testing/fixtures'
import {
  costLabel,
  familyName,
  fraction,
  latestRun,
  mentionLine,
  modeName,
  PILOT,
  progressLabel,
  providerName,
  ranking,
  runDate,
  scopeName,
  shareLabel,
  shareRows,
  sharesLine,
  statusName,
  tally,
  trendRows,
} from './results'

describe('the Prompt test’s measures', () => {
  it('sums cells by any filter, and writes shares', () => {
    const run = promptRun()
    expect(tally(run.cells)).toEqual({ answers: 12, named: 3, cited: 1 })
    expect(tally(run.cells, { provider: 'openai', mode: 'web' })).toEqual({ answers: 3, named: 2, cited: 1 })
    expect(tally(run.cells, { scope: 'english' })).toEqual({ answers: 0, named: 0, cited: 0 })
    expect([shareLabel(1, 6), shareLabel(0, 0), fraction(1, 4), fraction(1, 0)]).toEqual(['17%', '–', 0.25, 0])
    expect(sharesLine({ answers: 4, named: 1, cited: 0 })).toBe('25% name Tribelt · 0% cite it')
  })

  it('breaks a run down per provider, scope or belt family, both modes side by side', () => {
    const run = promptRun()
    expect(shareRows(run, 'provider')).toEqual([
      { key: 'openai', label: 'OpenAI', knowledge: { answers: 3, named: 1, cited: 0 }, web: { answers: 3, named: 2, cited: 1 } },
      { key: 'google', label: 'Google', knowledge: { answers: 3, named: 0, cited: 0 }, web: { answers: 3, named: 0, cited: 0 } },
    ])
    expect(shareRows(run, 'scope').map((r) => r.label)).toEqual([
      'Twente (Dutch)', 'Netherlands (Dutch)', 'Germany (German)', 'Dutch, no region', 'German, no region', 'English, no region',
    ])
    expect(shareRows(run, 'family')[0]).toEqual({ key: 'eyelink', label: 'Eyelink', knowledge: { answers: 3, named: 1, cited: 0 }, web: { answers: 3, named: 2, cited: 1 } })
  })

  it('follows the shares across runs, the pilot first', () => {
    const older = promptRun({ id: 'old', startedAt: '2026-10-01T06:00:00Z', status: 'stopped' })
    const empty = promptRun({ id: 'new', startedAt: '2026-12-01T06:00:00Z', status: 'running', cells: [] })
    const rows = trendRows([empty, promptRun(), older])
    expect(rows.map((r) => [r.label, r.note])).toEqual([
      [PILOT.label, PILOT.note],
      ['1 Oct 2026', 'Stopped at the budget'],
      ['1 Nov 2026', 'Done'],
    ])
    expect(rows[0]?.knowledge).toBeNull()
    expect(rows[2]?.overall).toEqual({ answers: 12, named: 3, cited: 1 })
    const openai = trendRows([promptRun()], { provider: 'openai', scope: 'twente' })
    expect(openai).toEqual([{ label: '1 Nov 2026', note: 'Done', overall: { answers: 6, named: 3, cited: 1 }, knowledge: { answers: 3, named: 1, cited: 0 }, web: { answers: 3, named: 2, cited: 1 } }])
  })

  it('names providers, scopes, families, modes and statuses', () => {
    expect([providerName('meta-llama'), providerName('cohere')]).toEqual(['Meta', 'cohere'])
    expect([scopeName('germany'), familyName('tri_flex_turn_curve'), modeName('web')]).toEqual(['Germany (German)', 'Curve-going (Tri-Flex)', 'With web search'])
    expect([statusName('running'), statusName('failed')]).toEqual(['Running', 'Failed'])
    // A value the contract adds later still reads as itself.
    expect([scopeName('mars' as never), familyName('x' as never), modeName('y' as never)]).toEqual(['mars', 'x', 'y'])
  })

  it('tells a run’s progress, cost and date', () => {
    expect(progressLabel(promptRun())).toBe('360 of 360 answers, 2 without an answer')
    expect(progressLabel(promptRun({ answered: 1200, failed: 0, tasks: 1260 }))).toBe('1,200 of 1,260 answers')
    expect([costLabel(0.42), costLabel(10), runDate(promptRun())]).toEqual(['$0.42', '$10.00', '1 Nov 2026'])
  })
})

describe('the latest run and the companies it names', () => {
  it('opens on the newest run with answers', () => {
    const running = promptRun({ id: 'new', status: 'running', cells: [] })
    expect(latestRun(promptResults({ runs: [running, promptRun()] }))?.id).toBe(promptRun().id)
    expect(latestRun(promptResults({ runs: [running] }))?.id).toBe('new')
    expect(latestRun(undefined)).toBeUndefined()
  })

  it('ranks the competitors and tells a card how often answers name its company', () => {
    const results = promptResults()
    expect(ranking(results.mentions).map((m) => m.name)).toEqual(['Twentebelt'])
    expect(mentionLine(results, 'twentebelt')).toBe('Named in 9 of 12 answers of the Prompt run of 1 Nov 2026, first in 4.')
    expect(mentionLine(promptResults({ mentions: results.mentions.map((m) => ({ ...m, first: 0 })) }), 'twentebelt')).toBe('Named in 9 of 12 answers of the Prompt run of 1 Nov 2026.')
    expect(mentionLine(results, 'esfo')).toBe('Named in none of the 12 answers of the Prompt run of 1 Nov 2026.')
    expect(mentionLine(promptResults({ runs: [] }), 'esfo')).toBeNull()
    expect(mentionLine(undefined, 'esfo')).toBeNull()
  })
})
