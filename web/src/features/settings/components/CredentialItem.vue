<script setup lang="ts">
import { computed, nextTick, ref, toRef, useTemplateRef } from 'vue'
import type { LlmCredential } from '../../../infrastructure/api/types.gen'
import { useCredentialTest, useKeyUsage, useRemoveCredential } from '../composables/useLlmSettings'
import { canRemove, credentialName, originLabel, sourceLabel, testOutcome } from '../domain/credentials'
import { usageLines } from '../domain/usage'

const props = defineProps<{ credential: LlmCredential }>()
const emit = defineEmits<{ removed: [] }>()

const { isTesting, outcome, error: testError, test } = useCredentialTest()
const { isRemoving, error: removeError, remove } = useRemoveCredential()
const confirming = ref(false)
// What the provider reports the key used, asked of it only when someone looks.
const showUsage = ref(false)
const { usage, isLoading: usageLoading, error: usageError } = useKeyUsage(toRef(() => props.credential.id), showUsage)
const removeButton = useTemplateRef<HTMLButtonElement>('removeButton')
const cancelButton = useTemplateRef<HTMLButtonElement>('cancelButton')

const titleId = computed(() => `key-${props.credential.id}`)
// The test just run, or else the last one on record.
const lastTest = computed(() => outcome.value ?? (props.credential.lastTest ? testOutcome(props.credential.lastTest) : null))

async function askToRemove(): Promise<void> {
  confirming.value = true
  await nextTick()
  // Cancel is the safe default, so it takes the focus.
  cancelButton.value?.focus()
}

async function cancel(): Promise<void> {
  confirming.value = false
  await nextTick()
  removeButton.value?.focus()
}

function confirmRemove(): void {
  remove(props.credential.id, () => {
    emit('removed')
  })
}
</script>

<template>
  <li class="flex flex-col gap-3 rounded-2xl bg-mist p-5 sm:p-6" :aria-labelledby="titleId">
    <div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
      <h3 :id="titleId" class="m-0 text-xl font-semibold">{{ credentialName(credential) }}</h3>
      <span class="rounded-full bg-white px-3 py-0.5 text-sm font-medium">{{ sourceLabel(credential) }}</span>
    </div>
    <p class="m-0 text-sm text-ink-2">
      Fingerprint <code class="font-mono text-black">{{ credential.fingerprint }}</code> ·
      {{ originLabel(credential) }}
    </p>
    <p v-if="!credential.readable" class="m-0 font-medium text-orange-ink">
      This key can no longer be read: it was sealed under another session key. Remove it and add it again.
    </p>
    <p class="m-0 flex items-start gap-2 text-sm" aria-live="polite">
      <template v-if="lastTest">
        <span aria-hidden="true" class="mt-1.5 size-2.5 flex-none rounded-full" :class="lastTest.ok ? 'bg-orange' : 'bg-ink-2'"></span>
        <span>{{ lastTest.text }}</span>
      </template>
      <span v-else class="text-muted">Not tested yet.</span>
    </p>
    <div v-if="showUsage" class="flex flex-col gap-1 rounded-xl bg-white p-4 text-sm" aria-live="polite" data-testid="key-usage">
      <p v-if="usageLoading" class="m-0 text-muted">Asking the provider…</p>
      <p v-else-if="usageError" role="alert" class="m-0 font-medium text-orange-ink">{{ usageError }}</p>
      <template v-else-if="usage">
        <p class="m-0 font-semibold">{{ usage.message }}</p>
        <p v-for="line in usageLines(usage)" :key="line" class="m-0">{{ line }}</p>
      </template>
    </div>
    <p v-if="testError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ testError }}</p>
    <p v-if="removeError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ removeError }}</p>

    <div class="flex flex-wrap items-center gap-3">
      <button
        type="button"
        :disabled="isTesting || !credential.readable"
        class="min-h-11 rounded-full border-2 border-black bg-white px-5 font-medium text-black hover:bg-black hover:text-white disabled:cursor-not-allowed disabled:border-grey disabled:text-muted disabled:hover:bg-white"
        :aria-describedby="titleId"
        @click="test(credential.id)"
      >
        {{ isTesting ? 'Testing…' : 'Test' }}
      </button>
      <button
        type="button"
        :disabled="!credential.readable"
        :aria-expanded="showUsage"
        class="min-h-11 rounded-full border-2 border-black bg-white px-5 font-medium text-black hover:bg-black hover:text-white disabled:cursor-not-allowed disabled:border-grey disabled:text-muted disabled:hover:bg-white"
        :aria-describedby="titleId"
        @click="showUsage = !showUsage"
      >
        {{ showUsage ? 'Hide usage' : 'Usage and cost' }}
      </button>
      <template v-if="canRemove(credential)">
        <button
          v-if="!confirming"
          ref="removeButton"
          type="button"
          class="min-h-11 rounded-full border-2 border-grey bg-white px-5 font-medium text-black hover:border-black"
          :aria-describedby="titleId"
          @click="askToRemove"
        >
          Remove
        </button>
        <div v-else role="group" :aria-label="`Remove ${credentialName(credential)}?`" class="flex flex-wrap items-center gap-3">
          <span class="font-medium">Remove this key? Analyses can no longer use it.</span>
          <button
            type="button"
            :disabled="isRemoving"
            class="min-h-11 rounded-full border-0 bg-black px-5 font-semibold text-white hover:bg-ink-2 disabled:opacity-70"
            @click="confirmRemove"
          >
            {{ isRemoving ? 'Removing…' : 'Remove key' }}
          </button>
          <button
            ref="cancelButton"
            type="button"
            class="min-h-11 rounded-full border-2 border-grey bg-white px-5 font-medium text-black hover:border-black"
            @click="cancel"
          >
            Cancel
          </button>
        </div>
      </template>
      <span v-else class="text-sm text-muted">Vault delivers this key; change or remove it in Vault.</span>
    </div>
  </li>
</template>
