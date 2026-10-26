<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import ProgressBar from '../../../shared/ui/ProgressBar.vue'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import SectionView from '../../shell/components/SectionView.vue'
import { useMe } from '../../shell/composables/useMe'
import { usePromptResults } from '../composables/usePromptTest'
import { costLabel, latestRun, modeName, progressLabel, providerName, ranking, runDate, sharesLine, statusName, tally } from '../domain/results'
import AnswerBrowser from './AnswerBrowser.vue'
import ShareTable from './ShareTable.vue'
import TrendTable from './TrendTable.vue'

const { results, isLoading, error } = usePromptResults()
const { isAdmin } = useMe()
const run = computed(() => latestRun(results.value))
const running = computed(() => results.value?.runs.find((r) => r.status === 'running'))
const competitors = computed(() => (results.value ? ranking(results.value.mentions).slice(0, 15) : []))
</script>

<template>
  <SectionView
    section="ai-search"
    title="Do AI assistants name Tribelt?"
    :source="['Prompt runs: live, from this app, through OpenRouter', 'The October pilot by hand, as the first point']"
  >
    <div class="flex flex-col gap-12 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)] stage:gap-x-24 stage:gap-y-10 stage:overflow-y-auto">
      <p class="m-0 max-w-4xl text-ink-2 stage:col-span-2 stage:max-w-none stage:text-[30px]">
        90 buyer questions in general terms, about each belt family, for a supplier, an application and a problem, in six
        scopes, asked of the cheapest model of each large provider, from its own knowledge and with web search. One fixed
        model reads every answer.
        <RouterLink to="/ai-search" class="font-medium">Back to AI search</RouterLink>
      </p>
      <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
      <p v-else-if="isLoading" class="m-0 text-muted">Loading the results…</p>
      <template v-else-if="results">
        <section aria-labelledby="run-title" class="flex flex-col gap-5 stage:gap-8">
          <SectionHeading id="run-title">{{ run ? `The run of ${runDate(run)}` : 'No Prompt run yet' }}</SectionHeading>
          <div v-if="running" class="flex flex-col gap-2" data-testid="run-progress">
            <ProgressBar
              :fraction="(running.answered + running.failed) / Math.max(running.tasks, 1)"
              label="Prompt run"
              :text="`${progressLabel(running)} · ${costLabel(running.costUsd)} so far`"
            />
            <ul aria-label="Per model" class="m-0 flex list-none flex-wrap gap-x-5 gap-y-1 p-0 text-sm text-ink-2 stage:text-[24px]">
              <li v-for="p in running.byProvider" :key="p.provider">{{ providerName(p.provider) }}: {{ p.answered }}</li>
            </ul>
          </div>
          <p v-if="!run" class="m-0 text-ink-2">
            The first run starts on the schedule, or when an administrator starts one in
            <RouterLink v-if="isAdmin" to="/settings/prompt-test">the Prompt test's settings</RouterLink><template v-else>Settings</template>.
          </p>
          <template v-else>
            <p class="m-0 text-sm text-ink-2 stage:text-[26px]">
              {{ statusName(run.status) }} · {{ progressLabel(run) }} · {{ costLabel(run.costUsd) }} · prompt set {{ run.promptSetVersion }},
              fact sheet {{ run.factSheetVersion }}
            </p>
            <dl class="m-0 grid grid-cols-1 gap-4 sm:grid-cols-2 stage:gap-8">
              <div v-for="mode in (['knowledge', 'web'] as const)" :key="mode" class="flex flex-col gap-1 rounded-2xl bg-mist p-4 stage:p-8">
                <dt class="text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[24px]">{{ modeName(mode) }}</dt>
                <dd class="m-0 font-display text-2xl font-semibold stage:text-[56px]">{{ sharesLine(tally(run.cells, { mode })) }}</dd>
              </div>
            </dl>
            <ShareTable :run="run" />
          </template>
        </section>

        <section aria-labelledby="trend-title" class="flex flex-col gap-5 stage:gap-8">
          <SectionHeading id="trend-title">Across runs</SectionHeading>
          <TrendTable :runs="results.runs" />
        </section>

        <section v-if="run" aria-labelledby="ranking-title" class="flex flex-col gap-4 stage:gap-6">
          <SectionHeading id="ranking-title">Who the answers name instead</SectionHeading>
          <p v-if="competitors.length === 0" class="m-0 text-ink-2">The answers name no competitor on the list.</p>
          <ol v-else aria-label="Competitors named" class="m-0 flex flex-col gap-1 pl-6 stage:text-[28px]">
            <li v-for="m in competitors" :key="m.competitorId">
              <RouterLink :to="{ path: '/competitors', query: { company: m.competitorId } }">{{ m.name }}</RouterLink>
              <span class="text-ink-2"> · {{ m.answers }} answers, first in {{ m.first }}</span>
            </li>
          </ol>
          <RouterLink v-if="isAdmin" to="/settings/candidates" class="self-start text-sm font-medium stage:text-[24px]">Companies named that are not on the list</RouterLink>
        </section>

        <section aria-labelledby="claims-title" class="flex flex-col gap-4 stage:gap-6">
          <SectionHeading id="claims-title">What they get wrong about Tribelt</SectionHeading>
          <p v-if="results.wrongClaims.length === 0" class="m-0 text-ink-2">No answer said anything about Tribelt that the fact sheet contradicts.</p>
          <ul v-else aria-label="Wrong claims" class="m-0 flex list-none flex-col gap-2 p-0 stage:text-[26px]">
            <li v-for="c in results.wrongClaims.slice(0, 20)" :key="`${c.answerId}-${c.claim}`" class="flex flex-col">
              <span class="font-medium">“{{ c.claim }}”</span>
              <span class="text-xs text-muted stage:text-[20px]">{{ c.model }} · {{ modeName(c.mode) }} · {{ runDate({ startedAt: c.at }) }}</span>
            </li>
          </ul>
        </section>

        <section v-if="run" aria-labelledby="answers-title" class="flex flex-col gap-5 stage:col-span-2 stage:gap-8">
          <SectionHeading id="answers-title">The answers</SectionHeading>
          <AnswerBrowser :key="run.id" :run="run" />
        </section>
      </template>
    </div>
  </SectionView>
</template>
