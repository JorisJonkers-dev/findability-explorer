<script setup lang="ts">
import { computed, ref } from 'vue'
import { useTermMap } from '../../terminology/composables/useTermSearch'
import { useCompetitors } from '../composables/useCompetitors'
import { creditedCompanies, filterRows, matrixRows, rowSummary } from '../domain/matrix'
import CompanyMatrix from './CompanyMatrix.vue'

const { competitors, isLoading, error } = useCompetitors()
const map = useTermMap()
const text = ref('')

const rows = computed(() => matrixRows(competitors.value, creditedCompanies(map.terms.value)))
const rivals = computed(() => rows.value.filter((r) => !r.groupCompany))
const group = computed(() => rows.value.filter((r) => r.groupCompany))
const shownRivals = computed(() => filterRows(rivals.value, text.value))
const shownGroup = computed(() => filterRows(group.value, text.value))
// The group's own makers, apart from Tribelt itself: "De Spiraal and Lesjöfors".
const sisters = computed(() => {
  const names = group.value.map((r) => r.name).filter((n) => !n.startsWith('Tribelt'))
  return names.length > 1 ? `${names.slice(0, -1).join(', ')} and ${String(names.at(-1))}` : names.join('')
})
</script>

<template>
  <div class="flex flex-col gap-7">
    <header class="flex max-w-[980px] flex-col gap-3">
      <h1 class="m-0 font-display text-[40px] leading-none font-semibold md:text-[52px]">Competitors</h1>
      <p class="m-0 max-w-[64ch] text-lg text-ink-2">
        {{ rivals.length > 0 ? rivals.length : 'The' }} makers of metal belts that sell in Europe, and their own name for
        each of Tribelt's belt families. An empty cell means the maker does not offer that kind of belt.
      </p>
    </header>

    <div class="flex flex-wrap items-end gap-3">
      <label class="flex min-w-0 flex-[0_1_360px] flex-col gap-2 text-sm font-medium text-ink-2">
        Find a company
        <input
          v-model="text"
          type="search"
          spellcheck="false"
          placeholder="Name or country"
          class="h-12 w-full rounded-lg border border-field bg-surface px-4 text-base font-medium text-ink"
        />
      </label>
      <p v-if="text.trim() !== ''" class="m-0 mb-3 text-[15px] text-muted" aria-live="polite">
        {{ rowSummary(shownRivals.length, rivals.length, 'competitors') }}
      </p>
    </div>

    <p v-if="error" role="alert" class="m-0 rounded-xl border border-line bg-surface p-6 text-ink-2">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the competitors…</p>
    <template v-else>
      <p v-if="shownRivals.length === 0" class="m-0 rounded-xl border border-line p-6 text-ink-2">
        No competitor matches “{{ text.trim() }}”.
      </p>
      <CompanyMatrix
        v-else
        :rows="shownRivals"
        label="Competitors"
        caption="Each competitor's name for each Tribelt belt family"
      />
      <p v-if="sisters !== ''" class="m-0 text-[15px] text-muted">
        {{ sisters }} belong to Tribelt's own group and are listed apart, below the competitors.
      </p>

      <section v-if="shownGroup.length > 0" aria-labelledby="group-title" class="mt-6 flex flex-col gap-4">
        <h2 id="group-title" class="m-0 font-display text-[28px] font-semibold">Tribelt's own group</h2>
        <CompanyMatrix
          :rows="shownGroup"
          label="Tribelt's own group"
          caption="Each group company's name for each Tribelt belt family"
        />
      </section>
    </template>
  </div>
</template>
