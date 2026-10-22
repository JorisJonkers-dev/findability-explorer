<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { tierFill } from '../../../shared/tiers'
import { languageLabel, scopeLabel, tierLabel } from '../../../shared/vocabulary'
import { scannedAt } from '../../scan/domain/results'
import { useTermMap } from '../../terminology/composables/useTermSearch'
import { useCompany } from '../composables/useCompany'
import { companyTerms, groupBeltNames, scanCountLabel, scanOutcome, sourceLabel } from '../domain/company'
import { countryName, creditedCompanies, languageNames, shortName } from '../domain/matrix'

/** How many terms show before the full list in the terminology. */
const FIRST_TERMS = 12

const props = defineProps<{ id: string }>()
const { detail, isLoading, notFound, error } = useCompany(computed(() => props.id))
const map = useTermMap()

const company = computed(() => detail.value?.competitor)
const name = computed(() => (company.value ? shortName(company.value.name, creditedCompanies(map.terms.value)) : ''))
const groups = computed(() => groupBeltNames(company.value?.beltNames ?? []))
const terms = computed(() => companyTerms(map.terms.value, name.value))
const scans = computed(() => detail.value?.scans ?? [])
const languages = computed(() => languageNames(company.value?.languages ?? []))
</script>

<template>
  <div class="flex flex-col gap-10">
    <RouterLink to="/competitors" class="self-start text-[15px] font-medium">
      <span aria-hidden="true">←</span> All competitors
    </RouterLink>

    <p v-if="notFound" role="alert" class="m-0 rounded-xl border border-line bg-surface p-6 text-ink-2">
      No company has this address. It may have been typed wrong.
    </p>
    <p v-else-if="error" role="alert" class="m-0 rounded-xl border border-line bg-surface p-6 text-ink-2">{{ error }}</p>
    <p v-else-if="isLoading || !company" class="m-0 text-muted">Loading the company…</p>

    <template v-else>
      <header class="flex max-w-[980px] flex-col gap-4">
        <h1 class="m-0 font-display text-[40px] leading-none font-semibold break-words md:text-[52px]">{{ name }}</h1>
        <p class="m-0 flex flex-wrap items-center gap-x-3 gap-y-1 text-lg text-ink-2">
          <span v-if="name !== company.name">{{ company.name }}.</span>
          <span>{{ countryName(company.country) }}.</span>
          <span v-if="languages !== ''">Website in {{ languages }}.</span>
          <span
            v-if="company.source === 'added'"
            class="rounded-full border border-straw/60 px-2.5 py-0.5 text-[13px] text-straw"
          >{{ sourceLabel(company) }}</span>
          <span v-else class="text-muted">{{ sourceLabel(company) }}</span>
        </p>
        <p v-if="company.groupCompany" class="m-0 text-ink-2">Part of Tribelt's own group, so never a competitor.</p>
        <div class="mt-2 flex flex-wrap items-center gap-3">
          <a
            v-if="company.websiteUrl"
            :href="company.websiteUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex h-12 items-center gap-2 rounded-lg border border-field px-5 font-semibold text-ink no-underline hover:text-straw-hover"
          >
            Visit website
            <svg aria-hidden="true" viewBox="0 0 16 16" class="size-4" fill="none" stroke="currentColor" stroke-width="1.75">
              <path d="M9 3h4v4M13 3 7 9M11 9.5V13H3V5h3.5" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
            <span class="sr-only">(opens in a new tab)</span>
          </a>
          <RouterLink
            v-if="company.websiteUrl"
            :to="{ path: '/scan', query: { competitor: company.id } }"
            class="inline-flex h-12 items-center rounded-lg bg-straw px-5 font-semibold text-on-straw no-underline hover:text-on-straw"
          >
            Scan this website
          </RouterLink>
          <p v-else class="m-0 text-muted">The research found no website for this company.</p>
          <span class="text-[15px] text-muted">{{ scanCountLabel(company.scans) }}</span>
        </div>
      </header>

      <section aria-labelledby="names-title" class="flex max-w-[980px] flex-col gap-4">
        <h2 id="names-title" class="m-0 font-display text-[28px] font-semibold">Its names for its belts</h2>
        <p v-if="groups.length === 0" class="m-0 text-ink-2">No belt names are known for this company yet.</p>
        <dl v-else class="m-0 grid grid-cols-1 gap-x-8 gap-y-4 md:grid-cols-[200px_1fr]">
          <template v-for="group in groups" :key="group.family">
            <dt class="font-semibold text-ink-2">{{ group.label }}</dt>
            <dd class="m-0 flex flex-col gap-1">
              <span v-for="belt in group.names" :key="belt">{{ belt }}</span>
            </dd>
          </template>
        </dl>
      </section>

      <section aria-labelledby="terms-title" class="flex max-w-[980px] flex-col gap-4">
        <h2 id="terms-title" class="m-0 font-display text-[28px] font-semibold">Terms it uses</h2>
        <p v-if="terms.length === 0" class="m-0 text-ink-2">The terminology map credits no terms to {{ name }}.</p>
        <template v-else>
          <p class="m-0 text-[15px] text-muted">
            {{ terms.length }} {{ terms.length === 1 ? 'term' : 'terms' }} in the terminology map, the most specific first.
          </p>
          <ul class="m-0 grid list-none grid-cols-[repeat(auto-fill,minmax(min(100%,300px),1fr))] gap-x-7 gap-y-3 p-0">
            <li v-for="term in terms.slice(0, FIRST_TERMS)" :key="`${term.term}|${term.language}`" class="flex flex-col gap-0.5">
              <span class="font-semibold break-words">{{ term.term }}</span>
              <span class="inline-flex items-center gap-2 text-sm text-muted">
                <span aria-hidden="true" class="size-2.5 flex-none rounded-full" :class="tierFill(term.tier)"></span>
                {{ tierLabel(term.tier) }}, {{ scopeLabel(term.family) }}, {{ languageLabel(term.language) }}
              </span>
            </li>
          </ul>
          <RouterLink :to="{ path: '/terminology', query: { user: name } }" class="self-start font-medium">
            {{ terms.length > FIRST_TERMS ? `See all ${String(terms.length)} terms` : 'See them' }} in the terminology
          </RouterLink>
        </template>
      </section>

      <section aria-labelledby="scans-title" class="flex max-w-[980px] flex-col gap-4">
        <h2 id="scans-title" class="m-0 font-display text-[28px] font-semibold">Its scans</h2>
        <p v-if="scans.length === 0" class="m-0 text-ink-2">Its website has not been scanned yet.</p>
        <ul v-else class="m-0 flex list-none flex-col p-0">
          <li v-for="scan in scans" :key="scan.id" class="border-t border-line">
            <RouterLink :to="`/scan/${scan.id}`" class="flex flex-wrap justify-between gap-x-4 gap-y-1 py-3 no-underline">
              <span class="font-medium">{{ scannedAt(scan.finishedAt ?? scan.createdAt) }}</span>
              <span class="text-muted">{{ scanOutcome(scan) }}</span>
            </RouterLink>
          </li>
        </ul>
      </section>
    </template>
  </div>
</template>
