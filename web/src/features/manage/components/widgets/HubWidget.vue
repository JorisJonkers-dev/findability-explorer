<script setup lang="ts">
import { RouterLink } from 'vue-router'
import ManageWidget from './ManageWidget.vue'

/** A card for a subpage on a section's index: what it holds, its key figures, and the way there. */
defineProps<{ title: string; text: string; figures: { value: string; label: string }[]; to: string; label: string; span?: 3 | 4 | 6 | 8 | 12 }>()
</script>

<template>
  <ManageWidget :title="title" :span="span">
    <p class="m-0 text-ink-2">{{ text }}</p>
    <dl class="m-0 flex flex-wrap gap-x-8 gap-y-2">
      <div v-for="f in figures" :key="f.label" class="flex flex-col-reverse">
        <dt class="text-sm text-ink-2">{{ f.label }}</dt>
        <dd class="m-0 font-display text-3xl font-semibold">{{ f.value }}</dd>
      </div>
    </dl>
    <template #foot>
      <RouterLink :to="to" class="inline-flex min-h-11 items-center rounded-full border-2 border-black px-5 font-semibold text-black no-underline hover:bg-black hover:text-white">{{ label }}</RouterLink>
    </template>
  </ManageWidget>
</template>
