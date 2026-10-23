<script setup lang="ts">
import { computed, toRef } from 'vue'
import { RouterLink } from 'vue-router'
import eyelink from '../../../assets/eyelink.webp'
import type { Term } from '../../../infrastructure/api/types.gen'
import { tierFill } from '../../../shared/tiers'
import { tierLabel } from '../../../shared/vocabulary'
import RoundedPhoto from '../../../shared/ui/RoundedPhoto.vue'
import { recentLabel } from '../../scan/domain/recent'
import { useCompany } from '../composables/useCompany'
import CompanyAdmin from './CompanyAdmin.vue'
import { type Company, companyTerms, scanCountLabel, scanOutcome } from '../domain/company'
import { beltRows, companyKind, termsTitle, whereLine } from '../domain/list'

const props = defineProps<{ company: Company; terms: readonly Term[]; isAdmin: boolean }>()
const emit = defineEmits<{ scan: [url: string]; removed: [] }>()

/** How many of its terms are shown as chips, and how many of its scans are listed. */
const TERMS_SHOWN = 12
const SCANS_SHOWN = 3

const { detail } = useCompany(toRef(() => props.company.id))
const used = computed(() => companyTerms(props.terms, props.company.shortName))
const scans = computed(() => (detail.value ? detail.value.scans.slice(0, SCANS_SHOWN) : []))
</script>

<template>
  <section aria-labelledby="company-name" class="flex flex-col gap-5 stage:min-h-0 stage:gap-9">
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
      <RouterLink :to="`/competitors/${company.id}/history`" class="font-medium stage:text-[30px]">Search terms and history</RouterLink>
    </div>
    <p v-if="company.archived" class="m-0 font-medium text-orange-ink stage:text-[28px]">Archived: hidden from the list and the re-scans.</p>
    <CompanyAdmin v-if="isAdmin" :company="company" @removed="emit('removed')" />

    <table class="w-full border-collapse stage:text-[30px]">
      <caption class="pb-2 text-left text-sm text-muted stage:pb-3 stage:text-[26px]">Its names for Tribelt's belt families</caption>
      <tbody>
        <tr v-for="row in beltRows(company)" :key="row.family" class="border-t border-grey align-top">
          <th scope="row" class="w-32 py-2 text-left font-medium text-ink-2 stage:w-[280px] stage:py-4">{{ row.label }}</th>
          <td class="py-2 stage:py-4" :class="row.offered ? 'text-black' : 'text-muted'">{{ row.name }}</td>
        </tr>
      </tbody>
    </table>

    <div class="flex flex-col gap-3 stage:min-h-0 stage:shrink stage:gap-4">
      <span class="text-sm text-muted stage:text-[26px]">{{ termsTitle(used.length) }}</span>
      <ul v-if="used.length > 0" class="m-0 flex list-none flex-wrap gap-2 p-0 stage:min-h-0 stage:gap-3 stage:overflow-hidden">
        <li
          v-for="t in used.slice(0, TERMS_SHOWN)"
          :key="`${t.term}-${t.language}`"
          class="inline-flex items-center gap-2 rounded-full bg-mist px-3 py-1 text-sm stage:gap-3 stage:px-[22px] stage:py-2 stage:text-[26px]"
        >
          <span aria-hidden="true" class="size-3 flex-none rounded-full stage:size-[18px]" :class="tierFill(t.tier)"></span>
          {{ t.term }}<span class="sr-only">, {{ tierLabel(t.tier) }}</span>
        </li>
      </ul>
      <p class="m-0 flex flex-wrap items-baseline gap-x-4 gap-y-1 text-sm text-muted stage:gap-x-6 stage:text-[24px]">
        <span>{{ scanCountLabel(company.scans) }}</span>
        <RouterLink v-for="scan in scans" :key="scan.id" :to="{ path: `/scan/${scan.id}`, query: { company: company.id } }">
          {{ recentLabel(scan) }}: {{ scanOutcome(scan) }}
        </RouterLink>
      </p>
    </div>

    <RoundedPhoto :src="eyelink" alt="Tribelt eyelink belts" class="stage:mt-auto stage:h-[380px] stage:flex-none" />
  </section>
</template>
