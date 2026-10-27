<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { tierFill } from '../../../shared/tiers'
import { tierLabel } from '../../../shared/vocabulary'
import { usePromptResults } from '../../prompt-test/composables/usePromptTest'
import { mentionLine } from '../../prompt-test/domain/results'
import CompanyReadiness from '../../readiness/components/CompanyReadiness.vue'
import ReadinessFiles from '../../readiness/components/ReadinessFiles.vue'
import ScanResult from '../../scan/components/ScanResult.vue'
import { useScan } from '../../scan/composables/useScan'
import SectionView from '../../shell/components/SectionView.vue'
import { useMe } from '../../shell/composables/useMe'
import { companyTabs } from '../../shell/domain/sections'
import { useTermMap } from '../../terminology/composables/useTermSearch'
import { ofLanguage } from '../../terminology/domain/languages'
import { useCompanies } from '../composables/useCompanies'
import { useRescan } from '../composables/useCompanyAdmin'
import { useCompany } from '../composables/useCompany'
import { rescanSummary } from '../domain/admin'
import { companyTerms, lastScanLabel } from '../domain/company'
import { beltRows, termsTitle, whereLine } from '../domain/list'
import CompanyAdmin from './CompanyAdmin.vue'

/** A competitor's latest scan: what the crawl found, which Tribelt belts it sells, and its AI readiness. */
const props = defineProps<{ companyId: string }>()

const router = useRouter()
const id = toRef(() => props.companyId)
const { isAdmin } = useMe()
const { byId, isLoading } = useCompanies()
const company = computed(() => byId(props.companyId))
const { detail, notFound, error } = useCompany(id)
const map = useTermMap()
// The scan's language filter narrows the map terms the site uses too.
const language = ref<string | null>(null)
const used = computed(() => (company.value ? ofLanguage(companyTerms(map.terms.value, company.value.shortName), language.value) : []))
const latest = computed(() => detail.value?.scans.find((s) => s.status === 'done'))
const { scan } = useScan(computed(() => latest.value?.id ?? ''))
const { results } = usePromptResults()
const mentioned = computed(() => mentionLine(results.value, props.companyId))
const { isQueuing, batch, error: rescanError, rescan } = useRescan()

const rows = computed(() => (company.value ? beltRows(company.value) : []))
const title = computed(() => {
  const c = company.value
  if (!c) return 'A competitor’s latest scan.'
  const sold = rows.value.filter((r) => r.offered).length
  const owner = c.role === 'tribelt' ? 'Tribelt' : c.shortName
  return sold === 0 ? `${owner}, as its latest scan found it.` : `${owner} sells ${String(sold)} of Tribelt's five belt families.`
})
</script>

<template>
  <SectionView section="competitors" :title="title" :tabs="companyTabs(companyId)">
    <template v-if="company" #aside>
      <p class="m-0 text-sm text-muted stage:text-right stage:text-[26px]">{{ whereLine(company) }}</p>
    </template>

    <p v-if="notFound" role="alert" class="m-0 text-orange-ink">No company has this id.</p>
    <p v-else-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading || !company" class="m-0 text-muted">Loading the company…</p>
    <div v-else class="flex flex-col gap-12 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_minmax(0,1fr)] stage:gap-24">
      <section aria-labelledby="crawl-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-6">
        <h2 id="crawl-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">The crawl</h2>
        <div class="flex flex-wrap items-center gap-3">
          <a
            v-if="company.websiteUrl"
            :href="company.websiteUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex min-h-11 items-center gap-2 rounded-full border-2 border-black px-5 font-medium text-black no-underline hover:bg-black hover:text-white stage:min-h-16 stage:px-8 stage:text-[26px]"
          >
            Visit website<span class="sr-only"> (opens in a new tab)</span>
          </a>
          <button
            v-if="isAdmin && company.websiteUrl"
            type="button"
            class="min-h-11 rounded-full border-0 bg-orange px-5 font-semibold text-black hover:bg-orange-soft disabled:opacity-60 stage:min-h-16 stage:px-8 stage:text-[26px]"
            :disabled="isQueuing"
            @click="rescan({ competitorIds: [company.id] })"
          >
            {{ company.scans > 0 ? 'Scan again' : 'Scan this website' }}
          </button>
        </div>
        <p v-if="batch" aria-live="polite" class="m-0 text-sm font-medium stage:text-[24px]">{{ rescanSummary(batch) }}</p>
        <p v-if="rescanError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ rescanError }}</p>
        <p class="m-0 text-ink-2 stage:text-[28px]">{{ lastScanLabel(company.lastScan) ?? 'No scan of its website has run yet.' }}</p>
        <div class="flex flex-col stage:min-h-0 stage:overflow-y-auto">
          <ScanResult v-if="scan" v-model:language="language" :scan="scan" />
          <p v-else-if="latest" class="m-0 text-muted">Loading the scan…</p>
        </div>
      </section>

      <section aria-labelledby="belts-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-6 stage:overflow-y-auto">
        <h2 id="belts-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Which Tribelt belts it sells</h2>
        <table class="w-full border-collapse stage:text-[28px]">
          <caption class="pb-2 text-left text-sm text-muted stage:text-[24px]">Its own names for Tribelt's belt families</caption>
          <tbody>
            <tr v-for="row in rows" :key="row.family" class="border-t border-grey align-top">
              <th scope="row" class="w-32 py-2 text-left font-medium text-ink-2 stage:w-[260px] stage:py-3">{{ row.label }}</th>
              <td class="py-2 stage:py-3" :class="row.offered ? 'text-black' : 'text-muted'">{{ row.name }}</td>
            </tr>
          </tbody>
        </table>
        <h3 class="m-0 text-sm font-semibold tracking-[0.04em] text-orange-ink uppercase stage:text-[24px]">Terms from the map used on the site</h3>
        <span class="text-sm text-muted stage:text-[24px]">{{ termsTitle(used.length) }}</span>
        <ul v-if="used.length > 0" aria-label="Its terms in the terminology map" class="m-0 flex list-none flex-wrap gap-2 p-0">
          <li
            v-for="t in used"
            :key="`${t.term}-${t.language}`"
            class="inline-flex items-center gap-2 rounded-full bg-mist px-3 py-1 text-sm stage:px-5 stage:py-2 stage:text-[24px]"
          >
            <span aria-hidden="true" class="size-3 flex-none rounded-full stage:size-4" :class="tierFill(t.tier)"></span>
            {{ t.term }}<span class="sr-only">, {{ tierLabel(t.tier) }}</span>
          </li>
        </ul>
        <p v-if="mentioned" class="m-0 text-sm text-ink-2 stage:text-[24px]">
          {{ mentioned }} <RouterLink to="/ai-search/prompt-test" class="underline underline-offset-4">The Prompt test</RouterLink>
        </p>
        <CompanyAdmin v-if="isAdmin" :company="company" @removed="router.push('/competitors/all')" />
      </section>

      <section aria-labelledby="ready-title" class="flex flex-col gap-6 stage:min-h-0 stage:overflow-y-auto">
        <h2 id="ready-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Ready for AI</h2>
        <CompanyReadiness :company-id="company.id" />
        <ReadinessFiles :company-id="company.id" />
      </section>
    </div>
  </SectionView>
</template>
