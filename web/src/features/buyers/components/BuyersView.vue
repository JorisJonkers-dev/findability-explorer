<script setup lang="ts">
import { computed, ref } from 'vue'
import { percent } from '../../../shared/percent'
import PillButton from '../../../shared/ui/PillButton.vue'
import PageView from '../../shell/components/PageView.vue'
import { buyerFigures, channelColumns, chartLabel, metric } from '../domain/channels'
import { CHANNELS, DEFAULT_METRIC, GOAL, METRICS, type MetricId, RECOMMENDATIONS, SOURCE, TABLE, TITLE } from '../domain/content'

const picked = ref<MetricId>(DEFAULT_METRIC)
const shown = computed(() => metric(picked.value))
const columns = computed(() => channelColumns(shown.value))
const figures = buyerFigures()

// Paid search, organic search, AI assistants: blue-grey, orange, black.
const COLUMN_FILL = ['bg-blue-grey', 'bg-orange', 'bg-black'] as const
</script>

<template>
  <PageView section="overview" eyebrow="Buyers" :back="{ to: '/', label: 'Back to Overview' }" :title="TITLE">
    <template #lede>
      <p class="m-0 text-muted">
        <template v-for="(line, index) in SOURCE" :key="line"><br v-if="index > 0" />{{ line }}</template>
      </p>
    </template>

    <section aria-label="Key figures" class="grid grid-cols-1 gap-5 md:grid-cols-3">
      <div v-for="f in figures" :key="f.big" class="flex flex-col gap-2.5 rounded-3xl bg-mist px-7 py-6">
        <span class="font-display text-5xl leading-none font-semibold tracking-[-0.02em]">{{ f.big }}</span>
        <span class="text-[17px] leading-snug font-semibold">{{ f.text }}</span>
      </div>
    </section>

    <div class="grid grid-cols-1 items-start gap-7 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)]">
      <section aria-labelledby="cmp-title" class="flex flex-col gap-4 rounded-[28px] border-2 border-solid border-grey p-7">
        <h2 id="cmp-title" class="m-0 font-display text-2xl font-semibold">Compare the channels</h2>
        <div role="group" aria-label="Measure" class="flex flex-wrap gap-2">
          <PillButton v-for="m in METRICS" :key="m.id" :pressed="picked === m.id" @click="picked = m.id">{{ m.label }}</PillButton>
        </div>
        <div role="img" :aria-label="chartLabel(shown)" class="grid h-72 grid-cols-3 items-stretch gap-6 border-b-[3px] border-black px-2 sm:gap-9 sm:px-4">
          <div v-for="(column, index) in columns" :key="column.channel" class="flex h-full flex-col justify-end gap-2">
            <span class="text-center font-display text-2xl font-semibold tracking-[-0.01em] sm:text-3xl">{{ column.value }}</span>
            <div
              class="rounded-t-lg motion-safe:transition-[height] motion-safe:duration-400 motion-safe:ease-out"
              :class="COLUMN_FILL[index]"
              :style="{ height: percent(column.fraction * 0.72) }"
            ></div>
          </div>
        </div>
        <div aria-hidden="true" class="grid grid-cols-3 gap-6 px-2 text-center text-sm font-bold sm:gap-9 sm:px-4 sm:text-base">
          <span v-for="channel in CHANNELS" :key="channel">{{ channel }}</span>
        </div>
        <p aria-live="polite" class="m-0 text-xl leading-tight font-bold">{{ shown.caption }}</p>
      </section>

      <div class="flex flex-col gap-5">
        <div class="overflow-x-auto">
          <table class="w-full border-collapse">
            <caption class="pb-2.5 text-left font-display text-xl font-semibold">Every figure</caption>
            <thead>
              <tr class="border-b-2 border-black text-right">
                <th scope="col" class="py-2.5 text-left font-medium text-muted">Measure</th>
                <th scope="col" class="py-2.5 pl-3">Paid</th>
                <th scope="col" class="py-2.5 pl-3">Organic</th>
                <th scope="col" class="py-2.5 pl-3">AI</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in TABLE" :key="row.measure" class="border-b border-grey text-right">
                <th scope="row" class="py-2.5 text-left font-normal">{{ row.measure }}</th>
                <td class="py-2.5 pl-3">{{ row.paid }}</td>
                <td class="py-2.5 pl-3 font-bold">{{ row.organic }}</td>
                <td class="py-2.5 pl-3">{{ row.ai }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <aside aria-labelledby="suggests-title" class="flex flex-col gap-2.5 rounded-[20px] bg-orange-pale px-6 py-5">
          <h2 id="suggests-title" class="m-0 text-sm font-bold tracking-[0.06em] text-orange-ink uppercase">What this export suggests</h2>
          <p v-for="r in RECOMMENDATIONS" :key="r.mark" class="m-0 leading-normal">
            <strong>{{ r.lead }}</strong> {{ r.text }}
          </p>
          <p class="m-0 text-ink-2">Goal {{ GOAL.goal }}, {{ GOAL.text.toLowerCase() }}: {{ GOAL.rows[0]?.value }} today, {{ GOAL.rows[1]?.value }} by the end of the year.</p>
        </aside>
      </div>
    </div>
  </PageView>
</template>
