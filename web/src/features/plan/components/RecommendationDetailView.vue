<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import PageFrame from '../../shell/components/PageFrame.vue'
import { useMe } from '../../shell/composables/useMe'
import { companyPath } from '../../shell/domain/sections'
import { usePlan } from '../composables/usePlan'
import { doneLine, effortWhy, formatMeasure, goalBar, goalOf, goalSentence, kindOf, lcFirst, measureName, nowSentence, placeLine } from '../domain/recommendations'
import { foundLine } from '../domain/routine'
import RecSection from './RecSection.vue'

const props = defineProps<{ id: string }>()
const { isAdmin } = useMe()
const { plan, isLoading, error, isSaving, saveError, tick } = usePlan()

const items = computed(() => plan.value?.improvements ?? [])
const it = computed(() => items.value.find((i) => i.id === props.id))
const checks = computed(() => plan.value?.readinessChecks ?? 0)
const goal = computed(() => (it.value && plan.value ? goalOf(it.value.measure, plan.value.watched) : undefined))
const bar = computed(() => (it.value ? goalBar(it.value, goal.value, checks.value) : undefined))
const barLabel = computed(() => {
  const r = it.value
  if (!r) return ''
  const f = (v: number) => formatMeasure(r.measure, v, checks.value)
  const parts = [r.tribelt === undefined ? 'Tribelt has no figure yet' : `Tribelt now ${f(r.tribelt)}`]
  if (r.leader) parts.push(`the best competitor ${f(r.leader.value)}`)
  if (goal.value !== undefined) parts.push(`the goal ${f(goal.value)}`)
  return parts.join(', ')
})
const found = computed(() => (it.value?.auto ? foundLine(it.value.parts) : ''))
const place = computed(() => placeLine(items.value, props.id))
/** A place along the bar, as a percentage with one decimal. */
const pct = (v: number) => `${String(Math.round(v * 1000) / 10)}%`

const back = { to: '/recommendations', label: 'Back to Recommendations' }
const manage = { to: '/manage/recommendations', label: 'Manage the recommendations' }
const PURPOSE = 'The steps to make this change, why it ranks where it does, and how to tell it is done.'
</script>

<template>
  <PageFrame
    section="recommendations"
    :title="it?.title ?? 'Recommendation'"
    :purpose="PURPOSE"
    :data-line="it ? 'Written by hand, and applies in general.' : undefined"
    :back="back"
    :manage="manage"
    help-topic="recommendations"
  >
    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the recommendation…</p>
    <p v-else-if="!it" class="m-0 text-ink-2">No recommendation has this address. <RouterLink to="/recommendations">See all the recommendations.</RouterLink></p>
    <template v-else>
      <div class="grid items-start gap-10 min-[1100px]:grid-cols-2">
        <RecSection title="Where, and how" heading-id="where-title">
          <p class="m-0">This applies to the whole site.</p>
        </RecSection>
        <RecSection title="Why it ranks here" heading-id="why-title">
          <div class="flex max-w-[64ch] flex-col gap-3.5">
            <p class="m-0">{{ it.why }}</p>
            <p class="m-0">{{ it.moves }}</p>
            <p class="m-0">Benefit {{ it.benefit }} of 5. Effort {{ it.effort }} of 5. {{ effortWhy(kindOf(it)) }}</p>
            <p v-if="place" class="m-0">{{ place }}</p>
          </div>
        </RecSection>
      </div>

      <RecSection title="The goal it improves" heading-id="goal-title">
        <div class="flex flex-col gap-3.5 rounded-3xl bg-mist p-5 sm:p-6">
          <p class="m-0">
            <strong>{{ measureName(it.measure) }}.</strong> {{ nowSentence(it, goal, checks) }} {{ goalSentence(it, checks) }}
          </p>
          <div v-if="bar" role="img" :aria-label="barLabel" class="relative h-3.5 rounded-full bg-grey">
            <span data-part="now" class="absolute inset-y-0 left-0 rounded-full bg-orange" :style="{ width: pct(bar.now) }"></span>
            <span
              v-if="bar.leader !== undefined"
              data-part="leader"
              class="absolute top-1/2 size-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-[3px] border-solid border-blue-grey bg-white"
              :style="{ left: pct(bar.leader) }"
            ></span>
            <span v-if="bar.goal !== undefined" data-part="goal" class="absolute -inset-y-1.5 w-[3px] -translate-x-1/2 bg-black" :style="{ left: pct(bar.goal) }"></span>
          </div>
          <div aria-hidden="true" class="flex flex-wrap gap-x-5 gap-y-1 text-sm text-ink-2">
            <span class="inline-flex items-center gap-2"><span class="size-3 rounded-sm bg-orange"></span>Tribelt now</span>
            <span v-if="it.leader" class="inline-flex items-center gap-2"><span class="size-3 rounded-full border-2 border-solid border-blue-grey bg-white"></span>The best competitor</span>
            <span v-if="goal !== undefined" class="inline-flex items-center gap-2"><span class="h-3.5 w-[3px] bg-black"></span>The goal</span>
          </div>
          <p v-if="it.leader" class="m-0">
            <RouterLink :to="companyPath(it.leader.competitorId)">{{ it.leader.name }}</RouterLink> is the best competitor on {{ lcFirst(measureName(it.measure)) }}. Look at
            how it does this.
          </p>
        </div>
      </RecSection>

      <div class="grid items-start gap-10 min-[1100px]:grid-cols-2">
        <RecSection title="Steps" heading-id="steps-title">
          <ol class="m-0 flex flex-col gap-2.5 pl-6">
            <li v-for="step in it.steps" :key="step" class="pl-1">{{ step }}</li>
          </ol>
        </RecSection>
        <RecSection title="When it counts as done" heading-id="done-title">
          <div class="flex flex-col gap-3">
            <p class="m-0">
              <span class="mr-2 rounded-full px-2.5 py-0.5 text-sm font-semibold" :class="it.done ? 'bg-orange-pale text-orange-ink' : 'bg-mist text-black'">{{
                it.done ? 'Done' : 'Open'
              }}</span>
              {{ it.doneWhen }}
            </p>
            <p v-if="found" class="m-0">{{ found }}</p>
            <p v-if="it.auto" class="m-0 text-ink-2">The latest scan of tribelt.nl checks this one.</p>
            <p v-if="it.done" role="status" class="m-0 font-medium">{{ doneLine(it) }}</p>
            <div v-if="isAdmin && !it.auto" class="flex flex-wrap items-center gap-3">
              <button
                type="button"
                :disabled="isSaving"
                class="min-h-12 cursor-pointer rounded-full px-5 font-bold"
                :class="it.done ? 'border-2 border-solid border-black bg-white text-black hover:bg-mist' : 'border-0 bg-black text-white hover:bg-ink-2'"
                @click="tick(it.id, !it.done)"
              >
                {{ it.done ? 'Undo, it is not done' : 'Mark as done' }}
              </button>
              <span v-if="!it.done" class="text-sm text-muted">A scan cannot check this one, so an administrator marks it.</span>
            </div>
            <p v-if="saveError" role="alert" class="m-0 font-medium text-orange-ink">{{ saveError }}</p>
          </div>
        </RecSection>
      </div>
    </template>
  </PageFrame>
</template>
