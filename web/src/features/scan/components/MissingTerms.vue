<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import type { KnownTerm } from '../../../infrastructure/api/types.gen'
import { tierFill } from '../../../shared/tiers'
import { scopeLabel, tierLabel } from '../../../shared/vocabulary'
import { missingTerms } from '../domain/results'

/** How many terms show before the list is opened in full. */
const FIRST = 10

const props = defineProps<{ knownTerms: KnownTerm[]; site: string }>()
const all = computed(() => missingTerms(props.knownTerms))
const expanded = ref(false)
const shown = computed(() => (expanded.value ? all.value : all.value.slice(0, FIRST)))
</script>

<template>
  <section aria-labelledby="gap-title" class="flex flex-col gap-[18px] rounded-xl border border-line bg-surface p-5 md:p-7">
    <div class="flex flex-col gap-1.5">
      <h3 id="gap-title" class="m-0 font-display text-[28px] leading-[1.1] font-semibold">Terms Tribelt is missing</h3>
      <p class="m-0 text-[15px] text-muted">
        <template v-if="all.length > 0">
          {{ site }} uses these on its pages. Tribelt's own website does not, so searches with them find {{ site }} first.
        </template>
        <template v-else>Tribelt's own website uses every term from the map that {{ site }} uses.</template>
      </p>
    </div>
    <ul v-if="all.length > 0" class="m-0 flex list-none flex-col p-0">
      <li
        v-for="term in shown"
        :key="`${term.term}|${term.language}`"
        class="grid grid-cols-[1fr_auto] gap-x-4 gap-y-1 border-t border-line py-3.5"
      >
        <span class="text-[17px] font-semibold break-words">{{ term.term }}</span>
        <span class="text-right font-semibold text-ink">{{ term.occurrences }}×</span>
        <span class="inline-flex items-center gap-2 text-sm text-muted">
          <span aria-hidden="true" class="size-2.5 flex-none rounded-full" :class="tierFill(term.tier)"></span>
          {{ tierLabel(term.tier) }}, {{ scopeLabel(term.family) }}
        </span>
        <RouterLink :to="{ path: '/terminology', query: { q: term.term } }" class="text-right text-sm">
          Who else uses it<span class="sr-only">: {{ term.term }}</span>
        </RouterLink>
      </li>
    </ul>
    <button
      v-if="all.length > FIRST"
      type="button"
      class="self-start rounded-md border border-field px-3 py-1.5 text-sm font-medium text-ink-2"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      {{ expanded ? 'Show fewer' : `Show all ${String(all.length)} terms` }}
    </button>
  </section>
</template>
