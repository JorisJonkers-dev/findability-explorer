<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { ClaudeStepSummary } from '../../../infrastructure/api/types.gen'
import DataTable from '../../manage/components/widgets/DataTable.vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import type { Column } from '../../manage/domain/table'
import { dollars } from '../../settings/domain/usage'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useClaudeSteps } from '../composables/useClaudeSteps'
import { savedOn, stepInfo } from '../domain/steps'

/** Every step that asks a Claude model: which model it uses, its instructions, and what it costs. */
const { steps, isLoading, error } = useClaudeSteps()
const COLUMNS: readonly Column<ClaudeStepSummary>[] = [
  { key: 'title', label: 'Step', value: (s) => s.title },
  { key: 'section', label: 'Section', value: (s) => stepInfo(s.id).section.label, wide: true },
  { key: 'model', label: 'Model', value: (s) => s.current.modelName },
  { key: 'version', label: 'Instructions', value: (s) => s.current.createdAt },
  { key: 'cost', label: 'Cost', value: (s) => s.jobCostUsd, num: true },
]
</script>

<template>
  <ManageLayout
    title="Claude models and instructions"
    help-topic="claude-steps"
    purpose="Every step that asks a Claude model: which model it uses, its instructions, and what it costs."
    data-line="The Anthropic key pays for all of them."
    :back="{ to: '/manage/keys', label: 'Back to Keys and spending' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the steps…</p>
    <WidgetGrid v-else>
      <ManageWidget title="The steps" :span="12">
        <p class="m-0 text-ink-2">
          Open a step to change its model or its instructions. The instructions for reading responses are set in the code, so every run is
          read the same way; only its model is chosen here. Each change is saved as a version you can go back to.
        </p>
        <DataTable :rows="steps" :columns="COLUMNS" :row-key="(s) => s.id" label="Claude steps">
          <template #cell-title="{ row }">
            <RouterLink :to="`/manage/claude-steps/${row.id}`" class="font-semibold">{{ row.title }}</RouterLink>
            <span class="block text-sm text-muted">{{ stepInfo(row.id).does }}</span>
          </template>
          <template #cell-version="{ row }">
            <template v-if="row.fixedInstructions">Set in the code</template>
            <template v-else>
              Version {{ row.current.number }}<span class="block text-sm text-muted">{{ savedOn(row.current.createdAt) }}</span>
            </template>
          </template>
          <template #cell-cost="{ row }">
            {{ dollars(row.jobCostUsd) }}<span class="block text-sm text-muted">{{ stepInfo(row.id).jobLabel }}</span>
          </template>
        </DataTable>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
