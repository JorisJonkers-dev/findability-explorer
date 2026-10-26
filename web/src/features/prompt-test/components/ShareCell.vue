<script setup lang="ts">
import { fraction, shareLabel } from '../domain/results'

/** A share as a figure over a thin bar: how many of the answers name (or cite) Tribelt. */
defineProps<{ part: number; whole: number }>()
</script>

<template>
  <span class="flex min-w-16 flex-col gap-1 stage:min-w-[140px] stage:gap-2">
    <span class="font-semibold tabular-nums">{{ shareLabel(part, whole) }}<span class="sr-only"> ({{ part }} of {{ whole }})</span></span>
    <span aria-hidden="true" class="block h-1.5 w-full rounded-full bg-grey stage:h-3">
      <span class="block h-full rounded-full bg-orange" :style="{ width: `${String(fraction(part, whole) * 100)}%` }"></span>
    </span>
  </span>
</template>
