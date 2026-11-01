<script setup lang="ts" generic="T">
import { computed, ref, useId } from 'vue'
import { directionOf, nextSort, type Sort, sortLabel, sortRows, type SortValue } from '../../../../shared/sort'
import { searchable, searchRows, shownLine, visibleRows } from '../../domain/table'
import { FIELD, LABEL, SECONDARY, TABLE, TABLE_BOX, TD, TH } from './look'

/** A column: its key, its heading, and the value it sorts by; num right-aligns a count. */
export type Column<R> = { key: string; label: string; value?: (row: R) => SortValue; num?: boolean }

/**
 * A Manage table: every column with a value sorts, a long table searches, and it shows the first
 * twelve rows until Show all. It never runs past its box: headings and cells wrap, and a row's
 * buttons wrap in their cell. A cell is the column's slot, cell-<key>, or else its value.
 */
const props = withDefaults(
  defineProps<{ rows: readonly T[]; columns: readonly Column<T>[]; rowKey: (row: T) => string; label: string; searchText?: (row: T) => string; empty?: string }>(),
  { searchText: undefined, empty: 'Nothing here yet.' },
)

const id = useId()
const sort = ref<Sort<string>>(null)
const query = ref('')
const all = ref(false)
const canSearch = computed(() => props.searchText !== undefined && searchable(props.rows.length))
const found = computed(() => (canSearch.value && props.searchText ? searchRows(props.rows, query.value, props.searchText) : [...props.rows]))
const sorted = computed(() => sortRows(found.value, sort.value, (row, key) => props.columns.find((c) => c.key === key)?.value?.(row) ?? ''))
const shown = computed(() => visibleRows(sorted.value, all.value))
const line = computed(() => shownLine(shown.value.length, sorted.value.length))

function sortBy(c: Column<T>): void {
  sort.value = nextSort(sort.value, c.key, c.num ? 'desc' : 'asc')
}
</script>

<template>
  <div class="flex min-w-0 flex-col gap-3">
    <label v-if="canSearch" :for="`${id}-search`" :class="LABEL" class="max-w-md">
      Search
      <input :id="`${id}-search`" v-model="query" type="search" :class="FIELD" />
    </label>
    <p v-if="rows.length === 0" class="m-0 text-ink-2">{{ empty }}</p>
    <div v-else :class="TABLE_BOX">
      <table :class="TABLE" :aria-label="label">
        <thead>
          <tr>
            <th
              v-for="c in columns"
              :key="c.key"
              scope="col"
              :class="[TH, c.num ? 'text-right' : '']"
              :aria-sort="directionOf(sort, c.key) === 'asc' ? 'ascending' : directionOf(sort, c.key) === 'desc' ? 'descending' : undefined"
            >
              <button
                v-if="c.value"
                type="button"
                :aria-label="sortLabel(c.label, directionOf(sort, c.key))"
                class="inline cursor-pointer border-0 bg-transparent p-0 text-left font-[inherit] text-[inherit] hover:text-black"
                @click="sortBy(c)"
              >
                {{ c.label }}<span aria-hidden="true" class="ml-1">{{ directionOf(sort, c.key) === 'asc' ? '▲' : directionOf(sort, c.key) === 'desc' ? '▼' : '' }}</span>
              </button>
              <template v-else>{{ c.label }}</template>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in shown" :key="rowKey(row)">
            <td v-for="c in columns" :key="c.key" :class="[TD, c.num ? 'text-right tabular-nums' : '']">
              <slot :name="`cell-${c.key}`" :row="row">{{ c.value ? c.value(row) : '' }}</slot>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <p v-if="rows.length > 0 && sorted.length === 0" class="m-0 text-ink-2">No row matches the search.</p>
    <div v-if="line" class="flex flex-wrap items-center gap-3">
      <span class="text-sm text-ink-2">{{ line }}</span>
      <button type="button" :class="SECONDARY" @click="all = true">Show all</button>
    </div>
  </div>
</template>
