<script setup lang="ts">
import { computed, nextTick, useTemplateRef } from 'vue'
import type { Analysis } from '../../../infrastructure/api/types.gen'
import { RouterLink } from 'vue-router'
import { useCompanies } from '../../competitors/composables/useCompanies'
import { scanCompany } from '../../competitors/domain/company'
import { pagePath, pagesReadTitle, readSummary, scannedAt, siteHost, wordsLabel } from '../domain/results'
import { isInProgress } from '../domain/status'
import EquivalentsTable from './EquivalentsTable.vue'
import MissingTerms from './MissingTerms.vue'
import PhraseList from './PhraseList.vue'
import ScanProgress from './ScanProgress.vue'
import TemperSpectrum from './TemperSpectrum.vue'

const props = defineProps<{ scan: Analysis }>()
const { companies } = useCompanies()

const summary = computed(() => props.scan.summary)
const host = computed(() => siteHost(summary.value.url))
const who = computed(() => scanCompany(summary.value, companies.value))
const site = computed(() => who.value.name)
const inProgress = computed(() => isInProgress(summary.value.status))
const when = computed(() =>
  inProgress.value
    ? `Started ${scannedAt(summary.value.createdAt)}.`
    : `Scanned ${scannedAt(summary.value.finishedAt ?? summary.value.createdAt)}.`,
)
const pagesTitle = computed(() => pagesReadTitle(props.scan.pages.length))
const pages = useTemplateRef<HTMLDetailsElement>('pages')

async function openPages(): Promise<void> {
  if (!pages.value) return
  pages.value.open = true
  await nextTick()
  pages.value.scrollIntoView({ block: 'start' })
  pages.value.querySelector('summary')?.focus()
}
</script>

<template>
  <section aria-labelledby="result-title" class="flex flex-col gap-7">
    <div class="flex flex-wrap items-end justify-between gap-4 border-b border-line pb-5">
      <div class="flex min-w-0 flex-col gap-1.5">
        <h2 id="result-title" class="m-0 font-display text-[32px] leading-[1.05] font-semibold break-words md:text-[40px]">
          {{ host }}
        </h2>
        <p class="m-0 text-muted">{{ when }} {{ readSummary(summary, scan.pages) }}</p>
        <p class="m-0 flex flex-wrap gap-x-5 gap-y-1 text-[15px]">
          <RouterLink v-if="who.company" :to="`/competitors/${who.company.id}`" class="font-medium">
            About {{ who.name }}
          </RouterLink>
          <a :href="summary.url" target="_blank" rel="noopener noreferrer" class="font-medium">
            Visit website<span class="sr-only"> (opens in a new tab)</span>
          </a>
        </p>
      </div>
      <a v-if="scan.pages.length > 0" href="#pages" class="font-medium" @click.prevent="openPages">
        See {{ pagesTitle.charAt(0).toLowerCase() + pagesTitle.slice(1) }}
      </a>
    </div>

    <ScanProgress v-if="inProgress" :summary="summary" />

    <div v-if="summary.status === 'failed'" class="flex max-w-[980px] flex-col gap-2 rounded-xl border border-danger/50 bg-surface p-6">
      <h3 class="m-0 font-display text-2xl font-semibold">The scan failed</h3>
      <p class="m-0 text-ink-2">{{ summary.failure ?? 'The site could not be read.' }}</p>
      <p class="m-0 text-ink-2">Check the address and scan again.</p>
    </div>

    <template v-if="summary.status === 'done'">
      <TemperSpectrum :known-terms="scan.knownTerms" />
      <div class="grid grid-cols-[repeat(auto-fit,minmax(min(100%,440px),1fr))] items-start gap-7">
        <MissingTerms :known-terms="scan.knownTerms" :site="site" />
        <PhraseList :keywords="scan.keywords" />
      </div>
      <EquivalentsTable :equivalents="scan.equivalents" :matching="scan.matching" :site="site" />
    </template>

    <details v-if="scan.pages.length > 0" id="pages" ref="pages" class="scroll-mt-6 border-t border-line pt-5">
      <summary class="font-display text-2xl font-semibold">{{ pagesTitle }}</summary>
      <ul
        class="m-0 mt-4 grid list-none grid-cols-[repeat(auto-fill,minmax(min(100%,320px),1fr))] gap-x-7 gap-y-2.5 p-0 text-[15px]"
      >
        <li v-for="page in scan.pages" :key="page.url" class="flex justify-between gap-3">
          <span class="min-w-0 break-all text-ink-2">{{ pagePath(page.url) }}</span>
          <span class="flex-none text-muted">{{ wordsLabel(page.wordCount) }}</span>
        </li>
      </ul>
    </details>
  </section>
</template>
