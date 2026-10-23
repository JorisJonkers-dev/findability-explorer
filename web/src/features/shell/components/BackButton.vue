<script setup lang="ts">
import { useBackToOverview } from '../composables/navigation'

const back = useBackToOverview()
</script>

<template>
  <button
    type="button"
    class="inline-flex min-h-11 cursor-zoom-out items-center gap-2 self-start rounded-full border-0 bg-black px-5 font-medium text-white hover:bg-ink-2 stage:min-h-24 stage:gap-[18px] stage:px-11 stage:text-[32px]"
    @click="back"
  >
    <svg aria-hidden="true" viewBox="0 0 24 24" class="size-5 stage:size-[34px]" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M15 5 8 12l7 7" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
    Overview
  </button>
</template>
