<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { useCompanies } from '../../competitors/composables/useCompanies'
import { scanCompany } from '../../competitors/domain/company'
import { useRecentScans } from '../composables/useRecentScans'
import { recentLabel } from '../domain/recent'

const route = useRoute()
const { scans, isLoading, error } = useRecentScans()
const { companies } = useCompanies()
</script>

<template>
  <section aria-labelledby="recent-title" class="flex flex-col gap-3 stage:gap-4">
    <h3 id="recent-title" class="m-0 text-sm font-normal text-muted stage:text-[26px]">Recent scans</h3>
    <p v-if="error" role="alert" class="m-0 text-sm text-orange-ink stage:text-[26px]">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-sm text-muted stage:text-[26px]">Loading…</p>
    <p v-else-if="scans.length === 0" class="m-0 text-sm text-muted stage:text-[26px]">No scans yet.</p>
    <ul v-else class="m-0 flex list-none flex-wrap gap-2 p-0 stage:gap-3">
      <li v-for="scan in scans" :key="scan.id">
        <RouterLink
          :to="{ path: `/scan/${scan.id}`, query: route.query }"
          class="inline-flex min-h-11 items-center gap-2 rounded-full border-2 border-black bg-white px-4 text-sm font-medium text-black no-underline hover:bg-black hover:text-white stage:min-h-[76px] stage:gap-3 stage:border-[3px] stage:px-[30px] stage:text-[28px]"
        >
          <span>{{ scanCompany(scan, companies).name }}</span><span class="sr-only">, </span>
          <span class="font-normal opacity-80">{{ recentLabel(scan) }}</span>
        </RouterLink>
      </li>
    </ul>
  </section>
</template>
