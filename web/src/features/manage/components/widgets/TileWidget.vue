<script setup lang="ts">
import ManageWidget from './ManageWidget.vue'

/** A widget with one key figure and a note below it. */
defineProps<{ title: string; value: string; note?: string; span?: 3 | 4 | 6 | 8 | 12 }>()
</script>

<template>
  <ManageWidget :title="title" :span="span">
    <p class="m-0 font-display text-4xl leading-none font-semibold" data-part="value">{{ value }}</p>
    <p v-if="note" class="m-0 text-ink-2" data-part="note">{{ note }}</p>
    <slot />
  </ManageWidget>
</template>
