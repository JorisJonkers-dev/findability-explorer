<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { companyPath } from '../../shell/domain/sections'
import { oxford, type RankBy, RANKINGS, rankRows, type Row, rowLabel, scoreText, shareText } from '../domain/compare'

/** Every competitor on the list, ranked, with Tribelt on top for comparison; the first few show until asked for the rest. */
const props = defineProps<{
  rows: readonly Row[]
  tribelt: Pick<Row, 'name' | 'share' | 'ready'> | undefined
  /** The group's own companies, which the list leaves out. */
  group: readonly string[]
  outOf: number
}>()

const FEW = 8
const by = ref<RankBy>('answers')
const all = ref(false)
const ranked = computed(() => rankRows(props.rows, by.value))
const shown = computed(() => (all.value ? ranked.value : ranked.value.slice(0, FEW)))
const COLUMNS = 'grid grid-cols-[minmax(0,1fr)_5.5rem_5.5rem_1.5rem] items-center gap-3 stage:grid-cols-[minmax(0,1fr)_320px_320px_48px] stage:gap-8 sm:grid-cols-[minmax(0,1fr)_12rem_12rem_1.5rem]'
</script>

<template>
  <section aria-labelledby="list-title" class="flex flex-col gap-4 stage:gap-8">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h2 id="list-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">The competitors</h2>
      <div role="group" aria-label="Rank by" class="flex flex-wrap gap-1.5 stage:gap-3">
        <button
          v-for="r in RANKINGS"
          :key="r.value"
          type="button"
          :aria-pressed="by === r.value"
          class="inline-flex min-h-11 cursor-pointer items-center rounded-full border-2 px-4 text-sm font-semibold stage:min-h-16 stage:px-7 stage:text-[24px]"
          :class="by === r.value ? 'border-black bg-black text-white' : 'border-silver bg-white text-black hover:border-black'"
          @click="by = r.value"
        >
          {{ r.label }}
        </button>
      </div>
    </div>
    <div aria-hidden="true" :class="COLUMNS" class="border-b-2 border-black px-2 pb-2 text-sm font-medium text-muted stage:text-[24px]">
      <span>Company</span><span class="text-right">AI answers that name it</span><span class="text-right">AI readiness checks</span><span></span>
    </div>
    <ol class="m-0 flex list-none flex-col p-0">
      <li v-if="tribelt" class="border-b border-grey bg-orange-pale">
        <div :class="COLUMNS" class="min-h-14 px-2 py-2 stage:min-h-24">
          <span class="flex flex-col"><span class="font-bold">{{ tribelt.name }}</span><span class="text-xs text-ink-2 stage:text-[20px]">For comparison</span></span>
          <span class="text-right text-lg font-extrabold tabular-nums">{{ shareText(tribelt.share) }}</span>
          <span class="text-right text-lg font-extrabold tabular-nums">{{ scoreText(tribelt.ready, outOf) }}</span>
          <span></span>
        </div>
      </li>
      <li v-for="r in shown" :key="r.id" class="border-b border-grey">
        <RouterLink :to="companyPath(r.id)" :aria-label="rowLabel(r, outOf)" :class="COLUMNS" class="min-h-14 px-2 py-2 text-black no-underline hover:bg-mist stage:min-h-24">
          <span class="font-semibold">{{ r.name }}</span>
          <span class="text-right tabular-nums" :class="r.share === undefined ? 'text-sm font-semibold text-muted' : 'text-lg font-extrabold'">{{ shareText(r.share) }}</span>
          <span class="text-right tabular-nums" :class="r.ready === undefined ? 'text-sm font-semibold text-muted' : 'text-lg font-extrabold'">{{ scoreText(r.ready, outOf) }}</span>
          <svg aria-hidden="true" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 5l7 7-7 7" /></svg>
        </RouterLink>
      </li>
    </ol>
    <button
      v-if="rows.length > FEW"
      type="button"
      :aria-expanded="all"
      class="min-h-11 cursor-pointer self-start rounded-full border-2 border-silver bg-white px-5 font-semibold hover:border-black stage:min-h-16 stage:px-8 stage:text-[26px]"
      @click="all = !all"
    >
      {{ all ? `Show only the first ${String(FEW)}` : `Show all ${String(rows.length)} competitors` }}
    </button>
    <p v-if="group.length > 0" class="m-0 text-sm text-ink-2 stage:text-[26px]">
      {{ oxford(group) }} {{ group.length === 1 ? 'is' : 'are' }} left out: {{ group.length === 1 ? 'it is a group company' : 'they are group companies' }} of Tribelt, not {{ group.length === 1 ? 'a competitor' : 'competitors' }}.
    </p>
  </section>
</template>
