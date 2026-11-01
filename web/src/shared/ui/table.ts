/**
 * The classes of a table in a page's mist block, whole so Tailwind finds them: the block that
 * scrolls a wide table on a phone, the table, its column heads, its cells, and its total row.
 */
export const TABLE = {
  box: 'overflow-x-auto rounded-3xl bg-mist px-3 py-3',
  table: 'w-full border-collapse text-left text-sm',
  head: 'border-b border-silver px-3 pt-1 pb-2 align-bottom text-xs font-semibold text-muted',
  cell: 'border-b border-grey px-3 py-2.5 align-top',
  foot: 'border-t-2 border-black px-3 py-2.5 align-top font-semibold',
} as const
