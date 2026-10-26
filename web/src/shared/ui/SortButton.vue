<script setup lang="ts">
import { computed } from 'vue'
import { type SortDirection, sortLabel } from '../sort'

/**
 * A sortable column's header: its label, and a chevron up (ascending) or down (descending) while
 * the table is sorted by it, or a faint pair while it is not. A click asks for the next sort.
 * label is what a screen reader names the column by; the slot may show a shorter one.
 */
const props = withDefaults(defineProps<{ label: string; direction: SortDirection | null; align?: 'start' | 'center' | 'end' }>(), {
  align: 'start',
})
const emit = defineEmits<{ sort: [] }>()
const justify = computed(() => ({ start: 'justify-start', center: 'justify-center', end: 'justify-end' })[props.align])
</script>

<template>
  <button
    type="button"
    :aria-label="sortLabel(label, direction)"
    :title="`Sort by ${label}`"
    :class="justify"
    class="inline-flex min-w-0 cursor-pointer items-center gap-1 border-0 bg-transparent p-0 font-[inherit] text-[inherit] hover:text-black stage:gap-2"
    @click="emit('sort')"
  >
    <span class="truncate"><slot>{{ label }}</slot></span>
    <svg aria-hidden="true" viewBox="0 0 10 14" class="h-3 w-2 flex-none stage:h-5 stage:w-3.5" fill="currentColor">
      <path d="M5 1 9 6H1z" :class="direction === 'asc' ? 'opacity-100' : direction === null ? 'opacity-30' : 'opacity-0'" />
      <path d="M5 13 1 8h8z" :class="direction === 'desc' ? 'opacity-100' : direction === null ? 'opacity-30' : 'opacity-0'" />
    </svg>
  </button>
</template>
