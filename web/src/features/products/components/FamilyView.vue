<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import BarList from '../../../shared/ui/charts/BarList.vue'
import FrameSection from '../../../shared/ui/FrameSection.vue'
import LanguageFilter from '../../../shared/ui/LanguageFilter.vue'
import { TABLE } from '../../../shared/ui/table'
import { languageLabel, tierLabel } from '../../../shared/vocabulary'
import PageFrame from '../../shell/components/PageFrame.vue'
import { companyPath, familyOfSlug } from '../../shell/domain/sections'
import { languageChoices, ofLanguage } from '../../terminology/domain/languages'
import { familyName, termPath } from '../../terminology/domain/searchTerms'
import { useProducts } from '../composables/useProducts'
import { responseBars, SELLERS_SHOWN, visibleSellers } from '../domain/products'
import type { Language } from '../../../infrastructure/api/types.gen'

/** A belt family: the terms buyers use for it by language, Tribelt's own names, who sells it, and the AI responses about it. */
const props = defineProps<{ slug: string }>()
const family = computed(() => familyOfSlug(props.slug))
const { products, isLoading, error } = useProducts()
const product = computed(() => products.value.find((p) => p.family === family.value))
const title = computed(() => (family.value ? familyName(family.value) : 'Belt family'))
const phrase = computed(() => title.value.charAt(0).toLowerCase() + title.value.slice(1))
const language = ref<string | null>(null)
const languages = computed(() => languageChoices(product.value?.terms.map((t) => t.language) ?? []))
const terms = computed(() => ofLanguage(product.value?.terms ?? [], language.value))
const usedHere = computed(() => terms.value.filter((t) => t.usedByTribelt).length)
const allSellers = ref(false)
const sellers = computed(() => visibleSellers(product.value?.sellers ?? [], allSellers.value))
const bars = computed(() => (product.value ? responseBars(product.value) : []))
const pathOf = (t: { term: string; language: string }) => termPath({ term: t.term, language: t.language as Language })
</script>

<template>
  <PageFrame
    section="terms"
    :title="title"
    purpose="The terms buyers use for these belts, and the companies that sell them."
    data-line="From the search term list, each site's newest scan, and the newest Prompt run."
    :back="{ to: '/terms', label: 'Back to Search terms' }"
    :manage="{ to: '/manage/terms', label: 'Manage the search term list' }"
    help-topic="terms"
  >
    <p v-if="!family" class="m-0 text-ink-2">That belt family does not exist.</p>
    <p v-else-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-5">{{ error }}</p>
    <p v-else-if="isLoading || !product" class="m-0 text-muted">Loading the belt family…</p>
    <template v-else>
      <div class="grid grid-cols-1 items-start gap-10 min-[1100px]:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]">
        <FrameSection id="terms" title="Terms by language">
          <LanguageFilter :choices="languages" :selected="language" @select="(l) => (language = l)" />
          <div tabindex="0" role="region" aria-labelledby="terms-title" :class="TABLE.box">
            <table :class="TABLE.table">
              <caption class="sr-only">Search terms for {{ phrase }}, the most used first</caption>
              <thead>
                <tr>
                  <th scope="col" :class="TABLE.head">Term</th>
                  <th scope="col" :class="TABLE.head">Language</th>
                  <th scope="col" :class="TABLE.head">Buyer intent</th>
                  <th scope="col" :class="[TABLE.head, 'text-right']">Competitors</th>
                  <th scope="col" :class="[TABLE.head, 'text-right']">On tribelt.nl</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="t in terms" :key="`${t.term}-${t.language}`">
                  <th scope="row" :class="[TABLE.cell, 'font-normal']"><RouterLink :to="pathOf(t)">{{ t.term }}</RouterLink></th>
                  <td :class="TABLE.cell">{{ languageLabel(t.language) }}</td>
                  <td :class="TABLE.cell">{{ tierLabel(t.tier) }}</td>
                  <td :class="[TABLE.cell, 'text-right']">{{ t.usedBy }}</td>
                  <td :class="[TABLE.cell, 'text-right font-semibold', t.usedByTribelt ? '' : 'text-orange-ink']">{{ t.usedByTribelt ? 'Yes' : 'No' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <p class="m-0 text-sm text-muted">{{ terms.length }} {{ terms.length === 1 ? 'term' : 'terms' }}, of which tribelt.nl uses {{ usedHere }}.</p>
        </FrameSection>

        <div class="flex min-w-0 flex-col gap-10">
          <FrameSection id="names" title="Tribelt’s own names">
            <div class="rounded-3xl bg-mist p-6">
              <p v-if="product.tribeltNames.length === 0" class="m-0 text-ink-2">No name is recorded.</p>
              <ul v-else class="m-0 flex list-disc flex-col gap-1 pl-5">
                <li v-for="n in product.tribeltNames" :key="n">{{ n }}</li>
              </ul>
            </div>
          </FrameSection>

          <FrameSection id="sellers" title="Who sells them">
            <p v-if="product.sellers.length === 0" class="m-0 rounded-3xl bg-mist p-6 text-ink-2">No competitor on the list sells these belts.</p>
            <ul v-else aria-label="Sellers" class="m-0 flex list-none flex-col rounded-3xl bg-mist px-6 py-3">
              <li v-for="s in sellers" :key="s.competitorId" class="flex flex-col gap-0.5 border-b border-grey py-2.5 last:border-b-0">
                <RouterLink :to="companyPath(s.competitorId)" class="font-semibold">{{ s.name }}</RouterLink>
                <span class="text-sm text-ink-2">Calls it {{ s.names.join(', ') }}</span>
              </li>
            </ul>
            <button
              v-if="product.sellers.length > SELLERS_SHOWN"
              type="button"
              :aria-expanded="allSellers"
              class="min-h-11 cursor-pointer self-start rounded-full border-2 border-silver bg-white px-5 text-sm font-medium text-black hover:border-black"
              @click="allSellers = !allSellers"
            >
              {{ allSellers ? `Show the first ${String(SELLERS_SHOWN)}` : `Show all ${String(product.sellers.length)}` }}
            </button>
          </FrameSection>
        </div>
      </div>

      <FrameSection
        id="responses"
        :title="`AI responses about ${phrase}`"
        intro="A response is what an AI assistant replies. Each bar is the share of responses that name Tribelt, for a region and language."
      >
        <div class="rounded-3xl bg-mist p-6">
          <p v-if="bars.length === 0" class="m-0 text-ink-2">The newest Prompt run asked no questions about these belts.</p>
          <BarList v-else :rows="bars" :max="1" label="The share of responses that name Tribelt" />
        </div>
        <RouterLink to="/ai-responses/all" class="self-start font-medium">See these responses</RouterLink>
      </FrameSection>
    </template>
  </PageFrame>
</template>
