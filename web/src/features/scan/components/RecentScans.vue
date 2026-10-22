<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { useRecentScans } from '../composables/useRecentScans'
import { recentLabel } from '../domain/recent'
import { siteHost } from '../domain/results'

const route = useRoute()
const { scans, isLoading, error } = useRecentScans()

function isCurrent(id: string): boolean {
  return route.path === `/scan/${id}`
}
</script>

<template>
  <!-- Beside the content the list is always open; above it, on a phone, it folds away. -->
  <section aria-labelledby="recent-title" class="hidden flex-col gap-1.5 md:flex">
    <h2 id="recent-title" class="m-0 text-[13px] font-normal text-muted">Recent scans</h2>
    <p v-if="error" class="m-0 px-3 text-sm text-muted">{{ error }}</p>
    <p v-else-if="!isLoading && scans.length === 0" class="m-0 px-3 text-sm text-muted">No scans yet.</p>
    <ul v-else class="m-0 flex list-none flex-col p-0">
      <li v-for="scan in scans" :key="scan.id">
        <RouterLink
          :to="`/scan/${scan.id}`"
          :aria-current="isCurrent(scan.id) ? 'page' : undefined"
          class="flex justify-between gap-2 rounded-md px-3 py-1.5 text-sm no-underline"
          :class="isCurrent(scan.id) ? 'bg-raised text-ink' : 'text-ink-2 hover:text-ink'"
        >
          <span class="min-w-0 truncate">{{ siteHost(scan.url) }}</span>
          <span class="flex-none text-muted">{{ recentLabel(scan) }}</span>
        </RouterLink>
      </li>
    </ul>
  </section>
  <details v-if="scans.length > 0" class="md:hidden">
    <summary class="text-sm font-medium text-ink-2">Recent scans</summary>
    <ul class="m-0 mt-2 flex list-none flex-wrap gap-x-4 gap-y-1 p-0">
      <li v-for="scan in scans" :key="scan.id">
        <RouterLink :to="`/scan/${scan.id}`" :aria-current="isCurrent(scan.id) ? 'page' : undefined" class="text-sm">
          {{ siteHost(scan.url) }} <span class="text-muted">{{ recentLabel(scan) }}</span>
        </RouterLink>
      </li>
    </ul>
  </details>
</template>
