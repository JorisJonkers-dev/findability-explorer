<script setup lang="ts">
import { RouterLink } from 'vue-router'
import GoalSection from '../../goals/components/GoalSection.vue'
import PageFrame from '../../shell/components/PageFrame.vue'
import { useMe } from '../../shell/composables/useMe'
import { useHome } from '../composables/useHome'

const home = useHome()
const { isAdmin } = useMe()
</script>

<template>
  <PageFrame
    section="overview"
    title="Overview"
    purpose="How easily buyers find Tribelt in search and AI assistants, measured against five goals."
    :data-line="home.dataLine.value"
    help-topic="goals"
  >
    <p v-if="home.error.value" role="alert" class="m-0 text-lg text-orange-ink">{{ home.error.value }}</p>
    <p v-else-if="home.isLoading.value" class="m-0 text-lg text-muted">Loading the goals…</p>

    <template v-else>
      <p data-part="summary" class="m-0 rounded-3xl bg-orange-pale px-6 py-5 font-display text-xl leading-snug font-medium text-pretty sm:px-8 sm:text-2xl">
        {{ home.summary.value }}
      </p>

      <GoalSection id="goals" title="Goals" intro="Click a goal to see who is ahead and what would improve it.">
        <div class="relative overflow-x-auto rounded-3xl bg-mist p-4 sm:p-5">
          <table class="w-full min-w-[640px] border-collapse text-left">
            <thead>
              <tr class="border-b border-silver text-sm text-ink-2">
                <th scope="col" class="py-2 pr-4 font-semibold">Goal</th>
                <th scope="col" class="py-2 pr-4 text-right font-semibold">Tribelt</th>
                <th scope="col" class="py-2 pr-4 text-right font-semibold">The goal</th>
                <th scope="col" class="py-2 pr-4 font-semibold">The best competitor</th>
                <th scope="col" class="w-[28%] py-2 font-semibold">Progress</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in home.rows.value" :key="r.target" class="border-b border-grey last:border-b-0">
                <th scope="row" class="py-3 pr-4 font-normal">
                  <RouterLink :to="r.path" :aria-label="r.aria" class="font-medium">{{ r.name }}</RouterLink>
                </th>
                <td class="py-3 pr-4 text-right font-bold" :class="r.measured ? '' : 'text-muted'">{{ r.now }}</td>
                <td class="py-3 pr-4 text-right">{{ r.goal }}</td>
                <td class="py-3 pr-4">{{ r.best }}</td>
                <td class="py-3">
                  <span role="img" :aria-label="r.barLabel" class="relative block h-[22px] min-w-[140px]">
                    <span class="absolute inset-x-0 top-1.5 h-2.5 rounded-[5px] bg-grey"></span>
                    <span data-mark="now" class="absolute top-1.5 left-0 h-2.5 rounded-[5px] bg-orange" :style="{ width: `${String(r.bar.now)}%` }"></span>
                    <span
                      v-if="r.bar.best !== undefined"
                      data-mark="best"
                      class="absolute top-1 size-3.5 rounded-full border-2 border-solid border-white bg-blue-grey"
                      :style="{ left: `calc(${String(r.bar.best)}% - 7px)` }"
                    ></span>
                    <span data-mark="goal" class="absolute top-0 h-[22px] w-[3px] bg-black" :style="{ left: `calc(${String(r.bar.goal)}% - 1px)` }"></span>
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <ul data-part="legend" aria-hidden="true" class="m-0 flex list-none flex-wrap gap-x-5 gap-y-2 p-0 text-sm text-ink-2">
          <li class="inline-flex items-center gap-2"><span class="h-2.5 w-[18px] rounded-[5px] bg-orange"></span>Tribelt now</li>
          <li class="inline-flex items-center gap-2"><span class="size-2.5 rounded-full bg-blue-grey"></span>The best competitor</li>
          <li class="inline-flex items-center gap-2"><span class="h-[18px] w-[3px] bg-black"></span>The goal</li>
        </ul>
      </GoalSection>

      <nav aria-label="Parts of the app">
        <ul class="m-0 grid list-none grid-cols-1 gap-4 p-0 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
          <li v-for="t in home.tiles.value" :key="t.id" class="flex">
            <RouterLink :to="t.path" class="flex min-h-[190px] w-full flex-col gap-3 rounded-3xl bg-mist p-6 text-black no-underline hover:bg-grey hover:text-black">
              <span class="flex gap-2.5 text-sm font-bold tracking-[0.06em] uppercase"><span class="text-orange-ink">{{ t.number }}</span><span class="text-ink-2">{{ t.label }}</span></span>
              <span class="font-display text-5xl leading-none font-semibold tracking-[-0.02em]">{{ t.big }}</span>
              <span class="leading-snug text-ink-2">{{ t.text }}</span>
            </RouterLink>
          </li>
        </ul>
      </nav>

      <GoalSection id="latest" title="Latest runs and scans" intro="When each process last ran, and what comes next.">
        <div class="relative overflow-x-auto rounded-3xl bg-mist p-4 sm:p-5">
          <table class="w-full min-w-[560px] border-collapse text-left">
            <thead>
              <tr class="border-b border-silver text-sm text-ink-2">
                <th scope="col" class="py-2 pr-4 font-semibold">Process</th>
                <th scope="col" class="py-2 pr-4 font-semibold">Latest</th>
                <th scope="col" class="py-2 pr-4 font-semibold">Next</th>
                <th v-if="isAdmin" scope="col" class="py-2 font-semibold"><span class="sr-only">Manage</span></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in home.latest.value" :key="r.process" class="border-b border-grey last:border-b-0">
                <th scope="row" class="py-3 pr-4 font-normal"><RouterLink :to="r.to" class="font-medium">{{ r.process }}</RouterLink></th>
                <td class="py-3 pr-4">{{ r.latest }}</td>
                <td class="py-3 pr-4">{{ r.next }}</td>
                <td v-if="isAdmin" class="py-3 text-right"><RouterLink :to="r.manage" class="font-medium">Manage</RouterLink></td>
              </tr>
            </tbody>
          </table>
        </div>
      </GoalSection>
    </template>
  </PageFrame>
</template>
