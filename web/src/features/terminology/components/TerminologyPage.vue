<script setup lang="ts">
import { computed } from 'vue'
import { tierBorder, tierFill } from '../../../shared/tiers'
import { familyLabel, languageLabel, tierLabel } from '../../../shared/vocabulary'
import { useTermFilters } from '../composables/useTermFilters'
import { useTermMap, useTermSearch } from '../composables/useTermSearch'
import {
  anyLanguageCount,
  companyName,
  familyOptions,
  hasFilters,
  languageOptions,
  resultSummary,
  splitUsedBy,
  tierOptions,
} from '../domain/filters'
import HighlightedText from './HighlightedText.vue'

const { filters, draft, update, toggle, clear } = useTermFilters()
const { terms, total, facets, isLoading, error } = useTermSearch(filters)
const map = useTermMap()

const tiers = computed(() => tierOptions(facets.value))
const languages = computed(() => languageOptions(facets.value, filters.value.language))
const families = computed(() => familyOptions(facets.value))
const summary = computed(() => resultSummary(total.value, map.total.value))
const caption = computed(() => {
  const q = filters.value.q.trim()
  return q === '' ? 'Terms in the terminology map' : `Terms matching ${q}`
})

function pickFamily(event: Event): void {
  update({ family: (event.target as HTMLSelectElement).value })
}

/** A company's button: picking it again shows every company. */
function isCurrentUser(usedBy: string): boolean {
  return filters.value.user !== '' && companyName(usedBy) === filters.value.user
}
</script>

<template>
  <div class="flex flex-col gap-7">
    <header class="flex max-w-[980px] flex-col gap-3">
      <h1 class="m-0 font-display text-[40px] leading-none font-semibold md:text-[52px]">Terminology</h1>
      <p class="m-0 max-w-[64ch] text-lg text-ink-2">
        Every name for a belt, material or specification that a competitor puts on its website. Search ignores accents
        and case, so förderband finds Förderband.
      </p>
    </header>

    <div class="flex flex-wrap items-end gap-3">
      <label class="flex min-w-0 flex-[1_1_360px] flex-col gap-2 text-sm font-medium text-ink-2">
        Search terms and companies
        <input
          v-model="draft"
          type="search"
          spellcheck="false"
          :maxlength="100"
          class="h-[52px] w-full rounded-lg border border-field bg-surface px-4 text-[17px] font-medium text-ink"
        />
      </label>
      <label class="flex flex-[0_1_220px] flex-col gap-2 text-sm font-medium text-ink-2">
        Belt family
        <select
          :value="filters.family ?? ''"
          class="h-[52px] w-full rounded-lg border border-field bg-surface px-3 text-base font-medium text-ink"
          @change="pickFamily"
        >
          <option value="">All families</option>
          <option v-for="family in families" :key="family.value" :value="family.value">
            {{ family.label }} ({{ family.count }})
          </option>
        </select>
      </label>
      <p class="m-0 mb-3.5 text-[15px] text-muted" aria-live="polite">{{ isLoading ? 'Searching…' : summary }}</p>
    </div>

    <div class="flex flex-col gap-3.5">
      <div role="group" aria-label="Kind of search" class="flex flex-wrap gap-2">
        <button
          v-for="tier in tiers"
          :key="tier.value"
          type="button"
          :aria-pressed="filters.tier === tier.value"
          class="inline-flex h-10 items-center gap-2 rounded-[20px] border px-3.5 text-[15px] font-medium"
          :class="filters.tier === tier.value ? ['bg-pressed text-ink', tierBorder(tier.value)] : 'border-field text-ink-2'"
          @click="toggle('tier', tier.value)"
        >
          <span aria-hidden="true" class="size-2.5 flex-none rounded-full" :class="tierFill(tier.value)"></span>
          {{ tier.label }} <span class="text-muted">{{ tier.count }}</span>
        </button>
      </div>
      <div role="group" aria-label="Language" class="flex flex-wrap gap-2">
        <button
          type="button"
          :aria-pressed="filters.language === null"
          class="h-9 rounded-[18px] border px-3 text-sm font-medium"
          :class="filters.language === null ? 'border-straw bg-pressed text-ink' : 'border-field text-ink-2'"
          @click="update({ language: '' })"
        >
          All languages <span class="text-muted">{{ anyLanguageCount(facets) }}</span>
        </button>
        <button
          v-for="language in languages"
          :key="language.value"
          type="button"
          :aria-pressed="filters.language === language.value"
          class="h-9 rounded-[18px] border px-3 text-sm font-medium"
          :class="filters.language === language.value ? 'border-straw bg-pressed text-ink' : 'border-field text-ink-2'"
          @click="toggle('language', language.value)"
        >
          {{ language.label }} <span class="text-muted">{{ language.count }}</span>
        </button>
      </div>
    </div>

    <div v-if="hasFilters(filters)" class="flex flex-wrap items-center gap-x-4 gap-y-2 text-[15px]">
      <p v-if="filters.user !== ''" class="m-0 text-ink-2">
        Terms used by <strong class="font-semibold text-ink">{{ filters.user }}</strong>.
        <button type="button" class="ml-1 text-straw underline hover:text-straw-hover" @click="update({ user: '' })">
          Show every company
        </button>
      </p>
      <button type="button" class="text-straw underline hover:text-straw-hover" @click="clear">Clear filters</button>
    </div>

    <p v-if="error" role="alert" class="m-0 rounded-xl border border-line bg-surface p-6 text-ink-2">{{ error }}</p>
    <p v-else-if="!isLoading && terms.length === 0" class="m-0 rounded-xl border border-line p-6 text-ink-2">
      No terms match. Try a shorter search or fewer filters.
    </p>
    <div
      v-else-if="terms.length > 0"
      role="region"
      aria-label="Matching terms"
      tabindex="0"
      class="relative overflow-x-auto rounded-xl border border-line"
    >
      <table class="w-full min-w-[760px] border-collapse">
        <caption class="sr-only">{{ caption }}</caption>
        <thead>
          <tr class="bg-surface text-left text-sm text-muted">
            <th scope="col" class="px-5 py-3.5 font-medium">Term</th>
            <th scope="col" class="px-5 py-3.5 font-medium">Language</th>
            <th scope="col" class="px-5 py-3.5 font-medium">Belt family</th>
            <th scope="col" class="px-5 py-3.5 font-medium">Used by</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="term in terms" :key="`${term.term}|${term.language}|${term.source}`" class="border-t border-line align-top">
            <td class="px-5 py-4">
              <span class="inline-flex items-baseline gap-2.5">
                <span aria-hidden="true" class="inline-block size-2.5 flex-none rounded-full" :class="tierFill(term.tier)"></span>
                <span class="text-[17px] font-semibold">
                  <HighlightedText :text="term.term" :query="filters.q" />
                  <span class="sr-only"> ({{ tierLabel(term.tier) }})</span>
                </span>
              </span>
            </td>
            <td class="px-5 py-4 whitespace-nowrap text-ink-2">{{ languageLabel(term.language) }}</td>
            <td class="px-5 py-4 whitespace-nowrap text-ink-2">{{ familyLabel(term.family) }}</td>
            <td class="px-5 py-4">
              <ul class="m-0 flex list-none flex-wrap gap-1.5 p-0">
                <li v-for="usedBy in term.usedBy" :key="usedBy">
                  <button
                    type="button"
                    :aria-pressed="isCurrentUser(usedBy)"
                    class="min-h-[30px] rounded-md border bg-surface px-2.5 text-left text-sm font-medium text-ink"
                    :class="isCurrentUser(usedBy) ? 'border-straw' : 'border-field'"
                    @click="toggle('user', companyName(usedBy))"
                  >
                    <HighlightedText :text="splitUsedBy(usedBy).company" :query="filters.q" />
                    <span v-if="splitUsedBy(usedBy).note !== ''" class="font-normal text-muted">
                      ({{ splitUsedBy(usedBy).note }})
                    </span>
                  </button>
                </li>
              </ul>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
