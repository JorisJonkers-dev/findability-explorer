import { describe, expect, it } from 'vitest'
import type { ComparisonGroup, ProfileTerm } from '../../../infrastructure/api/types.gen'
import { summary } from '../../../testing/fixtures'
import {
  byLanguage,
  changeCounts,
  changeDetail,
  changedGroups,
  defaultPair,
  finishedScans,
  firstGroupWithTerms,
  groupLabel,
  groupTabLabel,
  kindLabel,
  mapLabel,
  PROFILE_GROUPS,
  scanDate,
  scanModel,
  scanName,
  scanPages,
  scanState,
  sinceLabel,
} from './history'

const t = (phrase: string, language: string, overrides: Partial<ProfileTerm> = {}): ProfileTerm => ({
  phrase,
  language,
  score: 1,
  pages: 1,
  tier: 'product',
  inMap: false,
  firstScan: 1,
  ...overrides,
})

describe('the profile', () => {
  it('names the groups in their fixed order, with their counts', () => {
    expect(PROFILE_GROUPS.map(groupLabel)).toEqual(['Eyelink', 'Open spiral', 'Closed spiral', 'Tri-Flex', 'Other belts', 'Broad', 'Specification'])
    expect(groupTabLabel({ group: 'eyelink', terms: [t('a', 'en')] })).toBe('Eyelink (1)')
  })

  it('opens on the first group with terms, else the first, else none', () => {
    expect(firstGroupWithTerms([{ group: 'eyelink', terms: [] }, { group: 'spec', terms: [t('a', 'en')] }])).toBe('spec')
    expect(firstGroupWithTerms([{ group: 'eyelink', terms: [] }])).toBe('eyelink')
    expect(firstGroupWithTerms([])).toBeUndefined()
  })

  it('splits a group by language, the most terms first, ties by code', () => {
    const groups = byLanguage([t('a', 'nl'), t('b', 'en'), t('c', 'de'), t('d', 'en')])
    expect(groups.map((g) => [g.language, g.label, g.terms.map((x) => x.phrase)])).toEqual([
      ['en', 'English', ['b', 'd']],
      ['de', 'German', ['c']],
      ['nl', 'Dutch', ['a']],
    ])
  })

  it('says since when a term is found, and what the map makes of it', () => {
    expect(sinceLabel(3, 3)).toBe('new in this scan')
    expect(sinceLabel(1, 3)).toBe('since scan 1')
    expect(mapLabel({ inMap: true })).toBe('in the map')
    expect(mapLabel({ inMap: false, closestTerm: 'eyelink belt' })).toBe('not in the map, closest: eyelink belt')
    expect(mapLabel({ inMap: false })).toBe('not in the map')
  })
})

describe('the scans', () => {
  it('describes a scan', () => {
    expect(scanName({ number: 3 })).toBe('Scan 3')
    expect(scanName({})).toBe('Scan')
    expect(scanDate({ createdAt: '2026-10-22T23:30:00Z' })).toBe('22 Oct 2026')
    expect(scanPages({ status: 'done', pagesCrawled: 48, maxPages: 60 })).toBe('48 of 60 pages')
    expect(scanPages({ status: 'running', pagesCrawled: 4, maxPages: 60 })).toBe('4 of 60 pages so far')
    expect((['queued', 'running', 'failed', 'done'] as const).map((status) => scanState({ status }))).toEqual(['Queued', 'Scanning', 'Failed', 'Finished'])
    expect(scanModel({ matchingModel: 'anthropic: claude-haiku-5-5' })).toBe('anthropic: claude-haiku-5-5')
    expect(scanModel({})).toBe('no model')
  })

  it('compares the latest two finished scans at first', () => {
    const scans = [summary({ id: 'c', status: 'running' }), summary({ id: 'b' }), summary({ id: 'x', status: 'failed' }), summary({ id: 'a' })]
    expect(finishedScans(scans).map((s) => s.id)).toEqual(['b', 'a'])
    expect(defaultPair(scans)).toEqual({ from: 'a', to: 'b' })
    expect(defaultPair(scans.slice(0, 2))).toBeNull()
  })
})

describe('a comparison', () => {
  const change = { phrase: 'p', language: 'en', tier: 'product' as const, inMap: false }

  it('says what changed about a term', () => {
    expect(kindLabel('moved')).toBe('Moved')
    expect(changeDetail({ ...change, toScore: 30.46, toPages: 5 }, 'added')).toBe('score 30.5, 5 pages')
    expect(changeDetail(change, 'added')).toBe('score –, 0 pages')
    expect(changeDetail({ ...change, fromScore: 12, fromPages: 1 }, 'removed')).toBe('was 12, 1 page')
    expect(changeDetail(change, 'removed')).toBe('was –, 0 pages')
    expect(changeDetail({ ...change, fromScore: 12, toScore: 30, fromPages: 2, toPages: 5 }, 'moved')).toBe('score 12 → 30, pages 2 → 5')
    expect(changeDetail(change, 'moved')).toBe('score – → –, pages 0 → 0')
  })

  it('counts the changes and keeps the groups that have any', () => {
    const groups: ComparisonGroup[] = [
      { group: 'eyelink', added: [change, change], removed: [], moved: [change] },
      { group: 'broad', added: [], removed: [], moved: [] },
      { group: 'spec', added: [], removed: [change], moved: [] },
    ]
    expect(changeCounts(groups)).toEqual({ added: 2, removed: 1, moved: 1 })
    expect(changedGroups(groups).map((g) => g.group)).toEqual(['eyelink', 'spec'])
  })
})
