<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { usePromptResults } from '../composables/usePromptTest'
import { latestRun, modeName, PILOT, ranking, runDate, shareLabel, tally } from '../domain/results'

/** The Prompt test on the AI search poster: the latest run's shares, or the pilot until there is one. */
const { results } = usePromptResults()
const run = computed(() => latestRun(results.value))
const named = computed(() => (results.value ? ranking(results.value.mentions).slice(0, 6).map((m) => m.name) : []))
</script>

<template>
  <div class="flex flex-col gap-4 stage:gap-7" data-testid="prompt-summary">
    <template v-if="run">
      <p class="m-0 text-ink-2 stage:text-[30px]">Prompt run of {{ runDate(run) }}: {{ tally(run.cells).answers }} answers, read by one model.</p>
      <dl class="m-0 grid grid-cols-2 gap-4 stage:gap-8">
        <div v-for="mode in (['knowledge', 'web'] as const)" :key="mode" class="flex flex-col-reverse">
          <dt class="text-sm text-ink-2 stage:text-[26px]">name Tribelt, {{ modeName(mode).toLowerCase() }}</dt>
          <dd class="m-0 font-display text-4xl leading-none font-semibold stage:text-[96px]">
            {{ shareLabel(tally(run.cells, { mode }).named, tally(run.cells, { mode }).answers) }}
          </dd>
        </div>
      </dl>
    </template>
    <template v-else>
      <p class="m-0 text-ink-2 stage:text-[30px]">{{ PILOT.label }} ({{ PILOT.note }}): Tribelt named in {{ PILOT.overall.named }} of {{ PILOT.overall.answers }} searches. The monthly runs start soon.</p>
    </template>
    <div v-if="named.length > 0" class="flex flex-wrap items-center gap-2 stage:gap-3.5">
      <span class="text-sm text-muted stage:text-[28px]">Named instead:</span>
      <ul aria-label="Named instead" class="m-0 flex list-none flex-wrap gap-2 p-0 stage:gap-3.5">
        <li v-for="name in named" :key="name" class="rounded-full bg-mist px-3 py-1 text-sm font-medium stage:px-6 stage:py-2 stage:text-[28px]">{{ name }}</li>
      </ul>
    </div>
    <RouterLink to="/ai-search/prompt-test" class="self-start font-medium stage:text-[28px]">The Prompt test: every answer and the trends</RouterLink>
  </div>
</template>
