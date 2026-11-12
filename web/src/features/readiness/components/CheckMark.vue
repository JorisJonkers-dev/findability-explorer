<script setup lang="ts">
import type { CheckState } from '../domain/checks'

/**
 * A check's state as a mark: a tick, a tilde for partly, a cross, or a dash when not known. Its
 * shape says what it means and the colour only supports it; label is what a screen reader reads.
 */
defineProps<{ state: CheckState; label?: string }>()

const PATH: Record<CheckState, string> = {
  yes: 'M3.5 8.5 6.5 11.5 12.5 4.5',
  partly: 'M3 9.5C4.5 6.5 6.5 6.5 8 8s3.5 1.5 5-1.5',
  no: 'M4.5 4.5 11.5 11.5M11.5 4.5 4.5 11.5',
  unknown: 'M5 8H11',
}
const TONE: Record<CheckState, string> = {
  yes: 'bg-orange-pale text-orange-ink',
  partly: 'border-2 border-orange-soft bg-white text-orange-ink',
  no: 'bg-grey text-ink-2',
  unknown: 'bg-mist text-muted',
}
</script>

<template>
  <span :data-mark="state" class="inline-flex size-6 flex-none items-center justify-center rounded-full align-middle" :class="TONE[state]">
    <svg aria-hidden="true" viewBox="0 0 16 16" class="size-3.5" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path :d="PATH[state]" /></svg>
    <span v-if="label" class="sr-only">{{ label }}</span>
  </span>
</template>
