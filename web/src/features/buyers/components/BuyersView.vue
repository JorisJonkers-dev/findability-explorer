<script setup lang="ts">
import { computed, ref } from 'vue'
import mesh from '../../../assets/mesh.webp'
import { percent } from '../../../shared/percent'
import GoalCard from '../../../shared/ui/GoalCard.vue'
import RoundedPhoto from '../../../shared/ui/RoundedPhoto.vue'
import PillButton from '../../../shared/ui/PillButton.vue'
import RecommendationList from '../../../shared/ui/RecommendationList.vue'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import SectionView from '../../shell/components/SectionView.vue'
import { channelColumns, chartLabel, metric, volumeAgainstValue } from '../domain/channels'
import { CHANNELS, DEFAULT_METRIC, GOAL, METRICS, type MetricId, RECOMMENDATIONS, SOURCE, TABLE, TITLE } from '../domain/content'

const picked = ref<MetricId>(DEFAULT_METRIC)
const shown = computed(() => metric(picked.value))
const columns = computed(() => channelColumns(shown.value))
const volume = volumeAgainstValue()

// Paid search, organic search, AI assistants: blue-grey, orange, black.
const COLUMN_FILL = ['bg-blue-grey', 'bg-orange', 'bg-black'] as const
</script>

<template>
  <SectionView section="buyers" :title="TITLE" :source="SOURCE">
    <div class="flex flex-col gap-14 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)_minmax(0,0.8fr)] stage:gap-24">
      <section aria-labelledby="cmp-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-10">
        <SectionHeading id="cmp-title">Compare the channels</SectionHeading>
        <div role="group" aria-label="Measure" class="flex flex-wrap gap-2 stage:gap-4">
          <PillButton v-for="m in METRICS" :key="m.id" :pressed="picked === m.id" @click="picked = m.id">{{ m.label }}</PillButton>
        </div>
        <div
          role="img"
          :aria-label="chartLabel(shown)"
          class="grid h-64 grid-cols-3 items-stretch gap-6 border-b-2 border-black px-2 stage:h-auto stage:min-h-0 stage:flex-1 stage:gap-18 stage:border-b-[3px] stage:px-6"
        >
          <div v-for="(column, index) in columns" :key="column.channel" class="flex h-full flex-col justify-end gap-2 stage:gap-5">
            <span class="text-center font-display text-2xl leading-none font-semibold tracking-[-0.02em] stage:text-[88px]">{{ column.value }}</span>
            <div
              class="motion-safe:transition-[height] motion-safe:duration-400 motion-safe:ease-out"
              :class="COLUMN_FILL[index]"
              :style="{ height: percent(column.fraction * 0.8) }"
            ></div>
          </div>
        </div>
        <div aria-hidden="true" class="-mt-2 grid grid-cols-3 gap-6 px-2 text-center text-sm font-semibold stage:-mt-6 stage:gap-18 stage:px-6 stage:text-[32px]">
          <span v-for="channel in CHANNELS" :key="channel">{{ channel }}</span>
        </div>
        <p aria-live="polite" class="m-0 text-xl leading-tight font-medium stage:text-[44px] stage:leading-[1.25]">{{ shown.caption }}</p>
      </section>

      <section aria-labelledby="vv-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-10">
        <SectionHeading id="vv-title">Volume against value</SectionHeading>
        <p class="m-0 text-ink-2 stage:text-[30px]">Of the visitors from these three channels: who brings the visits, and who brings the contacts.</p>
        <div class="flex flex-col gap-5 stage:gap-9">
          <div v-for="row in volume" :key="row.channel" class="flex flex-col gap-2 stage:gap-3">
            <span class="font-semibold stage:text-[32px]">{{ row.channel }}</span>
            <dl class="m-0 grid grid-cols-[4.5rem_minmax(0,1fr)_3.5rem] items-center gap-x-3 gap-y-2 text-sm stage:grid-cols-[180px_minmax(0,1fr)_140px] stage:gap-x-5 stage:gap-y-3 stage:text-[26px]">
              <dt class="text-muted">Visits</dt>
              <dd aria-hidden="true" class="m-0 h-4 bg-mist stage:h-[30px]">
                <span class="block h-full bg-blue-grey" :style="{ width: percent(row.visitsFraction) }"></span>
              </dd>
              <dd class="m-0 text-right font-semibold">{{ row.visits }}</dd>
              <dt class="text-muted">Contacts</dt>
              <dd aria-hidden="true" class="m-0 h-4 bg-mist stage:h-[30px]">
                <span class="block h-full bg-orange" :style="{ width: percent(row.contactsFraction) }"></span>
              </dd>
              <dd class="m-0 text-right font-semibold">{{ row.contacts }}</dd>
            </dl>
          </div>
        </div>
        <div class="overflow-x-auto stage:mt-auto stage:overflow-visible">
          <table class="w-full border-collapse text-sm stage:text-[28px]">
            <caption class="pb-2 text-left text-muted stage:pb-4 stage:text-[26px]">All figures</caption>
            <thead>
              <tr class="border-b-2 border-black text-right stage:border-b-[3px]">
                <th scope="col" class="py-2 text-left font-medium text-muted stage:py-3.5">Measure</th>
                <th scope="col" class="py-2 pl-3 stage:py-3.5 stage:pl-6">Paid</th>
                <th scope="col" class="py-2 pl-3 stage:py-3.5 stage:pl-6">Organic</th>
                <th scope="col" class="py-2 pl-3 stage:py-3.5 stage:pl-6">AI</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in TABLE" :key="row.measure" class="border-b border-grey text-right">
                <th scope="row" class="py-2 text-left font-normal stage:py-3.5">{{ row.measure }}</th>
                <td class="py-2 pl-3 stage:py-3.5 stage:pl-6">{{ row.paid }}</td>
                <td class="py-2 pl-3 font-semibold stage:py-3.5 stage:pl-6">{{ row.organic }}</td>
                <td class="py-2 pl-3 stage:py-3.5 stage:pl-6">{{ row.ai }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <aside aria-labelledby="so-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-10">
        <RoundedPhoto :src="mesh" alt="Close-up of a Tribelt metal mesh belt" class="stage:h-[560px]" />
        <SectionHeading id="so-title">So we recommend</SectionHeading>
        <RecommendationList :items="RECOMMENDATIONS" />
        <GoalCard :goal="GOAL.goal" :text="GOAL.text" :rows="GOAL.rows" class="stage:mt-auto" />
      </aside>
    </div>
  </SectionView>
</template>
