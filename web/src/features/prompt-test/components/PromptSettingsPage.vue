<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import type { PromptCadence } from '../../../infrastructure/api/types.gen'
import { useLlmSettings } from '../../settings/composables/useLlmSettings'
import { credentialName } from '../../settings/domain/credentials'
import { useStartRun, usePromptResults, usePromptSettings, useUpdatePromptSettings } from '../composables/usePromptTest'
import { answerCost } from '../domain/grid'
import { costLabel, progressLabel, providerName, runDate, statusName } from '../domain/results'
import { addProvider, basisLine, modelLine, momentLabel, openRouterKeys, parseBudget, promptGroups } from '../domain/settings'
import SettingsShell from '../../shell/components/SettingsShell.vue'

const { settings, isLoading, forbidden, error } = usePromptSettings()
const llm = useLlmSettings()
const { results } = usePromptResults()
const { isSaving, notice, error: saveError, save } = useUpdatePromptSettings()
const { isStarting, notice: startNotice, error: startError, start } = useStartRun()

const keys = computed(() => openRouterKeys(llm.settings.value?.credentials ?? []))
const credentialId = ref('')
const providers = ref<string[]>([])
const newProvider = ref('')
const budget = ref('')
const cadence = ref<PromptCadence>('monthly')
const factSheet = ref('')
const formError = ref<string | null>(null)

// The form starts from the settings, and again after each save.
watch(
  settings,
  (s) => {
    if (!s) return
    credentialId.value = s.credentialId ?? ''
    providers.value = [...s.providers]
    budget.value = String(s.monthlyBudgetUsd)
    cadence.value = s.cadence
    factSheet.value = s.factSheet.text
  },
  { immediate: true },
)

const groups = computed(() => (settings.value ? promptGroups(settings.value.promptSet.prompts) : []))
const runs = computed(() => results.value?.runs ?? [])

function add(): void {
  const next = addProvider(providers.value, newProvider.value)
  formError.value = next.problem
  providers.value = next.list
  if (next.problem === null) newProvider.value = ''
}

function saveRuns(): void {
  const amount = parseBudget(budget.value)
  const list = providers.value
  if (amount === null) {
    formError.value = 'The budget is an amount in dollars above $0 and at most $1,000.'
    return
  }
  formError.value = null
  save(
    { ...(credentialId.value === '' ? {} : { credentialId: credentialId.value }), providers: list, monthlyBudgetUsd: amount, cadence: cadence.value },
    'The settings are saved.',
  )
}

function saveFactSheet(): void {
  save({ factSheet: factSheet.value }, 'The fact sheet is saved as a new version.')
}
</script>

<template>
  <SettingsShell title="Prompt test">
    <template #lead>
      <p class="m-0 max-w-[62ch] text-lg text-ink-2">
        What every Prompt run asks, of which providers, and what it costs. The questions stay the same, so each run compares with
        the ones before.
      </p>
      <p class="m-0"><RouterLink to="/ai-search/prompt-test" class="font-medium">The results</RouterLink></p>
    </template>

    <div v-if="forbidden" role="alert" class="flex flex-col gap-2 rounded-2xl bg-mist p-6">
      <p class="m-0 text-xl font-semibold">Only administrators can change these settings.</p>
    </div>
    <p v-else-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the settings…</p>
    <template v-else-if="settings">
      <div v-if="!settings.keyWorks" role="alert" class="flex flex-col gap-1 rounded-2xl border-2 border-orange-ink bg-orange-pale p-5">
        <p class="m-0 font-semibold">The OpenRouter key does not work, so no Prompt run can start.</p>
        <p class="m-0">Choose a working key below, or add one in <RouterLink to="/settings" class="underline underline-offset-4">Language models</RouterLink>.</p>
      </div>

      <section aria-labelledby="price-title" class="flex flex-col gap-4">
        <h2 id="price-title" class="m-0 font-display text-3xl font-semibold tracking-[-0.01em]">What runs cost</h2>
        <p class="m-0 text-ink-2">{{ basisLine(settings.prices) }}</p>
        <div class="overflow-x-auto">
          <table class="w-full border-collapse text-sm">
            <caption class="sr-only">The cost of one answer per provider, with and without web search, and of a run</caption>
            <thead>
              <tr class="border-b-2 border-black text-left text-muted">
                <th scope="col" class="py-1.5 font-medium">Provider and model</th>
                <th scope="col" class="px-2 py-1.5 text-right font-medium">One answer</th>
                <th scope="col" class="px-2 py-1.5 text-right font-medium">With web search</th>
                <th scope="col" class="px-2 py-1.5 text-right font-medium">A run</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in settings.prices.providers" :key="p.provider" class="border-b border-grey">
                <th scope="row" class="py-2 text-left font-medium">
                  {{ providerName(p.provider) }}<span class="block text-xs font-normal text-muted">{{ p.fallback ? `${p.model}, then ${p.fallback}` : p.model }}</span>
                </th>
                <td class="px-2 py-2 text-right">{{ answerCost(p.knowledgeUsd) }}</td>
                <td class="px-2 py-2 text-right">{{ answerCost(p.webUsd) }}</td>
                <td class="px-2 py-2 text-right font-semibold">{{ costLabel(p.runUsd) }}</td>
              </tr>
              <tr class="border-b border-grey">
                <th scope="row" class="py-2 text-left font-medium">Reading every answer<span class="block text-xs font-normal text-muted">{{ settings.extractorModel }}</span></th>
                <td class="px-2 py-2 text-right">{{ answerCost(settings.prices.readingUsd) }}</td>
                <td class="px-2 py-2"></td>
                <td class="px-2 py-2"></td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="border-b-2 border-black">
                <th scope="row" colspan="3" class="py-2 text-left">One run</th>
                <td class="px-2 py-2 text-right font-semibold">{{ costLabel(settings.prices.runUsd) }}</td>
              </tr>
              <tr>
                <th scope="row" colspan="3" class="py-2 text-left">
                  A month of {{ settings.cadence === 'weekly' ? 'weekly' : 'monthly' }} runs, against the budget of {{ costLabel(settings.prices.budgetUsd) }}
                </th>
                <td class="px-2 py-2 text-right font-semibold" :class="settings.prices.overBudget ? 'text-orange-ink' : ''">{{ costLabel(settings.prices.monthUsd) }}</td>
              </tr>
            </tfoot>
          </table>
        </div>
        <p v-if="settings.prices.overBudget" role="alert" class="m-0 font-semibold text-orange-ink">
          A month of runs costs more than the budget, so runs would stop before the month ends. Raise the budget or run less often.
        </p>
      </section>

      <section aria-labelledby="run-title" class="flex flex-col gap-4">
        <h2 id="run-title" class="m-0 font-display text-3xl font-semibold tracking-[-0.01em]">Runs</h2>
        <p class="m-0 text-ink-2">
          {{ costLabel(settings.spentThisMonthUsd) }} of {{ costLabel(settings.monthlyBudgetUsd) }} spent this month. The next
          scheduled run starts {{ momentLabel(settings.nextDueAt) }}.
        </p>
        <div>
          <button
            type="button"
            :disabled="isStarting"
            class="min-h-11 rounded-full border-0 bg-orange px-6 font-semibold text-black hover:bg-black hover:text-white disabled:opacity-70"
            @click="start"
          >
            {{ isStarting ? 'Starting…' : 'Run now' }}
          </button>
        </div>
        <p class="m-0 font-medium" aria-live="polite">{{ startNotice }}</p>
        <p v-if="startError" role="alert" class="m-0 font-medium text-orange-ink">{{ startError }}</p>
        <table v-if="runs.length > 0" class="w-full border-collapse text-sm">
          <caption class="pb-2 text-left text-muted">The latest runs, their tokens and cost</caption>
          <thead>
            <tr class="border-b-2 border-black text-left text-muted">
              <th scope="col" class="py-1.5 font-medium">Started</th>
              <th scope="col" class="px-2 py-1.5 font-medium">Status</th>
              <th scope="col" class="px-2 py-1.5 font-medium">Answers</th>
              <th scope="col" class="px-2 py-1.5 font-medium">Tokens in / out</th>
              <th scope="col" class="px-2 py-1.5 font-medium">Cost</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in runs" :key="r.id" class="border-b border-grey">
              <th scope="row" class="py-2 text-left font-medium">{{ runDate(r) }}<span class="block text-xs font-normal text-muted">{{ r.startedBy }}</span></th>
              <td class="px-2 py-2">{{ statusName(r.status) }}<span v-if="r.failure" class="block text-xs text-muted">{{ r.failure }}</span></td>
              <td class="px-2 py-2">{{ progressLabel(r) }}</td>
              <td class="px-2 py-2 tabular-nums">{{ r.inputTokens.toLocaleString('en-GB') }} / {{ r.outputTokens.toLocaleString('en-GB') }}</td>
              <td class="px-2 py-2 tabular-nums">{{ costLabel(r.costUsd) }}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <form aria-labelledby="choice-title" novalidate class="flex flex-col gap-4 rounded-2xl bg-mist p-5 sm:p-6" @submit.prevent="saveRuns">
        <h2 id="choice-title" class="m-0 font-display text-3xl font-semibold tracking-[-0.01em]">What runs ask with</h2>
        <label class="flex flex-col gap-1.5 font-medium">
          OpenRouter key
          <select v-model="credentialId" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base">
            <option value="" disabled>Choose a key</option>
            <option v-for="k in keys" :key="k.id" :value="k.id">{{ credentialName(k) }}</option>
          </select>
          <span v-if="keys.length === 0" class="text-sm font-normal text-ink-2">
            Add an OpenRouter key in <RouterLink to="/settings" class="underline underline-offset-4">the language-model settings</RouterLink> first.
          </span>
        </label>
        <fieldset class="m-0 flex flex-col gap-2 border-0 p-0">
          <legend class="mb-1.5 font-medium">Providers to ask</legend>
          <ul aria-label="Providers" class="m-0 flex list-none flex-col gap-1.5 p-0">
            <li v-for="p in providers" :key="p" class="flex flex-wrap items-center gap-3 rounded-xl bg-white px-3 py-2">
              <span class="min-w-32 flex-1">
                <span class="font-medium">{{ providerName(p) }}</span>
                <span class="block text-xs text-muted">{{ modelLine(settings.models, p) }}</span>
              </span>
              <button
                type="button"
                class="min-h-10 border-0 bg-transparent px-2 text-sm font-medium text-ink-2 underline underline-offset-4 disabled:opacity-50"
                :disabled="providers.length === 1"
                @click="providers = providers.filter((x) => x !== p)"
              >
                Remove<span class="sr-only"> {{ providerName(p) }}</span>
              </button>
            </li>
          </ul>
          <div class="flex flex-wrap items-end gap-2">
            <label class="flex flex-col gap-1 text-sm font-medium">
              Add a provider by its OpenRouter id
              <input
                v-model="newProvider"
                type="text"
                spellcheck="false"
                placeholder="meta-llama"
                class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base"
                @keydown.enter.prevent="add"
              />
            </label>
            <button type="button" class="min-h-11 rounded-full border-2 border-black bg-white px-5 font-semibold hover:bg-mist" @click="add">Add</button>
          </div>
          <p class="m-0 text-sm text-ink-2">Each provider's cheapest model answers, with a paid one when a free model cannot.</p>
        </fieldset>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="flex flex-col gap-1.5 font-medium">
            Monthly budget, in dollars
            <input v-model="budget" type="text" inputmode="decimal" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base" />
          </label>
          <label class="flex flex-col gap-1.5 font-medium">
            Schedule
            <select v-model="cadence" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base">
              <option value="monthly">Monthly, on the first at 06:00 UTC</option>
              <option value="weekly">Weekly, on Mondays at 06:00 UTC</option>
            </select>
          </label>
        </div>
        <p v-if="formError" role="alert" class="m-0 font-medium text-orange-ink">{{ formError }}</p>
        <div>
          <button type="submit" :disabled="isSaving" class="min-h-11 rounded-full border-0 bg-black px-6 font-semibold text-white hover:bg-ink-2 disabled:opacity-70">
            Save
          </button>
        </div>
      </form>

      <form aria-labelledby="facts-title" novalidate class="flex flex-col gap-4 rounded-2xl bg-mist p-5 sm:p-6" @submit.prevent="saveFactSheet">
        <h2 id="facts-title" class="m-0 font-display text-3xl font-semibold tracking-[-0.01em]">Fact sheet</h2>
        <p class="m-0 text-sm text-ink-2">
          Version {{ settings.factSheet.version }}<template v-if="settings.factSheet.updatedBy && settings.factSheet.updatedAt">,
            by {{ settings.factSheet.updatedBy }} on {{ momentLabel(settings.factSheet.updatedAt) }}</template>. Answers about Tribelt are judged against it.
        </p>
        <label class="flex flex-col gap-1.5 font-medium">
          What is true about Tribelt
          <textarea v-model="factSheet" rows="6" maxlength="4000" class="rounded-xl border-2 border-grey bg-white p-3 text-base"></textarea>
        </label>
        <div>
          <button type="submit" :disabled="isSaving" class="min-h-11 rounded-full border-0 bg-black px-6 font-semibold text-white hover:bg-ink-2 disabled:opacity-70">
            Save the fact sheet
          </button>
        </div>
      </form>
      <p class="m-0 font-medium" aria-live="polite">{{ notice }}</p>
      <p v-if="saveError" role="alert" class="m-0 font-medium text-orange-ink">{{ saveError }}</p>

      <section aria-labelledby="set-title" class="flex flex-col gap-4">
        <h2 id="set-title" class="m-0 font-display text-3xl font-semibold tracking-[-0.01em]">The {{ settings.promptSet.prompts.length }} buyer questions</h2>
        <p class="m-0 text-ink-2">Question set {{ settings.promptSet.version }}. Every run asks these, so each compares with the ones before.</p>
        <details v-for="g in groups" :key="g.family" class="rounded-2xl bg-mist p-4">
          <summary class="cursor-pointer font-semibold">{{ g.family }} ({{ g.prompts.length }})</summary>
          <ul class="m-0 mt-3 flex list-none flex-col gap-2 p-0 text-sm">
            <li v-for="p in g.prompts" :key="p.id" class="flex flex-col"><span class="text-xs text-muted">{{ p.scope }}</span>{{ p.text }}</li>
          </ul>
        </details>
      </section>
    </template>
  </SettingsShell>
</template>
