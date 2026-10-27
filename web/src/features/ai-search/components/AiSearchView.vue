<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { percent } from '../../../shared/percent'
import { formatValue } from '../../catching-up/domain/standings'
import SectionView from '../../shell/components/SectionView.vue'
import { AI_SEARCH_TABS } from '../../shell/domain/sections'
import { useAiSearchSummary } from '../composables/useAiSearchSummary'
import { ACCESS, CHART, chartLabel, isTribelt, keepingOutNote, linesOf, pct, scaleTop, shortDate, trendLine, tribeltNote, tribeltShares, yOf } from '../domain/summary'

const { summary, isLoading, error } = useAiSearchSummary()

const top = computed(() => (summary.value ? scaleTop(summary.value.trend, summary.value.goal) : 0.2))
const lines = computed(() => (summary.value ? linesOf(summary.value, top.value) : []))
const ticks = computed(() => [0, 1, 2, 3].map((i) => (top.value * i) / 3))
const shares = computed(() => (summary.value ? tribeltShares(summary.value) : undefined))
const bestScore = computed(() => Math.max(5, ...(summary.value?.ranking.map((r) => r.score) ?? [])))
const promptCard = computed(() => {
  const s = summary.value
  if (!s || s.questions === 0) return 'Every question and every answer, once the first Prompt run is in.'
  return `Each Prompt run asks ${String(s.providers)} assistants ${String(s.questions)} buyer questions. See every answer and who it names.`
})
</script>

<template>
  <SectionView section="ai-search" title="An AI assistant can only recommend what it has read." :tabs="AI_SEARCH_TABS">
    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading || !summary" class="m-0 text-muted">Loading the summary…</p>
    <div
      v-else
      class="flex flex-col gap-14 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1.15fr)_minmax(0,1.3fr)_minmax(0,0.9fr)] stage:gap-28"
    >
      <section aria-labelledby="share-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-7">
        <h2 id="share-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">How often the answers name Tribelt</h2>
        <div class="flex flex-wrap items-baseline gap-x-4 gap-y-1 stage:gap-6">
          <span class="font-display text-6xl leading-none font-semibold text-orange-ink stage:text-[120px]">{{ shares ? pct(shares.now) : '–' }}</span>
          <span class="text-ink-2 stage:text-[30px]">{{ trendLine(summary) }}</span>
        </div>
        <svg
          v-if="summary.trend.length > 0"
          role="img"
          :aria-label="chartLabel(summary)"
          viewBox="0 0 1300 640"
          class="block h-auto w-full font-sans"
        >
          <template v-for="(t, i) in ticks" :key="t">
            <line :x1="CHART.left" :x2="CHART.right" :y1="yOf(t, top)" :y2="yOf(t, top)" :class="i === 0 ? 'stroke-silver' : 'stroke-grey'" stroke-width="2" />
            <text :x="CHART.left - 14" :y="yOf(t, top) + 8" text-anchor="end" font-size="24" class="fill-muted">{{ pct(t) }}</text>
          </template>
          <line :x1="CHART.left" :x2="CHART.right" :y1="yOf(summary.goal, top)" :y2="yOf(summary.goal, top)" class="stroke-orange-ink" stroke-width="3" stroke-dasharray="12 10" />
          <text :x="CHART.left + 10" :y="yOf(summary.goal, top) - 12" font-size="24" font-weight="600" class="fill-orange-ink">Goal {{ pct(summary.goal) }}</text>
          <template v-for="l in lines" :key="l.name">
            <polyline
              :points="l.points"
              fill="none"
              :class="l.tribelt ? 'stroke-orange' : 'stroke-blue-grey'"
              :stroke-width="l.tribelt ? 8 : 4"
              stroke-linejoin="round"
              stroke-linecap="round"
            />
            <circle :cx="l.endX" :cy="l.endY" :r="l.tribelt ? 13 : 8" :class="l.tribelt ? 'fill-orange' : 'fill-blue-grey'" class="stroke-white" stroke-width="3" />
            <text
              :x="CHART.right + 20"
              :y="l.labelY"
              :font-size="l.tribelt ? 32 : 28"
              :font-weight="l.tribelt ? 700 : 400"
              :class="l.tribelt ? 'fill-orange-ink' : 'fill-black'"
            >
              {{ l.label }}
            </text>
          </template>
          <text :x="CHART.left" y="636" font-size="24" class="fill-muted">{{ shortDate(summary.trend[0]?.runStartedAt ?? '') }}</text>
          <text :x="CHART.right" y="636" text-anchor="end" font-size="24" class="fill-muted">{{ shortDate(summary.trend.at(-1)?.runStartedAt ?? '') }}</text>
        </svg>
        <RouterLink
          to="/ai-search/prompt-test"
          class="flex flex-col gap-2 rounded-2xl bg-mist px-6 py-5 text-black no-underline hover:bg-grey hover:text-black stage:mt-auto stage:gap-2.5 stage:rounded-[28px] stage:px-12 stage:py-10"
        >
          <span class="flex items-center justify-between text-xl font-semibold stage:text-[40px]">
            Prompt test
            <svg aria-hidden="true" viewBox="0 0 24 24" class="size-6 stage:size-10" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 5l7 7-7 7" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </span>
          <span class="text-ink-2 stage:text-[28px]">{{ promptCard }}</span>
        </RouterLink>
      </section>

      <section aria-labelledby="labs-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-7">
        <h2 id="labs-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Which AI labs may read tribelt.nl</h2>
        <div class="overflow-x-auto">
          <table class="w-full border-collapse text-sm stage:text-[28px]">
            <caption class="sr-only">Each crawler, what letting it in gets Tribelt, and whether tribelt.nl lets it in</caption>
            <thead>
              <tr class="border-b-[3px] border-black">
                <th scope="col" class="py-2 text-left font-medium text-muted stage:py-3">Crawler</th>
                <th scope="col" class="py-2 pl-4 text-left font-medium text-muted stage:py-3 stage:pl-5">What letting it in gets Tribelt</th>
                <th scope="col" class="py-2 pl-4 text-right font-medium text-muted stage:py-3 stage:pl-5">tribelt.nl</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="c in summary.crawlers" :key="c.bot" class="border-b border-grey">
                <th scope="row" class="py-2 text-left font-semibold stage:py-3.5">
                  {{ c.bot }}<span class="block text-xs font-normal text-muted stage:text-[22px]">{{ c.lab }}</span>
                </th>
                <td class="py-2 pl-4 stage:py-3.5 stage:pl-5">
                  <span class="inline-flex items-center gap-2 stage:gap-3">
                    <span aria-hidden="true" class="size-3 flex-none rounded-full stage:size-5" :class="c.feeds === 'today' ? 'bg-orange' : 'bg-blue-grey'"></span>
                    {{ c.gets }}<span class="sr-only">{{ c.feeds === 'today' ? ', shows up in answers today' : ', known to future models' }}</span>
                  </span>
                </td>
                <td class="py-2 pl-4 text-right font-semibold whitespace-nowrap stage:py-3.5 stage:pl-5" :class="ACCESS[c.access ?? 'unknown'].ok === false ? 'text-orange-ink' : ''">
                  <span aria-hidden="true" class="mr-1.5">{{ ACCESS[c.access ?? 'unknown'].mark }}</span>{{ ACCESS[c.access ?? 'unknown'].text }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <ul aria-label="Legend" class="m-0 flex list-none flex-wrap gap-x-6 gap-y-2 p-0 text-sm text-ink-2 stage:gap-9 stage:text-[24px]">
          <li class="inline-flex items-center gap-2"><span aria-hidden="true" class="size-3 rounded-full bg-orange stage:size-5"></span>Shows up in answers today</li>
          <li class="inline-flex items-center gap-2"><span aria-hidden="true" class="size-3 rounded-full bg-blue-grey stage:size-5"></span>Known to future models</li>
        </ul>
        <div class="flex flex-col gap-3 rounded-2xl bg-orange-pale px-6 py-5 stage:mt-auto stage:rounded-[28px] stage:px-12 stage:py-10">
          <span class="text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[26px]">Why let them in</span>
          <p class="m-0 stage:text-[32px]">
            Search crawlers decide what assistants can cite today. Training crawlers decide what next year's models already know.
            {{ tribeltNote(summary.crawlers) }}
          </p>
          <p class="m-0 font-semibold stage:text-[32px]">{{ keepingOutNote(summary.keepingOut) }}</p>
        </div>
      </section>

      <section aria-labelledby="ready-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-7">
        <h2 id="ready-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Ready for AI</h2>
        <span class="text-ink-2 stage:text-[28px]">Checks passed out of five</span>
        <p v-if="summary.ranking.length === 0" class="m-0 text-muted">No site has been scanned yet.</p>
        <ol v-else class="m-0 flex list-none flex-col gap-2 p-0">
          <li
            v-for="r in summary.ranking"
            :key="r.competitorId"
            class="grid grid-cols-[minmax(0,10rem)_minmax(0,1fr)_3rem] items-center gap-3 stage:min-h-14 stage:grid-cols-[300px_minmax(0,1fr)_70px] stage:gap-5"
          >
            <span class="truncate stage:text-[28px]" :class="isTribelt(r) ? 'font-bold text-orange-ink' : ''">{{ r.name }}</span>
            <span aria-hidden="true" class="relative h-4 bg-mist stage:h-6">
              <span class="absolute inset-y-0 left-0 rounded-r" :class="isTribelt(r) ? 'bg-orange' : 'bg-blue-grey'" :style="{ width: percent(r.score / bestScore) }"></span>
            </span>
            <span class="text-right font-semibold stage:text-[28px]">{{ formatValue('readinessChecks', r.score) }}</span>
          </li>
        </ol>
        <RouterLink
          to="/ai-search/readiness"
          class="flex flex-col gap-2 rounded-2xl bg-mist px-6 py-5 text-black no-underline hover:bg-grey hover:text-black stage:mt-auto stage:gap-2.5 stage:rounded-[28px] stage:px-12 stage:py-10"
        >
          <span class="flex items-center justify-between text-xl font-semibold stage:text-[40px]">
            AI readiness
            <svg aria-hidden="true" viewBox="0 0 24 24" class="size-6 stage:size-10" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 5l7 7-7 7" stroke-linecap="round" stroke-linejoin="round" />
            </svg>
          </span>
          <span class="text-ink-2 stage:text-[28px]">Crawler access, sitemaps, llms.txt and Markdown pages for all {{ summary.sites }} sites.</span>
        </RouterLink>
      </section>
    </div>
  </SectionView>
</template>
