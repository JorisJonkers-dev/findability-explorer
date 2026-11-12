import { describe, expect, it } from 'vitest'
import { competitor, summary } from '../../../testing/fixtures'
import { withShortNames } from './company'
import {
  bestCompetitor,
  columnValue,
  competitorRows,
  competitorTable,
  countriesOf,
  familiesCell,
  NO_FILTER,
  scanDate,
  summaryLine,
} from './table'

const companies = withShortNames(
  [
    competitor('Tribelt', { role: 'tribelt', country: 'NL' }),
    competitor('Twentebelt', {
      country: 'NL',
      beltNames: [
        { name: 'TwenteFlex', family: 'tri_flex_turn_curve' },
        { name: 'Oog', family: 'eyelink' },
        { name: 'Any', family: 'all' },
      ],
      lastScan: summary({ createdAt: '2026-10-20T08:00:00Z', finishedAt: '2026-10-20T09:00:00Z' }),
    }),
    competitor('Ashworth', { country: 'US', beltNames: [], lastScan: summary({ createdAt: '2026-10-25T08:00:00Z', finishedAt: undefined }) }),
    competitor('Old Belts', { country: 'DE', archived: true }),
    competitor('Sister', { role: 'group' }),
  ],
  [],
)
const share: Record<string, number> = { tribelt: 0.2, twentebelt: 0.5, ashworth: 0.5 }
const all = competitorRows(companies, { share: (id) => share[id], ready: (id) => (id === 'ashworth' ? 2.5 : undefined), terms: (id) => (id === 'twentebelt' ? 7 : 1) })

const rows = all
const row = (i: number) => {
  const r = rows[i]
  if (!r) throw new Error(`no row ${String(i)}`)
  return r
}

describe('the rows of the Competitors table', () => {
  it('hold Tribelt and every competitor, archived ones marked, and no group company', () => {
    expect(rows.map((r) => [r.id, r.tribelt, r.archived])).toEqual([
      ['tribelt', true, false],
      ['twentebelt', false, false],
      ['ashworth', false, false],
      ['old-belts', false, true],
    ])
    const twente = rows[1]
    expect(twente).toMatchObject({ name: 'Twentebelt', country: 'NL', families: ['Eyelink', 'Tri-Flex'], share: 0.5, ready: undefined, terms: 7, scannedAt: '2026-10-20T09:00:00Z' })
    expect(rows[2]?.scannedAt).toBe('2026-10-25T08:00:00Z')
    expect(rows[3]?.scannedAt).toBeUndefined()
  })

  it('show the belt families as n of 5, with their names on hover', () => {
    expect(familiesCell(row(1))).toEqual({ text: '2 of 5', title: 'Eyelink and Tri-Flex' })
    expect(familiesCell(row(2))).toEqual({ text: 'None recorded', title: '' })
  })

  it('sort by any column', () => {
    const r = row(2)
    expect(['name', 'country', 'families', 'share', 'ready', 'terms', 'scan'].map((k) => columnValue(r, k as never))).toEqual([
      'Ashworth',
      'US',
      0,
      0.5,
      2.5,
      1,
      '2026-10-25T08:00:00Z',
    ])
    const bare = row(3)
    expect([columnValue(bare, 'share'), columnValue(bare, 'ready'), columnValue(bare, 'scan')]).toEqual([-1, -1, ''])
  })
})

describe('the Competitors table', () => {
  it('pins Tribelt on top, leaves archived companies out, and sorts the rest, the most named first', () => {
    expect(competitorTable(rows, NO_FILTER, null).map((r) => r.id)).toEqual(['tribelt', 'ashworth', 'twentebelt'])
    expect(competitorTable(rows, NO_FILTER, { key: 'terms', direction: 'desc' }).map((r) => r.id)).toEqual(['tribelt', 'twentebelt', 'ashworth'])
    expect(competitorTable(rows, { ...NO_FILTER, archived: true }, { key: 'name', direction: 'asc' }).map((r) => r.id)).toEqual([
      'tribelt',
      'ashworth',
      'old-belts',
      'twentebelt',
    ])
  })

  it('filters by name, belt family, and country', () => {
    expect(competitorTable(rows, { ...NO_FILTER, q: ' TWEN' }, null).map((r) => r.id)).toEqual(['tribelt', 'twentebelt'])
    expect(competitorTable(rows, { ...NO_FILTER, family: 'eyelink' }, null).map((r) => r.id)).toEqual(['tribelt', 'twentebelt'])
    expect(competitorTable(rows, { ...NO_FILTER, country: 'US' }, null).map((r) => r.id)).toEqual(['tribelt', 'ashworth'])
    expect(countriesOf(rows)).toEqual(['DE', 'NL', 'US'])
  })
})

describe('the summary of Competitors', () => {
  it('names the best competitor, a tie going to the name first alphabetically', () => {
    expect(bestCompetitor(rows)?.name).toBe('Ashworth')
    expect(bestCompetitor(rows.slice(0, 1))).toBeUndefined()
    expect(summaryLine(rows[0], bestCompetitor(rows))).toBe('Tribelt is named in 20% of AI responses. The best competitor, Ashworth, is named in 50%.')
    expect(summaryLine(undefined, undefined)).toBe('No Prompt run has counted the AI responses yet.')
    expect(summaryLine({ ...row(0), share: undefined }, undefined)).toBe('No Prompt run has counted the AI responses yet.')
    expect(summaryLine(rows[0], undefined)).toBe('Tribelt is named in 20% of AI responses.')
  })

  it('writes a scan date as the pages do', () => {
    expect(scanDate('2026-10-20T09:00:00Z')).toBe('20 October 2026')
    expect(scanDate(undefined)).toBe('Not scanned')
  })
})
