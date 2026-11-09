<script setup lang="ts">
import { computed } from 'vue'
import { usePromptResults, usePromptSettings, useRetryRun } from '../../prompt-test/composables/usePromptTest'
import { costLabel } from '../../prompt-test/domain/results'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useStopWork } from '../composables/useManage'
import { dayLabel, endedOf, needsFinishing, runAsWork, runningWork, whenLabel } from '../domain/activity'
import { useStepModelName } from '../../claude-steps/composables/useClaudeSteps'
import { useLlmSettings } from '../../settings/composables/useLlmSettings'
import { leftUsd, readerOf } from '../domain/estimate'
import EndedPill from './widgets/EndedPill.vue'
import ManageWidget from './widgets/ManageWidget.vue'
import RunningWidget from './widgets/RunningWidget.vue'
import RunPanel from './widgets/RunPanel.vue'
import TileWidget from './widgets/TileWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/** Every Prompt run, how each ended, and how to finish one that stopped early or lost responses. */
const { results, error } = usePromptResults()
const { settings } = usePromptSettings()
const retrier = useRetryRun()
const stopper = useStopWork()

const NUMBER = new Intl.NumberFormat('en-GB')
const runs = computed(() => results.value?.runs ?? [])
const running = computed(() => runningWork(runs.value.filter((r) => r.status === 'running').map(runAsWork), settings.value?.prices.runUsd))
const failing = computed(() => runs.value.filter(needsFinishing))
const busy = computed(() => running.value.length > 0)
const left = computed(() => (settings.value ? leftUsd({ budgetUsd: settings.value.monthlyBudgetUsd, spentUsd: settings.value.spentThisMonthUsd }) : 0))
const last = computed(() => runs.value[0])
// Reading again is paid by the key that reads: the Anthropic key with the step's model while one
// works, else OpenRouter's reader.
const llm = useLlmSettings()
const readingModel = useStepModelName('reading')
const reader = computed(() => {
  const r = readerOf(settings.value?.extractorModel ?? '')
  return r.budget === 'Anthropic'
    ? { name: readingModel.value, budget: r.budget, left: llm.settings.value?.budget.leftUsd ?? 0 }
    : { name: r.model, budget: r.budget, left: left.value }
})
</script>

<template>
  <ManageLayout
    title="Runs"
    purpose="Every Prompt run, and how to finish one that stopped early."
    :back="{ to: '/manage/prompt-test', label: 'Back to Prompt test' }"
    :results="{ to: '/ai-responses', label: 'See the AI responses' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <WidgetGrid>
      <TileWidget title="Runs" :span="4" :value="NUMBER.format(runs.length)" :note="last ? `The last one was on ${dayLabel(last.startedAt)}.` : 'No run yet.'" />
      <TileWidget
        title="Need a retry"
        :span="4"
        :value="NUMBER.format(failing.length)"
        :note="failing.length > 0 ? 'Some responses failed in these runs.' : 'Every run is complete.'"
      />
      <ManageWidget title="Run status" :span="4">
        <span class="self-start rounded-full px-2.5 py-0.5 font-semibold" :class="busy ? 'bg-orange text-black' : 'bg-white text-black'" data-part="run-status">
          {{ busy ? 'A run is running' : 'Ready for a new run' }}
        </span>
        <p class="m-0 text-ink-2">The AI responses pages show the newest complete run.</p>
      </ManageWidget>

      <RunningWidget v-for="w in running" :key="w.id" :work="w" :stopping="stopper.isStopping.value" @stop="stopper.stop(w)" />
      <p v-if="stopper.notice.value" aria-live="polite" class="m-0 font-medium basis-full">{{ stopper.notice.value }}</p>
      <p v-if="stopper.error.value" role="alert" class="m-0 font-medium text-orange-ink basis-full">{{ stopper.error.value }}</p>

      <ManageWidget title="Finish the failed runs" :span="12">
        <template v-if="failing.length > 0">
          <p class="m-0 max-w-[80ch] text-ink-2">
            Retry asks the models again for the responses that failed, and only those. Reading again keeps the responses the models gave, and asks
            Claude Haiku to read the ones it missed the first time.
          </p>
          <div class="grid gap-4 min-[800px]:grid-cols-2">
            <RunPanel
              v-for="r in failing"
              :key="r.id"
              :run="r"
              :left="left"
              :reader="reader"
              :busy="retrier.isRetrying.value"
              :busy-note="busy ? 'Wait for the running work to finish or stop it.' : undefined"
              @retry="retrier.retry(r.id)"
              @reread="retrier.reread(r.id)"
            />
          </div>
          <p class="m-0 font-medium" aria-live="polite">{{ retrier.notice.value }}</p>
          <p v-if="retrier.error.value" role="alert" class="m-0 font-medium text-orange-ink">{{ retrier.error.value }}</p>
        </template>
        <p v-else class="m-0 text-ink-2">Every run is complete. Nothing needs a retry.</p>
      </ManageWidget>

      <ManageWidget title="All runs" :span="12">
        <p class="m-0 text-ink-2">A stopped or failed run stays here until you finish it.</p>
        <p v-if="runs.length === 0" class="m-0 text-ink-2">No run yet.</p>
        <div v-else class="overflow-x-auto" tabindex="0" role="region" aria-label="The runs table">
          <table class="w-full border-collapse text-left">
            <caption class="sr-only">Every Prompt run, newest first, and how each ended</caption>
            <thead>
              <tr class="border-b-2 border-black text-sm text-muted">
                <th scope="col" class="py-1.5 pr-3 font-medium">Date</th>
                <th scope="col" class="py-1.5 pr-3 text-right font-medium">Prompt set</th>
                <th scope="col" class="py-1.5 pr-3 text-right font-medium">Models</th>
                <th scope="col" class="py-1.5 pr-3 text-right font-medium">Responses</th>
                <th scope="col" class="py-1.5 pr-3 font-medium">How it ended</th>
                <th scope="col" class="py-1.5 text-right font-medium">Cost</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in runs" :key="r.id" class="border-b border-grey align-top">
                <th scope="row" class="py-2 pr-3 font-medium whitespace-nowrap">{{ whenLabel(r.startedAt) }}</th>
                <td class="py-2 pr-3 text-right whitespace-nowrap">Version {{ r.promptSetVersion }}</td>
                <td class="py-2 pr-3 text-right tabular-nums">{{ r.models.length }}</td>
                <td class="py-2 pr-3 text-right whitespace-nowrap tabular-nums">{{ NUMBER.format(r.answered) }} of {{ NUMBER.format(r.tasks) }}</td>
                <td class="py-2 pr-3"><EndedPill :ended="endedOf(r)" /></td>
                <td class="py-2 text-right tabular-nums">{{ costLabel(r.costUsd) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
