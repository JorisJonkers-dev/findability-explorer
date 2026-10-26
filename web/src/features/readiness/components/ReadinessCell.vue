<script setup lang="ts">
import type { Cell } from '../domain/readiness'

/** One cell's value, marked as following the recommendation (a dot), a gap (orange ink), or a fact. */
defineProps<{ cell: Cell }>()
</script>

<template>
  <span class="inline-flex items-center gap-1.5 whitespace-nowrap stage:gap-2.5" :class="cell.ok === false ? 'font-semibold text-orange-ink' : cell.ok === null ? 'text-ink-2' : ''">
    <span v-if="cell.ok !== null" aria-hidden="true" class="size-2.5 flex-none rounded-full stage:size-4" :class="cell.ok ? 'bg-orange' : 'border-2 border-orange-ink'"></span>
    {{ cell.text }}<span v-if="cell.ok !== null" class="sr-only">{{ cell.ok ? ', follows the advice' : ', a gap' }}</span>
  </span>
</template>
