<script setup lang="ts">
import { vScrollHint } from '../../../shared/scrollHint'
import { computed, ref, watch } from 'vue'
import type { LlmSettings } from '../../../infrastructure/api/types.gen'
import { useActiveModel, useLlmModels } from '../composables/useLlmSettings'
import { activeState, choosableCredentials, credentialName } from '../domain/credentials'
import { filterModels, modelLabel } from '../domain/models'

const props = defineProps<{ settings: LlmSettings }>()

const state = computed(() => activeState(props.settings))
const keys = computed(() => choosableCredentials(props.settings.credentials))
const activeKey = () => {
  const id = props.settings.active?.credentialId ?? ''
  return keys.value.some((k) => k.id === id) ? id : ''
}

const credentialId = ref(activeKey())
const model = ref(props.settings.active?.model ?? '')
const query = ref('')
const { models, isLoading, error: modelsError } = useLlmModels(credentialId)
const { isSaving, notice, error: saveError, save, turnOff } = useActiveModel()
const matches = computed(() => filterModels(models.value, query.value))

// Another key has other models: start its list afresh, on the active model when it is that key's.
watch(credentialId, (id) => {
  query.value = ''
  model.value = id === props.settings.active?.credentialId ? props.settings.active.model : ''
})

function submit(): void {
  if (credentialId.value === '' || model.value === '') return
  save({ credentialId: credentialId.value, model: model.value })
}
</script>

<template>
  <section aria-labelledby="model-title" class="flex flex-col gap-5">
    <h2 id="model-title" class="m-0 font-display text-3xl font-semibold tracking-[-0.01em]">Model for product matching</h2>

    <div class="flex flex-col gap-3 rounded-2xl bg-mist p-5 sm:p-6">
      <span class="text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase">In use now</span>
      <template v-if="state.kind === 'active'">
        <p class="m-0 text-lg">
          <strong class="font-semibold">{{ state.model }}</strong> with {{ state.credential }}<span v-if="state.isDefault">, the default</span>.
        </p>
        <p v-if="!state.working" class="m-0 font-medium text-orange-ink">
          Its key can no longer be read, so scans run without product matching until another key is chosen.
        </p>
        <div>
          <button
            type="button"
            :disabled="isSaving"
            class="min-h-11 rounded-full border-2 border-black bg-white px-5 font-medium text-black hover:bg-black hover:text-white disabled:opacity-70"
            @click="turnOff"
          >
            Turn matching off
          </button>
        </div>
      </template>
      <p v-else-if="state.kind === 'off'" class="m-0 text-lg">Product matching is off. Scans run without it.</p>
      <p v-else class="m-0 text-lg">No model is chosen, so scans run without product matching.</p>
    </div>

    <form aria-labelledby="model-title" class="flex flex-col gap-4" @submit.prevent="submit">
      <p v-if="keys.length === 0" class="m-0 text-ink-2">Add a key first: there is no key a model could use.</p>
      <template v-else>
        <label class="flex max-w-xl flex-col gap-2 font-medium">
          API key
          <select v-model="credentialId" class="h-12 rounded-xl border-2 border-grey bg-white px-3 text-base">
            <option value="" disabled>Choose a key</option>
            <option v-for="key in keys" :key="key.id" :value="key.id">{{ credentialName(key) }}</option>
          </select>
        </label>

        <template v-if="credentialId !== ''">
          <p v-if="isLoading" class="m-0 text-muted">Asking the provider for its models…</p>
          <p v-else-if="modelsError" role="alert" class="m-0 font-medium text-orange-ink">{{ modelsError }}</p>
          <template v-else>
            <label class="flex max-w-xl flex-col gap-2 font-medium">
              Find a model
              <input
                v-model="query"
                type="search"
                autocomplete="off"
                spellcheck="false"
                placeholder="haiku, mistral small…"
                aria-describedby="model-count"
                class="h-12 rounded-xl border-2 border-grey bg-white px-3 text-base"
              />
            </label>
            <p id="model-count" class="m-0 text-sm text-muted" aria-live="polite">
              <template v-if="matches.total === 0">No model matches.</template>
              <template v-else-if="matches.hidden > 0">
                Showing {{ matches.shown.length }} of {{ matches.total }} models: {{ matches.hidden }} more, refine the search.
              </template>
              <template v-else>{{ matches.total }} {{ matches.total === 1 ? 'model' : 'models' }}.</template>
            </p>
            <fieldset v-if="matches.shown.length > 0" v-scroll-hint class="scroll-hint m-0 max-h-96 overflow-y-auto rounded-2xl border-2 border-grey p-2">
              <legend class="sr-only">Models</legend>
              <label
                v-for="option in matches.shown"
                :key="option.id"
                class="flex min-h-11 items-center gap-3 rounded-xl px-3 py-2 hover:bg-mist"
                :class="model === option.id ? 'bg-mist font-semibold' : ''"
              >
                <input v-model="model" type="radio" name="model" :value="option.id" class="size-5 flex-none accent-black" />
                <span class="min-w-0 break-words">{{ modelLabel(option) }}</span>
              </label>
            </fieldset>
            <p v-if="model !== ''" class="m-0 text-sm text-ink-2">Chosen: <span class="font-semibold text-black">{{ model }}</span></p>
          </template>
        </template>

        <div class="flex flex-wrap items-center gap-4">
          <button
            type="submit"
            :disabled="isSaving || credentialId === '' || model === ''"
            class="min-h-12 rounded-full border-0 bg-black px-7 font-semibold text-white hover:bg-ink-2 disabled:cursor-not-allowed disabled:bg-silver disabled:text-black"
          >
            {{ isSaving ? 'Saving…' : 'Save' }}
          </button>
        </div>
      </template>
      <p class="m-0 font-medium" aria-live="polite">{{ notice }}</p>
      <p v-if="saveError" role="alert" class="m-0 font-medium text-orange-ink">{{ saveError }}</p>
    </form>
  </section>
</template>
