<script setup lang="ts">
import type { ColumnState } from '../columns'

/**
 * One column of a page's stage, with buttons to widen or minimise it. Minimised, it is a strip
 * that names the column and restores it; its content stays as it was. On a phone the columns
 * stack, so neither the buttons nor the strip show there.
 */
defineProps<{ title: string; state: ColumnState; canMinimise: boolean }>()
const emit = defineEmits<{ widen: []; minimise: []; restore: [] }>()

// Above the column, in the gap under the page's title, so they never cover its content.
const CONTROL =
  'inline-flex size-9 items-center justify-center rounded-full border-2 border-grey bg-white text-ink-2 hover:border-black hover:text-black disabled:opacity-40 stage:size-[52px] stage:border-[3px]'
</script>

<template>
  <div class="relative flex flex-col stage:grid stage:min-h-0 stage:grid-rows-[minmax(0,1fr)]">
    <div v-if="state !== 'min'" role="group" :aria-label="`${title}: width`" class="absolute right-0 bottom-full z-10 mb-2 hidden gap-2 stage:flex stage:gap-3">
      <button
        type="button"
        :class="CONTROL"
        :aria-pressed="state === 'wide'"
        :aria-label="state === 'wide' ? `Put ${title} back to its width` : `Widen ${title}`"
        :title="state === 'wide' ? 'Back to its width' : 'Widen'"
        @click="emit('widen')"
      >
        <svg aria-hidden="true" viewBox="0 0 16 16" class="size-4 stage:size-7" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <path v-if="state === 'wide'" d="M6 3 3 8l3 5M10 3l3 5-3 5" />
          <path v-else d="M3 3 1 8l2 5M13 3l2 5-2 5M5 8h6" />
        </svg>
      </button>
      <button
        type="button"
        :class="CONTROL"
        :disabled="!canMinimise"
        :aria-label="`Minimise ${title}`"
        title="Minimise"
        @click="emit('minimise')"
      >
        <svg aria-hidden="true" viewBox="0 0 16 16" class="size-4 stage:size-7" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round">
          <path d="M3 8h10" />
        </svg>
      </button>
    </div>
    <button
      v-if="state === 'min'"
      type="button"
      :aria-label="`Restore ${title}`"
      class="hidden h-full cursor-pointer flex-col items-center gap-6 rounded-3xl border-0 bg-mist py-10 text-black hover:bg-grey stage:flex"
      @click="emit('restore')"
    >
      <svg aria-hidden="true" viewBox="0 0 16 16" class="size-7 flex-none" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round">
        <path d="M8 3v10M3 8h10" />
      </svg>
      <span class="text-[28px] font-semibold [writing-mode:vertical-rl]">{{ title }}</span>
    </button>
    <!-- Kept mounted while minimised, so a column keeps its filters, sort and scroll; a phone always shows it. -->
    <div class="flex flex-col stage:grid stage:min-h-0 stage:grid-rows-[minmax(0,1fr)]" :class="state === 'min' ? 'stage:hidden' : ''">
      <slot />
    </div>
  </div>
</template>
