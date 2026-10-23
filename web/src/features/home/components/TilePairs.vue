<script setup lang="ts">
import { percent } from '../../../shared/percent'
import { HOME_PAIRS } from '../../buyers/domain/content'
</script>

<template>
  <span class="flex flex-col gap-3 stage:gap-[18px]">
    <span v-for="pair in HOME_PAIRS" :key="pair.title" class="flex flex-col gap-1 stage:gap-2.5">
      <span class="text-xs text-muted stage:text-[24px]">{{ pair.title }}</span>
      <span
        v-for="bar in pair.bars"
        :key="bar.label"
        class="grid grid-cols-[4rem_minmax(0,1fr)_3.25rem] items-center gap-2 text-xs stage:grid-cols-[130px_minmax(0,1fr)_120px] stage:gap-3.5 stage:text-[24px]"
      >
        <span>{{ bar.label }}</span>
        <span class="h-2.5 bg-grey stage:h-[22px]">
          <span class="block h-full" :class="bar.fill === 'orange' ? 'bg-orange' : 'bg-blue-grey'" :style="{ width: percent(bar.fraction) }"></span>
        </span>
        <span class="text-right font-semibold">{{ bar.value }}</span>
      </span>
    </span>
  </span>
</template>
