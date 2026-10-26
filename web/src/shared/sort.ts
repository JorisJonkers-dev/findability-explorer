/** Which way a column is sorted: ascending (A to Z, lowest first) or descending. */
export type SortDirection = 'asc' | 'desc'

/** A table's sort: the column and its direction, or null for the table's own order. */
export type Sort<K extends string> = { key: K; direction: SortDirection } | null

/** A value a column sorts by: text, a number, or whether a row has something (true sorts as 1). */
export type SortValue = string | number | boolean

/**
 * The sort after a click on a column's header: the column's first direction, then the other,
 * then the table's own order again. first is the direction that is most useful first: A to Z for
 * names, highest first for counts and for whether a company offers something.
 */
export function nextSort<K extends string>(current: Sort<K>, key: K, first: SortDirection): Sort<K> {
  if (current?.key !== key) return { key, direction: first }
  if (current.direction === first) return { key, direction: first === 'asc' ? 'desc' : 'asc' }
  return null
}

/** The column's direction in sort, or null when the table is not sorted by it. */
export function directionOf<K extends string>(sort: Sort<K>, key: K): SortDirection | null {
  return sort?.key === key ? sort.direction : null
}

const TEXT = new Intl.Collator('en', { sensitivity: 'base', numeric: true })

function compare(a: SortValue, b: SortValue): number {
  if (typeof a === 'string' && typeof b === 'string') return TEXT.compare(a, b)
  return Number(a) - Number(b)
}

/**
 * The rows sorted by the column sort names, through value; rows that tie keep their order, and
 * without a sort the rows come as they are.
 */
export function sortRows<T, K extends string>(rows: readonly T[], sort: Sort<K>, value: (row: T, key: K) => SortValue): T[] {
  if (sort === null) return [...rows]
  const sign = sort.direction === 'asc' ? 1 : -1
  return rows
    .map((row, index) => ({ row, index }))
    .sort((a, b) => sign * compare(value(a.row, sort.key), value(b.row, sort.key)) || a.index - b.index)
    .map(({ row }) => row)
}

/** How a sort reads for a screen reader: "Eyelink, sorted descending" or "Eyelink, not sorted". */
export function sortLabel(label: string, direction: SortDirection | null): string {
  if (direction === null) return `${label}, not sorted`
  return `${label}, sorted ${direction === 'asc' ? 'ascending' : 'descending'}`
}
