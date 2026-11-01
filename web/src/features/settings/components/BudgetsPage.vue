<script setup lang="ts">
import { computed } from 'vue'
import BigFigure from '../../manage/components/widgets/BigFigure.vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useLlmSettings } from '../composables/useLlmSettings'
import { budgetMeter } from '../domain/budget'
import BudgetMeter from './BudgetMeter.vue'

/** How much the language-model work may use in a month: the tokens used and left, and what a scan uses. */
const { settings, isLoading, error } = useLlmSettings()
const meter = computed(() => (settings.value ? budgetMeter(settings.value.budget) : null))
const NUMBER = new Intl.NumberFormat('en-GB')
</script>

<template>
  <ManageLayout title="Budgets" purpose="How much the language-model work may use in a month." :back="{ to: '/manage/keys', label: 'Back to Keys and spending' }">
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the budget…</p>
    <WidgetGrid v-else-if="settings && meter">
      <ManageWidget title="Monthly token budget" :span="8">
        <BudgetMeter :budget="settings.budget" />
        <p class="m-0 text-ink-2">
          The tokens count for product matching, the term review, the competitor review, and Claude Haiku reading Prompt test responses
          together. The month runs in UTC. Once the tokens are used, scans run without product matching and Haiku stops until the next month.
        </p>
      </ManageWidget>
      <ManageWidget title="A scan" :span="4">
        <BigFigure :value="NUMBER.format(settings.budget.monthlyTokens)" label="tokens a month" />
        <div class="flex flex-col gap-1" data-testid="per-scan">
          <p class="m-0 font-semibold">{{ meter.perScan }}</p>
          <p v-if="meter.left" class="m-0 text-ink-2">{{ meter.left }}</p>
        </div>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
