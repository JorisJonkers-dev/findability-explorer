<script setup lang="ts">
import { computed, nextTick, ref, useTemplateRef } from 'vue'
import { RouterLink } from 'vue-router'
import type { CompetitorCandidate } from '../../../infrastructure/api/types.gen'
import { companyPath } from '../../shell/domain/sections'
import { type CandidateDraft, candidateDecisionLine, candidateDraft, candidateEvidence, checkCandidateDraft } from '../domain/candidates'

/** One company the AI answers name, with what it takes to add it to the competitors or leave it out. */
const props = defineProps<{ candidate: CompetitorCandidate; busy: boolean }>()
const emit = defineEmits<{ add: [name: string, website: string]; dismiss: []; undo: [] }>()

const titleId = computed(() => `named-${props.candidate.id}`)
const decided = computed(() => candidateDecisionLine(props.candidate))
const editing = ref(false)
const draft = ref<CandidateDraft>(candidateDraft(props.candidate))
const draftError = ref<string | null>(null)
const nameInput = useTemplateRef<HTMLInputElement>('nameInput')

async function openForm(): Promise<void> {
  draft.value = candidateDraft(props.candidate)
  draftError.value = null
  editing.value = true
  await nextTick()
  nameInput.value?.focus()
}

function submit(): void {
  const check = checkCandidateDraft(draft.value)
  if (check.kind === 'invalid') {
    draftError.value = check.message
    return
  }
  emit('add', check.name, check.website)
}
</script>

<template>
  <li class="flex flex-col gap-3 rounded-2xl bg-mist p-5 sm:p-6" :aria-labelledby="titleId">
    <h3 :id="titleId" class="m-0 text-xl font-semibold">{{ candidate.name }}</h3>
    <p class="m-0 text-sm text-ink-2">{{ candidateEvidence(candidate) }}</p>
    <p v-if="candidate.addresses.length > 0" class="m-0 text-sm">
      <span class="text-muted">Websites the answers gave </span>
      <template v-for="(a, i) in candidate.addresses" :key="a"><template v-if="i > 0">, </template><a :href="/^https?:\/\//.test(a) ? a : `https://${a}`" target="_blank" rel="noopener noreferrer">{{ a }}</a></template>
    </p>
    <p v-else class="m-0 text-sm text-muted">The answers gave no website for it.</p>
    <p v-if="decided" class="m-0 text-sm font-medium">{{ decided }}</p>

    <form v-if="editing" novalidate :aria-label="`Add ${candidate.name}`" class="flex flex-col gap-3 rounded-xl bg-white p-4" @submit.prevent="submit">
      <div class="grid gap-3 sm:grid-cols-2">
        <label class="flex flex-col gap-1.5 text-sm font-medium">
          Name
          <input ref="nameInput" v-model="draft.name" type="text" maxlength="200" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base font-normal" />
        </label>
        <label class="flex flex-col gap-1.5 text-sm font-medium">
          Website
          <input v-model="draft.website" type="url" maxlength="500" placeholder="https://www.example.com" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base font-normal" />
        </label>
      </div>
      <p v-if="draftError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ draftError }}</p>
      <div class="flex flex-wrap gap-3">
        <button type="submit" :disabled="busy" class="min-h-11 rounded-full border-0 bg-black px-5 font-semibold text-white hover:bg-ink-2 disabled:opacity-70">
          Add to the competitors
        </button>
        <button type="button" class="min-h-11 rounded-full border-2 border-grey bg-white px-5 font-medium text-black hover:border-black" @click="editing = false">
          Cancel
        </button>
      </div>
    </form>

    <div v-else class="flex flex-wrap items-center gap-3">
      <template v-if="candidate.status === 'pending'">
        <button
          type="button"
          :disabled="busy"
          :aria-describedby="titleId"
          class="min-h-11 rounded-full border-0 bg-orange px-5 font-semibold text-black hover:bg-black hover:text-white disabled:opacity-70"
          @click="openForm"
        >
          Add…
        </button>
        <button
          type="button"
          :disabled="busy"
          :aria-describedby="titleId"
          class="min-h-11 rounded-full border-2 border-grey bg-white px-5 font-medium text-black hover:border-black disabled:opacity-70"
          @click="emit('dismiss')"
        >
          Not a competitor
        </button>
      </template>
      <template v-else>
        <RouterLink
          v-if="candidate.decision?.competitorId"
          :to="companyPath(candidate.decision.competitorId)"
          class="font-medium"
        >
          Open its page
        </RouterLink>
        <button
          type="button"
          :disabled="busy"
          :aria-describedby="titleId"
          class="min-h-11 rounded-full border-2 border-black bg-white px-5 font-medium text-black hover:bg-black hover:text-white disabled:opacity-70"
          @click="emit('undo')"
        >
          Undo
        </button>
      </template>
    </div>
  </li>
</template>
