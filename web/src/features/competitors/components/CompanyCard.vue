<script setup lang="ts">
import { vScrollHint } from '../../../shared/scrollHint'
import { computed, toRef } from 'vue'
import { RouterLink } from 'vue-router'
import eyelink from '../../../assets/eyelink.webp'
import type { Term, TermsSource } from '../../../infrastructure/api/types.gen'
import { tierFill } from '../../../shared/tiers'
import { tierLabel } from '../../../shared/vocabulary'
import ProgressBar from '../../../shared/ui/ProgressBar.vue'
import RoundedPhoto from '../../../shared/ui/RoundedPhoto.vue'
import { isInProgress, progress } from '../../scan/domain/status'
import { useCompany } from '../composables/useCompany'
import CompanyAdmin from './CompanyAdmin.vue'
import CompanyReadiness from '../../readiness/components/CompanyReadiness.vue'
import { usePromptResults } from '../../prompt-test/composables/usePromptTest'
import { mentionLine } from '../../prompt-test/domain/results'
import { type Company, companyTerms, lastScanLabel, scanCountLabel, scanLinkLabel } from '../domain/company'
import { archivedByHaiku } from '../domain/checks'
import { beltRows, companyKind, termsTitle, whereLine } from '../domain/list'

const props = defineProps<{ company: Company; terms: readonly Term[]; isAdmin: boolean; termsFrom?: TermsSource }>()
const emit = defineEmits<{ scan: [url: string]; removed: [] }>()

/** How many of its scans are listed; every one of its terms is, in a list that scrolls. */
const SCANS_SHOWN = 3

const { detail } = useCompany(toRef(() => props.company.id))
const used = computed(() => companyTerms(props.terms, props.company.shortName))
const scans = computed(() => (detail.value ? detail.value.scans.slice(0, SCANS_SHOWN) : []))
// The list carries the last scan; the company's own scans stand in until it does.
const newest = computed(() => props.company.lastScan ?? scans.value[0])
const lastScan = computed(() => lastScanLabel(newest.value))
/** The scan running now, if one is, for its progress bar. */
const running = computed(() => (newest.value && isInProgress(newest.value.status) ? newest.value : undefined))
const { results: prompts } = usePromptResults()
/** How often the latest Prompt run's answers name the company. */
const mentioned = computed(() => mentionLine(prompts.value, props.company.id))
</script>

<template>
  <!-- On the stage the whole card scrolls, so everything on it can be brought into view. -->
  <section
    v-scroll-hint
    aria-labelledby="company-name"
    class="scroll-hint flex flex-col gap-5 stage:min-h-0 stage:gap-9 stage:overflow-y-auto stage:pr-6"
  >
    <p class="sr-only" aria-live="polite">Showing {{ company.shortName }}.</p>
    <div class="flex flex-col gap-1.5 stage:gap-2.5">
      <span class="text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[26px]">{{ companyKind(company) }}</span>
      <h2 id="company-name" class="m-0 font-display text-4xl leading-none font-semibold tracking-[-0.02em] stage:text-[104px]">
        {{ company.shortName }}
      </h2>
      <span class="text-ink-2 stage:text-[32px]">{{ whereLine(company) }}</span>
    </div>

    <div class="flex flex-wrap items-center gap-3 stage:gap-4">
      <template v-if="company.websiteUrl">
        <a
          :href="company.websiteUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex min-h-11 items-center gap-2 rounded-full border-2 border-black px-5 font-medium text-black no-underline hover:bg-black hover:text-white stage:min-h-[88px] stage:gap-3.5 stage:border-[3px] stage:px-10 stage:text-[30px]"
        >
          Visit website
          <svg aria-hidden="true" viewBox="0 0 16 16" class="size-4 stage:size-7" fill="none" stroke="currentColor" stroke-width="1.6">
            <path d="M9 3h4v4M13 3 7 9M11 9.5V13H3V5h3.5" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <span class="sr-only">(opens in a new tab)</span>
        </a>
        <button
          v-if="isAdmin"
          type="button"
          class="min-h-11 rounded-full border-0 bg-orange px-5 font-semibold text-black hover:bg-black hover:text-white stage:min-h-[88px] stage:px-10 stage:text-[30px]"
          @click="emit('scan', company.websiteUrl)"
        >
          {{ company.scans > 0 ? 'Re-scan' : 'Scan this website' }}
        </button>
      </template>
      <p v-else class="m-0 text-muted stage:text-[30px]">The research found no website for this company.</p>
      <RouterLink :to="`/competitors/${company.id}`" class="font-medium stage:text-[30px]">Latest scan</RouterLink>
      <RouterLink :to="`/competitors/${company.id}/history`" class="font-medium stage:text-[30px]">Over time</RouterLink>
    </div>
    <div class="flex flex-col gap-0.5 border-l-4 border-orange pl-3 stage:gap-1 stage:border-l-[6px] stage:pl-6" data-testid="scan-stats">
      <p class="m-0 font-display text-xl font-semibold stage:text-[44px]">{{ scanCountLabel(company.scans, running !== undefined) }}</p>
      <p v-if="lastScan" class="m-0 stage:text-[28px]" :class="running ? 'font-medium text-orange-ink' : 'text-ink-2'">{{ lastScan }}</p>
      <p v-else-if="company.scans === 0" class="m-0 text-ink-2 stage:text-[28px]">No scan of its website has run yet.</p>
      <p v-if="newest?.blocked" class="m-0 text-sm text-ink-2 stage:text-[26px]" data-testid="blocked-note">
        Its site lets people in but turns automated readers away with bot protection, so the scan could not read it.
        Only the site's owner can let scanners in. Each scan still records its AI readiness.
      </p>
      <ProgressBar
        v-if="running"
        class="mt-1.5 stage:mt-3"
        :fraction="progress(running).fraction"
        :label="`Scan of ${company.shortName}`"
        :text="progress(running).label"
      />
    </div>
    <p v-if="company.archived" class="m-0 font-medium text-orange-ink stage:text-[28px]">
      {{ archivedByHaiku(company) ? 'Claude Haiku archived it as no competitor.' : 'Archived.' }} It is hidden from the list and the re-scans.
    </p>
    <CompanyAdmin v-if="isAdmin" :company="company" @removed="emit('removed')" />

    <CompanyReadiness :company-id="company.id" />
    <p v-if="mentioned" class="m-0 text-sm text-ink-2 stage:text-[26px]" data-testid="prompt-mentions">
      <span class="font-semibold text-black">AI answers:</span> {{ mentioned }}
      <RouterLink to="/ai-search/prompt-test" class="underline underline-offset-4">The Prompt test</RouterLink>
    </p>

    <table class="w-full border-collapse stage:text-[30px]">
      <caption class="pb-2 text-left text-sm text-muted stage:pb-3 stage:text-[26px]">Its names for Tribelt's belt families</caption>
      <tbody>
        <tr v-for="row in beltRows(company)" :key="row.family" class="border-t border-grey align-top">
          <th scope="row" class="w-32 py-2 text-left font-medium text-ink-2 stage:w-[280px] stage:py-4">{{ row.label }}</th>
          <td class="py-2 stage:py-4" :class="row.offered ? 'text-black' : 'text-muted'">{{ row.name }}</td>
        </tr>
      </tbody>
    </table>

    <div class="flex flex-col gap-3 stage:gap-4">
      <span class="text-sm text-muted stage:text-[26px]">{{ termsTitle(used.length, company.shortName === 'Tribelt' ? (termsFrom ?? 'scan') : undefined) }}</span>
      <ul
        v-if="used.length > 0"
        v-scroll-hint
        aria-label="Its terms in the terminology map"
        class="scroll-hint m-0 flex max-h-64 list-none flex-wrap content-start gap-2 overflow-y-auto p-0 stage:max-h-none stage:gap-3 stage:overflow-visible"
      >
        <li
          v-for="t in used"
          :key="`${t.term}-${t.language}`"
          class="inline-flex items-center gap-2 rounded-full bg-mist px-3 py-1 text-sm stage:gap-3 stage:px-[22px] stage:py-2 stage:text-[26px]"
        >
          <span aria-hidden="true" class="size-3 flex-none rounded-full stage:size-[18px]" :class="tierFill(t.tier)"></span>
          {{ t.term }}<span class="sr-only">, {{ tierLabel(t.tier) }}</span>
        </li>
      </ul>
      <p class="m-0 flex flex-wrap items-baseline gap-x-4 gap-y-1 text-sm text-muted stage:gap-x-6 stage:text-[24px]">
        <span v-if="scans.length > 0">Latest scans:</span>
        <RouterLink v-for="scan in scans" :key="scan.id" :to="{ path: `/scan/${scan.id}`, query: { company: company.id } }">
          {{ scanLinkLabel(scan) }}
        </RouterLink>
      </p>
    </div>

    <RoundedPhoto :src="eyelink" alt="Tribelt eyelink belts" class="stage:mt-auto stage:h-[380px] stage:flex-none" />
  </section>
</template>
