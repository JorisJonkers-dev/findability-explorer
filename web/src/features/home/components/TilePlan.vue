<script setup lang="ts">
import { MONTHS, ROUTINE, YEAR } from '../../plan/domain/routine'

const TONE = { orange: 'bg-orange text-black', black: 'bg-black text-white', 'blue-grey': 'bg-blue-grey text-black' } as const
</script>

<template>
  <span class="flex flex-col gap-2 stage:gap-[18px]">
    <span class="grid grid-cols-12 gap-1 text-[10px] text-muted stage:text-[20px]">
      <span v-for="m in MONTHS" :key="m">{{ m.charAt(0) }}</span>
    </span>
    <span class="grid grid-cols-12 gap-1">
      <span
        v-for="bar in YEAR"
        :key="bar.label"
        class="truncate px-1.5 py-1 text-xs font-semibold stage:h-11 stage:px-3 stage:py-1.5 stage:text-[22px]"
        :class="TONE[bar.tone]"
        :style="{ gridColumn: `${String(bar.start)} / span ${String(bar.span)}` }"
      >
        {{ bar.label }}
      </span>
    </span>
    <span class="flex flex-col gap-1 pt-1 text-sm stage:gap-2 stage:pt-2 stage:text-[26px]">
      <span v-for="step in ROUTINE" :key="step.number" class="grid grid-cols-[1.25rem_minmax(0,1fr)] gap-1 stage:grid-cols-[44px_minmax(0,1fr)] stage:gap-2">
        <span class="font-semibold text-orange-ink">{{ step.number }}</span><span>{{ step.title }}</span>
      </span>
    </span>
  </span>
</template>
