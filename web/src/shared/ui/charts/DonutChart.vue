<script setup lang="ts">
import { computed } from 'vue'
import { type Tone, TONE_BG, TONE_VAR } from './tones'

export type DonutSegment = { label: string; value: number; tone: Tone }

const props = defineProps<{ segments: readonly DonutSegment[]; centre: string; centreSub?: string }>()

const R = 80
const CIRC = 2 * Math.PI * R
const total = computed(() => props.segments.reduce((sum, s) => sum + Math.max(0, s.value), 0))
const arcs = computed(() => {
  let offset = 0
  return props.segments.map((s) => {
    const len = total.value > 0 ? (Math.max(0, s.value) / total.value) * CIRC : 0
    const arc = { ...s, dash: `${String(len)} ${String(CIRC - len)}`, offset: -offset }
    offset += len
    return arc
  })
})
const label = computed(() => `${props.centre}${props.centreSub ? ` ${props.centreSub}` : ''}. ${props.segments.map((s) => `${s.label}: ${String(s.value)}`).join('. ')}.`)
</script>

<template>
  <div class="flex flex-wrap items-center gap-x-8 gap-y-4">
    <svg role="img" :aria-label="label" viewBox="0 0 200 200" class="size-[220px] max-w-full flex-none">
      <circle cx="100" cy="100" :r="R" fill="none" stroke="var(--color-grey)" stroke-width="28" />
      <circle
        v-for="a in arcs"
        :key="a.label"
        cx="100"
        cy="100"
        :r="R"
        fill="none"
        stroke-width="28"
        transform="rotate(-90 100 100)"
        :stroke="TONE_VAR[a.tone]"
        :stroke-dasharray="a.dash"
        :stroke-dashoffset="a.offset"
      />
      <text x="100" :y="centreSub ? 100 : 108" text-anchor="middle" font-size="40" font-weight="800" fill="currentColor">{{ centre }}</text>
      <text v-if="centreSub" x="100" y="126" text-anchor="middle" font-size="14" fill="var(--color-ink-2)">{{ centreSub }}</text>
    </svg>
    <ul class="m-0 flex list-none flex-col gap-2 p-0">
      <li v-for="s in segments" :key="s.label" class="inline-flex items-center gap-2.5">
        <span aria-hidden="true" class="size-3.5 rounded-[4px]" :class="TONE_BG[s.tone]"></span>{{ s.label }}
      </li>
    </ul>
  </div>
</template>
