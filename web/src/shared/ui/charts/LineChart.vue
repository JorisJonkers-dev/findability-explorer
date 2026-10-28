<script setup lang="ts">
import { computed } from 'vue'

export type LinePoint = { x: number; y: number }

const props = defineProps<{ points: readonly LinePoint[]; target?: number; label?: string; format?: (y: number) => string }>()

const W = 600
const H = 240
const PAD = { l: 50, r: 20, t: 16, b: 28 }
const fmt = (y: number): string => (props.format ? props.format(y) : String(y))

const xs = computed(() => props.points.map((p) => p.x))
const top = computed(() => Math.max(...props.points.map((p) => p.y), props.target ?? 0, 0.0001))
const sx = (x: number): number => {
  const lo = Math.min(...xs.value)
  const hi = Math.max(...xs.value)
  return PAD.l + (hi === lo ? 0.5 : (x - lo) / (hi - lo)) * (W - PAD.l - PAD.r)
}
const sy = (y: number): number => H - PAD.b - (y / top.value) * (H - PAD.t - PAD.b)
const path = computed(() => props.points.map((p, i) => `${i === 0 ? 'M' : 'L'}${sx(p.x).toFixed(1)} ${sy(p.y).toFixed(1)}`).join(' '))
const aria = computed(() => `${props.label ?? 'Line chart'}. ${props.points.map((p) => `${String(p.x)}: ${fmt(p.y)}`).join('. ')}.${props.target === undefined ? '' : ` Target ${fmt(props.target)}.`}`)
</script>

<template>
  <svg role="img" :aria-label="aria" :viewBox="`0 0 ${String(W)} ${String(H)}`" class="block h-auto w-full">
    <line :x1="PAD.l" :x2="W - PAD.r" :y1="sy(0)" :y2="sy(0)" stroke="var(--color-silver)" stroke-width="2" />
    <text :x="PAD.l - 8" :y="sy(0) + 4" text-anchor="end" font-size="12" fill="var(--color-muted)">{{ fmt(0) }}</text>
    <text :x="PAD.l - 8" :y="sy(top) + 4" text-anchor="end" font-size="12" fill="var(--color-muted)">{{ fmt(top) }}</text>
    <line v-if="target !== undefined" :x1="PAD.l" :x2="W - PAD.r" :y1="sy(target)" :y2="sy(target)" stroke="var(--color-orange-ink)" stroke-width="2.5" stroke-dasharray="10 8" />
    <path v-if="points.length > 1" :d="path" fill="none" stroke="var(--color-orange)" stroke-width="4" stroke-linecap="round" stroke-linejoin="round" />
    <circle v-for="p in points" :key="p.x" :cx="sx(p.x)" :cy="sy(p.y)" r="5" fill="var(--color-orange)" stroke="#fff" stroke-width="2" />
  </svg>
</template>
