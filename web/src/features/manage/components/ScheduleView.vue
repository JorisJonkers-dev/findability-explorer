<script setup lang="ts">
import { computed, ref, useTemplateRef, watch } from 'vue'
import type { PromptCadence } from '../../../infrastructure/api/types.gen'
import { usePromptSettings, useUpdatePromptSettings } from '../../prompt-test/composables/usePromptTest'
import { costLabel } from '../../prompt-test/domain/results'
import { parseBudget } from '../../prompt-test/domain/settings'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { dayLabel, timeLabel } from '../domain/activity'
import { scheduleCosts, scheduleWarning } from '../domain/estimate'
import ManageWidget from './widgets/ManageWidget.vue'
import TileWidget from './widgets/TileWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/** How often a Prompt run starts by itself, what that costs, and the monthly budget it runs against. */
const { settings, isLoading, error } = usePromptSettings()
const { isSaving, notice, error: saveError, save } = useUpdatePromptSettings()

const budget = ref('')
const budgetError = ref<string | null>(null)
const budgetInput = useTemplateRef<HTMLInputElement>('budget-input')
watch(
  settings,
  (s) => {
    if (s) budget.value = String(s.monthlyBudgetUsd)
  },
  { immediate: true },
)

const costs = computed(() => (settings.value ? scheduleCosts(settings.value.prices, settings.value.cadence) : null))
const warning = computed(() => (settings.value ? scheduleWarning(settings.value.prices, settings.value.cadence) : null))
const CADENCES: { value: PromptCadence; label: string }[] = [
  { value: 'monthly', label: 'Monthly' },
  { value: 'weekly', label: 'Weekly' },
]

function setCadence(cadence: PromptCadence): void {
  if (settings.value?.cadence === cadence) return
  save({ cadence }, cadence === 'weekly' ? 'A Prompt run will start weekly.' : 'A Prompt run will start monthly.')
}

function saveBudget(): void {
  const amount = parseBudget(budget.value)
  if (amount === null) {
    budgetError.value = 'The budget is an amount in dollars above $0 and at most $1,000.'
    return
  }
  budgetError.value = null
  save({ monthlyBudgetUsd: amount }, `The monthly budget is ${costLabel(amount)}.`)
}
</script>

<template>
  <ManageLayout
    title="Schedule"
    purpose="Choose how often the Prompt test runs by itself."
    :back="{ to: '/manage/prompt-test', label: 'Back to Prompt test' }"
    :results="{ to: '/ai-responses', label: 'See the AI responses' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the schedule…</p>
    <WidgetGrid v-else-if="settings && costs">
      <ManageWidget v-if="warning" title="This schedule costs more than the budget" accent>
        <p class="m-0" role="alert">{{ warning.text }}</p>
        <template #foot>
          <button
            v-if="warning.setMonthly"
            type="button"
            :disabled="isSaving"
            class="min-h-11 rounded-full border-0 bg-black px-6 font-semibold text-white hover:bg-ink-2 disabled:opacity-60"
            @click="setCadence('monthly')"
          >
            Set to monthly
          </button>
          <button type="button" class="min-h-11 rounded-full border-2 border-black bg-white px-5 font-semibold hover:bg-mist" @click="budgetInput?.focus()">
            Change the budget
          </button>
        </template>
      </ManageWidget>

      <ManageWidget title="How often" :span="8">
        <p class="m-0 text-ink-2">A run starts by itself at 06:00 UTC, on the first of the month or on Mondays. Weekly costs about four times as much as monthly.</p>
        <div role="group" aria-label="How often" class="flex flex-wrap gap-2">
          <button
            v-for="c in CADENCES"
            :key="c.value"
            type="button"
            :aria-pressed="settings.cadence === c.value"
            :disabled="isSaving"
            class="min-h-11 rounded-full border-2 px-5 font-semibold"
            :class="settings.cadence === c.value ? 'border-black bg-black text-white' : 'border-silver bg-white text-black hover:border-black'"
            @click="setCadence(c.value)"
          >
            {{ c.label }}
          </button>
        </div>
      </ManageWidget>
      <TileWidget title="Next run" :span="4" :value="dayLabel(settings.nextDueAt)" :note="`At ${timeLabel(settings.nextDueAt)}, ${settings.cadence}.`" />

      <ManageWidget title="What it costs" :span="8">
        <dl class="m-0 flex flex-wrap gap-x-10 gap-y-3" data-part="costs">
          <div class="flex flex-col-reverse">
            <dt class="text-sm text-ink-2">a run</dt>
            <dd class="m-0 font-display text-3xl font-semibold">{{ costs.run }}</dd>
          </div>
          <div class="flex flex-col-reverse">
            <dt class="text-sm text-ink-2">a month</dt>
            <dd class="m-0 font-display text-3xl font-semibold">{{ costs.month }}</dd>
          </div>
          <div class="flex flex-col-reverse">
            <dt class="text-sm text-ink-2">a year, {{ costs.runsPerYear }} runs</dt>
            <dd class="m-0 font-display text-3xl font-semibold">{{ costs.year }}</dd>
          </div>
        </dl>
      </ManageWidget>

      <ManageWidget title="Monthly budget" :span="4">
        <form id="budget-form" novalidate class="flex flex-col gap-2" @submit.prevent="saveBudget">
          <label class="flex flex-col gap-1.5 font-medium">
            Monthly budget, in dollars
            <input ref="budget-input" v-model="budget" type="text" inputmode="decimal" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base" />
          </label>
          <p v-if="budgetError" role="alert" class="m-0 font-medium text-orange-ink">{{ budgetError }}</p>
        </form>
        <p class="m-0 text-sm text-ink-2">Runs, retries, and readings stop when the month’s budget is spent, and keep what they have done.</p>
        <template #foot>
          <button type="submit" form="budget-form" :disabled="isSaving" class="min-h-11 rounded-full border-0 bg-black px-6 font-semibold text-white hover:bg-ink-2 disabled:opacity-60">
            Save the budget
          </button>
        </template>
      </ManageWidget>
      <p class="m-0 font-medium min-[1100px]:col-span-12" aria-live="polite">{{ notice }}</p>
      <p v-if="saveError" role="alert" class="m-0 font-medium text-orange-ink min-[1100px]:col-span-12">{{ saveError }}</p>
    </WidgetGrid>
  </ManageLayout>
</template>
