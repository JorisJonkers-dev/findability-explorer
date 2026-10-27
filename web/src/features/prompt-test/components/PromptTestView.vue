<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { PromptMode, PromptScope } from '../../../infrastructure/api/types.gen'
import { saveText } from '../../../shared/download'
import { percent } from '../../../shared/percent'
import { useCompetitors } from '../../competitors/composables/useCompetitors'
import SectionView from '../../shell/components/SectionView.vue'
import { useMe } from '../../shell/composables/useMe'
import { AI_SEARCH_TABS } from '../../shell/domain/sections'
import { usePromptAnswers, usePromptResults, useRetryRun, useStartRun } from '../composables/usePromptTest'
import { answerCost, averageCost, byKey, counts, gridHeadline, headlineNumbers, MODE_FILTERS, mostNamed, parseFilter, retryPlan, SCOPE_FILTERS, toCsv } from '../domain/grid'
import { costLabel, latestRun, progressLabel, providerName, runDate, shareLabel, statusName } from '../domain/results'
import AnswerDetail from './AnswerDetail.vue'
import PromptGrid from './PromptGrid.vue'

const route = useRoute()
const router = useRouter()
const { isAdmin } = useMe()
const { results, isLoading, error } = usePromptResults()
const { competitors } = useCompetitors()
const { isStarting, notice: startNotice, error: startError, start } = useStartRun()
const { isRetrying, notice: retryNotice, error: retryError, retry } = useRetryRun()

// The run, the filters and the picked answer live in the address, so a reload or a link shows the same.
const run = computed(() => results.value?.runs.find((r) => r.id === route.query.run) ?? latestRun(results.value))
const mode = computed(() => parseFilter<PromptMode>(route.query.mode, MODE_FILTERS))
const scope = computed(() => parseFilter<PromptScope>(route.query.scope, SCOPE_FILTERS))
const selected = computed(() => (typeof route.query.answer === 'string' ? route.query.answer : undefined))
const { answers: list, isLoading: answersLoading, error: answersError } = usePromptAnswers(
  computed(() => run.value?.id),
  computed(() => ({})),
  computed(() => run.value?.status === 'running'),
)
const answers = computed(() => list.value?.items ?? [])
const picked = computed(() => (selected.value ? byKey(answers.value).get(selected.value) : undefined))

function nameOf(id: string): string {
  return competitors.value.find((c) => c.id === id)?.name ?? id
}
const title = computed(() => gridHeadline(answers.value, nameOf))
const numbers = computed(() => headlineNumbers(answers.value))
const named = computed(() => mostNamed(answers.value))
const most = computed(() => Math.max(...named.value.map((n) => n.share), 0.01))

const anyRunning = computed(() => results.value?.runs.some((r) => r.status === 'running') ?? false)
const plan = computed(() => (run.value ? retryPlan(run.value, answers.value) : { count: 0, estimate: 0 }))
const canRetry = computed(() => isAdmin.value && !anyRunning.value && !isRetrying.value)

function retryOne(): void {
  const a = picked.value
  if (run.value && a) retry(run.value.id, { promptId: a.promptId, provider: a.provider, mode: a.mode })
}

function setQuery(change: Record<string, string | undefined>): void {
  const query = { ...route.query, ...change }
  for (const [k, v] of Object.entries(query)) if (v === undefined || v === 'all') Reflect.deleteProperty(query, k)
  void router.replace({ query })
}

function download(): void {
  if (run.value) saveText(`prompt-run-${run.value.startedAt.slice(0, 10)}.csv`, toCsv(answers.value))
}
</script>

<template>
  <SectionView section="ai-search" :title="title" :tabs="AI_SEARCH_TABS">
    <template v-if="results && run" #aside>
      <div class="flex flex-wrap items-center gap-3 stage:justify-end stage:gap-4">
        <label class="inline-flex items-center gap-2 text-sm font-medium stage:gap-3 stage:text-[24px]">
          Run
          <select
            :value="run.id"
            class="min-h-11 rounded-lg border-2 border-silver bg-white px-3 stage:min-h-16 stage:px-5 stage:text-[24px]"
            @change="setQuery({ run: ($event.target as HTMLSelectElement).value, answer: undefined })"
          >
            <option v-for="r in results.runs" :key="r.id" :value="r.id">{{ runDate(r) }} ({{ statusName(r.status).toLowerCase() }})</option>
          </select>
        </label>
        <button
          type="button"
          class="inline-flex min-h-11 items-center rounded-full border-2 border-silver bg-white px-4 text-sm font-medium hover:border-black stage:min-h-16 stage:px-7 stage:text-[24px]"
          :disabled="answers.length === 0"
          @click="download"
        >
          Download as CSV
        </button>
        <button
          v-if="isAdmin"
          type="button"
          class="inline-flex min-h-11 items-center rounded-full border-0 bg-orange px-4 text-sm font-semibold text-black hover:bg-orange-soft disabled:opacity-60 stage:min-h-16 stage:px-7 stage:text-[24px]"
          :disabled="isStarting || results.runs.some((r) => r.status === 'running')"
          @click="start"
        >
          Run now
        </button>
        <button
          v-if="isAdmin && plan.count > 0"
          type="button"
          class="inline-flex min-h-11 items-center rounded-full border-2 border-black bg-white px-4 text-sm font-semibold hover:bg-mist disabled:opacity-60 stage:min-h-16 stage:px-7 stage:text-[24px]"
          :disabled="!canRetry || run.status === 'running'"
          @click="retry(run.id)"
        >
          Ask the {{ plan.count }} missing answers again, about {{ answerCost(plan.estimate) }}
        </button>
      </div>
      <p class="m-0 text-sm text-muted stage:text-right stage:text-[24px]">
        {{ progressLabel(run) }}. {{ statusName(run.status) }}. Cost {{ costLabel(run.costUsd) }}.
      </p>
      <p v-if="startNotice" class="m-0 text-sm font-medium stage:text-[24px]" aria-live="polite">{{ startNotice }}</p>
      <p v-if="startError" role="alert" class="m-0 text-sm font-medium text-orange-ink stage:text-[24px]">{{ startError }}</p>
      <p v-if="retryNotice" class="m-0 text-sm font-medium stage:text-[24px]" aria-live="polite">{{ retryNotice }}</p>
      <p v-if="retryError" role="alert" class="m-0 text-sm font-medium text-orange-ink stage:text-[24px]">{{ retryError }}</p>
    </template>

    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the results…</p>
    <div v-else-if="!run" class="flex flex-col gap-4">
      <p class="m-0 text-ink-2 stage:text-[30px]">
        No Prompt run yet. The first starts on the schedule, or when an administrator starts one<template v-if="isAdmin">
          in <RouterLink to="/settings/prompt-test" class="underline underline-offset-4">the Prompt test settings</RouterLink></template>.
      </p>
    </div>
    <div
      v-else
      class="flex flex-col gap-12 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1.55fr)_minmax(0,1fr)] stage:gap-24"
    >
      <section aria-labelledby="grid-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-6">
        <div class="flex flex-col gap-1">
          <h2 id="grid-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Every question and answer</h2>
          <span class="text-ink-2 stage:text-[26px]">Where each answer names Tribelt. Pick a square to read the answer.</span>
        </div>
        <p v-if="answersError" role="alert" class="m-0 text-orange-ink">{{ answersError }}</p>
        <p v-else-if="answersLoading" class="m-0 text-muted">Loading the answers…</p>
        <PromptGrid
          v-else
          :run="run"
          :answers="answers"
          :mode="mode"
          :scope="scope"
          :selected="selected"
          @mode="(m) => setQuery({ mode: m })"
          @scope="(s) => setQuery({ scope: s })"
          @select="(key) => setQuery({ answer: key })"
        />
      </section>

      <aside aria-label="Results and the selected answer" class="flex flex-col gap-6 stage:min-h-0 stage:gap-10 stage:overflow-y-auto">
        <dl class="m-0 grid grid-cols-3 gap-3 stage:gap-6">
          <div class="flex flex-col gap-1">
            <dt class="order-2 text-sm text-ink-2 stage:text-[24px]">of answers name Tribelt</dt>
            <dd class="order-1 m-0 font-display text-3xl font-semibold text-orange-ink stage:text-[88px]">{{ shareLabel(numbers.named, numbers.counted) }}</dd>
          </div>
          <div class="flex flex-col gap-1">
            <dt class="order-2 text-sm text-ink-2 stage:text-[24px]">when the buyer asks for Twente</dt>
            <dd class="order-1 m-0 font-display text-3xl font-semibold stage:text-[88px]">{{ shareLabel(numbers.twente, numbers.twenteCounted) }}</dd>
          </div>
          <div class="flex flex-col gap-1">
            <dt class="order-2 text-sm text-ink-2 stage:text-[24px]">of {{ numbers.web }} web answers cite tribelt.nl</dt>
            <dd class="order-1 m-0 font-display text-3xl font-semibold stage:text-[88px]">{{ numbers.cited }}</dd>
          </div>
        </dl>
        <figure class="m-0 flex flex-col gap-2 stage:gap-3">
          <figcaption class="flex flex-col">
            <span class="font-semibold stage:text-[32px]">Who the answers name</span>
            <span class="text-sm text-muted stage:text-[22px]">Share of the {{ numbers.counted }} answers that name each company</span>
          </figcaption>
          <div
            v-for="n in named"
            :key="n.id"
            class="grid grid-cols-[minmax(0,9rem)_minmax(0,1fr)_3rem] items-center gap-3 stage:grid-cols-[300px_minmax(0,1fr)_80px] stage:gap-4"
          >
            <span class="truncate text-sm stage:text-[24px]" :class="n.tribelt ? 'font-bold text-orange-ink' : ''">{{ nameOf(n.id) }}</span>
            <span aria-hidden="true" class="relative h-3 bg-white stage:h-5">
              <span class="absolute inset-y-0 left-0" :class="n.tribelt ? 'bg-orange' : 'bg-blue-grey'" :style="{ width: percent(n.share / most) }"></span>
            </span>
            <span class="text-right text-sm font-semibold stage:text-[24px]">{{ Math.round(n.share * 100) }}%</span>
          </div>
        </figure>
        <AnswerDetail :answer="picked" :competitors="competitors">
          <button
            v-if="isAdmin && picked && !counts(picked) && picked.failure !== undefined"
            type="button"
            class="inline-flex min-h-11 items-center self-start rounded-full border-0 bg-orange px-4 text-sm font-semibold text-black hover:bg-orange-soft disabled:opacity-60 stage:min-h-16 stage:px-7 stage:text-[24px]"
            :disabled="!canRetry || run.status === 'running'"
            @click="retryOne"
          >
            Ask {{ providerName(picked.provider) }} again, about {{ answerCost(averageCost(answers, picked.provider)) }}
          </button>
        </AnswerDetail>
      </aside>
    </div>
  </SectionView>
</template>
