<script setup lang="ts">
import { computed } from 'vue'
import type { Equivalent, MatchingStatus } from '../../../infrastructure/api/types.gen'
import { equivalentRows } from '../domain/results'
import { matchingText } from '../domain/status'

const props = defineProps<{ equivalents: Equivalent[]; matching: MatchingStatus; site: string }>()
const rows = computed(() => equivalentRows(props.equivalents))
const status = computed(() => matchingText(props.matching))
</script>

<template>
  <section aria-labelledby="eq-title" class="flex flex-col gap-[18px]">
    <div class="flex flex-col gap-1.5">
      <h3 id="eq-title" class="m-0 font-display text-[28px] leading-[1.1] font-semibold">Their products, Tribelt's equivalent</h3>
      <p class="m-0 max-w-[70ch] text-[15px] text-muted">
        Matched by Claude from the pages read. Put these names on Tribelt's product pages as "comparable to", so a buyer
        who knows only the installed belt still finds Tribelt.
      </p>
    </div>
    <p v-if="status !== null" class="m-0 rounded-xl border border-line px-5 py-4 text-ink-2">{{ status }}</p>
    <p v-else-if="rows.length === 0" class="m-0 rounded-xl border border-line px-5 py-4 text-ink-2">
      Claude found no products on these pages to match with Tribelt's.
    </p>
    <div
      v-else
      role="region"
      aria-label="Their products and Tribelt's equivalent"
      tabindex="0"
      class="relative overflow-x-auto rounded-xl border border-line"
    >
      <table class="w-full min-w-[640px] border-collapse">
        <thead>
          <tr class="bg-surface text-left text-sm text-muted">
            <th scope="col" class="px-5 py-3.5 font-medium">{{ site }} calls it</th>
            <th scope="col" class="px-5 py-3.5 font-medium">Tribelt makes</th>
            <th scope="col" class="px-5 py-3.5 font-medium">Words on their page</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.theirs" class="border-t border-line align-top">
            <td class="px-5 py-4 font-semibold">{{ row.theirs }}</td>
            <td class="px-5 py-4 font-medium text-straw">{{ row.ours }}</td>
            <td class="px-5 py-4 text-ink-2">{{ row.words }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
