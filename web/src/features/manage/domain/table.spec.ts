import { describe, expect, it } from 'vitest'
import { ROW_LIMIT, searchable, searchRows, shownLine, visibleRows } from './table'

const rows = Array.from({ length: 20 }, (_, i) => ({ name: i % 2 ? `Belt Works ${String(i)}` : `Mesh Co ${String(i)}` }))

describe('a long table', () => {
  it('searches only when it has more rows than it shows', () => {
    expect(searchable(ROW_LIMIT)).toBe(false)
    expect(searchable(ROW_LIMIT + 1)).toBe(true)
  })

  it('keeps the rows holding every word of the query, in any case', () => {
    expect(searchRows(rows, '', (r) => r.name)).toHaveLength(20)
    expect(searchRows(rows, '  belt  works 1 ', (r) => r.name).map((r) => r.name)).toEqual(['Belt Works 1', 'Belt Works 11', 'Belt Works 13', 'Belt Works 15', 'Belt Works 17', 'Belt Works 19'])
    expect(searchRows(rows, 'MESH 4', (r) => r.name).map((r) => r.name)).toEqual(['Mesh Co 4', 'Mesh Co 14'])
  })

  it('shows the first few rows until Show all, and says how many', () => {
    expect(visibleRows(rows, false)).toHaveLength(ROW_LIMIT)
    expect(visibleRows(rows, true)).toHaveLength(20)
    expect(shownLine(12, 20)).toBe('Showing 12 of 20.')
    expect(shownLine(20, 20)).toBeNull()
  })
})
