/** How many rows a long table shows before Show all. */
export const ROW_LIMIT = 12

/** Whether a table is long enough to search: more rows than it shows at first. */
export function searchable(count: number, limit = ROW_LIMIT): boolean {
  return count > limit
}

/** The rows whose text holds every word of the query, in any case; an empty query keeps them all. */
export function searchRows<T>(rows: readonly T[], query: string, text: (row: T) => string): T[] {
  const words = query.toLowerCase().split(/\s+/).filter((w) => w !== '')
  if (words.length === 0) return [...rows]
  return rows.filter((row) => {
    const haystack = text(row).toLowerCase()
    return words.every((w) => haystack.includes(w))
  })
}

/** The rows a table shows: all of them, or the first few until Show all. */
export function visibleRows<T>(rows: readonly T[], all: boolean, limit = ROW_LIMIT): T[] {
  return all ? [...rows] : rows.slice(0, limit)
}

/** The line under a cut table: "Showing 12 of 65.", or null when every row shows. */
export function shownLine(shown: number, total: number): string | null {
  return shown < total ? `Showing ${String(shown)} of ${String(total)}.` : null
}
