<script setup lang="ts">
import { computed } from 'vue'
import { type Ended, endedTone } from '../../domain/activity'

/** How a piece of work ended, as a pill; Running while it runs. */
const props = defineProps<{ ended: Ended | null }>()
const TONES = {
  good: 'bg-white text-black',
  warn: 'bg-orange-pale text-orange-ink',
  bad: 'bg-black text-white',
  live: 'bg-orange text-black',
} as const
const tone = computed(() => TONES[props.ended === null ? 'live' : endedTone(props.ended)])
</script>

<template>
  <span class="inline-block rounded-full px-2.5 py-0.5 text-sm font-semibold whitespace-nowrap" :class="tone">{{ ended ?? 'Running' }}</span>
</template>
