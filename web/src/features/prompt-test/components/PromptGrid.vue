<script setup lang="ts">
import { computed } from 'vue'
import type { PromptAnswer, PromptMode, PromptRun, PromptScope } from '../../../infrastructure/api/types.gen'
import { byKey, type Column, columnsOf, keyOf, MODE_FILTERS, PLACE_ORDER, PLACES, placeOf, type Row, rowsOf, SCOPE_FILTERS, tribeltPosition } from '../domain/grid'
import { familyName, modeName, scopeName } from '../domain/results'

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
const SHORT_MODE: Record<PromptMode, string> = { knowledge: 'Own', web: 'Web' }

function cellOf(row: Row, column: Column) {
  const key = keyOf({ promptId: row.promptId, provider: column.provider, mode: column.mode })
  const answer = found.value.get(key)
  const place = placeOf(answer)
  const position = answer ? tribeltPosition(answer) : undefined
  const where = `${familyName(row.family)}, ${scopeName(row.scope)}, ${column.name} ${modeName(column.mode).toLowerCase()}`
  const what = place === 'top3' || (place === 'later' && position !== undefined) ? `Tribelt named ${String(position)}` : PLACES[place].label
  return { key, place, mark: PLACES[place].mark(position), label: `${where}: ${what}` }
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
    <ul aria-label="Legend" class="m-0 flex list-none flex-wrap gap-x-5 gap-y-2 p-0 text-sm text-ink-2 stage:gap-x-8 stage:text-[24px]">
      <li v-for="p in PLACE_ORDER" :key="p" class="inline-flex items-center gap-2">
        <span aria-hidden="true" class="inline-flex size-6 items-center justify-center rounded border-2 text-xs font-semibold stage:size-9 stage:text-[20px]" :class="PLACES[p].cell">{{
          PLACES[p].mark(2)
        }}</span>
        {{ PLACES[p].label }}
      </li>
    </ul>
    <div class="overflow-x-auto stage:min-h-0 stage:overflow-y-auto">
      <table class="border-separate border-spacing-1 text-sm stage:border-spacing-2 stage:text-[24px]">
        <caption class="sr-only">Tribelt's place in each answer, by question, provider and search mode</caption>
        <thead>
          <tr>
            <th scope="col" class="text-left font-medium text-muted">Question</th>
            <th
              v-for="p in providers"
              :key="p.provider"
              scope="colgroup"
              :colspan="mode === 'all' ? 2 : 1"
              class="px-1 text-center font-semibold whitespace-nowrap"
            >
              {{ p.name }}<span class="block text-xs font-normal text-muted stage:text-[18px]">{{ p.model.split('/').at(-1) }}</span>
            </th>
          </tr>
          <tr>
            <td></td>
            <th v-for="c in columns" :key="`${c.provider}-${c.mode}`" scope="col" class="text-center text-xs font-medium text-muted stage:text-[20px]">
              {{ SHORT_MODE[c.mode] }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="g in grid" :key="g.row.promptId">
            <th scope="row" class="pr-3 text-left font-medium whitespace-nowrap" :title="g.row.question">
              {{ familyName(g.row.family) }}<span class="block text-xs font-normal text-muted stage:text-[20px]">{{ scopeName(g.row.scope) }}</span>
            </th>
            <td v-for="cell in g.cells" :key="cell.key" class="p-0">
              <button
                type="button"
                :aria-label="cell.label"
                :aria-pressed="selected === cell.key"
                class="flex size-9 items-center justify-center rounded border-2 font-semibold stage:size-16 stage:rounded-lg stage:text-[24px]"
                :class="[PLACES[cell.place].cell, selected === cell.key ? 'outline-3 outline-offset-2 outline-black' : '']"
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
