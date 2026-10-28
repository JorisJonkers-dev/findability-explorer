<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { languageLabel, familyLabel, tierLabel } from '../../../shared/vocabulary'
import DonutChart from '../../../shared/ui/charts/DonutChart.vue'
import LanguageFilter from '../../../shared/ui/LanguageFilter.vue'
import { languageChoices, ofLanguage } from '../../terminology/domain/languages'
import SectionView from '../../shell/components/SectionView.vue'
import { companyPath, familyOfSlug } from '../../shell/domain/sections'
import { useProducts } from '../composables/useProducts'
import { aiShare, familyHeadline, familyLede, SELLERS_SHOWN, shareText, termLanguages, termsOnlyOthers, visibleSellers } from '../domain/products'

const props = defineProps<{ slug: string }>()
const family = computed(() => familyOfSlug(props.slug))
const { products, isLoading, error } = useProducts()
const product = computed(() => products.value.find((p) => p.family === family.value))
const title = computed(() => (product.value ? familyHeadline(product.value) : family.value ? familyLabel(family.value) : 'No such belt family.'))
const language = ref<string | null>(null)
const languages = computed(() => languageChoices(product.value?.terms.map((t) => t.language) ?? []))
const terms = computed(() => ofLanguage(product.value?.terms ?? [], language.value))
const usedHere = computed(() => terms.value.filter((t) => t.usedByTribelt).length)
const allSellers = ref(false)
const sellers = computed(() => visibleSellers(product.value?.sellers ?? [], allSellers.value))
const share = computed(() => (product.value ? aiShare(product.value) : undefined))
</script>

<template>
  <SectionView section="terms" :title="title" :back="{ to: '/terms', label: 'Back to Search terms' }" :source="['Terminology map, our research of 22 October 2026']">
    <template v-if="product" #lede>
      <p class="m-0 max-w-[68ch] text-base text-ink-2 stage:text-[40px]">{{ familyLede(product) }}</p>
    </template>
    <p v-if="!family" class="m-0 text-ink-2">That belt family does not exist.</p>
    <p v-else-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading || !product" class="m-0 text-muted">Loading the belt family…</p>
    <div v-else class="flex flex-col gap-10 stage:min-h-0 stage:gap-16">
      <section aria-label="Key figures" class="grid grid-cols-1 gap-4 sm:grid-cols-3 stage:gap-14">
        <div class="flex flex-col gap-2 rounded-3xl bg-mist px-6 py-5 stage:gap-5 stage:rounded-[64px] stage:px-16 stage:py-12">
          <span class="font-display text-5xl leading-none font-extrabold stage:text-[140px]">{{ product.sellers.length }}</span>
          <span class="text-base font-semibold stage:text-[44px]">competitors sell it</span>
          <span class="text-sm text-muted stage:text-[36px]">Not counting Tribelt or its group companies.</span>
        </div>
        <div class="flex flex-col gap-2 rounded-3xl bg-orange-pale px-6 py-5 stage:gap-5 stage:rounded-[64px] stage:px-16 stage:py-12">
          <span class="font-display text-5xl leading-none font-extrabold text-orange-ink stage:text-[140px]">{{ product.termsUsed }} of {{ product.terms.length }}</span>
          <span class="text-base font-semibold stage:text-[44px]">of its search terms are used on tribelt.nl</span>
          <span class="text-sm text-muted stage:text-[36px]">{{ termLanguages(product) }}</span>
        </div>
        <div class="flex flex-col gap-2 rounded-3xl bg-mist px-6 py-5 stage:gap-5 stage:rounded-[64px] stage:px-16 stage:py-12">
          <span class="font-display text-5xl leading-none font-extrabold stage:text-[140px]" :class="share === undefined ? 'text-muted' : ''">{{ shareText(share) }}</span>
          <span class="text-base font-semibold stage:text-[44px]">
            {{ share === undefined ? 'No Prompt run has asked about it yet' : `of AI answers about ${familyLabel(product.family)} belts name Tribelt` }}
          </span>
          <RouterLink to="/ai-search/answers" class="text-sm font-bold stage:text-[36px]">Read those answers</RouterLink>
        </div>
      </section>

      <div class="grid grid-cols-1 gap-8 lg:grid-cols-[minmax(0,1.45fr)_minmax(0,1fr)] stage:min-h-0 stage:grid-cols-[minmax(0,1.45fr)_minmax(0,1fr)] stage:gap-16">
        <section aria-labelledby="terms-title" class="flex min-w-0 flex-col gap-4 stage:min-h-0 stage:gap-8">
          <h2 id="terms-title" class="m-0 font-display text-2xl font-extrabold stage:text-[64px]">How buyers search for it</h2>
          <LanguageFilter :choices="languages" :selected="language" @select="(l) => (language = l)" />
          <div class="relative overflow-auto stage:min-h-0 stage:flex-1">
            <table class="w-full border-collapse text-sm stage:text-[36px]">
              <caption class="sr-only">Search terms for {{ familyLabel(product.family) }}, the most used first</caption>
              <thead>
                <tr class="border-b-2 border-black">
                  <th scope="col" class="py-2 text-left font-medium text-muted">Term</th>
                  <th scope="col" class="py-2 pl-3 text-left font-medium text-muted">Language</th>
                  <th scope="col" class="py-2 pl-3 text-left font-medium text-muted">Kind</th>
                  <th scope="col" class="py-2 pl-3 text-right font-medium text-muted">Competitors</th>
                  <th scope="col" class="py-2 pl-3 text-right font-medium text-muted">On tribelt.nl</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="t in terms" :key="`${t.term}-${t.language}`" class="border-b border-grey">
                  <th scope="row" class="py-2.5 text-left font-bold stage:py-4">{{ t.term }}</th>
                  <td class="py-2.5 pl-3 stage:py-4">{{ languageLabel(t.language) }}</td>
                  <td class="py-2.5 pl-3 text-ink-2 stage:py-4">{{ tierLabel(t.tier) }}</td>
                  <td class="py-2.5 pl-3 text-right font-bold stage:py-4">{{ t.usedBy }}</td>
                  <td class="py-2.5 pl-3 text-right font-bold stage:py-4" :class="t.usedByTribelt ? '' : 'text-orange-ink'">{{ t.usedByTribelt ? 'Yes' : 'No' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <p class="m-0 text-sm text-muted stage:text-[34px]">
            {{ terms.length }} {{ terms.length === 1 ? 'term' : 'terms' }}, of which tribelt.nl uses {{ usedHere }}.
          </p>
        </section>

        <div class="flex min-w-0 flex-col gap-6 stage:gap-12">
          <section aria-labelledby="used-title" class="flex flex-col gap-3 rounded-3xl border-2 border-mist p-6 stage:gap-8 stage:rounded-[64px] stage:border-[6px] stage:p-14">
            <h2 id="used-title" class="m-0 font-display text-xl font-extrabold stage:text-[56px]">The terms tribelt.nl uses</h2>
            <DonutChart
              :segments="[
                { label: `${String(product.termsUsed)} used on tribelt.nl`, value: product.termsUsed, tone: 'orange' },
                { label: `${String(termsOnlyOthers(product))} used only by competitors`, value: termsOnlyOthers(product), tone: 'blue-grey' },
              ]"
              :centre="`${String(product.termsUsed)} of ${String(product.terms.length)}`"
            />
          </section>

          <section aria-labelledby="sellers-title" class="flex flex-col gap-3 stage:gap-8">
            <h2 id="sellers-title" class="m-0 font-display text-2xl font-extrabold stage:text-[64px]">Who sells it, and what they call it</h2>
            <ul aria-label="Sellers" class="m-0 flex list-none flex-col border-t-2 border-black p-0 stage:border-t-[6px]">
              <li v-for="s in sellers" :key="s.competitorId" class="flex flex-col gap-0.5 border-b border-grey px-2 py-2.5 stage:gap-2 stage:px-6 stage:py-6">
                <RouterLink :to="companyPath(s.competitorId)" class="text-base font-bold text-black stage:text-[44px]">{{ s.name }}</RouterLink>
                <span class="text-sm text-ink-2 stage:text-[36px]">Calls it {{ s.names.join(', ') }}</span>
              </li>
            </ul>
            <p v-if="product.sellers.length === 0" class="m-0 text-muted">No competitor sells it.</p>
            <button
              v-if="product.sellers.length > SELLERS_SHOWN"
              type="button"
              :aria-expanded="allSellers"
              class="min-h-11 cursor-pointer self-start rounded-full border-2 border-black bg-white px-5 text-sm font-bold text-black hover:bg-mist stage:min-h-[120px] stage:border-[6px] stage:px-12 stage:text-[40px]"
              @click="allSellers = !allSellers"
            >
              {{ allSellers ? `Show the first ${String(SELLERS_SHOWN)}` : `Show all ${String(product.sellers.length)}` }}
            </button>
          </section>
        </div>
      </div>
    </div>
  </SectionView>
</template>
