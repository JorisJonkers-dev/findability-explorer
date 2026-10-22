<script setup lang="ts">
import { computed } from 'vue'
import type { KnownTerm } from '../../../infrastructure/api/types.gen'
import { tierFill } from '../../../shared/tiers'
import { spectrum } from '../domain/results'

const props = defineProps<{ knownTerms: KnownTerm[] }>()
const view = computed(() => spectrum(props.knownTerms))
const filled = computed(() => view.value.bands.filter((b) => b.count > 0))
</script>

<template>
  <div class="flex max-w-[980px] flex-col gap-3">
    <p v-if="view.total === 0" class="m-0 text-[15px] text-ink-2">The site uses none of the terms in the terminology map.</p>
    <template v-else>
      <p class="m-0 text-[15px] text-ink-2">
        The site uses {{ view.total }} {{ view.total === 1 ? 'term' : 'terms' }} from the terminology map. Each colour is a
        kind of search:
      </p>
      <div role="img" :aria-label="view.description" class="flex h-3.5 gap-0.5 overflow-hidden rounded-[7px]">
        <div
          v-for="band in filled"
          :key="band.tier"
          :class="tierFill(band.tier)"
          :style="{ flexGrow: band.count, flexShrink: 1, flexBasis: 0 }"
        ></div>
      </div>
      <ul class="m-0 flex list-none flex-wrap gap-x-[22px] gap-y-2 p-0 text-sm text-ink-2">
        <li v-for="band in view.bands" :key="band.tier" class="inline-flex items-center gap-2">
          <span aria-hidden="true" class="size-2.5 flex-none rounded-full" :class="tierFill(band.tier)"></span>
          {{ band.label }} <span class="text-muted">{{ band.count }}</span>
        </li>
      </ul>
    </template>
  </div>
</template>
