<script setup lang="ts">
import { computed, ref, useTemplateRef } from 'vue'
import type { LlmProvider } from '../../../infrastructure/api/types.gen'
import { useAddCredential } from '../composables/useLlmSettings'
import { checkNewKey, MAX_KEY_LENGTH, MAX_LABEL_LENGTH, type NewKeyCheck } from '../domain/credentials'
import { PROVIDER_OPTIONS } from '../domain/providers'

const provider = ref<LlmProvider>('openrouter')
const label = ref('')
const apiKey = ref('')
const check = ref<NewKeyCheck>(null)
const added = ref<string | null>(null)
const labelInput = useTemplateRef<HTMLInputElement>('labelInput')
const keyInput = useTemplateRef<HTMLInputElement>('keyInput')
const { isAdding, error, add } = useAddCredential()

const fieldError = (field: 'label' | 'apiKey') => computed(() => (check.value?.field === field ? check.value.message : null))
const labelError = fieldError('label')
const keyError = fieldError('apiKey')

function submit(): void {
  added.value = null
  check.value = checkNewKey(label.value, apiKey.value)
  if (check.value) {
    ;(check.value.field === 'label' ? labelInput : keyInput).value?.focus()
    return
  }
  const name = label.value.trim()
  add({ provider: provider.value, label: name, apiKey: apiKey.value.trim() }, () => {
    // The key is never kept once it is stored.
    apiKey.value = ''
    label.value = ''
    added.value = `Added "${name}". Choose a model for it above.`
  })
}
</script>

<template>
  <section aria-labelledby="add-title" class="flex flex-col gap-5">
    <h2 id="add-title" class="m-0 font-display text-3xl font-semibold tracking-[-0.01em]">Add a key</h2>
    <form novalidate aria-labelledby="add-title" class="flex flex-col gap-4 rounded-2xl border-2 border-grey p-5 sm:p-6" @submit.prevent="submit">
      <div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
        <label class="flex flex-col gap-2 font-medium">
          Provider
          <select v-model="provider" class="h-12 rounded-xl border-2 border-grey bg-white px-3 text-base">
            <option v-for="option in PROVIDER_OPTIONS" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <label class="flex flex-col gap-2 font-medium">
          Name
          <input
            ref="labelInput"
            v-model="label"
            type="text"
            :maxlength="MAX_LABEL_LENGTH"
            autocomplete="off"
            placeholder="Course budget key"
            :aria-invalid="labelError !== null"
            :aria-describedby="labelError !== null ? 'add-label-error' : undefined"
            class="h-12 rounded-xl border-2 bg-white px-3 text-base"
            :class="labelError !== null ? 'border-orange-ink' : 'border-grey'"
          />
        </label>
      </div>
      <p v-if="labelError !== null" id="add-label-error" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ labelError }}</p>
      <label class="flex flex-col gap-2 font-medium">
        API key
        <input
          ref="keyInput"
          v-model="apiKey"
          type="password"
          autocomplete="off"
          spellcheck="false"
          :maxlength="MAX_KEY_LENGTH"
          :aria-invalid="keyError !== null"
          :aria-describedby="keyError !== null ? 'add-key-error' : 'add-key-hint'"
          class="h-12 rounded-xl border-2 bg-white px-3 font-mono text-base"
          :class="keyError !== null ? 'border-orange-ink' : 'border-grey'"
        />
      </label>
      <p v-if="keyError !== null" id="add-key-error" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ keyError }}</p>
      <p v-else id="add-key-hint" class="m-0 text-sm text-muted">The provider checks the key before it is stored.</p>
      <div class="flex flex-wrap items-center gap-4">
        <button
          type="submit"
          :disabled="isAdding"
          class="min-h-12 rounded-full border-0 bg-black px-7 font-semibold text-white hover:bg-ink-2 disabled:opacity-70"
        >
          {{ isAdding ? 'Checking the key…' : 'Add key' }}
        </button>
        <p class="m-0 font-medium" aria-live="polite">{{ added }}</p>
      </div>
      <p v-if="error" role="alert" class="m-0 font-medium text-orange-ink">{{ error }}</p>
    </form>
  </section>
</template>
