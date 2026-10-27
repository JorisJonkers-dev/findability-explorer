<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import triflex from '../../../assets/triflex.webp'
import { percent } from '../../../shared/percent'
import GoalCard from '../../../shared/ui/GoalCard.vue'
import RoundedPhoto from '../../../shared/ui/RoundedPhoto.vue'
import RecommendationList from '../../../shared/ui/RecommendationList.vue'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import SectionView from '../../shell/components/SectionView.vue'
import { AI_SEARCH_TABS } from '../../shell/domain/sections'
import PromptSummary from '../../prompt-test/components/PromptSummary.vue'
import {
  AD_SURFACES,
  ADS_CHECKED,
  type AdState,
  CRAWLERS,
  type CrawlerId,
  DEFAULT_CRAWLER,
  GOAL,
  MIRROR_FACTS,
  RECOMMENDATIONS,
  SOURCE,
  STATS,
  TICKS,
  TITLE,
} from '../domain/content'
import { crawler, dayPosition, ownerLine, rowLabel } from '../domain/timeline'

const picked = ref<CrawlerId>(DEFAULT_CRAWLER)
const selected = computed(() => crawler(picked.value))

const AD_DOT: Record<AdState, string> = { no: 'bg-orange-ink', notYet: 'bg-blue-grey', yes: 'bg-orange' }
</script>

<template>
  <SectionView section="ai-search" :title="TITLE" :source="SOURCE" :tabs="AI_SEARCH_TABS">
    <div class="flex flex-col gap-14 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)_minmax(0,0.8fr)] stage:gap-24">
      <section aria-labelledby="tl-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-9">
        <SectionHeading id="tl-title">Who read the mirror, and when</SectionHeading>
        <div class="flex flex-col gap-3 stage:gap-4">
          <dl class="m-0 grid grid-cols-2 gap-4 sm:grid-cols-4 stage:gap-8">
            <div v-for="stat in STATS" :key="stat.label" class="flex flex-col-reverse">
              <dt class="text-sm text-ink-2 stage:text-[26px]">{{ stat.label }}</dt>
              <dd class="m-0 font-display text-4xl leading-none font-semibold stage:text-[96px]" :class="{ 'text-orange-ink': stat.gap }">{{ stat.value }}</dd>
            </div>
          </dl>
          <p class="m-0 text-sm text-muted stage:text-[24px]">{{ MIRROR_FACTS }}</p>
        </div>
        <div class="flex flex-col gap-1.5">
          <div aria-hidden="true" class="grid grid-cols-[7.5rem_minmax(0,1fr)] gap-3 text-xs text-muted stage:grid-cols-[330px_minmax(0,1fr)] stage:gap-7 stage:text-[24px]">
            <span></span>
            <span class="relative h-5 stage:h-9">
              <span
                v-for="tick in TICKS"
                :key="tick.label"
                class="absolute whitespace-nowrap first:translate-x-0 last:-translate-x-full [&:not(:first-child):not(:last-child)]:-translate-x-1/2"
                :style="{ left: percent(dayPosition(tick.day)) }"
              >
                {{ tick.label }}
              </span>
            </span>
          </div>
          <ul aria-label="Crawlers" class="m-0 flex list-none flex-col p-0">
            <li v-for="row in CRAWLERS" :key="row.id" class="border-t-2 border-grey">
              <button
                type="button"
                :aria-pressed="picked === row.id"
                :aria-label="rowLabel(row)"
                class="grid min-h-14 w-full cursor-pointer grid-cols-[7.5rem_minmax(0,1fr)] items-center gap-3 border-0 py-1 text-left text-black stage:min-h-[110px] stage:grid-cols-[330px_minmax(0,1fr)] stage:gap-7 stage:py-2"
                :class="picked === row.id ? 'bg-mist' : 'bg-transparent hover:bg-mist'"
                @click="picked = row.id"
              >
                <span class="flex flex-col pl-2 stage:pl-5">
                  <span class="font-semibold stage:text-[34px]">{{ row.name }}</span>
                  <span class="text-xs text-muted stage:text-[24px]">{{ ownerLine(row) }}</span>
                </span>
                <span
                  aria-hidden="true"
                  class="relative h-8 bg-[repeating-linear-gradient(90deg,var(--color-mist)_0_2px,transparent_2px_100%)] bg-size-[4.3478%_100%] stage:h-18"
                >
                  <span
                    v-for="event in row.events"
                    :key="event.day"
                    class="absolute top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full"
                    :class="[
                      event.big ? 'size-5 bg-orange stage:size-16' : 'size-2.5 stage:size-[30px]',
                      event.big ? '' : row.verified === false ? 'border-2 border-blue-grey bg-white stage:border-[5px]' : 'bg-blue-grey',
                    ]"
                    :style="{ left: percent(dayPosition(event.day)) }"
                  ></span>
                  <span
                    v-if="row.events.length === 0"
                    class="absolute inset-0 flex items-center pl-3 text-sm font-semibold text-orange-ink stage:pl-6 stage:text-[28px]"
                  >
                    No visits in more than three weeks
                  </span>
                </span>
              </button>
            </li>
          </ul>
        </div>
        <div aria-live="polite" class="flex flex-col gap-2 rounded-2xl bg-mist p-5 stage:mt-auto stage:gap-3 stage:rounded-[28px] stage:px-12 stage:py-10">
          <span class="text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[26px]">{{ selected.name }} · {{ selected.purpose }}</span>
          <p class="m-0">{{ selected.did }}</p>
          <p class="m-0 font-semibold">{{ selected.means }}</p>
        </div>
      </section>

      <section aria-labelledby="pt-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-9">
        <SectionHeading id="pt-title">Did the assistant name Tribelt?</SectionHeading>
        <PromptSummary />
        <SectionHeading class="mt-4 stage:mt-6">Where AI ads can run today</SectionHeading>
        <table class="w-full border-collapse text-sm stage:text-[28px]">
          <caption class="sr-only">AI ad surfaces for buyers in the Netherlands and Germany</caption>
          <thead>
            <tr class="border-b-2 border-black stage:border-b-[3px]">
              <th scope="col" class="py-2 text-left font-medium text-muted stage:py-3.5">Surface</th>
              <th scope="col" class="py-2 pl-3 text-left font-medium text-muted stage:py-3.5 stage:pl-5">NL and DE</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="ad in AD_SURFACES" :key="ad.surface" class="border-b border-grey">
              <th scope="row" class="py-2.5 text-left font-medium stage:py-[18px]">{{ ad.surface }}</th>
              <td class="py-2.5 pl-3 stage:py-[18px] stage:pl-5">
                <span class="inline-flex items-center gap-2 stage:gap-3.5">
                  <span aria-hidden="true" class="size-3 flex-none rounded-full stage:size-6" :class="AD_DOT[ad.state]"></span>
                  {{ ad.text }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
        <span class="text-sm text-muted stage:text-[24px]">{{ ADS_CHECKED }}</span>
      </section>

      <aside aria-labelledby="so2-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-10">
        <RoundedPhoto :src="triflex" alt="A Tri-Flex turn-curve belt by Tribelt" class="stage:h-[560px]" />
        <SectionHeading id="so2-title">So we recommend</SectionHeading>
        <RecommendationList :items="RECOMMENDATIONS" />
        <RouterLink
          to="/ai-search/readiness"
          class="inline-flex min-h-11 items-center gap-2 self-start rounded-full border-2 border-black px-5 font-medium text-black no-underline hover:bg-black hover:text-white stage:min-h-[88px] stage:border-[3px] stage:px-10 stage:text-[30px]"
        >
          Who follows this: AI readiness of the competitors
        </RouterLink>
        <GoalCard :goal="GOAL.goal" :text="GOAL.text" :rows="GOAL.rows" class="stage:mt-auto" />
      </aside>
    </div>
  </SectionView>
</template>
