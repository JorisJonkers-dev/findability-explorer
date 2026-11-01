<script setup lang="ts">
import { toRef } from 'vue'
import { RouterLink } from 'vue-router'
import BarList from '../../../shared/ui/charts/BarList.vue'
import HistorySection from '../../prompt-test/components/HistorySection.vue'
import PageFrame from '../../shell/components/PageFrame.vue'
import { useGoalPage } from '../composables/useGoalPage'
import GoalSection from './GoalSection.vue'

const props = defineProps<{ measure: string }>()
const goal = useGoalPage(toRef(props, 'measure'))
const page = goal.page
const MARK = { yes: '✓', no: '✕', unknown: '?' } as const
</script>

<template>
  <PageFrame
    section="overview"
    :title="page?.name ?? 'Goal'"
    :purpose="page?.purpose ?? 'One of the five goals Tribelt is measured against.'"
    :data-line="goal.dataLine.value"
    :back="{ to: '/', label: 'Back to Overview' }"
    :manage="{ to: '/manage/recommendations', label: 'Set the goals' }"
    :help-topic="goal.helpTopic.value"
  >
    <p v-if="goal.error.value" role="alert" class="m-0 text-lg text-orange-ink">{{ goal.error.value }}</p>
    <p v-else-if="goal.isLoading.value" class="m-0 text-lg text-muted">Loading the goal…</p>
    <p v-else-if="goal.missing.value" class="m-0 text-lg">There is no goal at this address. Pick one from the Overview.</p>

    <template v-else-if="page">
      <p data-part="summary" class="m-0 rounded-3xl bg-orange-pale px-6 py-5 font-display text-xl leading-snug font-medium text-pretty sm:px-8 sm:text-2xl">
        {{ page.summary }}
      </p>

      <div class="grid items-start gap-10 min-[1100px]:grid-cols-2">
        <GoalSection id="compare" title="Tribelt against the competitors" :intro="page.comparison">
          <div class="rounded-3xl bg-mist p-5 sm:p-6">
            <BarList :rows="page.bars" :max="page.barsMax" :label="page.name" wide />
          </div>
        </GoalSection>

        <HistorySection
          v-if="goal.history.value"
          id="over-time"
          title="Over time"
          intro="One point for each Prompt run. A dashed ring marks a partial run."
          :measures="[goal.history.value]"
          :goal="page.goal"
        />
        <GoalSection v-else id="over-time" title="Over time">
          <p class="m-0 rounded-3xl bg-mist p-5 text-ink-2 sm:p-6">Only the newest scan of tribelt.nl is kept for this goal, so there is no line yet.</p>
        </GoalSection>
      </div>

      <GoalSection id="broken" title="Broken down">
        <p v-if="goal.breakdownError.value" role="alert" class="m-0 text-orange-ink">{{ goal.breakdownError.value }}</p>
        <p v-else-if="!goal.breakdown.value" class="m-0 text-muted">Loading…</p>
        <template v-else-if="goal.breakdown.value.kind === 'groups'">
          <p v-if="goal.breakdown.value.groups.length === 0" class="m-0 text-ink-2">No Prompt run has finished yet.</p>
          <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            <div v-for="g in goal.breakdown.value.groups" :key="g.title" class="flex flex-col gap-3 rounded-3xl bg-mist p-5 sm:p-6">
              <h3 class="m-0 text-lg font-semibold">{{ g.title }}</h3>
              <BarList :rows="g.rows" :max="1" :label="g.title" />
            </div>
          </div>
        </template>
        <ul v-else-if="goal.breakdown.value.kind === 'checks'" class="m-0 flex list-none flex-col gap-3 rounded-3xl bg-mist p-5 sm:p-6">
          <li v-for="c in goal.breakdown.value.checks" :key="c.label" class="grid grid-cols-[1.75rem_minmax(0,1fr)] gap-3">
            <span class="font-bold" :class="c.passed ? 'text-black' : 'text-orange-ink'">
              <span aria-hidden="true">{{ c.passed === null ? MARK.unknown : c.passed ? MARK.yes : MARK.no }}</span>
              <span class="sr-only">{{ c.passed === null ? 'Not known' : c.passed ? 'Passed' : 'Not passed' }}</span>
            </span>
            <span><strong>{{ c.label }}.</strong> {{ c.what }}</span>
          </li>
        </ul>
        <div v-else class="rounded-3xl bg-mist p-5 sm:p-6">
          <BarList :rows="goal.breakdown.value.rows" :max="goal.breakdown.value.max" label="Search terms tribelt.nl uses, per language" />
        </div>
      </GoalSection>

      <GoalSection id="improves" title="What improves it">
        <ul v-if="page.improvements.length" class="m-0 flex list-none flex-col rounded-3xl bg-mist px-5 py-2 sm:px-6">
          <li v-for="i in page.improvements" :key="i.id" class="flex flex-col gap-1 border-b border-grey py-3 last:border-b-0">
            <RouterLink :to="`/recommendations/${i.id}`" class="font-semibold">{{ i.title }}</RouterLink>
            <span class="text-sm text-muted">Benefit {{ i.benefit }}, effort {{ i.effort }}</span>
          </li>
        </ul>
        <p v-else class="m-0 text-ink-2">No open recommendation aims at this goal.</p>
      </GoalSection>
    </template>
  </PageFrame>
</template>
