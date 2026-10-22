<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import RecentScans from '../features/scan/components/RecentScans.vue'

const route = useRoute()

const links = [
  { to: '/', label: 'Overview' },
  { to: '/scan', label: 'Scan a competitor' },
  { to: '/terminology', label: 'Terminology' },
  { to: '/competitors', label: 'Competitors' },
] as const

/** A section's link stays marked on the pages under it, such as one scan under Scan a competitor. */
function isActive(to: string): boolean {
  if (to === '/') return route.path === '/'
  return route.path === to || route.path.startsWith(`${to}/`)
}
</script>

<template>
  <header
    class="flex flex-col gap-4 border-b border-line bg-sidebar px-4 py-4 md:sticky md:top-0 md:h-screen md:w-[232px] md:flex-none md:gap-7 md:overflow-y-auto md:border-r md:border-b-0 md:px-5 md:py-7"
  >
    <RouterLink to="/" class="flex flex-col gap-0.5 text-ink no-underline hover:text-ink">
      <span class="font-display text-2xl leading-none font-bold tracking-[0.01em]">Findability Explorer</span>
      <span class="text-[13px] text-muted">How metal belts are named, and found</span>
    </RouterLink>

    <nav aria-label="Main">
      <ul class="m-0 flex list-none flex-wrap gap-0.5 p-0 md:flex-col">
        <li v-for="link in links" :key="link.to">
          <RouterLink
            :to="link.to"
            :aria-current="isActive(link.to) ? 'page' : undefined"
            class="block rounded-md px-3 py-2 no-underline md:py-2.5"
            :class="isActive(link.to) ? 'bg-straw font-semibold text-on-straw hover:text-on-straw' : 'font-medium text-ink-2 hover:text-ink'"
          >
            {{ link.label }}
          </RouterLink>
        </li>
      </ul>
    </nav>

    <RecentScans />

    <p class="mt-auto mb-0 hidden text-[13px] text-muted md:block">
      Built for Tribelt B.V. in the Digital Strategy course, University of Twente.
    </p>
  </header>
</template>
