<script setup lang="ts">
import { computed, useId } from 'vue'

/**
 * One widget of a Manage page: a mist block with its title, an optional action on the right, a
 * body, and a footer for its buttons. span is its width in twelfths on a wide screen; a widget alone
 * on its row stretches to fill it, and on a narrow screen every widget takes the whole width.
 */
const props = withDefaults(defineProps<{ title: string; span?: 3 | 4 | 6 | 8 | 12; accent?: boolean }>(), { span: 6, accent: false })
const id = useId()

// Literal classes, so Tailwind finds them: the basis leaves room for the gaps, and growing fills the row.
const BASIS: Record<3 | 4 | 6 | 8 | 12, string> = {
  3: 'min-[700px]:basis-[calc(50%-0.75rem)] min-[1100px]:basis-[calc(25%-1.25rem)]',
  4: 'min-[700px]:basis-[calc(50%-0.75rem)] min-[1100px]:basis-[calc(33.333%-1.25rem)]',
  6: 'min-[1100px]:basis-[calc(50%-0.75rem)]',
  8: 'min-[1100px]:basis-[calc(66.666%-1.25rem)]',
  12: '',
}
const width = computed(() => BASIS[props.span])
</script>

<template>
  <section
    :aria-labelledby="id"
    :data-span="span"
    class="flex min-w-0 grow basis-full flex-col gap-4 rounded-3xl p-6"
    :class="[width, accent ? 'bg-orange-pale' : 'bg-mist']"
  >
    <div class="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
      <h2 :id="id" class="m-0 font-display text-xl leading-tight font-semibold">{{ title }}</h2>
      <slot name="action" />
    </div>
    <div class="flex min-w-0 flex-col gap-3">
      <slot />
    </div>
    <div v-if="$slots.foot" data-part="foot" class="mt-auto flex flex-wrap items-center gap-3 pt-2">
      <slot name="foot" />
    </div>
  </section>
</template>
