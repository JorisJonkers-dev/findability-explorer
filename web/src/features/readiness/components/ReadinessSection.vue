<script setup lang="ts">
/** A section of the readiness pages: the orange dash, its title, an optional action on the right, an intro, and its body. */
defineProps<{ id: string; title: string; intro?: string }>()
</script>

<template>
  <section :aria-labelledby="id" class="flex min-w-0 flex-col gap-3">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h2 :id="id" class="m-0 flex items-center gap-3 font-display text-2xl leading-tight font-semibold tracking-[-0.01em]">
        <span aria-hidden="true" class="h-1.5 w-7 flex-none bg-orange"></span>{{ title }}
      </h2>
      <slot name="aside" />
    </div>
    <p v-if="intro" class="m-0 max-w-[70ch] text-ink-2">{{ intro }}</p>
    <slot />
  </section>
</template>
