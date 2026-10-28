<script setup lang="ts">
import type { Figure } from '../domain/compare'

/** The page's key figures, each a big number with what it counts and a note. */
defineProps<{ figures: readonly Figure[] }>()
</script>

<template>
  <section aria-label="Key figures" class="grid gap-4 sm:grid-cols-3">
    <div v-for="f in figures" :key="f.text" class="flex flex-col gap-1.5 rounded-3xl bg-mist p-6">
      <span class="font-display text-5xl leading-none font-bold tabular-nums">{{ f.big }}</span>
      <span class="text-lg font-semibold text-black">{{ f.text }}</span>
      <span class="text-sm text-ink-2">{{ f.note }}</span>
    </div>
  </section>
</template>
