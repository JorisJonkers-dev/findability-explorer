<script setup lang="ts">
import { computed } from 'vue'
import type { AnalysisSummary } from '../../../infrastructure/api/types.gen'
import { percent } from '../../../shared/percent'
import { siteHost } from '../domain/results'
import { progress } from '../domain/status'

const props = defineProps<{ summary: AnalysisSummary }>()
const state = computed(() => progress(props.summary))
// While the scan waits its turn the whole belt runs, dimmed; then it fills as pages are read.
const width = computed(() => (state.value.fraction === null ? '100%' : percent(Math.max(state.value.fraction, 0.04))))
</script>

<template>
  <div class="flex flex-col gap-3 rounded-2xl bg-white p-5 stage:gap-4 stage:rounded-[28px] stage:p-9">
    <p aria-live="polite" class="m-0 text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[24px]">{{ state.label }}</p>
    <p class="m-0 font-display text-2xl font-semibold stage:text-[56px]">{{ siteHost(summary.url) }}</p>
    <p class="m-0 text-sm text-ink-2 stage:text-[28px]">
      The scan reads up to {{ summary.maxPages }} pages, respecting robots.txt, then ranks the phrases and matches the products.
    </p>
    <div
      role="progressbar"
      aria-label="Scan progress"
      :aria-valuemin="0"
      :aria-valuemax="summary.maxPages"
      :aria-valuenow="state.fraction === null ? undefined : summary.pagesCrawled"
      :aria-valuetext="state.label"
      class="h-3 overflow-hidden rounded-full bg-grey stage:h-5"
    >
      <div class="belt h-full rounded-full" :class="{ 'opacity-40': state.fraction === null }" :style="{ width }"></div>
    </div>
  </div>
</template>
