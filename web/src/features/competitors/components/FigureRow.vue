<script setup lang="ts">
import type { Figure } from '../domain/compare'

/** The page's key figures, each a big number with what it counts and a note. */
defineProps<{ figures: readonly Figure[] }>()
</script>

<template>
  <section aria-label="Key figures" class="grid gap-4 sm:grid-cols-3 stage:gap-10">
    <div v-for="f in figures" :key="f.text" class="flex flex-col gap-1.5 rounded-3xl bg-mist p-6 stage:gap-3 stage:rounded-[36px] stage:p-12">
      <span class="font-display text-5xl leading-none font-bold tabular-nums stage:text-[120px]">{{ f.big }}</span>
      <span class="text-lg font-semibold text-black stage:text-[40px]">{{ f.text }}</span>
      <span class="text-sm text-ink-2 stage:text-[28px]">{{ f.note }}</span>
    </div>
  </section>
</template>
