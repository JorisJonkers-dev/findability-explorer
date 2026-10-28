<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'

/** The link at the top of a detail page, such as "Back to AI search". */
defineProps<{ to: RouteLocationRaw; label: string }>()
</script>

<template>
  <RouterLink
    :to="to"
    class="inline-flex min-h-11 items-center gap-2 self-start text-base font-bold text-black no-underline hover:text-orange-ink stage:min-h-[72px] stage:gap-4 stage:text-[32px]"
  >
    <svg aria-hidden="true" viewBox="0 0 24 24" class="size-[18px] stage:size-8" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M15 5l-7 7 7 7" />
    </svg>
    {{ label }}
  </RouterLink>
</template>
