<script setup lang="ts">
/** A labelled section of the Overview or a goal page: the orange dash before its title, an intro, and its body. */
defineProps<{ id: string; title: string; intro?: string }>()
</script>

<template>
  <section :aria-labelledby="`${id}-title`" class="flex min-w-0 flex-col gap-3">
    <h2 :id="`${id}-title`" class="m-0 flex items-center gap-3 font-display text-2xl leading-tight font-semibold tracking-[-0.01em]">
      <span aria-hidden="true" class="h-1 w-7 flex-none bg-orange"></span>{{ title }}
    </h2>
    <p v-if="intro" class="m-0 max-w-[72ch] text-ink-2">{{ intro }}</p>
    <slot />
  </section>
</template>
