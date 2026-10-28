<script setup lang="ts">
import { computed } from 'vue'
import type { PromptAnswer, PromptMode, PromptRun, PromptScope } from '../../../infrastructure/api/types.gen'
import { byKey, cellLabel, gridFamily, gridScope, type Column, columnsOf, keyOf, LEGEND, markOf, MODE_COLUMNS, MODE_FILTERS, PLACES, type Row, rowsOf, SCOPE_FILTERS, SQUARE, squareOf } from '../domain/grid'

const props = defineProps<{
  run: PromptRun
  answers: readonly PromptAnswer[]
  mode: PromptMode | 'all'
  scope: PromptScope | 'all'
  selected: string | undefined
}>()
const emit = defineEmits<{ mode: [PromptMode | 'all']; scope: [PromptScope | 'all']; select: [string] }>()

const columns = computed(() => columnsOf(props.run, props.mode))
const providers = computed(() => columnsOf(props.run, 'knowledge'))
const found = computed(() => byKey(props.answers))
const grid = computed(() => rowsOf(props.answers, props.scope).map((row) => ({ row, cells: columns.value.map((c) => cellOf(row, c)) })))

function cellOf(row: Row, column: Column) {
  const key = keyOf({ promptId: row.promptId, provider: column.provider, mode: column.mode })
  const square = squareOf(found.value.get(key))
  return { key, place: square.place, mark: markOf(square), label: cellLabel(row, column, square) }
}
</script>

<template>
  <div class="flex flex-col gap-4 stage:min-h-0 stage:gap-6">
    <div class="flex flex-wrap gap-x-8 gap-y-3">
      <div role="group" aria-label="Search mode" class="flex flex-wrap gap-1.5">
        <button
          v-for="f in MODE_FILTERS"
          :key="f.value"
          type="button"
          :aria-pressed="mode === f.value"
          class="inline-flex min-h-11 items-center rounded-full border-2 px-4 text-sm font-medium stage:min-h-16 stage:px-7 stage:text-[26px]"
          :class="mode === f.value ? 'border-black bg-black text-white' : 'border-silver bg-white text-ink-2 hover:border-black'"
          @click="emit('mode', f.value)"
        >
          {{ f.label }}
        </button>
      </div>
      <div role="group" aria-label="Where the buyer asks" class="flex flex-wrap gap-1.5">
        <button
          v-for="f in SCOPE_FILTERS"
          :key="f.value"
          type="button"
          :aria-pressed="scope === f.value"
          class="inline-flex min-h-11 items-center rounded-full border-2 px-4 text-sm font-medium stage:min-h-16 stage:px-7 stage:text-[26px]"
          :class="scope === f.value ? 'border-black bg-black text-white' : 'border-silver bg-white text-ink-2 hover:border-black'"
          @click="emit('scope', f.value)"
        >
          {{ f.label }}
        </button>
      </div>
    </div>
    <ul aria-label="What the squares mean" class="m-0 flex list-none flex-wrap items-center gap-x-5 gap-y-2 p-0 text-sm text-ink-2 stage:gap-x-8 stage:text-[24px]">
      <li v-for="s in LEGEND" :key="s.place" class="inline-flex items-center gap-2">
        <span aria-hidden="true" :class="[SQUARE, PLACES[s.place].cell]">{{ markOf(s) }}</span>
        {{ PLACES[s.place].label }}
      </li>
    </ul>
    <div class="overflow-x-auto stage:min-h-0 stage:overflow-y-auto">
      <!-- Fixed layout: every answer column is equally wide, however long a model's name. -->
      <table class="w-full min-w-[640px] table-fixed border-separate border-spacing-1 text-sm stage:min-w-0 stage:border-spacing-2 stage:text-[24px]">
        <caption class="sr-only">Where each answer names Tribelt, by question, provider and search mode</caption>
        <colgroup>
          <col class="w-44 stage:w-[400px]" />
          <col v-for="c in columns" :key="`${c.provider}-${c.mode}`" />
        </colgroup>
        <thead>
          <tr>
            <th scope="col" class="text-left font-medium text-muted">Question</th>
            <th
              v-for="p in providers"
              :key="p.provider"
              scope="colgroup"
              :colspan="mode === 'all' ? 2 : 1"
              class="px-1 text-center font-semibold whitespace-nowrap"
              :title="p.model"
            >
              <span class="block truncate">{{ p.name }}</span><span class="block truncate text-xs font-normal text-muted stage:text-[18px]">{{ p.model.split('/').at(-1) }}</span>
            </th>
          </tr>
          <tr>
            <td></td>
            <th v-for="c in columns" :key="`${c.provider}-${c.mode}`" scope="col" class="text-center text-xs font-medium text-muted stage:text-[20px]" :title="MODE_COLUMNS[c.mode].long">
              <span aria-hidden="true">{{ MODE_COLUMNS[c.mode].short }}</span><span class="sr-only">{{ MODE_COLUMNS[c.mode].long }}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="g in grid" :key="g.row.promptId">
            <th scope="row" class="truncate pr-3 text-left font-medium whitespace-nowrap" :title="g.row.question">
              {{ gridFamily(g.row.family) }}<span class="block truncate text-xs font-normal text-muted stage:text-[20px]">{{ gridScope(g.row.scope) }}</span>
            </th>
            <td v-for="cell in g.cells" :key="cell.key" class="p-0">
              <button
                type="button"
                :aria-label="cell.label"
                :title="cell.label"
                :aria-pressed="selected === cell.key"
                :class="[SQUARE, PLACES[cell.place].cell, selected === cell.key ? 'outline-3 outline-offset-2 outline-black' : '']"
                @click="emit('select', cell.key)"
              >
                <span aria-hidden="true">{{ cell.mark }}</span>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
