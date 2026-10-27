<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { percent } from '../../../shared/percent'
import { tierFill } from '../../../shared/tiers'
import { familyLabel, languageLabel, tierLabel } from '../../../shared/vocabulary'
import LanguageFilter from '../../../shared/ui/LanguageFilter.vue'
import { languageChoices, ofLanguage } from '../../terminology/domain/languages'
import { scopeName } from '../../prompt-test/domain/results'
import SectionView from '../../shell/components/SectionView.vue'
import { companyPath, familyOfSlug, PRODUCT_TABS } from '../../shell/domain/sections'
import { useProducts } from '../composables/useProducts'
import { aiShare, productHeadline, shareText } from '../domain/products'

const props = defineProps<{ slug: string }>()
const family = computed(() => familyOfSlug(props.slug))
const { products, isLoading, error } = useProducts()
const product = computed(() => products.value.find((p) => p.family === family.value))
const title = computed(() => {
  if (!family.value) return 'No such belt family.'
  return product.value ? productHeadline(product.value) : familyLabel(family.value)
})
const TIERS = ['product', 'spec', 'application', 'problem', 'broad'] as const
const language = ref<string | null>(null)
const languages = computed(() => languageChoices(product.value?.terms.map((t) => t.language) ?? []))
const terms = computed(() => ofLanguage(product.value?.terms ?? [], language.value))
</script>

<template>
  <SectionView section="products" :title="title" :tabs="PRODUCT_TABS">
    <p v-if="!family" class="m-0 text-ink-2">Pick a belt family above.</p>
    <p v-else-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading || !product" class="m-0 text-muted">Loading the belt family…</p>
    <div v-else class="flex flex-col gap-12 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1.2fr)_minmax(0,1.2fr)_minmax(0,0.9fr)] stage:gap-24">
      <section aria-labelledby="terms-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-6">
        <div class="flex flex-wrap items-baseline justify-between gap-3">
          <h2 id="terms-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">How buyers search for it</h2>
          <span class="stage:text-[28px]">tribelt.nl uses <strong class="text-orange-ink">{{ product.termsUsed }} of {{ product.terms.length }}</strong></span>
        </div>
        <LanguageFilter :choices="languages" :selected="language" @select="(l) => (language = l)" />
        <div class="relative overflow-auto stage:min-h-0 stage:flex-1">
          <table class="w-full border-collapse text-sm stage:text-[26px]">
            <caption class="sr-only">Search terms for {{ familyLabel(product.family) }}, the most used first</caption>
            <thead>
              <tr class="border-b-2 border-black">
                <th scope="col" class="py-2 text-left font-medium text-muted">Term</th>
                <th scope="col" class="py-2 pl-3 text-left font-medium text-muted">Language</th>
                <th scope="col" class="py-2 pl-3 text-right font-medium text-muted">Used by</th>
                <th scope="col" class="py-2 pl-3 text-right font-medium text-muted">tribelt.nl</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="t in terms" :key="`${t.term}-${t.language}`" class="border-b border-grey">
                <th scope="row" class="py-2 text-left font-medium stage:py-3">
                  <span class="inline-flex items-center gap-2"><span aria-hidden="true" class="size-3 flex-none rounded-full stage:size-4" :class="tierFill(t.tier)"></span>{{ t.term }}</span>
                  <span class="sr-only">, {{ tierLabel(t.tier) }}</span>
                </th>
                <td class="py-2 pl-3 stage:py-3">{{ languageLabel(t.language) }}</td>
                <td class="py-2 pl-3 text-right stage:py-3">{{ t.usedBy }}</td>
                <td class="py-2 pl-3 text-right font-semibold stage:py-3" :class="t.usedByTribelt ? 'text-orange-ink' : 'text-muted'">{{ t.usedByTribelt ? '✓ Yes' : '✕ No' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <ul aria-label="Kinds of term" class="m-0 flex list-none flex-wrap gap-x-5 gap-y-1 p-0 text-sm text-ink-2 stage:text-[22px]">
          <li v-for="tier in TIERS" :key="tier" class="inline-flex items-center gap-2"><span aria-hidden="true" class="size-3 rounded-full stage:size-4" :class="tierFill(tier)"></span>{{ tierLabel(tier) }}</li>
        </ul>
      </section>

      <section aria-labelledby="sellers-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-6">
        <h2 id="sellers-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Who sells it and what they call it</h2>
        <ul aria-label="Sellers" class="m-0 flex list-none flex-col p-0 stage:min-h-0 stage:overflow-y-auto">
          <li class="flex flex-col gap-1 border-b-2 border-orange bg-orange-pale p-3 stage:p-5">
            <span class="font-bold text-orange-ink stage:text-[28px]">Tribelt</span>
            <span class="text-sm stage:text-[24px]">{{ product.tribeltNames.join(', ') || 'Tribelt names none for it yet.' }}</span>
          </li>
          <li v-for="s in product.sellers" :key="s.competitorId" class="flex items-start gap-3 border-b border-grey p-3 stage:gap-5 stage:p-5">
            <span class="flex min-w-0 flex-1 flex-col gap-1">
              <RouterLink :to="companyPath(s.competitorId)" class="font-semibold text-black stage:text-[28px]">{{ s.name }}</RouterLink>
              <span class="text-sm text-ink-2 stage:text-[24px]">{{ s.names.join(', ') }}</span>
            </span>
            <a v-if="s.websiteUrl" :href="s.websiteUrl" target="_blank" rel="noopener" :aria-label="`Open the website of ${s.name} in a new tab`" class="text-ink-2">
              <svg aria-hidden="true" width="20" height="20" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M5 2H2v8h8V7M7 2h3v3M10 2 5.5 6.5" /></svg>
            </a>
          </li>
        </ul>
        <p v-if="product.sellers.length === 0" class="m-0 text-muted">No competitor sells it.</p>
      </section>

      <aside aria-labelledby="answers-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-8">
        <h2 id="answers-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">In AI answers</h2>
        <p v-if="product.answers === 0" class="m-0 text-ink-2">No Prompt run has asked about it yet.</p>
        <template v-else>
          <div class="flex flex-wrap items-baseline gap-x-4">
            <span class="font-display text-5xl font-semibold text-orange-ink stage:text-[120px]">{{ shareText(aiShare(product)) }}</span>
            <span class="text-ink-2 stage:text-[28px]">of the {{ product.answers }} answers about it name Tribelt</span>
          </div>
          <div class="flex flex-col gap-2">
            <div v-for="s in product.byScope" :key="s.scope" class="grid grid-cols-[minmax(0,9rem)_minmax(0,1fr)_4rem] items-center gap-3 stage:grid-cols-[300px_minmax(0,1fr)_120px]">
              <span class="text-sm stage:text-[26px]">{{ scopeName(s.scope) }}</span>
              <span aria-hidden="true" class="h-3 bg-mist stage:h-5"><span class="block h-full bg-orange" :style="{ width: percent(s.answers === 0 ? 0 : s.named / s.answers) }"></span></span>
              <span class="text-right text-sm font-semibold stage:text-[26px]">{{ s.named }} of {{ s.answers }}</span>
            </div>
          </div>
          <div v-if="product.mostNamed.length > 0" class="flex flex-col gap-2">
            <span class="text-sm text-muted stage:text-[24px]">Named most often</span>
            <ol aria-label="Named most often" class="m-0 flex list-none flex-wrap gap-2 p-0">
              <li
                v-for="m in product.mostNamed"
                :key="m.competitorId"
                class="rounded-full border-2 px-3 py-1 text-sm stage:px-5 stage:text-[24px]"
                :class="m.role === 'tribelt' ? 'border-orange font-bold text-orange-ink' : 'border-grey'"
              >
                {{ m.name }} {{ Math.round(m.share * 100) }}%
              </li>
            </ol>
          </div>
          <RouterLink to="/ai-search/prompt-test" class="self-start font-medium underline underline-offset-4 stage:text-[26px]">All the answers in the Prompt test</RouterLink>
        </template>
      </aside>
    </div>
  </SectionView>
</template>
