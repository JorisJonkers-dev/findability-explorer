<script setup lang="ts">
import { computed } from 'vue'
import type { AnalysisSummary } from '../../../infrastructure/api/types.gen'
import { progress } from '../domain/status'

const props = defineProps<{ summary: AnalysisSummary }>()
const state = computed(() => progress(props.summary))
// While the scan waits its turn the whole belt runs, dimmed; then it fills as pages are read.
const width = computed(() => (state.value.fraction === null ? '100%' : `${String(Math.max(state.value.fraction * 100, 4))}%`))
</script>

<template>
  <div class="flex max-w-[980px] flex-col gap-2.5">
    <p class="m-0 text-[15px] text-ink-2" aria-live="polite">{{ state.label }}…</p>
    <div
      role="progressbar"
      aria-label="Scan progress"
      :aria-valuemin="0"
      :aria-valuemax="summary.maxPages"
      :aria-valuenow="state.fraction === null ? undefined : summary.pagesCrawled"
      :aria-valuetext="state.label"
      class="h-3.5 overflow-hidden rounded-[7px] bg-raised"
    >
      <div class="belt h-full rounded-[7px]" :class="{ 'opacity-40': state.fraction === null }" :style="{ width }"></div>
    </div>
  </div>
</template>
