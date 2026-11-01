<script setup lang="ts">
import { useId } from 'vue'

/**
 * A widget of a Manage page: a flat mist block on the twelve-column grid, with its title, an action
 * beside the title, its body, and a foot for its buttons. A lone widget stretches to the full width.
 */
const props = withDefaults(defineProps<{ title: string; span?: 3 | 4 | 6 | 8 | 12; accent?: boolean }>(), { span: 12, accent: false })

// Written out in full, so Tailwind finds each class.
const SPANS = {
  3: 'min-[1100px]:col-span-3',
  4: 'min-[1100px]:col-span-4',
  6: 'min-[1100px]:col-span-6',
  8: 'min-[1100px]:col-span-8',
  12: 'min-[1100px]:col-span-12',
} as const
const id = useId()
const span = SPANS[props.span]
</script>

<template>
  <section :aria-labelledby="id" class="flex min-w-0 flex-col gap-4 rounded-3xl p-6" :class="[span, accent ? 'bg-orange-pale' : 'bg-mist']">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <h2 :id="id" class="m-0 font-display text-xl leading-tight font-semibold">{{ title }}</h2>
      <div v-if="$slots.action" class="flex flex-wrap items-center gap-3 text-sm"><slot name="action" /></div>
    </div>
    <div v-if="$slots.default" class="flex min-w-0 flex-col gap-3"><slot /></div>
    <div v-if="$slots.foot" class="mt-auto flex flex-wrap items-center gap-3"><slot name="foot" /></div>
  </section>
</template>
