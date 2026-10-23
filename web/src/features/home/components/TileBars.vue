<script setup lang="ts">
import { percent } from '../../../shared/percent'
import type { TileBar } from '../domain/tiles'

defineProps<{
  bars: readonly TileBar[]
  /** The first part's fill. */
  fill: 'orange' | 'blue-grey'
  /** A label to set in bold, such as the crawler the tile is about. */
  strong?: string
  /** Whether a zero value is pointed out in orange ink. */
  markZeros?: boolean
  legend: readonly { label: string; fill: 'orange' | 'blue-grey' }[]
}>()
</script>

<template>
  <span class="flex flex-col gap-1.5 stage:gap-[18px]">
    <span
      v-for="bar in bars"
      :key="bar.label"
      class="grid grid-cols-[6.5rem_minmax(0,1fr)_2.5rem] items-center gap-2 text-xs stage:grid-cols-[190px_minmax(0,1fr)_70px] stage:gap-3.5 stage:text-[24px]"
    >
      <span :class="bar.label === strong ? 'font-semibold' : ''">{{ bar.label }}</span>
      <span class="flex h-2.5 bg-grey stage:h-[22px]">
        <span class="block h-full" :class="fill === 'orange' ? 'bg-orange' : 'bg-blue-grey'" :style="{ width: percent(bar.first) }"></span>
        <span class="block h-full bg-blue-grey" :style="{ width: percent(bar.second) }"></span>
      </span>
      <span class="text-right font-semibold" :class="markZeros && bar.zero ? 'text-orange-ink' : ''">{{ bar.value }}</span>
    </span>
    <span class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted stage:gap-6 stage:text-[22px]">
      <span v-for="item in legend" :key="item.label" class="inline-flex items-center gap-1.5 stage:gap-2.5">
        <span class="size-2.5 stage:size-5" :class="item.fill === 'orange' ? 'bg-orange' : 'bg-blue-grey'"></span>{{ item.label }}
      </span>
    </span>
  </span>
</template>
