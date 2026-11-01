<script setup lang="ts">
import LinesChart from '../../../shared/ui/charts/LinesChart.vue'
import { useHistory } from '../composables/useHistory'
import type { Measure } from '../domain/history'
import VersionFilter from './VersionFilter.vue'

/**
 * A Prompt test measure over time, in a mist block: the Prompt set filter and its note, a line per
 * measure with the goal, the notes on breaks and partial runs, and the figures as a folded table.
 */
const props = withDefaults(defineProps<{ measures: readonly Measure[]; goal?: number; label?: string }>(), { goal: undefined, label: 'Over time' })
const h = useHistory(props.measures)
const cell = (v: number | undefined): string => (v === undefined ? '' : `${String(Math.round(v * 100))}%`)
</script>

<template>
  <div class="flex min-w-0 flex-col gap-4 rounded-3xl bg-mist p-4 sm:p-6">
    <p v-if="h.error.value" role="alert" class="m-0 text-orange-ink">{{ h.error.value }}</p>
    <p v-else-if="h.isLoading.value" class="m-0 text-muted">Loading the history…</p>
    <template v-else>
      <VersionFilter :options="h.options.value" :version="h.version.value" :note="h.note.value" @choose="h.choose" />
      <p v-if="!h.hasPoints.value" class="m-0 text-muted">No complete runs yet.</p>
      <template v-else>
        <LinesChart :series="h.series.value" :breaks="h.breaks.value" :goal="goal" :label="label" />
        <ul v-if="h.notes.value.length" data-part="notes" class="m-0 flex flex-col gap-1 pl-5 text-sm text-ink-2">
          <li v-for="n in h.notes.value" :key="n">{{ n }}</li>
        </ul>
        <details>
          <summary class="cursor-pointer font-medium text-orange-ink">Show the figures as a table</summary>
          <div class="mt-3 overflow-x-auto">
            <table class="w-full border-collapse text-left text-sm">
              <caption class="sr-only">{{ label }}, per run</caption>
              <thead>
                <tr class="border-b-2 border-black">
                  <th scope="col" class="py-2 pr-4 font-semibold">Date</th>
                  <th v-for="m in measures" :key="m.label" scope="col" class="py-2 pr-4 text-right font-semibold">{{ m.label }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in h.rows.value" :key="r.x" class="border-b border-grey">
                  <th scope="row" class="py-2 pr-4 font-normal">{{ r.when }}<template v-if="r.partial"> (partial run)</template></th>
                  <td v-for="(v, i) in r.values" :key="i" class="py-2 pr-4 text-right">{{ cell(v) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </details>
      </template>
    </template>
  </div>
</template>
