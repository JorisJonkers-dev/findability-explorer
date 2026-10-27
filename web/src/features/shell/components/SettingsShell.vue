<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useTermCandidates } from '../../review/composables/useTermReview'
import { useMe } from '../composables/useMe'
import SectionEyebrow from './SectionEyebrow.vue'
import TopBar from './TopBar.vue'

/** A settings page's frame: the top bar, the settings menu, the eyebrow, the title and the lead. */
defineProps<{ title: string }>()
const route = useRoute()
const { me } = useMe()
// How many terms wait in Term review: the menu shows it on every settings page.
const { counts } = useTermCandidates(ref('pending'))
const waiting = computed(() => counts.value?.pending)

const PAGES = [
  { label: 'Language models', path: '/settings' },
  { label: 'Prompt test', path: '/settings/prompt-test' },
  { label: 'Term review', path: '/settings/terms' },
] as const
</script>

<template>
  <div class="mx-auto flex max-w-7xl flex-col gap-10 px-4 pt-3 pb-20 sm:px-8 lg:pt-6">
    <TopBar />
    <div class="flex flex-col gap-10 lg:grid lg:grid-cols-[220px_minmax(0,1fr)] lg:gap-14">
      <nav aria-label="Settings" class="flex flex-wrap gap-2 lg:flex-col lg:gap-1 lg:pt-24">
        <RouterLink
          v-for="page in PAGES"
          :key="page.path"
          :to="page.path"
          :aria-current="route.path === page.path ? 'page' : undefined"
          class="inline-flex min-h-11 items-center justify-between gap-3 rounded-xl px-4 font-medium no-underline"
          :class="route.path === page.path ? 'bg-black text-white hover:text-white' : 'text-ink-2 hover:bg-mist hover:text-black'"
        >
          {{ page.label }}
          <span
            v-if="page.path === '/settings/terms' && waiting !== undefined && waiting > 0"
            class="rounded-full bg-orange px-2 text-sm font-semibold text-black"
            :aria-label="`${String(waiting)} waiting`"
          >{{ waiting }}</span>
        </RouterLink>
      </nav>
      <div class="flex max-w-5xl min-w-0 flex-col gap-10">
        <header class="flex flex-col gap-4">
          <div class="flex flex-wrap items-center justify-between gap-4">
            <SectionEyebrow>Settings</SectionEyebrow>
            <span v-if="me" class="text-sm text-muted">Signed in as {{ me.name }}{{ me.isAdmin ? ' (administrator)' : '' }}</span>
          </div>
          <h1 class="m-0 font-display text-4xl leading-[0.95] font-semibold tracking-[-0.03em] sm:text-6xl">{{ title }}</h1>
          <slot name="lead" />
        </header>
        <slot />
      </div>
    </div>
  </div>
</template>
