<script setup lang="ts">
import { toRef } from 'vue'
import { RouterLink } from 'vue-router'
import type { Language } from '../../../infrastructure/api/types.gen'
import { companyPath } from '../../shell/domain/sections'
import { useTermPages } from '../composables/useTermPages'
import { pageLabel, unrecordedLine } from '../domain/termPages'

/**
 * Where one search term is used, in two columns: the competitors' pages that use it, from each
 * one's newest scan, and tribelt.nl's. The term page shows it.
 */
const props = defineProps<{ term: string; language: Language }>()

const { split, isLoading, error } = useTermPages(
  toRef(() => props.term),
  toRef(() => props.language),
)
const PAGE = 'font-medium break-words'
</script>

<template>
  <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
  <p v-else-if="isLoading" class="m-0 text-muted">Loading the pages that use it…</p>
  <div v-else class="grid gap-10 min-[1100px]:grid-cols-2">
    <section aria-labelledby="term-pages-competitors" class="flex min-w-0 flex-col gap-3">
      <h2 id="term-pages-competitors" class="m-0 flex items-center gap-3 font-display text-2xl leading-tight font-semibold">
        <span aria-hidden="true" class="h-1.5 w-7 flex-none bg-orange"></span>Where competitors use it
      </h2>
      <p v-if="split.others.length === 0" class="m-0 rounded-2xl bg-mist p-5 text-ink-2">No competitor’s newest scan found this term.</p>
      <ul v-else class="m-0 flex list-none flex-col gap-4 rounded-2xl bg-mist p-5">
        <li v-for="c in split.others" :key="c.competitorId" class="flex flex-col gap-1">
          <RouterLink :to="companyPath(c.competitorId)" class="self-start font-semibold">{{ c.name }}</RouterLink>
          <p v-if="c.pages.length === 0" class="m-0 text-sm text-muted">{{ unrecordedLine }}</p>
          <ul v-else class="m-0 flex list-none flex-col gap-1 p-0 text-sm">
            <li v-for="p in c.pages" :key="p.url">
              <a :href="p.url" target="_blank" rel="noopener" :class="PAGE">{{ pageLabel(p) }}</a>
            </li>
          </ul>
        </li>
      </ul>
    </section>
    <section aria-labelledby="term-pages-tribelt" class="flex min-w-0 flex-col gap-3">
      <h2 id="term-pages-tribelt" class="m-0 flex items-center gap-3 font-display text-2xl leading-tight font-semibold">
        <span aria-hidden="true" class="h-1.5 w-7 flex-none bg-orange"></span>On tribelt.nl
      </h2>
      <p v-if="!split.tribelt" class="m-0 rounded-2xl bg-mist p-5 text-ink-2">tribelt.nl’s newest scan did not find this term.</p>
      <div v-else class="rounded-2xl bg-mist p-5">
        <p v-if="split.tribelt.pages.length === 0" class="m-0 text-sm text-muted">{{ unrecordedLine }}</p>
        <ul v-else class="m-0 flex list-none flex-col gap-1 p-0 text-sm">
          <li v-for="p in split.tribelt.pages" :key="p.url">
            <a :href="p.url" target="_blank" rel="noopener" :class="PAGE">{{ pageLabel(p) }}</a>
          </li>
        </ul>
      </div>
    </section>
  </div>
</template>
