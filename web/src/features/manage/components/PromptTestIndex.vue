<script setup lang="ts">
import StepCard from '../../claude-steps/components/StepCard.vue'
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import ProgressBar from '../../../shared/ui/ProgressBar.vue'
import { usePromptResults, usePromptSettings, useStartRun } from '../../prompt-test/composables/usePromptTest'
import { costLabel } from '../../prompt-test/domain/results'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useStopWork } from '../composables/useManage'
import { dayLabel, lastEnded, needsFinishing, runAsWork, runningWork, timeLabel } from '../domain/activity'
import { budgetFraction, leftUsd } from '../domain/estimate'
import HubWidget from './widgets/HubWidget.vue'
import ManageWidget from './widgets/ManageWidget.vue'
import RunningWidget from './widgets/RunningWidget.vue'
import StartButton from './widgets/StartButton.vue'
import TileWidget from './widgets/TileWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/**
 * The Prompt test's index: the last run, the next, the cost of one, and the budget left; Start a
 * Prompt run with its estimate; the running run with its Stop; then a card per subpage.
 */
const { settings, isLoading, error } = usePromptSettings()
const { results } = usePromptResults()
const starter = useStartRun()
const stopper = useStopWork()

const NUMBER = new Intl.NumberFormat('en-GB')
const runs = computed(() => results.value?.runs ?? [])
const last = computed(() => lastEnded(runs.value))
const running = computed(() => runningWork(runs.value.filter((r) => r.status === 'running').map(runAsWork), settings.value?.prices.runUsd))
const finishing = computed(() => runs.value.filter(needsFinishing).length)
const questions = computed(() => settings.value?.promptSet.prompts.length ?? 0)
const models = computed(() => settings.value?.providers.length ?? 0)
const budget = computed(() => (settings.value ? { budgetUsd: settings.value.monthlyBudgetUsd, spentUsd: settings.value.spentThisMonthUsd } : { budgetUsd: 0, spentUsd: 0 }))
const left = computed(() => leftUsd(budget.value))
const cadence = computed(() => (settings.value?.cadence === 'weekly' ? 'Weekly' : 'Monthly'))

const disabledNote = computed(() => {
  if (running.value.length > 0) return 'A Prompt run is already running. Stop it or wait for it to finish.'
  if (settings.value && !settings.value.keyWorks) return 'Add a working OpenRouter key first, in Keys and spending.'
  return undefined
})
</script>

<template>
  <ManageLayout
    title="Prompt test"
    purpose="Find out which companies AI assistants name when buyers ask about metal conveyor belts."
    :results="{ to: '/ai-responses', label: 'See the AI responses' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the Prompt test…</p>
    <WidgetGrid v-else-if="settings">
      <TileWidget
        title="Last run"
        :span="3"
        :value="last ? dayLabel(last.startedAt) : 'None yet'"
        :note="last ? `${NUMBER.format(last.answered)} of ${NUMBER.format(last.tasks)} responses` : 'No run has finished.'"
      />
      <TileWidget title="Next run" :span="3" :value="dayLabel(settings.nextDueAt)" :note="`${cadence}, at ${timeLabel(settings.nextDueAt)}`" />
      <TileWidget title="Cost per run" :span="3" :value="costLabel(settings.prices.runUsd)" :note="`${models} models, ${questions} questions`" />
      <ManageWidget title="Budget left" :span="3">
        <div class="flex flex-wrap items-baseline justify-between gap-2"><span>OpenRouter spent</span><strong>{{ costLabel(budget.spentUsd) }} of {{ costLabel(budget.budgetUsd) }}</strong></div>
        <ProgressBar :fraction="budgetFraction(budget)" label="OpenRouter spent" :text="`${costLabel(budget.spentUsd)} of ${costLabel(budget.budgetUsd)}`" />
        <p class="m-0 text-ink-2" data-part="budget-left">{{ costLabel(left) }} left this month</p>
      </ManageWidget>

      <ManageWidget title="Start a Prompt run" :span="12">
        <p class="m-0 max-w-[70ch] text-ink-2">
          Asks {{ NUMBER.format(questions) }} questions of {{ NUMBER.format(models) }} models, each from its own knowledge and with web search,
          {{ NUMBER.format(questions * models * 2) }} responses in all. When a run finishes, the AI responses pages show its results.
        </p>
        <template #foot>
          <div class="flex flex-col gap-2">
            <StartButton
              label="Start a Prompt run"
              budget="OpenRouter"
              :usd="settings.prices.runUsd"
              :left="left"
              :disabled-note="disabledNote"
              :busy="starter.isStarting.value"
              @start="starter.start()"
            />
            <RouterLink v-if="!settings.keyWorks" to="/manage/keys" class="self-start font-medium">Open Keys and spending</RouterLink>
            <p class="m-0 font-medium" aria-live="polite">{{ starter.notice.value }}</p>
            <p v-if="starter.error.value" role="alert" class="m-0 font-medium text-orange-ink">{{ starter.error.value }}</p>
          </div>
        </template>
      </ManageWidget>

      <RunningWidget v-for="w in running" :key="w.id" :work="w" :stopping="stopper.isStopping.value" @stop="stopper.stop(w)" />
      <p v-if="stopper.notice.value" aria-live="polite" class="m-0 font-medium basis-full">{{ stopper.notice.value }}</p>
      <p v-if="stopper.error.value" role="alert" class="m-0 font-medium text-orange-ink basis-full">{{ stopper.error.value }}</p>

      <HubWidget
        title="Questions"
        :span="4"
        text="The buyer questions, published as versions."
        :figures="[
          { value: NUMBER.format(questions), label: 'questions' },
          { value: String(settings.promptSet.version), label: 'version' },
        ]"
        to="/manage/prompt-test/questions"
        label="See the questions"
      />
      <HubWidget
        title="Models"
        :span="4"
        text="The AI models that are asked, and what each costs."
        :figures="[
          { value: NUMBER.format(models), label: 'models' },
          { value: costLabel(settings.prices.runUsd), label: 'a run' },
        ]"
        to="/manage/prompt-test/models"
        label="Change the models"
      />
      <HubWidget
        title="Schedule"
        :span="4"
        text="How often a run starts by itself."
        :figures="[{ value: cadence, label: 'how often' }]"
        to="/manage/prompt-test/schedule"
        label="Change the schedule"
      />
      <HubWidget
        title="Fact sheet"
        :span="6"
        text="The text about Tribelt that the responses are checked against."
        :figures="[{ value: String(settings.factSheet.version), label: 'version' }]"
        to="/manage/prompt-test/fact-sheet"
        label="Change the fact sheet"
      />
      <HubWidget
        title="Runs"
        :span="6"
        text="Every run, how it ended, and how to finish one that stopped early."
        :figures="[
          { value: NUMBER.format(runs.length), label: 'runs' },
          { value: NUMBER.format(finishing), label: 'need a retry' },
        ]"
        to="/manage/prompt-test/runs"
        label="See the runs"
      />
      <StepCard step-id="reading" />
    </WidgetGrid>
  </ManageLayout>
</template>
