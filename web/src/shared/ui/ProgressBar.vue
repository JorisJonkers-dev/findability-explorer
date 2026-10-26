<script setup lang="ts">
import { computed } from 'vue'
import { percent } from '../percent'

/**
 * A thin progress bar. fraction is from 0 to 1, or null while the work waits to start: then the
 * whole belt runs, dimmed. label names the bar; text says where it is, for screen readers.
 */
const props = withDefaults(defineProps<{ fraction: number | null; label: string; text: string; thick?: boolean }>(), { thick: false })
const width = computed(() => (props.fraction === null ? '100%' : percent(Math.max(props.fraction, 0.02))))
</script>

<template>
  <div
    role="progressbar"
    :aria-label="label"
    :aria-valuemin="0"
    :aria-valuemax="100"
    :aria-valuenow="fraction === null ? undefined : Math.round(fraction * 100)"
    :aria-valuetext="text"
    class="overflow-hidden rounded-full bg-grey"
    :class="thick ? 'h-3 stage:h-5' : 'h-1.5 stage:h-3'"
  >
    <div class="belt h-full rounded-full" :class="{ 'opacity-40': fraction === null }" :style="{ width }"></div>
  </div>
</template>
