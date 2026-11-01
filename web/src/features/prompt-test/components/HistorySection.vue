<script setup lang="ts">
import type { Measure } from '../domain/history'
import HistoryPanel from './HistoryPanel.vue'

/** A labelled section, the orange dash before its title, holding a Prompt test measure over time. */
defineProps<{ id: string; title: string; intro: string; measures: readonly Measure[]; goal?: number }>()
</script>

<template>
  <section :aria-labelledby="`${id}-title`" class="flex min-w-0 flex-col gap-3">
    <h2 :id="`${id}-title`" class="m-0 flex items-center gap-3 font-display text-2xl leading-tight font-semibold tracking-[-0.01em]">
      <span aria-hidden="true" class="h-1 w-7 flex-none bg-orange"></span>{{ title }}
    </h2>
    <p class="m-0 max-w-[72ch] text-ink-2">{{ intro }}</p>
    <HistoryPanel :measures="measures" :goal="goal" :label="title" />
  </section>
</template>
