<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import ProgressBar from '../../../shared/ui/ProgressBar.vue'
import { useCompetitorCandidates, usePromptSettings } from '../../prompt-test/composables/usePromptTest'
import { costLabel } from '../../prompt-test/domain/results'
import { useTermCandidates } from '../../review/composables/useTermReview'
import { useLlmSettings } from '../../settings/composables/useLlmSettings'
import { useActivity, useStopWork, useWorkScans } from '../composables/useManage'
import { missingKeys, queuedScans, recentWork, runningWork, spendingRows, waitLine, whenLabel } from '../domain/activity'
import { budgetFraction, leftUsd } from '../domain/estimate'
import EndedPill from './widgets/EndedPill.vue'
import ManageWidget from './widgets/ManageWidget.vue'
import RunningWidget from './widgets/RunningWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/**
 * Activity's widgets: what runs, each with its Stop; what comes next; what waits for an administrator; the
 * two budgets; what the month cost by kind of work; and the last 20 pieces of work.
 */
const { activity, error } = useActivity()
const { scans } = useWorkScans()
const { settings } = usePromptSettings()
const llm = useLlmSettings()
const { counts: terms, review: termReview } = useTermCandidates(ref('pending'))
const { counts: candidates } = useCompetitorCandidates(ref('pending'))
const stopper = useStopWork()

const running = computed(() => runningWork(activity.value?.running ?? [], settings.value?.prices.runUsd))
const runningSpan = computed(() => (running.value.length === 1 ? 12 : running.value.length === 2 ? 6 : 4))
const recent = computed(() => recentWork(activity.value?.recent ?? []))
const queued = computed(() => queuedScans(scans.value))
const termsWaiting = computed(() => terms.value?.pending ?? 0)
const candidatesWaiting = computed(() => candidates.value?.pending ?? 0)
const keysMissing = computed(() => (llm.settings.value ? missingKeys(llm.settings.value.credentials) : []))
const anyWaiting = computed(() => termsWaiting.value > 0 || candidatesWaiting.value > 0 || keysMissing.value.length > 0)

const openRouter = computed(() => {
  const s = settings.value
  if (!s) return null
  const budget = { budgetUsd: s.monthlyBudgetUsd, spentUsd: s.spentThisMonthUsd }
  const fraction = budgetFraction(budget)
  return {
    spent: `${costLabel(s.spentThisMonthUsd)} of ${costLabel(s.monthlyBudgetUsd)}`,
    fraction,
    left: costLabel(leftUsd(budget)),
    used: `${String(Math.round(fraction * 100))}%`,
  }
})
const anthropic = computed(() => {
  const b = llm.settings.value?.budget
  if (!b) return null
  const budget = { budgetUsd: b.monthlyBudgetUsd, spentUsd: b.spentThisMonthUsd }
  const fraction = budgetFraction(budget)
  return {
    spent: `${costLabel(b.spentThisMonthUsd)} of ${costLabel(b.monthlyBudgetUsd)}`,
    fraction,
    left: costLabel(b.leftUsd),
    used: `${String(Math.round(fraction * 100))}%`,
  }
})
const spending = computed(() => spendingRows(activity.value?.spending.lines ?? []))

const ROW = 'flex flex-col gap-1 border-b border-grey pb-3 last:border-b-0 last:pb-0'
</script>

<template>
  <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
  <WidgetGrid>
    <template v-if="running.length > 0">
      <RunningWidget v-for="w in running" :key="w.id" :work="w" :span="runningSpan" :stopping="stopper.isStopping.value" @stop="stopper.stop(w)" />
    </template>
    <ManageWidget v-else title="Running now" :span="12">
      <p class="m-0 text-ink-2">Nothing is running.</p>
    </ManageWidget>
    <p v-if="stopper.notice.value" aria-live="polite" class="m-0 font-medium basis-full">{{ stopper.notice.value }}</p>
    <p v-if="stopper.error.value" role="alert" class="m-0 font-medium text-orange-ink basis-full">{{ stopper.error.value }}</p>

    <ManageWidget title="Coming up" :span="6">
      <div :class="ROW">
        <h3 class="m-0 text-base font-semibold">Next Prompt run</h3>
        <p v-if="settings" class="m-0 flex flex-wrap gap-x-4 text-ink-2" data-part="next-run">
          <span>{{ whenLabel(settings.nextDueAt) }}</span>
          <span>About {{ costLabel(settings.prices.runUsd) }} a run, {{ settings.cadence }}</span>
        </p>
        <RouterLink to="/manage/prompt-test/schedule" class="self-start font-medium">Open Schedule</RouterLink>
      </div>
      <div :class="ROW">
        <h3 class="m-0 text-base font-semibold">Term review</h3>
        <p class="m-0 text-ink-2" data-part="term-review">
          <template v-if="termReview?.running">The term review is reviewing the waiting search terms.</template>
          <template v-else-if="termReview?.on">On, so waiting search terms are reviewed automatically.</template>
          <template v-else-if="termsWaiting > 0">Off, so {{ waitLine(termsWaiting, 'search term', 'search terms') }} for you.</template>
          <template v-else>Off. No search terms are waiting.</template>
          <RouterLink to="/manage/terms" class="underline underline-offset-4">Open Search term list</RouterLink>.
        </p>
      </div>
      <div :class="ROW">
        <h3 class="m-0 text-base font-semibold">Website scans</h3>
        <p class="m-0 text-ink-2" data-part="scan-queue">
          {{ queued > 0 ? `${waitLine(queued, 'scan', 'scans')} in the queue.` : 'No scans are waiting.' }}
          <RouterLink to="/manage/scans" class="underline underline-offset-4">Open Website scans</RouterLink>.
        </p>
        <button
          v-if="queued > 0"
          type="button"
          class="min-h-11 self-start rounded-full border-2 border-orange-ink bg-white px-4 font-semibold text-orange-ink hover:bg-orange-pale"
          :disabled="stopper.isStopping.value"
          @click="stopper.removeQueue()"
        >
          Remove the waiting scans
        </button>
      </div>
    </ManageWidget>

    <ManageWidget title="Waiting for you" :span="6" :accent="anyWaiting">
      <template v-if="anyWaiting">
        <div v-if="termsWaiting > 0" :class="ROW">
          <RouterLink to="/manage/terms/waiting" class="self-start font-semibold">{{ waitLine(termsWaiting, 'search term', 'search terms') }} for the term review</RouterLink>
          <span class="text-ink-2">Accept the ones buyers use, or let the review decide.</span>
        </div>
        <div v-if="candidatesWaiting > 0" :class="ROW">
          <RouterLink to="/manage/competitors/candidates" class="self-start font-semibold">{{ waitLine(candidatesWaiting, 'competitor candidate', 'competitor candidates') }}</RouterLink>
          <span class="text-ink-2">Add the ones that compete and dismiss the rest.</span>
        </div>
        <div v-if="keysMissing.includes('openrouter')" :class="ROW">
          <RouterLink to="/manage/keys" class="self-start font-semibold">The OpenRouter key is missing</RouterLink>
          <span class="text-ink-2">No Prompt run can start without it.</span>
        </div>
        <div v-if="keysMissing.includes('anthropic')" :class="ROW">
          <RouterLink to="/manage/keys" class="self-start font-semibold">The Anthropic key is missing</RouterLink>
          <span class="text-ink-2">The term review and the candidate checks cannot start without it.</span>
        </div>
      </template>
      <p v-else class="m-0 text-ink-2">Nothing is waiting for you.</p>
    </ManageWidget>

    <ManageWidget title="OpenRouter budget" :span="3">
      <template #action>
        <RouterLink to="/manage/prompt-test/schedule" class="font-medium">Change<span class="sr-only"> the OpenRouter budget</span></RouterLink>
      </template>
      <template v-if="openRouter">
        <div class="flex flex-wrap items-baseline justify-between gap-2"><span>OpenRouter spent</span><strong>{{ openRouter.spent }}</strong></div>
        <ProgressBar :fraction="openRouter.fraction" label="OpenRouter spent" :text="openRouter.spent" />
        <p class="m-0 flex flex-wrap gap-x-5 text-ink-2" data-part="openrouter-left">
          <span><strong class="text-black">{{ openRouter.left }}</strong> left</span>
          <span><strong class="text-black">{{ openRouter.used }}</strong> used</span>
        </p>
      </template>
    </ManageWidget>
    <ManageWidget title="Anthropic budget" :span="3">
      <template #action>
        <RouterLink to="/manage/keys" class="font-medium">Change<span class="sr-only"> the Anthropic budget</span></RouterLink>
      </template>
      <template v-if="anthropic">
        <div class="flex flex-wrap items-baseline justify-between gap-2"><span>Anthropic spent</span><strong>{{ anthropic.spent }}</strong></div>
        <ProgressBar :fraction="anthropic.fraction" label="Anthropic spent" :text="anthropic.spent" />
        <p class="m-0 flex flex-wrap gap-x-5 text-ink-2" data-part="anthropic-left">
          <span><strong class="text-black">{{ anthropic.left }}</strong> left</span>
          <span><strong class="text-black">{{ anthropic.used }}</strong> used</span>
        </p>
      </template>
    </ManageWidget>

    <ManageWidget title="Spending by kind of work" :span="6">
      <div class="overflow-x-auto">
        <table class="w-full border-collapse text-left">
          <caption class="sr-only">What each kind of work spent this month, and the key that paid</caption>
          <thead>
            <tr class="border-b-2 border-black text-sm text-muted">
              <th scope="col" class="py-1.5 pr-3 font-medium">Kind of work</th>
              <th scope="col" class="py-1.5 pr-3 font-medium">Paid from</th>
              <th scope="col" class="py-1.5 text-right font-medium">Spent this month</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="k in spending.rows" :key="`${k.kind} ${k.paidFrom}`" class="border-b border-grey">
              <th scope="row" class="py-2 pr-3 font-medium">{{ k.kind }}</th>
              <td class="py-2 pr-3">{{ k.paidFrom }}</td>
              <td class="py-2 text-right tabular-nums">{{ costLabel(k.usd) }}</td>
            </tr>
          </tbody>
          <tfoot>
            <tr class="border-t-2 border-black">
              <th scope="row" colspan="2" class="py-2 pr-3 font-semibold">All work</th>
              <td class="py-2 text-right font-semibold tabular-nums">{{ costLabel(spending.totalUsd) }}</td>
            </tr>
          </tfoot>
        </table>
      </div>
    </ManageWidget>

    <ManageWidget title="Recent work" :span="12">
      <template #action><span class="text-ink-2">The last 20 pieces of work</span></template>
      <p v-if="recent.length === 0" class="m-0 text-ink-2">No work has ended yet.</p>
      <div v-else class="overflow-x-auto">
        <table class="w-full border-collapse text-left">
          <caption class="sr-only">The last 20 pieces of work, newest first, and how each ended</caption>
          <thead>
            <tr class="border-b-2 border-black text-sm text-muted">
              <th scope="col" class="py-1.5 pr-3 font-medium">When</th>
              <th scope="col" class="py-1.5 pr-3 font-medium">Work</th>
              <th scope="col" class="py-1.5 pr-3 font-medium">How it ended</th>
              <th scope="col" class="py-1.5 pr-3 font-medium">What it did</th>
              <th scope="col" class="py-1.5 pr-3 text-right font-medium">Cost</th>
              <th scope="col" class="py-1.5 font-medium"><span class="sr-only">Open</span></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="w in recent" :key="w.id" class="border-b border-grey align-top">
              <td class="py-2 pr-3 whitespace-nowrap">{{ whenLabel(w.at) }}</td>
              <th scope="row" class="py-2 pr-3 font-medium">{{ w.work }}</th>
              <td class="py-2 pr-3"><EndedPill :ended="w.ended" /></td>
              <td class="py-2 pr-3">{{ w.did }}</td>
              <td class="py-2 pr-3 text-right tabular-nums">{{ costLabel(w.costUsd) }}</td>
              <td class="py-2"><RouterLink :to="w.to" class="font-medium">Open<span class="sr-only">{{ ` ${w.work}, ${whenLabel(w.at)}` }}</span></RouterLink></td>
            </tr>
          </tbody>
        </table>
      </div>
    </ManageWidget>
  </WidgetGrid>
</template>
