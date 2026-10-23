<script setup lang="ts">
import type { Recommendation } from './types'

defineProps<{ items: readonly Recommendation[] }>()
</script>

<template>
  <ol class="m-0 flex list-none flex-col gap-4 p-0 stage:gap-7">
    <li v-for="item in items" :key="item.lead" class="grid grid-cols-[2rem_minmax(0,1fr)] gap-2 stage:grid-cols-[64px_minmax(0,1fr)] stage:gap-3">
      <span class="font-display text-2xl leading-none font-semibold text-orange-ink stage:text-[48px]">{{ item.mark }}</span>
      <span><strong class="font-semibold">{{ item.lead }}</strong> {{ item.text }}</span>
    </li>
  </ol>
</template>
