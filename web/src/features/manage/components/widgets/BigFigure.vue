<script setup lang="ts">
/** A key figure in a widget: the value large, what it counts below, and an optional note. */
defineProps<{ value: string; label: string; note?: string }>()
</script>

<template>
  <p class="m-0 flex flex-col gap-1">
    <span data-part="value" class="font-display text-4xl leading-none font-semibold tracking-[-0.02em]">{{ value }}</span>
    <span class="text-ink-2">{{ label }}</span>
    <span v-if="note" class="text-sm text-muted">{{ note }}</span>
  </p>
</template>
