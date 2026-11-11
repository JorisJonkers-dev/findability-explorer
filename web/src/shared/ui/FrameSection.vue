<script setup lang="ts">
import { computed, useId } from 'vue'

/**
 * A section of a Results page: its title after the orange dash, an optional sentence on what it
 * shows, an optional link or control on the right, and its body. Labelled by its title, whose id is
 * `<id>-title` when the page names the section, so a page and its tests can point at it.
 */
const props = defineProps<{ id?: string; title: string; intro?: string }>()
const generated = useId()
const titleId = computed(() => (props.id === undefined ? generated : `${props.id}-title`))
</script>

<template>
  <section :aria-labelledby="titleId" class="flex min-w-0 flex-col gap-4">
    <header class="flex flex-col gap-2">
      <div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <h2 :id="titleId" class="m-0 flex items-center gap-3 font-display text-2xl leading-tight font-semibold tracking-[-0.01em]">
          <span aria-hidden="true" class="h-1 w-7 flex-none rounded-sm bg-orange"></span>
          {{ title }}
        </h2>
        <slot name="aside" />
      </div>
      <p v-if="intro" class="m-0 max-w-[70ch] text-ink-2">{{ intro }}</p>
    </header>
    <slot />
  </section>
</template>
