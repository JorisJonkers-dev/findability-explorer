<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useLlmSettings } from '../../settings/composables/useLlmSettings'
import { credentialName } from '../../settings/domain/credentials'
import { usePromptSettings, useUpdatePromptSettings } from '../../prompt-test/composables/usePromptTest'
import { costLabel } from '../../prompt-test/domain/results'
import { openRouterKeys } from '../../prompt-test/domain/settings'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useModelCatalogue } from '../composables/useModelCatalogue'
import {
  addModel,
  catalogueIntro,
  catalogueRows,
  type CatalogueRow,
  filterCatalogue,
  listRows,
  type ListRow,
  MAX_MODELS,
  providerOptions,
  removeModel,
  saveLine,
  savedList,
  sameList,
} from '../domain/models'
import type { Column } from '../domain/table'
import DataTable from './widgets/DataTable.vue'
import { FIELD, LABEL, PRIMARY, ROW_BUTTON, SECONDARY } from './widgets/look'
import ManageWidget from './widgets/ManageWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/**
 * The Model list: the OpenRouter models every run asks, each its own column, and the key that pays;
 * Save says what the changes cost and that the charts mark the date; Add a model lists every text
 * model on OpenRouter, the recommended five first.
 */
const { settings, isLoading, error } = usePromptSettings()
const llm = useLlmSettings()
const { isSaving, notice, error: saveError, save } = useUpdatePromptSettings()
const keyWorks = computed(() => settings.value?.keyWorks ?? false)
const catalogue = useModelCatalogue(keyWorks)

const keys = computed(() => openRouterKeys(llm.settings.value?.credentials ?? []))
const credentialId = ref('')
const draft = ref<string[]>([])
const saved = computed(() => (settings.value ? savedList(settings.value) : []))

watch(
  settings,
  (s) => {
    if (!s) return
    credentialId.value = s.credentialId ?? ''
    draft.value = savedList(s)
  },
  { immediate: true },
)

const questions = computed(() => settings.value?.promptSet.prompts.length ?? 0)
const reading = computed(() => settings.value?.prices.readingUsd ?? 0)
const changed = computed(() => !sameList(saved.value, draft.value) || credentialId.value !== (settings.value?.credentialId ?? ''))
const line = computed(() => saveLine(saved.value, draft.value, catalogue.models.value, questions.value, reading.value))
const rows = computed(() => listRows(draft.value, catalogue.models.value, questions.value))

const provider = ref('all')
const freeOnly = ref(false)
const providers = computed(() => providerOptions(catalogue.models.value))
const offered = computed(() => catalogueRows(filterCatalogue(catalogue.models.value, provider.value, freeOnly.value), draft.value, questions.value))
const full = computed(() => draft.value.length >= MAX_MODELS)

function submit(): void {
  save({ ...(credentialId.value === '' ? {} : { credentialId: credentialId.value }), models: draft.value }, 'The model list is saved. Charts will mark a break where it changed.')
}
function undo(): void {
  draft.value = [...saved.value]
  credentialId.value = settings.value?.credentialId ?? ''
}

const LIST_COLUMNS: readonly Column<ListRow>[] = [
  { key: 'provider', label: 'Provider', value: (r) => r.provider },
  { key: 'name', label: 'Model', value: (r) => r.name },
  { key: 'cost', label: 'Cost per run', value: (r) => r.runUsd, num: true },
  { key: 'free', label: 'Free or not', value: (r) => (r.free ? 'Free to ask' : 'Paid') },
  { key: 'remove', label: 'Change' },
]
const catalogueColumns = computed<readonly Column<CatalogueRow>[]>(() => [
  { key: 'provider', label: 'Provider', value: (r) => r.provider },
  { key: 'name', label: 'Model', value: (r) => r.name },
  { key: 'listed', label: 'On OpenRouter since', value: (r) => r.listedAt, wide: true },
  { key: 'cost', label: `Cost per run of ${String(questions.value)} questions`, value: (r) => r.runUsd, num: true },
  { key: 'add', label: 'Add' },
])
const PILL = 'inline-flex rounded-full px-2 py-0.5 text-xs font-semibold whitespace-nowrap'
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
      <div v-if="!settings.keyWorks" role="alert" class="flex basis-full flex-col gap-1 rounded-3xl border-2 border-orange-ink bg-orange-pale p-5">
        <p class="m-0 font-semibold">The OpenRouter key does not work, so no Prompt run can start.</p>
        <p class="m-0">Choose a working key below, or add one in <RouterLink to="/manage/keys" class="underline underline-offset-4">Keys and spending</RouterLink>.</p>
      </div>

      <ManageWidget title="The model list" :span="8">
        <p class="m-0 text-ink-2">These models are asked in every run, {{ questions }} questions each. Each one is its own column on the AI responses page.</p>
        <label :class="LABEL" class="max-w-md">
          OpenRouter key
          <select v-model="credentialId" :class="FIELD">
            <option value="" disabled>Choose a key</option>
            <option v-for="k in keys" :key="k.id" :value="k.id">{{ credentialName(k) }}</option>
          </select>
          <span v-if="keys.length === 0" class="font-normal">
            Add an OpenRouter key in <RouterLink to="/manage/keys" class="underline underline-offset-4">Keys and spending</RouterLink> first.
          </span>
        </label>
        <DataTable :rows="rows" :columns="LIST_COLUMNS" :row-key="(r) => r.id" label="The model list" empty="The list is empty. Add a model below, or a run has nobody to ask.">
          <template #cell-cost="{ row }">{{ row.known ? costLabel(row.runUsd) : 'Not on OpenRouter now' }}</template>
          <template #cell-free="{ row }">
            <span :class="[PILL, row.free ? 'bg-orange-pale text-orange-ink' : 'bg-white text-ink-2']">{{ row.free ? 'Free to ask' : 'Paid' }}</span>
          </template>
          <template #cell-remove="{ row }">
            <button type="button" :class="ROW_BUTTON" @click="draft = removeModel(draft, row.id)">Remove<span class="sr-only">{{ ' ' + row.name }}</span></button>
          </template>
        </DataTable>
      </ManageWidget>

      <ManageWidget title="Save the model list" :span="4" :accent="changed">
        <p class="m-0 text-ink-2">{{ line }}</p>
        <template #foot>
          <button type="button" :class="PRIMARY" :disabled="!changed || draft.length === 0 || isSaving" @click="submit">Save the model list</button>
          <button type="button" :class="SECONDARY" :disabled="!changed" @click="undo">Undo the changes</button>
          <p class="m-0 w-full font-medium" aria-live="polite">{{ notice }}</p>
          <p v-if="saveError" role="alert" class="m-0 w-full font-medium text-orange-ink">{{ saveError }}</p>
        </template>
      </ManageWidget>

      <ManageWidget title="Add a model" :span="12">
        <p v-if="!settings.keyWorks" class="m-0 text-ink-2">OpenRouter’s models can be listed once the key works.</p>
        <p v-else-if="catalogue.error.value" role="alert" class="m-0 font-medium text-orange-ink">{{ catalogue.error.value }}</p>
        <p v-else-if="catalogue.isLoading.value" class="m-0 text-muted">Reading OpenRouter’s models…</p>
        <template v-else>
          <p class="m-0 text-ink-2">{{ catalogueIntro(catalogue.models.value, catalogue.readAt.value, questions) }}</p>
          <div class="flex flex-wrap items-end gap-4">
            <label :class="LABEL" class="max-w-xs">
              Provider
              <select v-model="provider" :class="FIELD">
                <option v-for="p in providers" :key="p.value" :value="p.value">{{ p.label }}</option>
              </select>
            </label>
            <label class="inline-flex min-h-12 cursor-pointer items-center gap-2 font-medium">
              <input v-model="freeOnly" type="checkbox" class="size-5 accent-black" />
              Free only
            </label>
          </div>
          <p v-if="full" class="m-0 text-ink-2">The list holds at most {{ MAX_MODELS }} models. Remove one to add another.</p>
          <p v-if="offered.length === 0" class="m-0 text-ink-2">No model matches these filters.</p>
          <DataTable v-else :rows="offered" :columns="catalogueColumns" :row-key="(r) => r.id" label="OpenRouter’s models" :search-text="(r) => r.search">
            <template #cell-name="{ row }">
              {{ row.name }}<span v-if="row.recommended" :class="PILL" class="ml-2 bg-orange-pale text-orange-ink">Recommended</span>
            </template>
            <template #cell-listed="{ row }">{{ row.listed }}</template>
            <template #cell-cost="{ row }">
              {{ costLabel(row.runUsd) }}<span v-if="row.free" class="block text-xs text-muted">Free, apart from web search</span>
            </template>
            <template #cell-add="{ row }">
              <span v-if="row.onList" class="text-sm text-muted">On the list</span>
              <button v-else type="button" :class="ROW_BUTTON" :disabled="full" @click="draft = addModel(draft, row.id)">
                Add<span class="sr-only">{{ ' ' + row.name }}</span>
              </button>
            </template>
          </DataTable>
        </template>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
