import { computed, ref, watch } from 'vue'
import { type ColumnState, columnTemplate, minimised, parseLayout, restored, widened } from './columns'

const KEY = 'findability-explorer:columns:'

function read(page: string, count: number): ColumnState[] | null {
  try {
    return parseLayout(globalThis.localStorage.getItem(KEY + page), count)
  } catch {
    return null
  }
}

function write(page: string, states: readonly ColumnState[]): void {
  try {
    globalThis.localStorage.setItem(KEY + page, JSON.stringify(states))
  } catch {
    // A browser without storage keeps the layout for this visit only.
  }
}

/**
 * The layout of a page's stage columns, base being each column's designed share: widen, minimise
 * and restore a column. The browser remembers it per page, as a viewer's own convenience.
 */
export function useColumnLayout(page: string, base: readonly number[]) {
  const states = ref<ColumnState[]>(read(page, base.length) ?? base.map(() => 'normal'))
  watch(states, (now) => { write(page, now); }, { deep: true })
  return {
    states,
    /** The stage grid's columns, for a grid-template-columns style. */
    template: computed(() => columnTemplate(base, states.value)),
    widen: (i: number) => (states.value = widened(states.value, i)),
    minimise: (i: number) => (states.value = minimised(states.value, i)),
    restore: (i: number) => (states.value = restored(states.value, i)),
    /** Whether column i may be minimised: not the last one open. */
    canMinimise: (i: number) => states.value[i] !== 'min' && states.value.filter((s) => s !== 'min').length > 1,
  }
}
