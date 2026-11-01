import { describe, expect, it } from 'vitest'
import { competitor } from '../../../testing/fixtures'
import { archivedLine, listCompanies, listLine, statusOf } from './companies'
import { MODEL_DECIDER } from './candidates'

const day = (iso: string) => iso.slice(0, 10)

describe('the companies on the list', () => {
  it('keeps the competitors, archived ones too, A to Z', () => {
    const companies = [competitor('Zeta'), competitor('Tribelt', { role: 'tribelt' }), competitor('Alpha', { archived: true }), competitor('Group', { role: 'group' })]
    expect(listCompanies(companies).map((c) => c.name)).toEqual(['Alpha', 'Zeta'])
  })

  it('counts them in a line', () => {
    expect(listLine([{ archived: false }, { archived: false }, { archived: true }])).toBe('2 companies count as competitors and 1 is archived.')
    expect(listLine([{ archived: false }, { archived: true }, { archived: true }])).toBe('1 company counts as a competitor and 2 are archived.')
    expect(listLine([{ archived: false }])).toBe('1 company counts as a competitor.')
  })

  it('says what each is, and who archived it when', () => {
    expect(statusOf({ archived: false })).toBe('Competitor')
    expect(statusOf({ archived: true, archivedBy: 'joris' })).toBe('Archived')
    expect(statusOf({ archived: true, archivedBy: MODEL_DECIDER })).toBe('Archived by Haiku')
    expect(archivedLine({ archived: false }, day)).toBeNull()
    expect(archivedLine({ archived: true, archivedBy: 'joris', archivedAt: '2026-10-27T09:00:00Z' }, day)).toBe(
      'Archived by joris on 2026-10-27. It no longer counts as a competitor, and its scans stay.',
    )
    expect(archivedLine({ archived: true, archivedBy: MODEL_DECIDER }, day)).toBe('Archived by Claude Haiku. It no longer counts as a competitor, and its scans stay.')
    expect(archivedLine({ archived: true }, day)).toBe('Archived. It no longer counts as a competitor, and its scans stay.')
  })
})
