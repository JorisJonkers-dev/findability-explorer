<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { ClaudeStepId } from '../../../infrastructure/api/types.gen'
import BigFigure from '../../manage/components/widgets/BigFigure.vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import { SECONDARY } from '../../manage/components/widgets/look'
import { dollars } from '../../settings/domain/usage'
import { useClaudeStep } from '../composables/useClaudeSteps'
import { stepInfo } from '../domain/steps'

/**
 * A Claude step on a page where it works: on a section's index a card with what one job costs, and
 * on a settings page a widget with its model and version. Either links to the step's page.
 */
const props = withDefaults(defineProps<{ stepId: ClaudeStepId; variant?: 'card' | 'widget' }>(), { variant: 'card' })
const id = computed(() => props.stepId)
const { step } = useClaudeStep(id)
const info = computed(() => stepInfo(props.stepId))
const link = computed(() => (step.value?.fixedInstructions ? 'Change the model' : 'Change the model or instructions'))
</script>

<template>
  <ManageWidget :title="variant === 'card' ? (step?.title ?? 'Claude step') : 'Model and instructions'" :span="variant === 'card' ? 4 : 6">
    <template v-if="step">
      <p v-if="variant === 'card'" class="m-0 text-ink-2">{{ info.does }} Done by {{ step.current.modelName }}.</p>
      <BigFigure v-if="variant === 'card'" :value="dollars(step.jobCostUsd)" :label="info.jobLabel" />
      <p v-else class="m-0 text-ink-2">
        {{ step.current.modelName }} does this step, with instructions version {{ step.current.number }}. About {{ dollars(step.jobCostUsd) }}
        for {{ info.jobLabel }}.
      </p>
    </template>
    <p v-else class="m-0 text-muted">Loading the step…</p>
    <template #foot>
      <RouterLink :to="`/manage/claude-steps/${stepId}`" :class="SECONDARY">{{ link }}</RouterLink>
    </template>
  </ManageWidget>
</template>
