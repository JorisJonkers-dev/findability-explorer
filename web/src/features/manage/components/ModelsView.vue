<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useLlmSettings } from '../../settings/composables/useLlmSettings'
import { credentialName } from '../../settings/domain/credentials'
import { usePromptSettings, useUpdatePromptSettings } from '../../prompt-test/composables/usePromptTest'
import { answerCost } from '../../prompt-test/domain/grid'
import { costLabel, providerName } from '../../prompt-test/domain/results'
import { addProvider, basisLine, modelLine, openRouterKeys } from '../../prompt-test/domain/settings'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import ManageWidget from './widgets/ManageWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/**
 * The providers whose cheapest model every run asks, the OpenRouter key that pays, and what each
 * costs. The model list from OpenRouter's catalogue replaces this with its own ticket.
 */
const { settings, isLoading, error } = usePromptSettings()
const llm = useLlmSettings()
const { isSaving, notice, error: saveError, save } = useUpdatePromptSettings()

const keys = computed(() => openRouterKeys(llm.settings.value?.credentials ?? []))
const credentialId = ref('')
const providers = ref<string[]>([])
const newProvider = ref('')
const formError = ref<string | null>(null)

watch(
  settings,
  (s) => {
    if (!s) return
    credentialId.value = s.credentialId ?? ''
    providers.value = [...s.providers]
  },
  { immediate: true },
)

function add(): void {
  const next = addProvider(providers.value, newProvider.value)
  formError.value = next.problem
  providers.value = next.list
  if (next.problem === null) newProvider.value = ''
}

function submit(): void {
  save({ ...(credentialId.value === '' ? {} : { credentialId: credentialId.value }), providers: providers.value }, 'The model list is saved.')
}
</script>

<template>
  <ManageLayout
    title="Models"
    purpose="The AI models that are asked the buyer questions."
    :back="{ to: '/manage/prompt-test', label: 'Back to Prompt test' }"
    :results="{ to: '/ai-responses', label: 'See the AI responses' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the models…</p>
    <WidgetGrid v-else-if="settings">
      <div v-if="!settings.keyWorks" role="alert" class="flex flex-col gap-1 rounded-3xl border-2 border-orange-ink bg-orange-pale p-5 min-[1100px]:col-span-12">
        <p class="m-0 font-semibold">The OpenRouter key does not work, so no Prompt run can start.</p>
        <p class="m-0">Choose a working key below, or add one in <RouterLink to="/manage/keys" class="underline underline-offset-4">Keys and spending</RouterLink>.</p>
      </div>

      <ManageWidget title="The model list" :span="8">
        <form id="models-form" novalidate class="flex flex-col gap-4" @submit.prevent="submit">
          <label class="flex flex-col gap-1.5 font-medium">
            OpenRouter key
            <select v-model="credentialId" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base">
              <option value="" disabled>Choose a key</option>
              <option v-for="k in keys" :key="k.id" :value="k.id">{{ credentialName(k) }}</option>
            </select>
            <span v-if="keys.length === 0" class="text-sm font-normal text-ink-2">
              Add an OpenRouter key in <RouterLink to="/manage/keys" class="underline underline-offset-4">Keys and spending</RouterLink> first.
            </span>
          </label>
          <fieldset class="m-0 flex flex-col gap-2 border-0 p-0">
            <legend class="mb-1.5 font-medium">Providers to ask, each with its cheapest model</legend>
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
                  Remove<span class="sr-only">{{ ' ' + providerName(p) }}</span>
                </button>
              </li>
            </ul>
            <div class="flex flex-wrap items-end gap-2">
              <label class="flex min-w-0 flex-1 flex-col gap-1 text-sm font-medium">
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
          </fieldset>
          <p v-if="formError" role="alert" class="m-0 font-medium text-orange-ink">{{ formError }}</p>
        </form>
      </ManageWidget>
      <ManageWidget title="Save the model list" :span="4">
        <p class="m-0 text-ink-2">A run of {{ settings.promptSet.prompts.length }} questions costs about {{ costLabel(settings.prices.runUsd) }} with the saved list.</p>
        <template #foot>
          <button type="submit" form="models-form" :disabled="isSaving" class="min-h-11 rounded-full border-0 bg-black px-6 font-semibold text-white hover:bg-ink-2 disabled:opacity-60">
            Save the model list
          </button>
          <p class="m-0 w-full font-medium" aria-live="polite">{{ notice }}</p>
          <p v-if="saveError" role="alert" class="m-0 w-full font-medium text-orange-ink">{{ saveError }}</p>
        </template>
      </ManageWidget>

      <ManageWidget title="What runs cost">
        <p class="m-0 text-ink-2">{{ basisLine(settings.prices) }}</p>
        <div class="overflow-x-auto" tabindex="0" role="region" aria-label="The cost table">
          <table class="w-full border-collapse text-sm">
            <caption class="sr-only">The cost of one response per provider, with and without web search, and of a run</caption>
            <thead>
              <tr class="border-b-2 border-black text-left text-muted">
                <th scope="col" class="py-1.5 font-medium">Provider and model</th>
                <th scope="col" class="px-2 py-1.5 text-right font-medium">One response</th>
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
                <th scope="row" class="py-2 text-left font-medium">Reading every response<span class="block text-xs font-normal text-muted">{{ settings.extractorModel }}</span></th>
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
            </tfoot>
          </table>
        </div>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
