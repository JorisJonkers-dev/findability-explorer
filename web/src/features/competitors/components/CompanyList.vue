<script setup lang="ts">
import { computed, ref } from 'vue'
import { familyLabel, PRODUCT_FAMILIES } from '../../../shared/vocabulary'
import PillButton from '../../../shared/ui/PillButton.vue'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import type { Company } from '../domain/company'
import { countLabel, countryCode, familyDots, filterCompanies, type Region, REGIONS, rowLabel } from '../domain/list'

const props = withDefaults(
  defineProps<{ companies: readonly Company[]; selectedId: string | undefined; loading: boolean; archived?: readonly Company[] }>(),
  { archived: () => [] },
)
const emit = defineEmits<{ pick: [id: string] }>()

const query = ref('')
const region = ref<Region>('all')
const shown = computed(() => filterCompanies(props.companies, query.value, region.value))

/** The column heads, short on a phone. */
const SHORT: Record<(typeof PRODUCT_FAMILIES)[number], string> = {
  eyelink: 'Eye',
  open_spiral: 'Open',
  closed_spiral: 'Closed',
  tri_flex_turn_curve: 'Tri',
  other: 'Other',
}
</script>

<template>
  <section aria-labelledby="companies-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-8">
    <SectionHeading id="companies-title">Who makes which belt</SectionHeading>
    <div class="flex flex-wrap items-center gap-3 stage:gap-4">
      <label for="company-q" class="sr-only">Find a company</label>
      <input
        id="company-q"
        v-model="query"
        type="search"
        placeholder="Find a company or country"
        spellcheck="false"
        class="h-11 w-full rounded-full border-2 border-grey bg-white px-5 text-base sm:w-80 stage:h-[88px] stage:w-[640px] stage:border-[3px] stage:px-8 stage:text-[30px]"
      />
      <div role="group" aria-label="Show" class="flex flex-wrap gap-2 stage:gap-3">
        <PillButton v-for="r in REGIONS" :key="r.value" :pressed="region === r.value" @click="region = r.value">{{ r.label }}</PillButton>
      </div>
    </div>

    <div class="flex flex-col stage:min-h-0 stage:flex-1">
      <div
        aria-hidden="true"
        class="grid grid-cols-[minmax(0,1fr)_repeat(5,2.25rem)_2rem] items-end gap-1 border-b-2 border-black px-2 pb-2 text-center text-xs font-semibold stage:grid-cols-[minmax(0,1fr)_repeat(5,150px)_110px] stage:gap-3 stage:border-b-[3px] stage:px-5 stage:pb-4 stage:text-[24px]"
      >
        <span class="text-left font-medium text-muted">Company</span>
        <span v-for="family in PRODUCT_FAMILIES" :key="family">
          <span class="stage:hidden">{{ SHORT[family] }}</span><span class="hidden stage:inline">{{ family === 'other' ? 'Other' : familyLabel(family) }}</span>
        </span>
        <span>Country</span>
      </div>
      <p v-if="loading" class="m-0 py-4 text-muted">Loading the companies…</p>
      <ul v-else aria-label="Companies" class="m-0 max-h-[26rem] list-none overflow-y-auto p-0 stage:max-h-none stage:min-h-0 stage:flex-1">
        <li v-for="company in shown" :key="company.id" class="border-b border-grey">
          <button
            type="button"
            :aria-pressed="company.id === selectedId"
            :aria-label="rowLabel(company)"
            class="grid min-h-11 w-full cursor-pointer grid-cols-[minmax(0,1fr)_repeat(5,2.25rem)_2rem] items-center gap-1 border-0 px-2 py-1 text-left text-black stage:min-h-[76px] stage:grid-cols-[minmax(0,1fr)_repeat(5,150px)_110px] stage:gap-3 stage:px-5 stage:py-1.5"
            :class="company.id === selectedId ? 'bg-mist' : 'bg-transparent hover:bg-mist'"
            @click="emit('pick', company.id)"
          >
            <span class="truncate stage:text-[32px]" :class="company.id === selectedId ? 'font-semibold' : ''">{{ company.shortName }}</span>
            <span v-for="dot in familyDots(company)" :key="dot.family" class="flex justify-center">
              <span class="size-4 rounded-full stage:size-[30px]" :class="dot.offers ? 'bg-orange' : 'bg-grey'"></span>
            </span>
            <span class="text-center text-sm text-ink-2 stage:text-[26px]">{{ countryCode(company.country) }}</span>
          </button>
        </li>
      </ul>
    </div>

    <div v-if="archived.length > 0" class="flex flex-wrap items-center gap-2 text-sm stage:gap-3 stage:text-[24px]">
      <span class="text-muted">Archived:</span>
      <button
        v-for="company in archived"
        :key="company.id"
        type="button"
        :aria-pressed="company.id === selectedId"
        class="min-h-11 rounded-full border-2 border-grey bg-white px-3 text-sm text-black hover:border-black stage:min-h-[64px] stage:border-[3px] stage:px-6 stage:text-[24px]"
        @click="emit('pick', company.id)"
      >
        {{ company.shortName }}
      </button>
    </div>
    <p class="m-0 flex flex-wrap gap-x-6 gap-y-1 text-sm text-muted stage:gap-8 stage:text-[24px]">
      <span class="inline-flex items-center gap-2 stage:gap-3"><span aria-hidden="true" class="size-3.5 rounded-full bg-orange stage:size-6"></span>Offers this family</span>
      <span class="inline-flex items-center gap-2 stage:gap-3"><span aria-hidden="true" class="size-3.5 rounded-full bg-grey stage:size-6"></span>Does not</span>
      <span aria-live="polite" class="ml-auto">{{ loading ? 'Loading…' : countLabel(shown.length, companies.length) }}</span>
    </p>
  </section>
</template>
