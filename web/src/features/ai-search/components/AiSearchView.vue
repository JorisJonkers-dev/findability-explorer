<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import DonutChart from '../../../shared/ui/charts/DonutChart.vue'
import PillButton from '../../../shared/ui/PillButton.vue'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import { tally } from '../../prompt-test/domain/results'
import SectionView from '../../shell/components/SectionView.vue'
import { useAiSearchPage } from '../composables/useAiSearchPage'
import { pct } from '../domain/summary'
import { type Scope, slice } from '../domain/voice'

const page = useAiSearchPage()
const scope = ref<Scope>('all')
const SCOPES: readonly { id: Scope; label: string }[] = [
  { id: 'all', label: 'All answers' },
  { id: 'twente', label: 'Answers about Twente' },
]
const shown = computed(() => (page.summary.value ? slice(scope.value, page.cells.value, page.summary.value, page.plan.value) : undefined))
const total = computed(() => tally(page.cells.value).answers)
</script>

<template>
  <SectionView section="ai-search" :title="page.headline.value">
    <template #lede>
      <p class="m-0 max-w-[64ch] text-base text-ink-2 text-pretty sm:text-lg stage:max-w-[2300px] stage:text-[40px]">{{ page.lede.value }}</p>
    </template>

    <p v-if="page.error.value" role="alert" class="m-0 text-orange-ink">{{ page.error.value }}</p>
    <p v-else-if="page.isLoading.value || !page.summary.value || !shown" class="m-0 text-muted">Loading the summary…</p>
    <div v-else class="flex flex-col gap-10 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1fr)_minmax(0,0.95fr)] stage:gap-24">
      <div class="flex flex-col gap-8 stage:min-h-0 stage:gap-12">
        <section aria-label="Key figures" class="grid gap-4 sm:grid-cols-3 stage:gap-8">
          <RouterLink
            v-for="f in page.figures.value"
            :key="f.text"
            :to="f.to"
            class="flex flex-col gap-2 rounded-3xl bg-mist px-6 py-5 text-black no-underline hover:bg-grey hover:text-black stage:gap-3 stage:rounded-[40px] stage:px-10 stage:py-8"
          >
            <span class="font-display text-5xl leading-none font-semibold tracking-[-0.02em] stage:text-[120px]">{{ f.big }}</span>
            <span class="font-semibold stage:text-[34px]">{{ f.text }}</span>
            <span class="text-sm text-muted stage:text-[26px]">{{ f.note }}</span>
          </RouterLink>
        </section>

        <section aria-labelledby="donut-title" class="flex flex-col gap-5 rounded-[28px] border-2 border-grey p-6 stage:gap-9 stage:rounded-[48px] stage:border-[3px] stage:p-14">
          <SectionHeading id="donut-title">How often the answers name Tribelt</SectionHeading>
          <div role="group" aria-label="Which answers" class="flex flex-wrap gap-2 stage:gap-4">
            <PillButton v-for="s in SCOPES" :key="s.id" :pressed="scope === s.id" @click="scope = s.id">{{ s.label }}</PillButton>
          </div>
          <div class="[&_ul]:text-base [&_svg]:size-[220px] stage:[&_svg]:size-[480px] stage:[&_ul]:gap-5 stage:[&_ul]:text-[34px] stage:[&>div]:gap-x-16">
            <DonutChart
              :centre="shown.centre"
              :centre-sub="shown.sub"
              :segments="[
                { label: shown.namedText, value: shown.named, tone: 'orange' },
                { label: shown.restText, value: shown.total - shown.named, tone: 'blue-grey' },
              ]"
            />
          </div>
          <p v-if="shown.goal" class="m-0 text-ink-2 stage:text-[30px]">{{ shown.goal }}</p>
          <RouterLink
            to="/ai-search/answers"
            class="inline-flex min-h-12 items-center self-start rounded-full bg-black px-6 font-bold text-white no-underline hover:bg-orange-ink hover:text-white stage:min-h-[96px] stage:px-12 stage:text-[34px]"
          >
            See every answer
          </RouterLink>
        </section>
      </div>

      <div class="flex flex-col gap-8 stage:min-h-0 stage:gap-10">
        <section aria-labelledby="named-title" class="flex flex-col gap-3 stage:gap-5">
          <div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
            <SectionHeading id="named-title">Who the answers name</SectionHeading>
            <span class="text-sm text-muted stage:text-[26px]">Share of the {{ total }} answers that name each company</span>
          </div>
          <p v-if="page.rows.value.length === 0" class="m-0 text-muted">No Prompt run has finished yet.</p>
          <ol v-else class="m-0 list-none border-t-2 border-black p-0 stage:border-t-4">
            <li v-for="r in page.rows.value" :key="r.id" class="border-b border-grey">
              <component
                :is="r.to ? RouterLink : 'div'"
                :to="r.to"
                class="grid min-h-14 grid-cols-[2.5rem_minmax(0,1fr)_4.5rem_1.5rem] items-center gap-3 rounded-xl px-2 py-1.5 text-black no-underline stage:min-h-[104px] stage:grid-cols-[100px_minmax(0,1fr)_180px_48px] stage:gap-6 stage:px-4"
                :class="[r.to ? 'hover:bg-mist hover:text-black' : '', r.tribelt ? 'bg-orange-pale text-orange-ink' : '']"
              >
                <span class="text-lg font-extrabold text-muted stage:text-[40px]">{{ r.rank }}</span>
                <span class="flex flex-col">
                  <span class="text-lg stage:text-[40px]" :class="r.to ? 'font-semibold' : 'font-bold'">{{ r.name }}</span>
                  <span v-if="r.note" class="text-sm text-muted stage:text-[26px]">{{ r.note }}</span>
                </span>
                <span class="text-right text-xl font-extrabold stage:text-[44px]">{{ pct(r.share) }}</span>
                <svg v-if="r.to" aria-hidden="true" viewBox="0 0 24 24" class="size-[18px] stage:size-9" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M9 5l7 7-7 7" />
                </svg>
                <span v-else></span>
              </component>
            </li>
          </ol>
          <p class="m-0 text-sm text-muted stage:text-[26px]">One answer can name several companies, so the shares add up to more than 100%.</p>
        </section>

        <RouterLink
          to="/ai-search/readiness"
          class="flex items-center justify-between gap-6 rounded-3xl bg-mist px-7 py-6 text-black no-underline hover:bg-grey hover:text-black stage:mt-auto stage:rounded-[40px] stage:px-14 stage:py-10"
        >
          <span class="flex flex-col gap-1.5">
            <span class="text-xl font-extrabold stage:text-[44px]">AI readiness by site</span>
            <span class="text-ink-2 stage:text-[30px]">{{ page.readinessLine.value }}</span>
          </span>
          <svg aria-hidden="true" viewBox="0 0 24 24" class="size-6 flex-none stage:size-12" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 5l7 7-7 7" />
          </svg>
        </RouterLink>
      </div>
    </div>
  </SectionView>
</template>
