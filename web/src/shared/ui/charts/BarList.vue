<script setup lang="ts">
import { computed } from 'vue'
import { type Tone, TONE_BG } from './tones'

export type BarRow = { label: string; value: number; display: string; tone: Tone; highlight?: boolean; leader?: number }

const props = defineProps<{ rows: readonly BarRow[]; max: number; target?: number; label?: string }>()

const at = (v: number): number => (props.max > 0 ? Math.min(100, Math.max(0, (v / props.max) * 100)) : 0)
const aria = computed(
  () => `${props.label ?? 'Bars'}. ${props.rows.map((r) => `${r.label}: ${r.display}`).join('. ')}.${props.target === undefined ? '' : ` Target ${String(props.target)}.`}`,
)
</script>

<template>
  <ul role="img" :aria-label="aria" class="m-0 flex list-none flex-col gap-3 p-0">
    <li v-for="r in rows" :key="r.label" class="grid grid-cols-[minmax(0,9rem)_minmax(0,1fr)_auto] items-center gap-3">
      <span class="truncate text-sm" :class="r.highlight ? 'font-bold text-black' : 'text-ink-2'">{{ r.label }}</span>
      <span class="relative h-[22px]">
        <span class="absolute inset-x-0 top-1.5 h-2.5 rounded-[5px] bg-grey"></span>
        <span class="absolute top-1.5 left-0 h-2.5 rounded-[5px]" :class="TONE_BG[r.tone]" :style="{ width: `${String(at(r.value))}%` }"></span>
        <span v-if="target !== undefined" class="absolute top-0 h-[22px] w-[3px] bg-black" :style="{ left: `calc(${String(at(target))}% - 1px)` }"></span>
        <span
          v-if="r.leader !== undefined"
          class="absolute top-1 size-3.5 rounded-full border-2 border-solid border-white bg-blue-grey"
          :style="{ left: `calc(${String(at(r.leader))}% - 7px)` }"
        ></span>
      </span>
      <span class="text-sm tabular-nums" :class="r.highlight ? 'font-bold text-black' : 'text-ink-2'">{{ r.display }}</span>
    </li>
  </ul>
</template>
