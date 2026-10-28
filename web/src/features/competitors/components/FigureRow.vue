<script setup lang="ts">
import { computed } from 'vue'
import type { Figure } from '../domain/compare'

/** The page's key figures, each a big number with what it counts and a note. */
const props = defineProps<{ figures: readonly Figure[] }>()

// The cards share the row whatever their count (the page can have fewer than three figures).
const columns = computed(() => (props.figures.length >= 3 ? 'sm:grid-cols-3' : props.figures.length === 2 ? 'sm:grid-cols-2' : 'sm:grid-cols-1'))
</script>

<template>
  <section aria-label="Key figures" :class="columns" class="grid gap-4">
    <div v-for="f in figures" :key="f.text" class="flex flex-col gap-1.5 rounded-3xl bg-mist p-6">
      <span class="font-display text-5xl leading-none font-bold tabular-nums">{{ f.big }}</span>
      <span class="text-lg font-semibold text-black">{{ f.text }}</span>
      <span class="text-sm text-ink-2">{{ f.note }}</span>
    </div>
  </section>
</template>
