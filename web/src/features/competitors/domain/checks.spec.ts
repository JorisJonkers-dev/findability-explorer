import { describe, expect, it } from 'vitest'
import type { CompetitorCheck } from '../../../infrastructure/api/types.gen'
import { competitor } from '../../../testing/fixtures'
import { archivedByHaiku, checksById, haikuArchived, isFlagged, listCheckLine, verdictLabel, verdictReason } from './checks'

function check(overrides: Partial<CompetitorCheck> = {}): CompetitorCheck {
  return { competitorId: 'fenner', competitor: false, sure: true, reason: 'Makes rubber belting.', model: 'haiku', at: '2026-10-27T09:00:00Z', ...overrides }
}

describe("Haiku's check of the list", () => {
  it('finds a company’s check by its id', () => {
    expect(checksById([check()]).get('fenner')?.reason).toBe('Makes rubber belting.')
  })

  it('says what Haiku made of a company, and why', () => {
    expect(verdictLabel(undefined)).toBeNull()
    expect(verdictLabel(check())).toBe('Haiku says not a competitor')
    expect(verdictLabel(check({ competitor: true }))).toBe('Haiku says competitor')
    expect(verdictLabel(check({ sure: false }))).toBe('Haiku is unsure')
    expect(verdictReason(check())).toBe('Makes rubber belting.')
    expect(verdictReason(check({ reason: '' }))).toBe('Haiku gave no reason for it.')
    expect(verdictReason(undefined)).toBe('Haiku gave no reason for it.')
  })

  it('flags a company on the list Haiku is unsure of, until a person keeps it', () => {
    const c = competitor('Intralox')
    expect(isFlagged(c, check({ sure: false }))).toBe(true)
    expect(isFlagged(c, check())).toBe(false)
    expect(isFlagged(c, undefined)).toBe(false)
    expect(isFlagged({ ...c, keptBy: 'joris' }, check({ sure: false }))).toBe(false)
    expect(isFlagged({ ...c, archived: true }, check({ sure: false }))).toBe(false)
  })

  it('lists what Haiku archived, the latest first', () => {
    const older = competitor('Fenner', { archived: true, archivedBy: 'language model', archivedAt: '2026-10-26T09:00:00Z' })
    const newer = competitor('Ammeraal', { archived: true, archivedBy: 'language model', archivedAt: '2026-10-27T09:00:00Z' })
    const same = competitor('Beltco', { archived: true, archivedBy: 'language model', archivedAt: '2026-10-27T09:00:00Z' })
    const undated = competitor('Undated', { archived: true, archivedBy: 'language model' })
    const byPerson = competitor('Gone', { archived: true, archivedBy: 'admin-1' })
    expect(archivedByHaiku(older)).toBe(true)
    expect(archivedByHaiku(byPerson)).toBe(false)
    expect(archivedByHaiku({ archived: false, archivedBy: 'language model' })).toBe(false)
    expect(haikuArchived([older, undated, byPerson, same, newer, competitor('Here')]).map((c) => c.name)).toEqual(['Ammeraal', 'Beltco', 'Fenner', 'Undated'])
  })

  it('says how far Haiku is', () => {
    expect(listCheckLine({ running: true }, 3)).toBe('Haiku is checking the companies on the list.')
    expect(listCheckLine({ running: false, problem: 'noKey' }, 3)).toContain('No Anthropic key')
    expect(listCheckLine({ running: false, problem: 'failed' }, 3)).toBe('Haiku failed on its last try. Try again.')
    expect(listCheckLine({ running: false }, 1)).toBe('Haiku has not checked 1 company on the list yet.')
    expect(listCheckLine({ running: false }, 12)).toBe('Haiku has not checked 12 companies on the list yet.')
    expect(listCheckLine({ running: false }, 0)).toBe('Haiku has checked every company on the list.')
  })
})
