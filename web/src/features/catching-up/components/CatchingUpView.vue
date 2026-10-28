<script setup lang="ts">
import { computed } from 'vue'
import { percent } from '../../../shared/percent'
import { RouterLink } from 'vue-router'
import SectionView from '../../shell/components/SectionView.vue'
import { statusLine } from '../../plan/domain/routine'
import { companyPath } from '../../shell/domain/sections'
import { useStandings } from '../composables/useStandings'
import { barsOf, headline, measureLabel, noteOf, termsSourceLine } from '../domain/standings'

const { measures, trailing, open, termsFrom, readinessChecks, isLoading, error } = useStandings()
const title = computed(() => (isLoading.value || error.value ? 'Where Tribelt trails the leaders.' : headline(measures.value, trailing.value)))

const FILL = { tribelt: 'bg-orange', typical: 'bg-blue-grey', leader: 'bg-black' } as const
const LEGEND = [
  { label: 'Tribelt', fill: 'bg-orange' },
  { label: 'Typical competitor', fill: 'bg-blue-grey' },
  { label: 'The leader', fill: 'bg-black' },
] as const
</script>

<template>
  <SectionView section="catching-up" :title="title">
    <template #aside>
      <ul aria-label="Legend" class="m-0 flex list-none flex-wrap gap-x-6 gap-y-2 p-0 text-sm text-ink-2 stage:gap-9 stage:text-[28px]">
        <li v-for="l in LEGEND" :key="l.label" class="inline-flex items-center gap-2 stage:gap-3">
          <span aria-hidden="true" class="size-4 stage:size-7" :class="l.fill"></span>{{ l.label }}
        </li>
      </ul>
    </template>

    <div class="flex flex-col gap-12 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)] stage:gap-32">
      <section aria-labelledby="field-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-7">
        <h2 id="field-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Tribelt against the field</h2>
        <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
        <p v-else-if="isLoading" class="m-0 text-muted">Loading the comparison…</p>
        <figure
          v-for="m in measures"
          v-else
          :key="m.measure"
          class="m-0 flex flex-col gap-2 border-t-2 border-grey pt-4 stage:gap-2 stage:pt-5"
        >
          <figcaption class="flex flex-wrap items-baseline justify-between gap-x-4 font-semibold stage:text-[34px]">
            {{ measureLabel(m.measure, readinessChecks) }}
            <span class="text-sm font-normal stage:text-[26px]" :class="m.leads ? 'font-semibold text-orange-ink' : 'text-muted'">{{ noteOf(m) }}</span>
          </figcaption>
          <div
            v-for="b in barsOf(m)"
            :key="b.tone"
            class="grid grid-cols-[minmax(0,9rem)_minmax(0,1fr)_4.5rem] items-center gap-3 stage:min-h-11 stage:grid-cols-[380px_minmax(0,1fr)_180px] stage:gap-6"
          >
            <span class="truncate text-sm stage:text-[26px]" :class="b.tone === 'tribelt' ? 'font-bold text-orange-ink' : ''">{{ b.who }}</span>
            <span aria-hidden="true" class="relative h-4 stage:h-7">
              <span
                class="absolute inset-y-0 left-0 rounded-r-md"
                :class="[FILL[b.tone], b.length > 0 ? 'min-w-1.5' : '']"
                :style="{ width: percent(b.length) }"
              ></span>
            </span>
            <span class="text-right text-sm font-semibold stage:text-[28px]" :class="b.tone === 'tribelt' ? 'text-orange-ink' : ''">{{ b.value }}</span>
          </div>
          <p v-if="m.measure === 'termsUsed' && termsSourceLine(termsFrom)" class="m-0 text-sm text-muted stage:text-[24px]">{{ termsSourceLine(termsFrom) }}</p>
        </figure>
      </section>

      <section aria-labelledby="todo-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-7">
        <h2 id="todo-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">What to do, in order</h2>
        <p v-if="!isLoading && !error && open.length === 0" class="m-0 text-ink-2">All planned improvements are done.</p>
        <ol v-else class="m-0 flex list-none flex-col p-0">
          <li
            v-for="(it, i) in open"
            :key="it.id"
            class="grid grid-cols-[2.5rem_minmax(0,1fr)_auto] gap-3 border-t-2 border-grey py-4 stage:grid-cols-[96px_minmax(0,1fr)_240px] stage:gap-6 stage:py-7"
          >
            <span class="font-display text-3xl leading-none font-semibold text-orange-ink stage:text-[72px]">{{ i + 1 }}</span>
            <span class="flex flex-col gap-1">
              <strong class="font-semibold stage:text-[36px]">{{ it.title }}</strong>
              <span class="text-sm text-ink-2 stage:text-[28px]">{{ it.moves }}</span>
              <span class="text-sm font-medium stage:text-[28px]">{{ statusLine(it, readinessChecks, termsFrom) }}</span>
            </span>
            <RouterLink v-if="it.leader" :to="companyPath(it.leader.competitorId)" class="flex flex-col text-right no-underline">
              <span class="text-xs text-muted stage:text-[22px]">Look at</span>
              <span class="font-semibold stage:text-[28px]">{{ it.leader.name }}</span>
            </RouterLink>
          </li>
        </ol>
        <RouterLink to="/plan" class="self-start font-semibold stage:text-[28px]">The whole plan</RouterLink>
      </section>
    </div>
  </SectionView>
</template>
