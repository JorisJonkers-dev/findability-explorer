<script setup lang="ts">
/** A section of a recommendations page: the orange dash, its title, an optional action on the right, an intro, and its body. */
defineProps<{ title: string; headingId: string }>()
</script>

<template>
  <section :aria-labelledby="headingId" class="flex min-w-0 flex-col gap-4">
    <header class="flex flex-col gap-2">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 :id="headingId" class="m-0 flex items-center gap-3 font-display text-2xl font-semibold tracking-[-0.01em]">
          <span aria-hidden="true" class="h-1 w-7 flex-none bg-orange"></span>{{ title }}
        </h2>
        <slot name="aside" />
      </div>
      <p v-if="$slots.intro" class="m-0 text-ink-2"><slot name="intro" /></p>
    </header>
    <slot />
  </section>
</template>
