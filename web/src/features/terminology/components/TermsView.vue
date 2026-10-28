<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { TONE_VAR } from '../../../shared/ui/charts/tones'
import { useOverview } from '../../home/composables/useOverview'
import { usePlan } from '../../plan/composables/usePlan'
import { useProducts } from '../../products/composables/useProducts'
import { familyTitle, tribeltCalls } from '../../products/domain/products'
import SectionView from '../../shell/components/SectionView.vue'
import { familyPath } from '../../shell/domain/sections'
import { useTermMap } from '../composables/useTermSearch'
import { americanEnglishCount, DONUT_RADIUS, donutArcs, languageSlices, pickedLanguage, termsFigures, termsHeadline } from '../domain/overview'

const { overview } = useOverview()
const { plan } = usePlan()
const { terms } = useTermMap()
const { products, error } = useProducts()

const goal = computed(() => plan.value?.watched.find((w) => w.target === 'termsUsed')?.goal)
const title = computed(() => termsHeadline(overview.value?.terms))
const figures = computed(() => termsFigures(overview.value?.terms, goal.value))
const slices = computed(() => languageSlices(overview.value?.terms))

// The language the donut stresses; the one with the most terms until another is picked.
const chosen = ref<string | null>(null)
const pickedSlice = computed(() => slices.value.find((s) => s.value === chosen.value) ?? slices.value[0])
const arcs = computed(() => donutArcs(slices.value, pickedSlice.value?.value ?? null))
const picked = computed(() => pickedLanguage(pickedSlice.value, americanEnglishCount(terms.value)))
const donutLabel = computed(
  () => `The ${String(overview.value?.terms.total ?? 0)} terms by language: ${slices.value.map((s) => `${s.label} ${String(s.count)}`).join(', ')}.`,
)
const sliceColour = (value: string, tone: string): string => (value === pickedSlice.value?.value ? 'var(--color-orange)' : TONE_VAR[tone as keyof typeof TONE_VAR])
</script>

<template>
  <SectionView section="terms" :title="title" :source="['Search term list, our research of 22 October 2026', 'Competitors: also found by live scans']">
    <template #lede>
      <p class="m-0 max-w-[64ch] text-base text-ink-2 stage:text-[40px]">
        The list of search terms buyers use for metal conveyor belts also holds the words competitors use. When tribelt.nl doesn't use a term, Tribelt doesn't turn up for
        that search.
      </p>
    </template>
    <div class="flex flex-col gap-10 stage:min-h-0 stage:gap-16">
      <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-5">{{ error }}</p>

      <section aria-label="Key figures" class="grid grid-cols-1 gap-4 sm:grid-cols-3 stage:gap-14">
        <div v-for="f in figures" :key="f.text" class="flex flex-col gap-2 rounded-3xl bg-mist px-6 py-5 stage:gap-5 stage:rounded-[64px] stage:px-16 stage:py-12">
          <span class="font-display text-5xl leading-none font-extrabold tracking-[-0.02em] stage:text-[150px]">{{ f.big }}</span>
          <span class="text-base font-semibold stage:text-[44px]">{{ f.text }}</span>
          <span v-if="f.note" class="text-sm text-muted stage:text-[36px]">{{ f.note }}</span>
        </div>
      </section>

      <div class="grid grid-cols-1 gap-8 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.3fr)] stage:min-h-0 stage:grid-cols-[minmax(0,0.9fr)_minmax(0,1.3fr)] stage:gap-16">
        <section aria-labelledby="lang-title" class="flex flex-col gap-5 rounded-[28px] border-2 border-mist p-6 stage:gap-10 stage:rounded-[72px] stage:border-[6px] stage:p-16">
          <h2 id="lang-title" class="m-0 font-display text-2xl font-extrabold stage:text-[68px]">The terms by language</h2>
          <div class="flex flex-wrap items-center gap-x-6 gap-y-4 stage:flex-nowrap stage:gap-14">
            <svg role="img" :aria-label="donutLabel" viewBox="0 0 200 200" class="size-[200px] flex-none stage:size-[520px]">
              <circle
                v-for="(a, i) in arcs"
                :key="a.value"
                cx="100"
                cy="100"
                :r="DONUT_RADIUS"
                fill="none"
                transform="rotate(-90 100 100)"
                :stroke="sliceColour(a.value, slices[i]?.tone ?? 'silver')"
                :stroke-width="a.width"
                :stroke-dasharray="a.dash"
                :stroke-dashoffset="a.offset"
              />
              <text x="100" y="98" text-anchor="middle" font-size="38" font-weight="800" fill="currentColor">{{ overview?.terms.total ?? 0 }}</text>
              <text x="100" y="122" text-anchor="middle" font-size="14" fill="var(--color-ink-2)">terms</text>
            </svg>
            <div role="group" aria-label="Pick a language" class="flex min-w-[12rem] flex-1 flex-col gap-1.5 stage:gap-4">
              <button
                v-for="s in slices"
                :key="s.value"
                type="button"
                :aria-pressed="s.value === pickedSlice?.value"
                :aria-label="`${s.label}: ${String(s.count)} terms`"
                class="flex min-h-12 w-full cursor-pointer items-center gap-3 rounded-xl border-2 bg-transparent px-3.5 text-black hover:bg-mist stage:min-h-[128px] stage:gap-8 stage:rounded-[32px] stage:border-[6px] stage:px-9"
                :class="s.value === pickedSlice?.value ? 'border-black bg-white' : 'border-transparent'"
                @click="chosen = s.value"
              >
                <span aria-hidden="true" class="size-3.5 flex-none rounded-[4px] stage:size-9" :style="{ background: sliceColour(s.value, s.tone) }"></span>
                <span class="flex-1 text-left text-base font-semibold stage:text-[44px]">{{ s.label }}</span>
                <span class="text-base font-extrabold stage:text-[44px]">{{ s.count }}</span>
              </button>
            </div>
          </div>
          <div v-if="picked" class="flex flex-col gap-1.5 rounded-[18px] bg-orange-pale px-5 py-4 stage:gap-4 stage:rounded-[48px] stage:px-12 stage:py-10">
            <span class="text-base font-bold stage:text-[46px]">{{ picked.title }}</span>
            <span class="text-sm text-ink-2 stage:text-[38px] stage:leading-[1.5]">{{ picked.text }}</span>
          </div>
        </section>

        <section aria-labelledby="families-title" class="flex flex-col gap-4 stage:gap-8">
          <div class="flex flex-wrap items-baseline justify-between gap-3">
            <h2 id="families-title" class="m-0 font-display text-2xl font-extrabold stage:text-[68px]">The belt families</h2>
            <span class="text-sm text-muted stage:text-[36px]">Open a family to see its terms, who sells it, and the AI answers about it.</span>
          </div>
          <ul aria-label="Belt families" class="m-0 list-none border-t-2 border-black p-0 stage:border-t-[6px]">
            <li v-for="p in products" :key="p.family" class="border-b border-grey stage:border-b-[3px]">
              <RouterLink
                :to="familyPath(p.family)"
                class="grid min-h-[72px] grid-cols-[minmax(0,1fr)_auto_1.5rem] items-center gap-4 rounded-xl px-2 py-3 text-black no-underline hover:bg-mist hover:text-black stage:min-h-[200px] stage:grid-cols-[minmax(0,1fr)_auto_64px] stage:gap-10 stage:rounded-[32px] stage:px-6 stage:py-8"
              >
                <span class="flex min-w-0 flex-col gap-1 stage:gap-3">
                  <span class="text-lg font-bold stage:text-[52px]">{{ familyTitle(p.family) }}</span>
                  <span class="text-sm text-muted stage:text-[36px]">{{ tribeltCalls(p) }}</span>
                </span>
                <span class="flex flex-col items-end gap-0.5">
                  <span class="text-sm font-bold text-ink-2 stage:text-[40px]">{{ p.termsUsed }} of {{ p.terms.length }}</span>
                  <span class="text-xs text-muted stage:text-[32px]">terms tribelt.nl uses</span>
                </span>
                <svg aria-hidden="true" viewBox="0 0 24 24" class="size-[18px] stage:size-14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M9 5l7 7-7 7" />
                </svg>
              </RouterLink>
            </li>
          </ul>
          <p class="m-0 text-sm text-muted stage:text-[36px]">
            Every term in the search term list, ranked by how many competitors use it, is on
            <RouterLink to="/terms/all" class="font-bold">its own page</RouterLink>.
          </p>
        </section>
      </div>
    </div>
  </SectionView>
</template>
