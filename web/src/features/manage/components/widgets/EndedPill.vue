<script setup lang="ts">
import { computed } from 'vue'
import { type Ending, endingLabel, endingTone } from '../../../../shared/endings'

/** How a piece of work ended, as a pill; Running while it runs. */
const props = defineProps<{ ended: Ending | null }>()
const TONES = {
  good: 'bg-white text-black',
  warn: 'bg-orange-pale text-orange-ink',
  bad: 'bg-black text-white',
  live: 'bg-orange text-black',
} as const
const tone = computed(() => TONES[props.ended === null ? 'live' : endingTone(props.ended)])
</script>

<template>
  <span class="inline-block rounded-full px-2.5 py-0.5 text-sm font-semibold whitespace-nowrap" :class="tone">{{ ended === null ? 'Running' : endingLabel(ended) }}</span>
</template>
