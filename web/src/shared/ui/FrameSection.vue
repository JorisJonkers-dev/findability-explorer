<script setup lang="ts">
/**
 * A section of a page in the frame: the orange dash before its title, an optional sentence under
 * it, an optional link or control on the right, and its body. Labelled by its title.
 */
defineProps<{ id: string; title: string; intro?: string }>()
</script>

<template>
  <section :aria-labelledby="`${id}-title`" class="flex min-w-0 flex-col gap-4">
    <header class="flex flex-col gap-1.5">
      <div class="flex flex-wrap items-center justify-between gap-x-4 gap-y-2">
        <h2 :id="`${id}-title`" class="m-0 flex items-center gap-3 font-display text-2xl leading-tight font-semibold tracking-[-0.01em]">
          <span aria-hidden="true" class="h-1.5 w-7 flex-none bg-orange"></span>{{ title }}
        </h2>
        <slot name="aside" />
      </div>
      <p v-if="intro" class="m-0 max-w-[72ch] text-ink-2">{{ intro }}</p>
    </header>
    <slot />
  </section>
</template>
