<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { ACTIONS, FIELD, LABEL, PRIMARY } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useBudgetChange, useLlmSettings } from '../composables/useLlmSettings'
import { budgetMeter, checkBudget } from '../domain/budget'
import BudgetMeter from './BudgetMeter.vue'

/** How much the work Claude does may spend in a month: spent and left, and the budget in dollars with Save. */
const { settings, isLoading, error } = useLlmSettings()
const { isSaving, error: saveError, save } = useBudgetChange()
const meter = computed(() => (settings.value ? budgetMeter(settings.value.budget) : null))
const typed = ref('')
const problem = ref<string | null>(null)
const notice = ref<string | null>(null)

watch(
  () => settings.value?.budget.monthlyBudgetUsd,
  (usd) => {
    if (usd !== undefined) typed.value = usd.toFixed(2)
  },
  { immediate: true },
)

function submit(): void {
  notice.value = null
  const check = checkBudget(typed.value)
  if (check.kind === 'invalid') {
    problem.value = check.message
    return
  }
  problem.value = null
  save(check.usd, () => {
    notice.value = 'The budget is saved.'
  })
}
</script>

<template>
  <ManageLayout title="Budgets" purpose="How much the work may spend in a month." :back="{ to: '/manage/keys', label: 'Back to Keys and spending' }">
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the budget…</p>
    <WidgetGrid v-else-if="settings && meter">
      <ManageWidget title="Anthropic monthly budget" :span="8">
        <BudgetMeter :budget="settings.budget" />
        <p class="m-0 text-ink-2">
          Product matching, the term review, the competitor review, and Claude Haiku reading responses spend it together. Work paid from
          this key stops when this month's spending reaches the budget. Work already done is kept. The month starts again on the 1st (UTC).
        </p>
        <form novalidate aria-label="Anthropic monthly budget" class="flex flex-col gap-3" @submit.prevent="submit">
          <label :class="LABEL">
            Budget in dollars
            <input v-model="typed" type="text" inputmode="decimal" :class="FIELD" />
          </label>
          <p v-if="problem" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ problem }}</p>
          <p v-if="saveError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ saveError }}</p>
          <div :class="ACTIONS">
            <button type="submit" :class="PRIMARY" :disabled="isSaving">Save the budget</button>
            <span class="font-medium" aria-live="polite">{{ notice }}</span>
          </div>
        </form>
      </ManageWidget>
      <ManageWidget title="A scan" :span="4">
        <div class="flex flex-col gap-1" data-testid="per-scan">
          <p class="m-0 font-semibold">{{ meter.perScan }}</p>
          <p v-if="meter.left" class="m-0 text-ink-2">{{ meter.left }}</p>
        </div>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
