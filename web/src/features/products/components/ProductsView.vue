<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { percent } from '../../../shared/percent'
import { familyLabel } from '../../../shared/vocabulary'
import SectionView from '../../shell/components/SectionView.vue'
import { familySlug, PRODUCT_TABS } from '../../shell/domain/sections'
import { useProducts } from '../composables/useProducts'
import { aiShare, shareText, standoutText, topTerm } from '../domain/products'

const { products, standouts, isLoading, error } = useProducts()
const mostSellers = computed(() => Math.max(1, ...products.value.map((p) => p.sellers.length)))
</script>

<template>
  <SectionView section="products" title="Tribelt's belt families against the field." :tabs="PRODUCT_TABS">
    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the belt families…</p>
    <div v-else class="flex flex-col gap-10 stage:grid stage:min-h-0 stage:grid-rows-[minmax(0,1fr)_auto] stage:gap-14">
      <nav aria-label="Belt families" class="grid grid-cols-1 gap-4 sm:grid-cols-2 stage:min-h-0 stage:grid-cols-5 stage:gap-12">
        <RouterLink
          v-for="p in products"
          :key="p.family"
          :to="`/products/${familySlug(p.family)}`"
          class="flex flex-col gap-4 rounded-2xl bg-mist p-5 text-black no-underline hover:bg-grey hover:text-black stage:min-h-0 stage:gap-8 stage:rounded-[28px] stage:p-12"
        >
          <span class="flex flex-col gap-1">
            <span class="font-display text-2xl font-semibold stage:text-[56px]">{{ familyLabel(p.family) }}</span>
            <span class="text-sm text-ink-2 stage:text-[24px]">{{ p.tribeltNames.join(', ') || 'Tribelt names none' }}</span>
          </span>
          <span class="flex flex-col">
            <span class="font-display text-4xl font-semibold stage:text-[120px]" :class="(aiShare(p) ?? 0) >= 0.25 ? 'text-orange-ink' : ''">{{ shareText(aiShare(p)) }}</span>
            <span class="text-sm text-ink-2 stage:text-[26px]">of AI answers about it name Tribelt</span>
          </span>
          <span class="flex flex-col gap-1.5">
            <span class="flex justify-between gap-2 text-sm stage:text-[26px]"><span>Competitors selling it</span><strong>{{ p.sellers.length }}</strong></span>
            <span aria-hidden="true" class="block h-2 bg-white stage:h-3"><span class="block h-full bg-blue-grey" :style="{ width: percent(p.sellers.length / mostSellers) }"></span></span>
          </span>
          <span class="flex flex-col gap-1.5">
            <span class="flex justify-between gap-2 text-sm stage:text-[26px]"><span>Search terms Tribelt uses</span><strong>{{ p.termsUsed }} of {{ p.terms.length }}</strong></span>
            <span aria-hidden="true" class="block h-2 bg-white stage:h-3"><span class="block h-full bg-orange" :style="{ width: percent(p.terms.length === 0 ? 0 : p.termsUsed / p.terms.length) }"></span></span>
          </span>
          <span v-if="topTerm(p)" class="flex flex-col">
            <span class="text-xs text-muted stage:text-[22px]">Most used name</span>
            <span class="font-semibold stage:text-[30px]">{{ topTerm(p) }}</span>
          </span>
          <span class="mt-auto font-medium text-orange-ink stage:text-[26px]">Open {{ familyLabel(p.family) }}</span>
        </RouterLink>
      </nav>
      <section v-if="standouts.length > 0" aria-label="What stands out" class="flex flex-col gap-3 stage:grid stage:grid-cols-3 stage:gap-12">
        <p v-for="s in standouts" :key="s.finding" class="m-0 stage:text-[32px]">
          <strong>{{ standoutText(s, products).lead }}</strong> {{ standoutText(s, products).rest }}
        </p>
      </section>
    </div>
  </SectionView>
</template>
