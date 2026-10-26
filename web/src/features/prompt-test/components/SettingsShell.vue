<script setup lang="ts">
import { RouterLink } from 'vue-router'
import AccountBar from '../../shell/components/AccountBar.vue'
import SectionEyebrow from '../../shell/components/SectionEyebrow.vue'

/** A settings page's frame: back to the settings, the account, the eyebrow, the title and the lead. */
defineProps<{ title: string }>()
</script>

<template>
  <div class="mx-auto flex max-w-5xl flex-col gap-10 px-4 pt-5 pb-20 sm:px-8 lg:pt-10">
    <header class="flex flex-col gap-8">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <RouterLink
          to="/settings"
          class="inline-flex min-h-11 items-center gap-2 rounded-full bg-black px-5 font-medium text-white no-underline hover:bg-ink-2 hover:text-white"
        >
          <svg aria-hidden="true" viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M15 5 8 12l7 7" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          Settings
        </RouterLink>
        <AccountBar tone="light" />
      </div>
      <div class="flex flex-col gap-4">
        <SectionEyebrow>Settings</SectionEyebrow>
        <h1 class="m-0 font-display text-4xl leading-[0.95] font-semibold tracking-[-0.03em] sm:text-6xl">{{ title }}</h1>
        <slot name="lead" />
      </div>
    </header>
    <slot />
  </div>
</template>
