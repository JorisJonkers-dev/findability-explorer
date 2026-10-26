import { describe, expect, it } from 'vitest'
import { directionOf, nextSort, sortLabel, sortRows } from './sort'

describe('nextSort', () => {
  it('cycles a column through its first direction, the other, and back to the table order', () => {
    const one = nextSort(null, 'name', 'asc')
    expect(one).toEqual({ key: 'name', direction: 'asc' })
    const two = nextSort(one, 'name', 'asc')
    expect(two).toEqual({ key: 'name', direction: 'desc' })
    expect(nextSort(two, 'name', 'asc')).toBeNull()
    // A count starts highest first.
    expect(nextSort(null, 'total', 'desc')).toEqual({ key: 'total', direction: 'desc' })
    expect(nextSort({ key: 'total', direction: 'desc' }, 'total', 'desc')).toEqual({ key: 'total', direction: 'asc' })
    // Another column starts over.
    expect(nextSort(two, 'country', 'asc')).toEqual({ key: 'country', direction: 'asc' })
  })

  it('tells a column its direction', () => {
    expect(directionOf({ key: 'name', direction: 'desc' }, 'name')).toBe('desc')
    expect(directionOf({ key: 'name', direction: 'desc' }, 'country')).toBeNull()
    expect(directionOf(null, 'name')).toBeNull()
  })
})

describe('sortRows', () => {
  const rows = [
    { name: 'Pelger', country: 'DE', eyelink: false, terms: 3 },
    { name: 'ashworth', country: 'US', eyelink: true, terms: 10 },
    { name: 'Märtens', country: 'DE', eyelink: true, terms: 49 },
  ]
  type Key = keyof (typeof rows)[number]
  const value = (r: (typeof rows)[number], k: Key) => r[k]
  const names = (sorted: typeof rows) => sorted.map((r) => r.name)

  it('sorts text without regard to case, numbers by size, and offers first when descending', () => {
    expect(names(sortRows(rows, { key: 'name' as Key, direction: 'asc' }, value))).toEqual(['ashworth', 'Märtens', 'Pelger'])
    expect(names(sortRows(rows, { key: 'terms' as Key, direction: 'desc' }, value))).toEqual(['Märtens', 'ashworth', 'Pelger'])
    expect(names(sortRows(rows, { key: 'eyelink' as Key, direction: 'desc' }, value))).toEqual(['ashworth', 'Märtens', 'Pelger'])
  })

  it('keeps the order of rows that tie, and the table order without a sort', () => {
    expect(names(sortRows(rows, { key: 'country' as Key, direction: 'asc' }, value))).toEqual(['Pelger', 'Märtens', 'ashworth'])
    expect(names(sortRows(rows, null, value))).toEqual(['Pelger', 'ashworth', 'Märtens'])
  })
})

describe('sortLabel', () => {
  it('says how a column is sorted', () => {
    expect(sortLabel('Eyelink', 'desc')).toBe('Eyelink, sorted descending')
    expect(sortLabel('Company', 'asc')).toBe('Company, sorted ascending')
    expect(sortLabel('Company', null)).toBe('Company, not sorted')
  })
})
