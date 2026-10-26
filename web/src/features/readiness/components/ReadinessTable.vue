<script setup lang="ts">
import { vScrollHint } from '../../../shared/scrollHint'
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import type { AiReadinessRow } from '../../../infrastructure/api/types.gen'
import { directionOf, nextSort, type Sort, type SortDirection } from '../../../shared/sort'
import SortButton from '../../../shared/ui/SortButton.vue'
import { BAND_LABELS, cell, COLUMNS, type Gap, GAPS, type ReadinessColumn, tableRows } from '../domain/readiness'
import ReadinessCell from './ReadinessCell.vue'

const props = defineProps<{ rows: readonly AiReadinessRow[] }>()

// Sorted by a column while one is picked; Tribelt and the mirror stay on top whatever the sort.
const sort = ref<Sort<ReadinessColumn>>(null)
const gap = ref<Gap | ''>('')
const bands = computed(() => tableRows(props.rows, sort.value, gap.value === '' ? null : gap.value))
const shown = computed(() => bands.value.reduce((n, b) => n + b.rows.length, 0))

function sortBy(column: ReadinessColumn, first: SortDirection): void {
  sort.value = nextSort(sort.value, column, first)
}
</script>

<template>
  <div class="flex flex-col gap-4 stage:min-h-0 stage:gap-7">
    <div class="flex flex-wrap items-end gap-4 stage:gap-8">
      <label class="flex flex-col gap-1.5 text-sm text-ink-2 stage:gap-2 stage:text-[26px]">
        Show the competitors with a gap
        <select
          v-model="gap"
          class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base text-black stage:h-[76px] stage:border-[3px] stage:px-5 stage:text-[28px]"
        >
          <option value="">All companies</option>
          <option v-for="g in GAPS" :key="g.value" :value="g.value">{{ g.label }}</option>
        </select>
      </label>
      <p class="m-0 flex flex-wrap gap-x-6 gap-y-1 text-sm text-muted stage:gap-x-8 stage:text-[24px]">
        <span class="inline-flex items-center gap-2 stage:gap-3"><span aria-hidden="true" class="size-3 rounded-full bg-orange stage:size-5"></span>Follows the report's advice</span>
        <span class="inline-flex items-center gap-2 stage:gap-3"><span aria-hidden="true" class="size-3 rounded-full border-2 border-orange-ink stage:size-5"></span>A gap</span>
        <span aria-live="polite">{{ shown }} of {{ rows.length }} companies</span>
      </p>
    </div>
    <div v-scroll-hint class="scroll-hint overflow-auto stage:min-h-0 stage:flex-1">
      <table class="w-max min-w-full border-collapse text-sm stage:text-[24px]">
        <caption class="sr-only">AI readiness of Tribelt, the mirror, Tribelt's group and the competitors, per search and AI crawler and signal</caption>
        <thead>
          <tr class="border-b-2 border-black stage:border-b-[3px]">
            <th
              v-for="column in COLUMNS"
              :key="column.key"
              scope="col"
              class="bg-white px-2 py-2 text-left align-bottom font-semibold first:sticky first:left-0 first:z-10 stage:px-4 stage:py-4"
            >
              <SortButton :label="column.label" :direction="directionOf(sort, column.key)" @sort="sortBy(column.key, column.first)">{{ column.short }}</SortButton>
            </th>
          </tr>
        </thead>
        <tbody v-for="band in bands" :key="band.band" :aria-label="BAND_LABELS[band.band]">
          <tr>
            <th :colspan="COLUMNS.length" scope="colgroup" class="sticky left-0 bg-mist px-2 py-1.5 text-left text-xs font-semibold tracking-[0.06em] text-orange-ink uppercase stage:px-4 stage:py-3 stage:text-[22px]">
              {{ BAND_LABELS[band.band] }}
            </th>
          </tr>
          <tr v-for="row in band.rows" :key="row.competitorId" class="border-b border-grey">
            <th scope="row" class="sticky left-0 z-10 bg-white px-2 py-2 text-left font-medium stage:px-4 stage:py-3.5">
              <RouterLink :to="{ path: '/competitors', query: { company: row.competitorId } }" class="whitespace-nowrap text-black">{{ row.name }}</RouterLink>
            </th>
            <td v-for="column in COLUMNS.slice(1)" :key="column.key" class="px-2 py-2 stage:px-4 stage:py-3.5">
              <ReadinessCell :cell="cell(row, column.key)" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
