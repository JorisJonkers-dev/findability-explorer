<script setup lang="ts">
import { computed, nextTick, ref, useTemplateRef } from 'vue'
import type { TermAcceptance, TermCandidate } from '../../../infrastructure/api/types.gen'
import { FAMILIES, familyLabel, LANGUAGES, languageLabel, TIERS, tierLabel } from '../../../shared/vocabulary'
import { atLabel } from '../../settings/domain/time'
import { type AcceptDraft, checkDraft, decisionLine, draftOf, evidenceLine, suggestionLine, verdictLine } from '../domain/candidates'

const props = defineProps<{ candidate: TermCandidate; busy: boolean }>()
const emit = defineEmits<{ accept: [body: TermAcceptance]; reject: []; undo: [] }>()

const titleId = computed(() => `candidate-${props.candidate.id}`)
const editing = ref(false)
const draft = ref<AcceptDraft>(draftOf(props.candidate))
const draftError = ref<string | null>(null)
const termInput = useTemplateRef<HTMLInputElement>('termInput')
const decided = computed(() => decisionLine(props.candidate))
const verdict = computed(() => verdictLine(props.candidate))
// The term the language model suggests, ready to accept in one step.
const suggested = computed(() => (props.candidate.verdict?.term ? draftOf(props.candidate) : null))

function acceptSuggested(): void {
  if (!suggested.value) return
  const check = checkDraft(suggested.value)
  if (check.kind === 'ok') {
    emit('accept', check.body)
    return
  }
  // Something about it needs a person: show it in the form.
  void openForm().then(() => {
    draftError.value = check.message
  })
}

async function openForm(): Promise<void> {
  draft.value = draftOf(props.candidate)
  draftError.value = null
  editing.value = true
  await nextTick()
  termInput.value?.focus()
}

function submit(): void {
  const check = checkDraft(draft.value)
  if (check.kind === 'invalid') {
    draftError.value = check.message
    return
  }
  emit('accept', check.body)
}
</script>

<template>
  <li class="flex flex-col gap-3 rounded-2xl bg-mist p-5 sm:p-6" :aria-labelledby="titleId">
    <div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
      <h3 :id="titleId" class="m-0 text-xl font-semibold">{{ candidate.phrase }}</h3>
      <span class="text-sm text-ink-2">Best score {{ candidate.bestScore }} · up to {{ candidate.maxPages }} pages</span>
    </div>
    <p class="m-0 text-sm text-ink-2">{{ evidenceLine(candidate) }} · last seen {{ atLabel(candidate.lastSeenAt) }}</p>
    <p class="m-0 text-sm">
      <span class="text-muted">Found at </span>{{ candidate.companies.map((c) => c.name).join(', ') }}
    </p>
    <p class="m-0 text-sm"><span class="text-muted">Suggested: </span>{{ suggestionLine(candidate) }}</p>
    <p v-if="verdict" class="m-0 text-sm">{{ verdict }}</p>
    <p v-if="candidate.status === 'pending' && suggested" class="m-0 text-sm">
      <span class="text-muted">It suggests </span>“{{ suggested.term }}”, {{ languageLabel(suggested.language) }}, {{ tierLabel(suggested.tier) }},
      {{ familyLabel(suggested.family) }}.
    </p>
    <p v-if="decided" class="m-0 text-sm font-medium">{{ decided }}</p>
    <p v-if="candidate.decision?.term" class="m-0 text-sm text-ink-2">
      In the map as “{{ candidate.decision.term.term }}”, {{ languageLabel(candidate.decision.term.language) }},
      {{ tierLabel(candidate.decision.term.tier) }}, {{ familyLabel(candidate.decision.term.family) }}.
    </p>

    <form v-if="editing" novalidate :aria-label="`Accept ${candidate.phrase}`" class="flex flex-col gap-3 rounded-xl bg-white p-4" @submit.prevent="submit">
      <div class="grid gap-3 sm:grid-cols-2">
        <label class="flex flex-col gap-1.5 text-sm font-medium">
          Term
          <input
            ref="termInput"
            v-model="draft.term"
            type="text"
            maxlength="200"
            class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base font-normal"
          />
        </label>
        <label class="flex flex-col gap-1.5 text-sm font-medium">
          Used by (comma-separated)
          <input v-model="draft.usedBy" type="text" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base font-normal" />
        </label>
        <label class="flex flex-col gap-1.5 text-sm font-medium">
          Language
          <select v-model="draft.language" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base font-normal">
            <option v-for="l in LANGUAGES" :key="l" :value="l">{{ languageLabel(l) }}</option>
          </select>
        </label>
        <label class="flex flex-col gap-1.5 text-sm font-medium">
          Family
          <select v-model="draft.family" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base font-normal">
            <option v-for="f in FAMILIES" :key="f" :value="f">{{ familyLabel(f) }}</option>
          </select>
        </label>
        <label class="flex flex-col gap-1.5 text-sm font-medium">
          Intent tier
          <select v-model="draft.tier" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base font-normal">
            <option v-for="t in TIERS" :key="t" :value="t">{{ tierLabel(t) }}</option>
          </select>
        </label>
      </div>
      <p v-if="draftError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ draftError }}</p>
      <div class="flex flex-wrap gap-3">
        <button type="submit" :disabled="busy" class="min-h-11 rounded-full border-0 bg-black px-5 font-semibold text-white hover:bg-ink-2 disabled:opacity-70">
          Add to the map
        </button>
        <button type="button" class="min-h-11 rounded-full border-2 border-grey bg-white px-5 font-medium text-black hover:border-black" @click="editing = false">
          Cancel
        </button>
      </div>
    </form>

    <div v-else class="flex flex-wrap items-center gap-3">
      <template v-if="candidate.status === 'pending'">
        <button
          v-if="suggested"
          type="button"
          :disabled="busy"
          :aria-describedby="titleId"
          class="min-h-11 rounded-full border-0 bg-black px-5 font-semibold text-white hover:bg-ink-2 disabled:opacity-70"
          @click="acceptSuggested"
        >
          Accept as suggested
        </button>
        <button
          type="button"
          :disabled="busy"
          :aria-describedby="titleId"
          class="min-h-11 rounded-full border-0 bg-orange px-5 font-semibold text-black hover:bg-black hover:text-white disabled:opacity-70"
          @click="openForm"
        >
          Accept…
        </button>
        <button
          type="button"
          :disabled="busy"
          :aria-describedby="titleId"
          class="min-h-11 rounded-full border-2 border-grey bg-white px-5 font-medium text-black hover:border-black disabled:opacity-70"
          @click="emit('reject')"
        >
          Reject
        </button>
      </template>
      <button
        v-else
        type="button"
        :disabled="busy"
        :aria-describedby="titleId"
        class="min-h-11 rounded-full border-2 border-black bg-white px-5 font-medium text-black hover:bg-black hover:text-white disabled:opacity-70"
        @click="emit('undo')"
      >
        Undo
      </button>
    </div>
  </li>
</template>
