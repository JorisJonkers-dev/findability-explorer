<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { visitorsLine } from '../../buyers/domain/channels'
import { headline } from '../../catching-up/domain/standings'
import { numberRows } from '../../numbers/domain/numbers'
import PageView from '../../shell/components/PageView.vue'
import { useHome } from '../composables/useHome'
import { sectionCards } from '../domain/cards'

const { facts, overview, isLoading, error } = useHome()

const rows = computed(() => (facts.value ? numberRows(facts.value) : []))
const cards = computed(() => (facts.value && overview.value ? sectionCards(overview.value, facts.value.plan) : []))
const title = computed(() => (facts.value ? headline(facts.value.standings.measures, facts.value.standings.trailing) : 'Overview'))
</script>

<template>
  <PageView section="overview" :title="title">
    <template #lede>
      <p class="m-0 max-w-[62ch] text-lg leading-normal text-ink-2">
        This page shows how easily buyers find Tribelt in search and in AI answers, compared with its competitors. Pick a number to see who is
        ahead and what to do about it.
      </p>
    </template>

    <p v-if="error" role="alert" class="m-0 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the numbers…</p>

    <template v-else>
      <section aria-labelledby="watch-title" class="flex flex-col gap-3.5">
        <div class="flex flex-wrap items-baseline justify-between gap-3">
          <h2 id="watch-title" class="m-0 font-display text-3xl font-semibold">Numbers to watch</h2>
          <ul aria-hidden="true" class="m-0 flex list-none flex-wrap gap-x-5 gap-y-2 p-0 text-sm text-ink-2">
            <li class="inline-flex items-center gap-2"><span class="h-2.5 w-[18px] rounded-[5px] bg-orange"></span>Tribelt now</li>
            <li class="inline-flex items-center gap-2"><span class="h-[18px] w-[3px] bg-black"></span>Target for the year</li>
            <li class="inline-flex items-center gap-2"><span class="size-2.5 rounded-full bg-blue-grey"></span>Leading competitor</li>
          </ul>
        </div>
        <ul class="m-0 list-none border-t-2 border-black p-0">
          <li v-for="r in rows" :key="r.target" class="border-b border-grey">
            <RouterLink
              :to="r.path"
              :aria-label="r.aria"
              class="grid grid-cols-[minmax(0,1fr)_auto_1.75rem] items-center gap-x-4 gap-y-3 rounded-xl px-2 py-5 text-black no-underline hover:bg-mist hover:text-black md:grid-cols-[minmax(0,1.25fr)_minmax(0,2fr)_9rem_1.75rem] md:gap-x-7"
            >
              <span class="flex min-w-0 flex-col gap-1.5">
                <span class="text-lg font-bold">{{ r.label }}</span>
                <span v-if="r.improvedBy" class="text-sm text-muted">{{ r.improvedBy }}</span>
              </span>
              <span class="order-last col-span-3 flex min-w-0 flex-col gap-2 md:order-none md:col-span-1">
                <span aria-hidden="true" class="relative block h-[22px]">
                  <span class="absolute inset-x-0 top-1.5 h-2.5 rounded-[5px] bg-grey"></span>
                  <span class="absolute top-1.5 left-0 h-2.5 rounded-[5px] bg-orange" :style="{ width: `${String(r.fill)}%` }"></span>
                  <span class="absolute top-0 h-[22px] w-[3px] bg-black" :style="{ left: `calc(${String(r.goalAt)}% - 1px)` }"></span>
                  <span
                    v-if="r.leaderAt !== undefined"
                    class="absolute top-1 size-3.5 rounded-full border-2 border-solid border-white bg-blue-grey"
                    :style="{ left: `calc(${String(r.leaderAt)}% - 7px)` }"
                  ></span>
                </span>
                <span class="text-sm text-ink-2">{{ r.progress }}</span>
              </span>
              <span class="flex flex-col items-end gap-0.5">
                <span :class="r.measured ? 'font-display text-3xl leading-none font-semibold' : 'text-lg leading-tight font-bold text-muted'">{{ r.now }}</span>
                <span class="text-sm text-muted">Target {{ r.goal }}</span>
              </span>
              <svg aria-hidden="true" viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M9 5l7 7-7 7" />
              </svg>
            </RouterLink>
          </li>
        </ul>
      </section>

      <section aria-labelledby="sections-title" class="flex flex-col gap-3.5">
        <h2 id="sections-title" class="m-0 font-display text-3xl font-semibold">The sections</h2>
        <ul class="m-0 grid list-none grid-cols-1 gap-5 p-0 sm:grid-cols-2 lg:grid-cols-4">
          <li v-for="c in cards" :key="c.id" class="flex">
            <RouterLink :to="c.path" class="flex min-h-[220px] w-full flex-col gap-3 rounded-3xl bg-mist p-6 text-black no-underline hover:bg-grey hover:text-black">
              <span class="flex gap-2.5 text-sm font-bold tracking-[0.06em]"><span class="text-orange-ink">{{ c.number }}</span><span class="text-ink-2">{{ c.label }}</span></span>
              <span class="font-display text-5xl leading-none font-semibold tracking-[-0.02em]">{{ c.big }}</span>
              <span class="leading-snug text-ink-2">{{ c.text }}</span>
              <span class="mt-auto inline-flex items-center gap-1.5 font-bold">
                {{ c.cta }}
                <svg aria-hidden="true" viewBox="0 0 24 24" class="size-4" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 5l7 7-7 7" /></svg>
              </span>
            </RouterLink>
          </li>
        </ul>
      </section>

      <RouterLink
        to="/buyers"
        class="grid grid-cols-1 items-center gap-6 rounded-3xl bg-orange-pale px-8 py-7 text-black no-underline hover:text-black hover:brightness-95 md:grid-cols-[minmax(0,1fr)_auto]"
      >
        <span class="flex flex-col gap-2">
          <span class="text-sm font-bold tracking-[0.06em] text-orange-ink">WHERE VISITORS COME FROM</span>
          <span class="max-w-[50ch] font-display text-2xl leading-tight font-semibold">{{ visitorsLine() }}</span>
          <span class="text-ink-2">See how the channels compare.</span>
        </span>
        <span class="inline-flex min-h-12 items-center rounded-full bg-black px-6 font-bold whitespace-nowrap text-white">See the visitor figures</span>
      </RouterLink>
    </template>
  </PageView>
</template>
