<script setup lang="ts">
import { useId } from 'vue'

/** A section of a Results page: its title after the orange dash, an optional sentence on what it shows, and its body. */
defineProps<{ title: string; intro?: string }>()
const id = useId()
</script>

<template>
  <section :aria-labelledby="id" class="flex min-w-0 flex-col gap-4">
    <header class="flex flex-col gap-2">
      <h2 :id="id" class="m-0 flex items-center gap-3 font-display text-2xl leading-tight font-semibold tracking-[-0.01em]">
        <span aria-hidden="true" class="h-1 w-7 flex-none rounded-sm bg-orange"></span>
        {{ title }}
      </h2>
      <p v-if="intro" class="m-0 max-w-[70ch] text-ink-2">{{ intro }}</p>
    </header>
    <slot />
  </section>
</template>
