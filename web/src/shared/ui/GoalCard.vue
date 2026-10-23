<script setup lang="ts">
import { percent } from '../percent'
import type { GoalRow } from './types'

defineProps<{ goal: string; text: string; rows: readonly GoalRow[] }>()
</script>

<template>
  <div class="flex flex-col gap-3 rounded-2xl bg-mist p-5 stage:gap-4 stage:rounded-[28px] stage:p-10">
    <span class="text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[26px]">Goal {{ goal }}</span>
    <span class="stage:text-[32px]">{{ text }}</span>
    <div class="grid grid-cols-[4.5rem_minmax(0,1fr)_4rem] items-center gap-3 text-sm stage:grid-cols-[160px_minmax(0,1fr)_120px] stage:gap-4 stage:text-[26px]">
      <template v-for="row in rows" :key="row.label">
        <span>{{ row.label }}</span>
        <span class="h-4 bg-grey stage:h-[30px]" aria-hidden="true">
          <span class="block h-full" :class="row.fill === 'today' ? 'bg-blue-grey' : 'bg-orange'" :style="{ width: percent(row.fraction) }"></span>
        </span>
        <span class="text-right font-semibold">{{ row.value }}</span>
      </template>
    </div>
  </div>
</template>
