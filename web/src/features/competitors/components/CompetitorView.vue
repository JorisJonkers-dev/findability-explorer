<script setup lang="ts">
import { computed, toRef } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import PageView from '../../shell/components/PageView.vue'
import { useMe } from '../../shell/composables/useMe'
import { companyPath, familySlug } from '../../shell/domain/sections'
import { useCompetitorDetail } from '../composables/useCompetitorDetail'
import { useRescan } from '../composables/useCompanyAdmin'
import { rescanSummary } from '../domain/admin'
import { detailFigures } from '../domain/compare'
import { lastScanLabel } from '../domain/company'
import CompanyAdmin from './CompanyAdmin.vue'
import FigureRow from './FigureRow.vue'
import MeasureChart from './MeasureChart.vue'

/** One competitor against Tribelt: how often AI names it, how ready its site is, and which Tribelt belts it sells. */
const props = defineProps<{ companyId: string }>()

const router = useRouter()
const id = toRef(() => props.companyId)
const { isAdmin } = useMe()
const { company, isLoading, notFound, error, title, where, belts, latestScan, measures } = useCompetitorDetail(id)
const { isQueuing, batch, error: rescanError, rescan } = useRescan()
const figures = computed(() => detailFigures(measures.value))
</script>

<template>
  <PageView section="competitors" :title="title" :back="{ to: '/competitors', label: 'Back to Competitors' }">
    <template v-if="company" #lede>
      <p class="m-0 max-w-3xl text-lg text-ink-2">{{ where }} {{ lastScanLabel(company.lastScan) ?? 'No scan of its website has run yet' }}.</p>
    </template>
    <p v-if="notFound" role="alert" class="m-0 text-orange-ink">No company has this id.</p>
    <p v-else-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading || !company" class="m-0 text-muted">Loading the company…</p>
    <div v-else class="flex flex-col gap-12">
      <div class="flex flex-wrap items-center gap-3">
        <a
          v-if="company.websiteUrl"
          :href="company.websiteUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex min-h-11 items-center gap-2 rounded-full border-2 border-black px-5 font-medium text-black no-underline hover:bg-black hover:text-white"
        >
          Visit its website<span class="sr-only"> (opens in a new tab)</span>
        </a>
        <RouterLink
          :to="`${companyPath(company.id)}/history`"
          class="inline-flex min-h-11 items-center rounded-full border-2 border-silver px-5 font-medium text-black no-underline hover:border-black"
        >
          See its scan history
        </RouterLink>
        <RouterLink
          v-if="latestScan"
          :to="`/scan/${latestScan.id}`"
          class="inline-flex min-h-11 items-center rounded-full border-2 border-silver px-5 font-medium text-black no-underline hover:border-black"
        >
          Open its latest scan
        </RouterLink>
        <button
          v-if="isAdmin && company.websiteUrl"
          type="button"
          class="min-h-11 cursor-pointer rounded-full border-0 bg-orange px-5 font-semibold text-black hover:bg-orange-soft disabled:opacity-60"
          :disabled="isQueuing"
          @click="rescan({ competitorIds: [company.id] })"
        >
          {{ company.scans > 0 ? 'Scan again' : 'Scan this website' }}
        </button>
      </div>
      <p v-if="batch" aria-live="polite" class="m-0 text-sm font-medium">{{ rescanSummary(batch) }}</p>
      <p v-if="rescanError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ rescanError }}</p>
      <FigureRow :figures="figures" />
      <div class="grid gap-12 lg:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] lg:items-start">
        <MeasureChart :name="company.shortName" :measures="measures" />
        <section aria-labelledby="belts-title" class="flex flex-col gap-4">
          <h2 id="belts-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em]">Which Tribelt belts it sells</h2>
          <ul class="m-0 flex list-none flex-col p-0">
            <li v-for="row in belts" :key="row.family" class="flex flex-col gap-0.5 border-t border-grey py-3">
              <RouterLink :to="`/terms/families/${familySlug(row.family)}`" class="font-semibold">{{ row.label }}</RouterLink>
              <span :class="row.offered ? 'text-black' : 'text-muted'" class="">{{ row.name }}</span>
            </li>
          </ul>
        </section>
      </div>
      <CompanyAdmin v-if="isAdmin" :company="company" @removed="router.push('/competitors')" />
    </div>
  </PageView>
</template>
