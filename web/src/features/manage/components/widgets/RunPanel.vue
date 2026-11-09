<script setup lang="ts">
import { computed } from 'vue'
import type { PromptRun } from '../../../../infrastructure/api/types.gen'
import { useRetryPlan } from '../../../prompt-test/composables/usePromptTest'
import { dayLabel } from '../../domain/activity'
import type { BudgetName } from '../../domain/estimate'
import StartButton from './StartButton.vue'

/**
 * A run with failed responses: Retry the failed responses, paid from the OpenRouter budget, and
 * Read them again with the model that reads now, paid from its key's budget, each with its estimate
 * from the run's retry plan.
 */
const props = defineProps<{
  run: PromptRun
  /** What is left of the OpenRouter budget, which asking again spends. */
  left: number
  /** The model that reads the responses again, and the budget, by key and what is left, it spends. */
  reader: { name: string; budget: BudgetName; left: number }
  busy: boolean
  busyNote?: string
}>()
defineEmits<{ retry: []; reread: [] }>()

const NUMBER = new Intl.NumberFormat('en-GB')
const { plan } = useRetryPlan(
  computed(() => props.run.id),
  computed(() => true),
)
const missing = computed(() => props.run.tasks - props.run.answered)
const ask = computed(() => plan.value?.askTasks ?? missing.value)
const read = computed(() => plan.value?.readTasks ?? 0)
const count = (n: number) => (n === 1 ? '1 response' : `${NUMBER.format(n)} responses`)
</script>

<template>
  <section :aria-label="`Run of ${dayLabel(run.startedAt)}`" class="flex flex-col gap-3 rounded-2xl bg-white p-5">
    <h3 class="m-0 font-display text-lg font-semibold">Run of {{ dayLabel(run.startedAt) }}</h3>
    <p class="m-0 text-ink-2">{{ NUMBER.format(missing) }} of {{ NUMBER.format(run.tasks) }} responses failed.</p>
    <StartButton
      :label="`Retry the ${NUMBER.format(ask)} failed responses`"
      budget="OpenRouter"
      :usd="plan?.askCostUsd ?? 0"
      :left="left"
      :note="count(ask)"
      :disabled-note="busyNote ?? (plan && ask === 0 ? 'No question of these can be asked again.' : undefined)"
      :busy="busy"
      @start="$emit('retry')"
    />
    <StartButton
      :label="`Read them again with ${reader.name}`"
      :budget="reader.budget"
      :usd="plan?.readCostUsd ?? 0"
      :left="reader.left"
      :note="count(read)"
      :disabled-note="busyNote ?? (read === 0 ? 'Every response was read.' : undefined)"
      :busy="busy"
      @start="$emit('reread')"
    />
  </section>
</template>
